// Package litelists contains persistence operations for the read-only lite-list APIs.
package litelists

import (
	"context"
	"encoding/json"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

type Page struct {
	Results json.RawMessage
	Count   int
}

// ListOptions controls the database portion of cursor/page handling. Offset is
// deliberately supported as well: the Django cursor paginator resolves cursors
// before the repository is called.
type ListOptions struct {
	Limit           int
	Offset          int
	IncludeArchived bool
	OrderBy         string
}

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("lite lists database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) page(ctx context.Context, query string, args ...any) (Page, error) {
	var results []byte
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&results, &count); err != nil {
		return Page{}, err
	}
	if len(results) == 0 {
		results = []byte("[]")
	}
	return Page{Results: results, Count: count}, nil
}

func paging(o ListOptions, defaultOrder string, allowed map[string]string) (string, []any) {
	order := defaultOrder
	if v, ok := allowed[o.OrderBy]; ok {
		order = v
	}
	limit := o.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := o.Offset
	if offset < 0 {
		offset = 0
	}
	return fmt.Sprintf(" ORDER BY %s LIMIT %d OFFSET %d", order, limit, offset), nil
}

var projectOrder = map[string]string{"-created_at": "p.created_at DESC", "created_at": "p.created_at ASC", "name": "p.name ASC", "-name": "p.name DESC"}
var cycleOrder = map[string]string{"-created_at": "c.created_at DESC", "created_at": "c.created_at ASC", "name": "c.name ASC", "-name": "c.name DESC"}
var moduleOrder = map[string]string{"-created_at": "m.created_at DESC", "created_at": "m.created_at ASC", "name": "m.name ASC", "-name": "m.name DESC"}

func (r *Repository) ListProjects(ctx context.Context, user uuid.UUID, slug string, o ListOptions) (Page, error) {
	p, _ := paging(o, "p.created_at DESC", projectOrder)
	archived := " AND p.archived_at IS NULL"
	if o.IncludeArchived {
		archived = ""
	}
	q := `SELECT COALESCE(json_agg(x.item), '[]'::json), COALESCE(max(x.total),0) FROM (
 SELECT json_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,
 'icon_prop',p.icon_prop,'emoji',p.emoji,'description',p.description,'cover_image_url',
 CASE WHEN p.cover_image_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||p.cover_image_asset_id::text||'/' ELSE NULLIF(p.cover_image,'') END,
 'archived_at',p.archived_at) item, p.created_at
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE w.slug=$2 AND w.deleted_at IS NULL AND p.deleted_at IS NULL` + archived +
		` AND (p.network=2 OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL))` +
		p + `) x`
	return r.page(ctx, q, user, slug)
}

func (r *Repository) ListCycles(ctx context.Context, slug string, projectID uuid.UUID, o ListOptions) (Page, error) {
	p, _ := paging(o, "c.created_at DESC", cycleOrder)
	q := `SELECT COALESCE(json_agg(x.item), '[]'::json), COALESCE(max(x.total),0) FROM (
 SELECT json_build_object('id',c.id,'created_at',c.created_at,'updated_at',c.updated_at,'created_by',c.created_by_id,'updated_by',c.updated_by_id,'deleted_at',c.deleted_at,'name',c.name,'description',c.description,'start_date',c.start_date,'end_date',c.end_date,'owned_by',c.owned_by_id,'project',c.project_id,'workspace',c.workspace_id,'external_source',c.external_source,'external_id',c.external_id,'logo_props',c.logo_props,'progress_snapshot',c.progress_snapshot,'sort_order',c.sort_order,'timezone',c.timezone,'version',c.version,'view_props',c.view_props,'archived_at',c.archived_at) item,c.created_at,count(*) OVER () total
 FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE w.slug=$1 AND c.project_id=$2 AND w.deleted_at IS NULL AND c.deleted_at IS NULL AND c.archived_at IS NULL` + p + `) x`
	return r.page(ctx, q, slug, projectID)
}

func (r *Repository) ListModules(ctx context.Context, slug string, projectID uuid.UUID, o ListOptions) (Page, error) {
	p, _ := paging(o, "m.created_at DESC", moduleOrder)
	q := `SELECT COALESCE(json_agg(x.item), '[]'::json), COALESCE(max(x.total),0) FROM (
 SELECT json_build_object('id',m.id,'created_at',m.created_at,'updated_at',m.updated_at,'created_by',m.created_by_id,'updated_by',m.updated_by_id,'deleted_at',m.deleted_at,'name',m.name,'description',m.description,'description_html',m.description_html,'description_text',m.description_text,'start_date',m.start_date,'target_date',m.target_date,'status',m.status,'lead',m.lead_id,'project',m.project_id,'workspace',m.workspace_id,'external_source',m.external_source,'external_id',m.external_id,'logo_props',m.logo_props,'view_props',m.view_props,'sort_order',m.sort_order,'archived_at',m.archived_at,'members',COALESCE((SELECT json_agg(mm.member_id) FROM module_members mm WHERE mm.module_id=m.id AND mm.deleted_at IS NULL),'[]'::json)) item,m.created_at,count(*) OVER () total
 FROM modules m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.project_id=$2 AND w.deleted_at IS NULL AND m.deleted_at IS NULL AND m.archived_at IS NULL` + p + `) x`
	return r.page(ctx, q, slug, projectID)
}

const memberJSON = `json_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'avatar',u.avatar,'avatar_url',CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' ELSE NULLIF(u.avatar,'') END,'display_name',u.display_name,'role',m.role,'is_active',m.is_active,'is_bot',u.is_bot)`

func (r *Repository) ListWorkspaceMembers(ctx context.Context, slug string, o ListOptions) (Page, error) {
	p, _ := paging(o, "m.created_at DESC", map[string]string{"-created_at": "m.created_at DESC", "created_at": "m.created_at ASC"})
	q := `SELECT COALESCE(json_agg(x.item), '[]'::json), count(*) FROM (SELECT ` + memberJSON + ` item,m.created_at FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id JOIN users u ON u.id=m.member_id WHERE w.slug=$1 AND w.deleted_at IS NULL AND m.deleted_at IS NULL` + p + `) x`
	return r.page(ctx, q, slug)
}

func (r *Repository) ListProjectMembers(ctx context.Context, slug string, projectID uuid.UUID, o ListOptions) (Page, error) {
	p, _ := paging(o, "m.created_at DESC", map[string]string{"-created_at": "m.created_at DESC", "created_at": "m.created_at ASC"})
	q := `SELECT COALESCE(json_agg(x.item), '[]'::json), count(*) FROM (SELECT ` + memberJSON + ` item,m.created_at FROM project_members m JOIN projects p0 ON p0.id=m.project_id JOIN workspaces w ON w.id=p0.workspace_id JOIN users u ON u.id=m.member_id WHERE w.slug=$1 AND m.project_id=$2 AND w.deleted_at IS NULL AND p0.deleted_at IS NULL AND m.deleted_at IS NULL` + p + `) x`
	return r.page(ctx, q, slug, projectID)
}
