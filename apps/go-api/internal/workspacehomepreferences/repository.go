package workspacehomepreferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"uuid"
)

var ErrNotFound = errors.New("workspace home preference not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace home preferences database is required")
	}
	return &Repository{db}, nil
}

const prefJSON = `json_build_object('key',p.key,'is_enabled',p.is_enabled,'sort_order',p.sort_order)`
const prefListJSON = `json_build_object('key',p.key,'is_enabled',p.is_enabled,'config',p.config,'sort_order',p.sort_order)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}
func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `
WITH w AS (
	SELECT id FROM workspaces WHERE slug=$2 AND deleted_at IS NULL
), defaults(key,sort_order) AS (
	VALUES ('quick_links',999.0),('recents',998.0),('my_stickies',997.0)
), inserted AS (
	INSERT INTO workspace_home_preferences (
		id,workspace_id,user_id,key,is_enabled,config,sort_order,
		created_at,updated_at,created_by_id,updated_by_id,deleted_at
	)
	SELECT gen_random_uuid(),w.id,$1,d.key,true,'{}'::jsonb,d.sort_order,
		NOW(),NOW(),NULL,NULL,NULL
	FROM w CROSS JOIN defaults d
	ON CONFLICT (workspace_id,user_id,key) WHERE deleted_at IS NULL
	DO UPDATE SET key=workspace_home_preferences.key
	RETURNING id,workspace_id,user_id,key,is_enabled,config,sort_order,created_at
), preferences AS (
	SELECT p.id,p.workspace_id,p.user_id,p.key,p.is_enabled,p.config,p.sort_order,p.created_at
	FROM workspace_home_preferences p JOIN w ON w.id=p.workspace_id
	WHERE p.user_id=$1 AND p.deleted_at IS NULL
	  AND NOT EXISTS (
		SELECT 1 FROM inserted i
		WHERE i.workspace_id=p.workspace_id AND i.user_id=p.user_id AND i.key=p.key
	  )
	UNION ALL
	SELECT id,workspace_id,user_id,key,is_enabled,config,sort_order,created_at FROM inserted
)
SELECT COALESCE(json_agg(json_build_object('key',p.key,'is_enabled',p.is_enabled,'config',p.config,'sort_order',p.sort_order)
	ORDER BY p.created_at DESC,p.sort_order ASC,p.key ASC),'[]'::json)
FROM preferences p`, user, slug).Scan(&b)
	return b, err
}
func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug, key string) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `SELECT `+prefJSON+` FROM workspace_home_preferences p JOIN workspaces w ON w.id=p.workspace_id WHERE p.key=$3 AND w.slug=$2 AND p.user_id=$1 AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, user, slug, key).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug, key string, newKey *string, enabled *bool, sort *float64) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `UPDATE workspace_home_preferences p SET key=COALESCE($4,p.key),is_enabled=COALESCE($5,p.is_enabled),sort_order=COALESCE($6,p.sort_order),updated_at=NOW(),updated_by_id=$1 FROM workspaces w WHERE p.workspace_id=w.id AND p.key=$3 AND w.slug=$2 AND p.user_id=$1 AND p.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING `+prefJSON, user, slug, key, newKey, enabled, sort).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
