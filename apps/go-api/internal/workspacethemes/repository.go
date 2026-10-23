package workspacethemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("workspace theme not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace themes database is required")
	}
	return &Repository{db}, nil
}

const themeJSON = `json_build_object('id',t.id,'workspace',t.workspace_id,'actor',t.actor_id,'created_at',t.created_at,'updated_at',t.updated_at,'created_by',t.created_by_id,'updated_by',t.updated_by_id,'deleted_at',t.deleted_at,'name',t.name,'colors',t.colors)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}
func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(theme ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+themeJSON+` AS theme,t.created_at FROM workspace_themes t JOIN workspaces w ON w.id=t.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND t.deleted_at IS NULL AND w.deleted_at IS NULL) x`, slug, user).Scan(&b)
	return b, err
}
func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `SELECT `+themeJSON+` FROM workspace_themes t JOIN workspaces w ON w.id=t.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id WHERE t.id=$1 AND w.slug=$2 AND wm.member_id=$3 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND t.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, user).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug, name string, colors json.RawMessage) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `INSERT INTO workspace_themes AS t(workspace_id,actor_id,name,colors,created_by_id) SELECT w.id,$1,$2,$3,$1 FROM workspaces w WHERE w.slug=$4 AND w.deleted_at IS NULL RETURNING `+themeJSON, user, name, colors, slug).Scan(&b)
	return b, err
}

func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, name *string, colors json.RawMessage) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `UPDATE workspace_themes t SET name=COALESCE($4,name), colors=COALESCE($5,colors), updated_at=NOW(),updated_by_id=$3 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE t.id=$1 AND t.workspace_id=w.id AND w.slug=$2 AND wm.member_id=$3 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND t.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING `+themeJSON, id, slug, user, name, colors).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE workspace_themes t SET deleted_at=NOW(),updated_at=NOW(),updated_by_id=$1 FROM workspaces w,workspace_members wm WHERE t.id=$2 AND t.workspace_id=w.id AND w.slug=$3 AND wm.workspace_id=w.id AND wm.member_id=$1 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND t.deleted_at IS NULL AND w.deleted_at IS NULL`, user, id, slug)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
