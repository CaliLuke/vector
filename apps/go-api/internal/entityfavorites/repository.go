// Package entityfavorites contains the persistence operations used by the
// entity-specific favorite endpoints.  These endpoints deliberately return
// no representation on writes (the Django contract is HTTP 204).
package entityfavorites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

var ErrNotFound = errors.New("entity favorite not found")
var ErrAlreadyExists = errors.New("entity favorite already exists")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

// Authorized reports whether the user can access the workspace (and, when
// supplied, the project-scoped favorite collection).
func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string, projectID *uuid.UUID) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.slug=$2 AND w.deleted_at IS NULL`
	args := []any{user, slug}
	if projectID != nil {
		query = `SELECT EXISTS (SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE pm.member_id=$1 AND pm.project_id=$3 AND pm.is_active=true AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND w.slug=$2 AND w.deleted_at IS NULL`
		args = append(args, *projectID)
	}
	query += `)`
	var ok bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("entity favorites database is required")
	}
	return &Repository{db: db}, nil
}

const entityData = `CASE f.entity_type
WHEN 'project' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props) FROM projects e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'page' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',(SELECT pp.project_id FROM project_pages pp JOIN projects ep ON ep.id=pp.project_id AND ep.deleted_at IS NULL WHERE pp.page_id=e.id AND pp.deleted_at IS NULL ORDER BY pp.created_at DESC LIMIT 1)) FROM pages e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'cycle' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM cycles e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'module' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM modules e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'view' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM issue_views e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
ELSE NULL END`

const favoriteJSON = `json_build_object('id',f.id,'entity_type',f.entity_type,'entity_identifier',f.entity_identifier,'entity_data',` + entityData + `,'name',f.name,'is_folder',f.is_folder,'sequence',f.sequence,'parent',f.parent_id,'workspace_id',f.workspace_id,'project_id',f.project_id)`

// List returns the UserFavorite serializer shape, ordered newest first.
func (r *Repository) List(ctx context.Context, user uuid.UUID, slug, entityType string, projectID *uuid.UUID) (json.RawMessage, error) {
	var body []byte
	query := `SELECT COALESCE(json_agg(favorite ORDER BY created_at DESC),'[]'::json) FROM (SELECT ` + favoriteJSON + ` AS favorite,f.created_at FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.user_id=$1 AND w.slug=$2 AND f.entity_type=$3 AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND (f.project_id IS NULL OR EXISTS (SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id WHERE pm.project_id=f.project_id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL AND p.deleted_at IS NULL))`
	args := []any{user, slug, entityType}
	if projectID != nil {
		query += ` AND f.project_id=$4`
		args = append(args, *projectID)
	}
	query += `) x`
	if err := r.db.QueryRow(ctx, query, args...).Scan(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug, entityType string, entityID, projectID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `INSERT INTO user_favorites (id,workspace_id,user_id,entity_type,entity_identifier,project_id,sequence,created_by_id,created_at,updated_at) SELECT $6,w.id,$1,$3,$4,$5,CASE WHEN MAX(previous.sequence) IS NULL THEN 65535 ELSE MAX(previous.sequence)+10000 END,$1,NOW(),NOW() FROM workspaces w LEFT JOIN user_favorites previous ON previous.workspace_id=w.id AND previous.deleted_at IS NULL WHERE w.slug=$2 AND w.deleted_at IS NULL GROUP BY w.id`, user, slug, entityType, entityID, projectID, uuid.NewV7())
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug, entityType string, entityID, projectID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_favorites f USING workspaces w WHERE f.workspace_id=w.id AND f.user_id=$1 AND w.slug=$2 AND f.entity_type=$3 AND f.entity_identifier=$4 AND f.project_id=$5 AND f.deleted_at IS NULL AND w.deleted_at IS NULL`, user, slug, entityType, entityID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListProjects(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	return r.List(ctx, user, slug, "project", nil)
}
func (r *Repository) CreateProject(ctx context.Context, user uuid.UUID, slug string, projectID uuid.UUID) error {
	return r.Create(ctx, user, slug, "project", projectID, projectID)
}
func (r *Repository) DeleteProject(ctx context.Context, user uuid.UUID, slug string, projectID uuid.UUID) error {
	return r.Delete(ctx, user, slug, "project", projectID, projectID)
}

func (r *Repository) ListCycles(ctx context.Context, user uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	return r.List(ctx, user, slug, "cycle", &projectID)
}
func (r *Repository) CreateCycle(ctx context.Context, user uuid.UUID, slug string, projectID, cycleID uuid.UUID) error {
	return r.Create(ctx, user, slug, "cycle", cycleID, projectID)
}
func (r *Repository) DeleteCycle(ctx context.Context, user uuid.UUID, slug string, projectID, cycleID uuid.UUID) error {
	return r.Delete(ctx, user, slug, "cycle", cycleID, projectID)
}

func (r *Repository) ListModules(ctx context.Context, user uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	return r.List(ctx, user, slug, "module", &projectID)
}
func (r *Repository) CreateModule(ctx context.Context, user uuid.UUID, slug string, projectID, moduleID uuid.UUID) error {
	return r.Create(ctx, user, slug, "module", moduleID, projectID)
}
func (r *Repository) DeleteModule(ctx context.Context, user uuid.UUID, slug string, projectID, moduleID uuid.UUID) error {
	return r.Delete(ctx, user, slug, "module", moduleID, projectID)
}

func (r *Repository) ListViews(ctx context.Context, user uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	return r.List(ctx, user, slug, "view", &projectID)
}
func (r *Repository) CreateView(ctx context.Context, user uuid.UUID, slug string, projectID, viewID uuid.UUID) error {
	return r.Create(ctx, user, slug, "view", viewID, projectID)
}
func (r *Repository) DeleteView(ctx context.Context, user uuid.UUID, slug string, projectID, viewID uuid.UUID) error {
	return r.Delete(ctx, user, slug, "view", viewID, projectID)
}

func (r *Repository) CreatePage(ctx context.Context, user uuid.UUID, slug string, projectID, pageID uuid.UUID) error {
	return r.Create(ctx, user, slug, "page", pageID, projectID)
}
func (r *Repository) DeletePage(ctx context.Context, user uuid.UUID, slug string, projectID, pageID uuid.UUID) error {
	return r.Delete(ctx, user, slug, "page", pageID, projectID)
}
