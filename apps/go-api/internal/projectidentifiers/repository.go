package projectidentifiers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"uuid"
)

var (
	ErrNameRequired  = errors.New("project identifier name is required")
	ErrProjectExists = errors.New("cannot delete an identifier of an existing project")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("project identifiers database is required")
	}
	return &Repository{db: db}, nil
}

func normalizeName(name string) (string, error) {
	name = cases.Upper(language.Und).String(strings.TrimSpace(name))
	if name == "" {
		return "", ErrNameRequired
	}
	return name, nil
}

// Authorized matches ProjectIdentifierEndpoint's workspace-level permission:
// active workspace members and administrators may inspect or remove names.
func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM workspace_members wm
JOIN workspaces w ON w.id=wm.workspace_id
WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20)
  AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL
)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug, name string) (json.RawMessage, error) {
	name, err := normalizeName(name)
	if err != nil {
		return nil, err
	}

	var body []byte
	err = r.db.QueryRow(ctx, `SELECT json_build_object(
  'exists', count(*)::int,
  'identifiers', COALESCE(json_agg(json_build_object('id',pi.id,'name',pi.name,'project',pi.project_id) ORDER BY pi.created_at DESC), '[]'::json)
)
FROM project_identifiers pi
JOIN workspaces w ON w.id=pi.workspace_id
WHERE pi.name=$1 AND w.slug=$2
  AND pi.deleted_at IS NULL AND w.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM workspace_members wm
              WHERE wm.workspace_id=w.id AND wm.member_id=$3
                AND wm.role IN (15,20) AND wm.is_active=true
                AND wm.deleted_at IS NULL)`, name, slug, user).Scan(&body)
	return body, err
}

func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug, name string) error {
	name, err := normalizeName(name)
	if err != nil {
		return err
	}

	var projectExists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM projects p
JOIN workspaces w ON w.id=p.workspace_id
WHERE p.identifier=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
)`, name, slug).Scan(&projectExists)
	if err != nil {
		return err
	}
	if projectExists {
		return ErrProjectExists
	}

	// Django's queryset delete is a soft delete and intentionally does not
	// require a matching row; DELETE therefore remains idempotent.
	_, err = r.db.Exec(ctx, `UPDATE project_identifiers pi
SET deleted_at=NOW()
FROM workspaces w
JOIN workspace_members wm ON wm.workspace_id=w.id
WHERE pi.workspace_id=w.id AND pi.name=$1 AND w.slug=$2 AND wm.member_id=$3
  AND wm.role IN (15,20) AND pi.deleted_at IS NULL
  AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL`, name, slug, user)
	return err
}
