package projectmemberpreferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type HTTPError struct {
	Status int
	Body   map[string]any
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("project member preferences HTTP %d", e.Status)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("project member preferences database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Authorize(ctx context.Context, actor, slug, projectID string) error {
	var allowed bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM project_members pm
		JOIN projects p ON p.id=pm.project_id
		JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		AND pm.is_active AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL
		AND pm.role IN (5,15,20)
	)`, actor, projectID, slug).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return &HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}

func (r *Repository) Patch(ctx context.Context, actor, slug, projectID, memberID string, preferences map[string]any) (map[string]any, error) {
	var currentRaw []byte
	err := r.db.QueryRow(ctx, `SELECT pm.preferences FROM project_members pm
		JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.project_id=$1 AND pm.member_id=$2 AND w.slug=$3
		AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, projectID, memberID, slug).Scan(&currentRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	if err != nil {
		return nil, err
	}
	current := map[string]any{}
	if len(currentRaw) > 0 {
		if unmarshalErr := json.Unmarshal(currentRaw, &current); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}
	for key, value := range preferences {
		current[key] = value
	}
	encoded, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `UPDATE project_members SET preferences=$1::jsonb,updated_at=clock_timestamp(),updated_by_id=$2
		WHERE project_id=$3 AND member_id=$4 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$5)`, encoded, actor, projectID, memberID, slug)
	if err != nil {
		return nil, err
	}
	return map[string]any{"preferences": current}, nil
}

// Lookup preserves the source object's lookup before request-body evaluation.
func (r *Repository) Lookup(ctx context.Context, slug, projectID, memberID string) error {
	var present bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.project_id=$1 AND pm.member_id=$2 AND w.slug=$3 AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, projectID, memberID, slug).Scan(&present)
	if err != nil {
		return err
	}
	if !present {
		return &HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	return nil
}
