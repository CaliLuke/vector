package analyticviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

var ErrNotFound = errors.New("analytic view not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("analytic views database is required")
	}
	return &Repository{db: db}, nil
}

const viewJSON = `json_build_object('id',a.id,'created_at',a.created_at,'updated_at',a.updated_at,'created_by',a.created_by_id,'updated_by',a.updated_by_id,'deleted_at',a.deleted_at,'workspace',a.workspace_id,'name',a.name,'description',a.description,'query',a.query,'query_dict',a.query_dict)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}
func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(v ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+viewJSON+` AS v,a.created_at FROM analytic_views a JOIN workspaces w ON w.id=a.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND a.deleted_at IS NULL AND w.deleted_at IS NULL) x`, slug, user).Scan(&b)
	return b, err
}
func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `SELECT `+viewJSON+` FROM analytic_views a JOIN workspaces w ON w.id=a.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id WHERE a.id=$1 AND w.slug=$2 AND wm.member_id=$3 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND a.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, user).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug, name, description string, query, queryDict json.RawMessage) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `INSERT INTO analytic_views AS a(workspace_id,name,description,query,query_dict,created_by_id) SELECT w.id,$1,$2,$3::jsonb,$4::jsonb,$5 FROM workspaces w WHERE w.slug=$6 AND w.deleted_at IS NULL RETURNING `+viewJSON, name, description, query, queryDict, user, slug).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, name *string, desc *string, query json.RawMessage, queryPresent bool) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `UPDATE analytic_views a SET name=COALESCE($4,a.name),description=COALESCE($5,a.description),query=CASE WHEN $6 THEN $7::jsonb ELSE a.query END,updated_at=NOW(),updated_by_id=$3 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE a.id=$1 AND a.workspace_id=w.id AND w.slug=$2 AND wm.member_id=$3 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND a.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING `+viewJSON, id, slug, user, name, desc, queryPresent, query).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE analytic_views a SET deleted_at=NOW(),updated_at=NOW(),updated_by_id=$3 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE a.id=$1 AND a.workspace_id=w.id AND w.slug=$2 AND wm.member_id=$3 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND a.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, user)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
