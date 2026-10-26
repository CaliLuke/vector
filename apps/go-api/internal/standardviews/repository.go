// Package standardviews contains the persistence contract for IssueView.
//
// The package deliberately returns the serializer-shaped JSON produced by
// PostgreSQL. This keeps the adapter independent of generated Loom types and
// lets the migration compare the complete DRF representation.
package standardviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound  = errors.New("view not found")
	ErrForbidden = errors.New("view permission denied")
	ErrLocked    = errors.New("view is locked")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("standard views database is required")
	}
	return &Repository{db: db}, nil
}

// ViewInput contains writable IssueViewSerializer fields. Unknown fields are
// ignored by Django's serializer; callers should validate those fields before
// constructing this value.
type ViewInput struct {
	Name              *string         `json:"name,omitempty"`
	Description       *string         `json:"description,omitempty"`
	Filters           json.RawMessage `json:"filters,omitempty"`
	DisplayFilters    json.RawMessage `json:"display_filters,omitempty"`
	DisplayProperties json.RawMessage `json:"display_properties,omitempty"`
	RichFilters       json.RawMessage `json:"rich_filters,omitempty"`
	LogoProps         json.RawMessage `json:"logo_props,omitempty"`
	ArchivedAt        json.RawMessage `json:"archived_at,omitempty"`
}

const viewJSON = `jsonb_build_object('id',v.id,'created_at',v.created_at,'updated_at',v.updated_at,'deleted_at',v.deleted_at,'name',v.name,'description',v.description,'query',v.query,'filters',v.filters,'display_filters',v.display_filters,'display_properties',v.display_properties,'rich_filters',v.rich_filters,'access',v.access,'sort_order',v.sort_order,'logo_props',v.logo_props,'owned_by',v.owned_by_id,'is_locked',v.is_locked,'archived_at',v.archived_at,'created_by',v.created_by_id,'updated_by',v.updated_by_id,'workspace',v.workspace_id,'project',v.project_id)`
const viewJSONFavorite = viewJSON + ` || jsonb_build_object('is_favorite',COALESCE(fav.is_favorite,false))`

func scanJSON(row pgx.Row) (json.RawMessage, error) {
	var body []byte
	if err := row.Scan(&body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return body, nil
}

func (r *Repository) AuthorizedWorkspace(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE w.slug=$2 AND w.deleted_at IS NULL AND wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) AuthorizedProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id WHERE p.id=$3 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL)`, user, slug, project).Scan(&ok)
	return ok, err
}

const favoriteJoinProject = `LEFT JOIN LATERAL (SELECT true AS is_favorite FROM user_favorites f WHERE f.user_id=$1 AND f.entity_type='view' AND f.entity_identifier=v.id AND f.project_id=v.project_id AND f.workspace_id=v.workspace_id AND f.deleted_at IS NULL LIMIT 1) fav ON true`
const favoriteJoinWorkspace = `LEFT JOIN LATERAL (SELECT true AS is_favorite FROM user_favorites f WHERE f.user_id=$1 AND f.entity_type='view' AND f.entity_identifier=v.id AND f.project_id IS NULL AND f.workspace_id=v.workspace_id AND f.deleted_at IS NULL LIMIT 1) fav ON true`

func (r *Repository) ListProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) (json.RawMessage, error) {
	q := `SELECT COALESCE(json_agg(item ORDER BY is_favorite DESC,name ASC),'[]'::json) FROM (SELECT ` + viewJSONFavorite + ` AS item,COALESCE(fav.is_favorite,false) AS is_favorite,v.name FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id JOIN projects p ON p.id=v.project_id JOIN project_members pm ON pm.project_id=p.id ` + favoriteJoinProject + ` WHERE w.slug=$2 AND v.project_id=$3 AND p.archived_at IS NULL AND v.deleted_at IS NULL AND w.deleted_at IS NULL AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL AND (v.owned_by_id=$1 OR v.access=1) AND (pm.role<>5 OR p.guest_view_all_features OR v.owned_by_id=$1)) x`
	var body []byte
	err := r.db.QueryRow(ctx, q, user, slug, project).Scan(&body)
	return body, err
}

func (r *Repository) GetProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID) (json.RawMessage, error) {
	q := `SELECT ` + viewJSONFavorite + ` FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id JOIN projects p ON p.id=v.project_id JOIN project_members pm ON pm.project_id=p.id ` + favoriteJoinProject + ` WHERE v.id=$4 AND w.slug=$2 AND v.project_id=$3 AND p.archived_at IS NULL AND v.deleted_at IS NULL AND w.deleted_at IS NULL AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL AND (v.owned_by_id=$1 OR v.access=1) AND (pm.role<>5 OR p.guest_view_all_features OR v.owned_by_id=$1)`
	return scanJSON(r.db.QueryRow(ctx, q, user, slug, project, id))
}

func (r *Repository) ListWorkspace(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	q := `SELECT COALESCE(json_agg(item ORDER BY created_at DESC),'[]'::json) FROM (SELECT ` + viewJSON + ` AS item,v.created_at FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id ` + favoriteJoinWorkspace + ` WHERE w.slug=$2 AND v.project_id IS NULL AND v.deleted_at IS NULL AND w.deleted_at IS NULL AND wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL AND (v.owned_by_id=$1 OR v.access=1) AND (wm.role<>5 OR v.owned_by_id=$1)) x`
	var body []byte
	err := r.db.QueryRow(ctx, q, user, slug).Scan(&body)
	return body, err
}

func (r *Repository) GetWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	q := `SELECT ` + viewJSON + ` FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id JOIN workspace_members wm ON wm.workspace_id=w.id ` + favoriteJoinWorkspace + ` WHERE v.id=$3 AND w.slug=$2 AND v.project_id IS NULL AND v.deleted_at IS NULL AND w.deleted_at IS NULL AND wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL AND (v.owned_by_id=$1 OR v.access=1) AND (wm.role<>5 OR v.owned_by_id=$1)`
	return scanJSON(r.db.QueryRow(ctx, q, user, slug, id))
}

// RecordRecentVisit mirrors recent_visited_task. It intentionally does not
// make a view request fail when the best-effort activity write fails.
func (r *Repository) RecordRecentVisit(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `WITH workspace AS (SELECT id FROM workspaces WHERE slug=$2 AND deleted_at IS NULL), existing AS (SELECT v.id FROM user_recent_visits v JOIN workspace w ON w.id=v.workspace_id WHERE v.entity_name='view' AND v.entity_identifier=$3 AND v.user_id=$1 AND v.project_id IS NOT DISTINCT FROM $4 AND v.deleted_at IS NULL LIMIT 1), evicted AS (UPDATE user_recent_visits v SET deleted_at=NOW() WHERE v.id=(SELECT v2.id FROM user_recent_visits v2 JOIN workspace w ON w.id=v2.workspace_id WHERE v2.user_id=$1 AND v2.deleted_at IS NULL ORDER BY v2.created_at ASC OFFSET 19 LIMIT 1) AND NOT EXISTS (SELECT 1 FROM existing)) INSERT INTO user_recent_visits (workspace_id,project_id,entity_name,entity_identifier,user_id,created_by_id,updated_by_id,created_at,updated_at,visited_at) SELECT w.id,$4,'view',$3,$1,$1,$1,NOW(),NOW(),NOW() FROM workspace w WHERE NOT EXISTS (SELECT 1 FROM existing)`, user, slug, id, project)
	return err
}

func (r *Repository) CreateProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID, in ViewInput) (json.RawMessage, error) {
	return r.create(ctx, user, slug, &project, in)
}

func (r *Repository) CreateWorkspace(ctx context.Context, user uuid.UUID, slug string, in ViewInput) (json.RawMessage, error) {
	return r.create(ctx, user, slug, nil, in)
}

func (r *Repository) create(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, in ViewInput) (json.RawMessage, error) {
	input, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	query, err := normalizeFilters(in.Filters, "POST")
	if err != nil {
		return nil, err
	}
	q := `WITH input AS (SELECT $1::jsonb body), target AS (SELECT w.id workspace_id,$3::uuid project_id FROM workspaces w WHERE w.slug=$2 AND w.deleted_at IS NULL), inserted AS (INSERT INTO issue_views (id,workspace_id,project_id,name,description,query,filters,display_filters,display_properties,rich_filters,logo_props,owned_by_id,created_by_id,created_at,updated_at,is_locked,access,sort_order) SELECT $6::uuid,t.workspace_id,t.project_id,body->>'name',COALESCE(body->>'description',''),$5::jsonb,COALESCE(body->'filters','{}'::jsonb),COALESCE(body->'display_filters','{"group_by":null,"order_by":"-created_at","type":null,"sub_issue":true,"show_empty_groups":true,"layout":"list","calendar_date_range":""}'::jsonb),COALESCE(body->'display_properties','{"assignee":true,"attachment_count":true,"created_on":true,"due_date":true,"estimate":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}'::jsonb),COALESCE(body->'rich_filters','{}'::jsonb),COALESCE(body->'logo_props','{}'::jsonb),$4,$4,NOW(),NOW(),false,1,COALESCE((SELECT MAX(x.sort_order)+10000 FROM issue_views x WHERE x.workspace_id=t.workspace_id AND x.project_id IS NOT DISTINCT FROM t.project_id AND x.deleted_at IS NULL),65535) FROM input,target t RETURNING *) SELECT ` + viewJSON + ` FROM inserted v LEFT JOIN LATERAL (SELECT false AS is_favorite) fav ON true`
	var projectArg any
	if project != nil {
		projectArg = *project
	}
	newID := uuid.NewV7()
	return scanJSON(r.db.QueryRow(ctx, q, input, slug, projectArg, user, query, newID))
}

func (r *Repository) UpdateProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	return r.update(ctx, user, slug, &project, id, in, false)
}
func (r *Repository) UpdateWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	return r.update(ctx, user, slug, nil, id, in, false)
}

// PatchProject and PatchWorkspace implement the explicit DRF partial-update
// actions. Unlike PUT, PATCH is restricted to the view owner and rejects
// locked views.
func (r *Repository) PatchProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	return r.update(ctx, user, slug, &project, id, in, true)
}
func (r *Repository) PatchWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	return r.update(ctx, user, slug, nil, id, in, true)
}
func (r *Repository) update(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, id uuid.UUID, in ViewInput, ownerOnly bool) (json.RawMessage, error) {
	input, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	query, err := normalizeFilters(in.Filters, "PATCH")
	if err != nil {
		return nil, err
	}
	projectClause := "v.project_id IS NULL"
	args := []any{input, user, slug, id, query}
	if project != nil {
		projectClause = "v.project_id=$6"
		args = append(args, *project)
	}
	permission := "v.access=1 OR v.owned_by_id=$2"
	if ownerOnly {
		permission = "v.owned_by_id=$2 AND v.is_locked=false"
	}
	q := `WITH input AS (SELECT $1::jsonb body), changed AS (UPDATE issue_views v SET name=CASE WHEN (SELECT body ? 'name' FROM input) THEN (SELECT body->>'name' FROM input) ELSE v.name END,description=CASE WHEN (SELECT body ? 'description' FROM input) THEN (SELECT body->>'description' FROM input) ELSE v.description END,filters=CASE WHEN (SELECT body ? 'filters' FROM input) THEN (SELECT body->'filters' FROM input) ELSE v.filters END,query=CASE WHEN (SELECT body ? 'filters' FROM input) THEN $5::jsonb ELSE '{}'::jsonb END,display_filters=CASE WHEN (SELECT body ? 'display_filters' FROM input) THEN (SELECT body->'display_filters' FROM input) ELSE v.display_filters END,display_properties=CASE WHEN (SELECT body ? 'display_properties' FROM input) THEN (SELECT body->'display_properties' FROM input) ELSE v.display_properties END,rich_filters=CASE WHEN (SELECT body ? 'rich_filters' FROM input) THEN (SELECT body->'rich_filters' FROM input) ELSE v.rich_filters END,logo_props=CASE WHEN (SELECT body ? 'logo_props' FROM input) THEN (SELECT body->'logo_props' FROM input) ELSE v.logo_props END,archived_at=CASE WHEN (SELECT body ? 'archived_at' FROM input) THEN NULLIF((SELECT body->>'archived_at' FROM input),'')::timestamptz ELSE v.archived_at END,updated_at=NOW(),updated_by_id=$2 WHERE v.id=$4 AND v.workspace_id=(SELECT w.id FROM workspaces w WHERE w.slug=$3) AND ` + projectClause + ` AND v.deleted_at IS NULL AND (` + permission + `) RETURNING *) SELECT ` + viewJSON + ` FROM changed v LEFT JOIN LATERAL (SELECT false AS is_favorite) fav ON true`
	body, err := scanJSON(r.db.QueryRow(ctx, q, args...))
	if errors.Is(err, ErrNotFound) {
		if ownerOnly {
			var locked bool
			if e := r.db.QueryRow(ctx, `SELECT is_locked FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id WHERE v.id=$1 AND w.slug=$2 AND v.deleted_at IS NULL`, id, slug).Scan(&locked); e == nil && locked {
				return nil, ErrLocked
			}
		}
		return nil, ErrForbidden
	}
	return body, err
}

func (r *Repository) DeleteProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID) error {
	return r.remove(ctx, user, slug, &project, id)
}
func (r *Repository) DeleteWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	return r.remove(ctx, user, slug, nil, id)
}
func (r *Repository) remove(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, id uuid.UUID) error {
	projectClause := "v.project_id IS NULL"
	args := []any{user, slug, id}
	if project != nil {
		projectClause = "v.project_id=$4"
		args = append(args, *project)
	}
	// A single statement keeps removal and dependent cleanup atomic, including
	// when a cleanup trigger or constraint rejects the change.
	q := `WITH removed AS (
 UPDATE issue_views v SET deleted_at=NOW()
 FROM workspaces w LEFT JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL
 WHERE v.id=$3 AND v.workspace_id=w.id AND w.slug=$2 AND ` + projectClause + `
 AND v.deleted_at IS NULL AND (v.owned_by_id=$1 OR wm.role=20)
 RETURNING v.id,v.workspace_id,v.project_id
), favorites AS (
 UPDATE user_favorites f SET deleted_at=NOW() FROM removed v
 WHERE f.workspace_id=v.workspace_id AND f.entity_identifier=v.id AND f.entity_type='view'
 AND f.project_id IS NOT DISTINCT FROM v.project_id AND f.deleted_at IS NULL
), visits AS (
 DELETE FROM user_recent_visits r USING removed v
 WHERE v.project_id IS NOT NULL AND r.workspace_id=v.workspace_id AND r.project_id=v.project_id
 AND r.entity_identifier=v.id AND r.entity_name='view'
)
SELECT EXISTS(SELECT 1 FROM removed)`
	var removed bool
	if err := r.db.QueryRow(ctx, q, args...).Scan(&removed); err != nil {
		return err
	}
	if !removed {
		return ErrForbidden
	}
	return nil
}

// CheckMutation preserves lookup and permission errors before field validation.
// Invalid requests never write; one query observes all PATCH precedence checks.
func (r *Repository) CheckMutation(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, id uuid.UUID, partial bool) error {
	if !partial {
		var exists bool
		var projectID any
		if project != nil {
			projectID = *project
		}
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id
  LEFT JOIN projects p ON p.id=v.project_id WHERE v.id=$4 AND w.slug=$2 AND v.deleted_at IS NULL
  AND v.project_id IS NOT DISTINCT FROM $3::uuid AND (v.owned_by_id=$1 OR v.access=1)
  AND ($3::uuid IS NULL OR (p.archived_at IS NULL AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=$3 AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL))))`, user, slug, projectID, id).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return nil
	}
	var allowed, exists, locked, owner bool
	var projectID any
	if project != nil {
		projectID = *project
	}
	err := r.db.QueryRow(ctx, `WITH membership AS (
 SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id
 WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL
 ), target AS (
 SELECT v.is_locked, v.owned_by_id FROM issue_views v JOIN workspaces w ON w.id=v.workspace_id
 WHERE v.id=$4 AND w.slug=$2 AND v.deleted_at IS NULL AND ($3::uuid IS NULL OR v.project_id=$3)
 ) SELECT
 EXISTS(SELECT 1 FROM membership) AND (
 EXISTS(SELECT 1 FROM issue_views WHERE id=$4 AND created_by_id=$1 AND deleted_at IS NULL)
 OR ($3::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM membership WHERE role=20)
 AND EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND pm.project_id=$3 AND w.slug=$2 AND pm.is_active=true AND pm.deleted_at IS NULL))),
 EXISTS(SELECT 1 FROM target), COALESCE((SELECT is_locked FROM target),false), COALESCE((SELECT owned_by_id=$1 FROM target),false)`, user, slug, projectID, id).Scan(&allowed, &exists, &locked, &owner)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrPermission
	}
	if !exists {
		return ErrNotFound
	}
	if locked {
		return ErrLocked
	}
	if !owner {
		return ErrForbidden
	}
	return nil
}

var ErrPermission = errors.New("standard view mutation permission denied")
