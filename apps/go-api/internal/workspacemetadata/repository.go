package workspacemetadata

import (
	"context"
	"encoding/json"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
)

// Database is the subset of pgx used by this repository.
type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace metadata database is required")
	}
	return &Repository{db: db}, nil
}

// Authorized implements WorkspaceViewerPermission and WorkspaceEntityPermission
// for their safe (GET) methods: every active workspace member, including guests,
// may read these collections.
func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) ListLabels(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(label ORDER BY created_at DESC),'[]'::json) FROM (SELECT json_build_object('parent',l.parent_id,'name',l.name,'color',l.color,'id',l.id,'project_id',l.project_id,'workspace_id',l.workspace_id,'sort_order',l.sort_order) AS label,l.created_at FROM labels l JOIN workspaces w ON w.id=l.workspace_id JOIN projects p ON p.id=l.project_id WHERE w.slug=$1 AND w.deleted_at IS NULL AND l.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL)) x`, slug, user).Scan(&body)
	return body, err
}

func (r *Repository) ListStates(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `WITH visible AS (SELECT s.id,s.project_id,s.workspace_id,s.name,s.color,s.group AS group_name,s."default" AS default_value,s.description,s.sequence,row_number() OVER (PARTITION BY s.group ORDER BY s.sequence,s.id)::double precision / count(*) OVER (PARTITION BY s.group) AS item_order FROM states s JOIN workspaces w ON w.id=s.workspace_id JOIN projects p ON p.id=s.project_id WHERE w.slug=$1 AND w.deleted_at IS NULL AND s.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND s.is_triage=false AND s.group <> 'triage' AND EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL)) SELECT COALESCE(json_agg(json_build_object('id',id,'project_id',project_id,'workspace_id',workspace_id,'name',name,'color',color,'group',group_name,'default',default_value,'description',description,'sequence',sequence,'order',item_order) ORDER BY sequence,id),'[]'::json) FROM visible`, slug, user).Scan(&body)
	return body, err
}

// Labels and States are concise aliases for callers that expose the two
// collection names directly.
func (r *Repository) Labels(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	return r.ListLabels(ctx, user, slug)
}

func (r *Repository) States(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	return r.ListStates(ctx, user, slug)
}
