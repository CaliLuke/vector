// Package userproperties contains the persistence operations for the four
// user-property resources exposed by the workspace API.
package userproperties

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"uuid"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("user properties not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

// AuthorizedProject applies the same workspace membership check as Authorized,
// and also requires an active membership in the requested project. The Django
// cycle and module endpoints use project-scoped permissions.
func (r *Repository) AuthorizedProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id JOIN project_members pm ON pm.workspace_id=w.id AND pm.member_id=wm.member_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL AND pm.project_id=$3 AND pm.is_active=true AND pm.deleted_at IS NULL)`, user, slug, project).Scan(&ok)
	return ok, err
}

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("user properties database is required")
	}
	return &Repository{db: db}, nil
}

const defaults = `'{"priority":null,"state":null,"state_group":null,"assignees":null,"created_by":null,"labels":null,"start_date":null,"target_date":null,"subscriber":null}'::jsonb`
const displayDefaults = `'{"group_by":null,"order_by":"-created_at","type":null,"sub_issue":true,"show_empty_groups":true,"layout":"list","calendar_date_range":""}'::jsonb`
const propertyDefaults = `'{"assignee":true,"attachment_count":true,"created_on":true,"due_date":true,"estimate":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}'::jsonb`
const workspaceDisplayDefaults = `'{"display_filters":{"group_by":null,"order_by":"-created_at","type":null,"sub_issue":true,"show_empty_groups":true,"layout":"list","calendar_date_range":""}}'::jsonb`
const workspacePropertyDefaults = `'{"display_properties":{"assignee":true,"attachment_count":true,"created_on":true,"due_date":true,"estimate":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}}'::jsonb`
const preferencesDefaults = `'{"pages":{"block_display":true},"navigation":{"default_tab":"work_items","hide_in_more_menu":[]}}'::jsonb`
const projectJSON = `json_build_object('id',p.id,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'updated_by',p.updated_by_id,'deleted_at',p.deleted_at,'workspace',p.workspace_id,'project',p.project_id,'user',p.user_id,'filters',p.filters,'display_filters',p.display_filters,'display_properties',p.display_properties,'rich_filters',p.rich_filters,'preferences',p.preferences,'sort_order',p.sort_order)`
const workspaceJSON = `json_build_object('id',p.id,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'updated_by',p.updated_by_id,'deleted_at',p.deleted_at,'workspace',p.workspace_id,'user',p.user_id,'filters',p.filters,'display_filters',p.display_filters,'display_properties',p.display_properties,'rich_filters',p.rich_filters,'navigation_project_limit',p.navigation_project_limit,'navigation_control_preference',p.navigation_control_preference)`
const cycleJSON = `json_build_object('id',p.id,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'updated_by',p.updated_by_id,'deleted_at',p.deleted_at,'workspace',p.workspace_id,'project',p.project_id,'cycle',p.cycle_id,'user',p.user_id,'filters',p.filters,'display_filters',p.display_filters,'display_properties',p.display_properties,'rich_filters',p.rich_filters)`
const moduleJSON = `json_build_object('id',p.id,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'updated_by',p.updated_by_id,'deleted_at',p.deleted_at,'workspace',p.workspace_id,'project',p.project_id,'module',p.module_id,'user',p.user_id,'filters',p.filters,'display_filters',p.display_filters,'display_properties',p.display_properties,'rich_filters',p.rich_filters)`

func scan(r pgx.Row) (json.RawMessage, error) { var b []byte; err := r.Scan(&b); return b, err }
func result(b json.RawMessage, err error) (json.RawMessage, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}

func (r *Repository) GetWorkspace(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	q := `WITH w AS (SELECT id FROM workspaces WHERE slug=$2 AND deleted_at IS NULL), ins AS (INSERT INTO workspace_user_properties (id,created_at,updated_at,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,navigation_project_limit,navigation_control_preference,created_by_id) SELECT $3,NOW(),NOW(),id,$1,` + defaults + `,` + workspaceDisplayDefaults + `,` + workspacePropertyDefaults + `,'{}'::jsonb,10,'ACCORDION',$1 FROM w ON CONFLICT (workspace_id,user_id) WHERE deleted_at IS NULL DO UPDATE SET id=workspace_user_properties.id RETURNING *) SELECT ` + workspaceJSON + ` FROM ins p`
	return result(scan(r.db.QueryRow(ctx, q, user, slug, googleuuid.New())))
}

func (r *Repository) PatchWorkspace(ctx context.Context, user uuid.UUID, slug string, patch map[string]json.RawMessage) (json.RawMessage, error) {
	return r.patchWorkspace(ctx, user, slug, patch)
}
func (r *Repository) patchWorkspace(ctx context.Context, user uuid.UUID, slug string, m map[string]json.RawMessage) (json.RawMessage, error) {
	filters, df, dp, rf := m["filters"], m["display_filters"], m["display_properties"], m["rich_filters"]
	var b []byte
	limit, preference := m["navigation_project_limit"], m["navigation_control_preference"]
	q := `UPDATE workspace_user_properties p SET filters=CASE WHEN $3 THEN $4::jsonb ELSE p.filters END,display_filters=CASE WHEN $5 THEN $6::jsonb ELSE p.display_filters END,display_properties=CASE WHEN $7 THEN $8::jsonb ELSE p.display_properties END,rich_filters=CASE WHEN $9 THEN $10::jsonb ELSE p.rich_filters END,navigation_project_limit=CASE WHEN $11 THEN $12::integer ELSE p.navigation_project_limit END,navigation_control_preference=CASE WHEN $13 THEN $14::text ELSE p.navigation_control_preference END,updated_at=NOW(),updated_by_id=$1 FROM workspaces w WHERE w.id=p.workspace_id AND w.slug=$2 AND p.user_id=$1 AND p.deleted_at IS NULL RETURNING ` + workspaceJSON
	err := r.db.QueryRow(ctx, q, user, slug, filters != nil, filters, df != nil, df, dp != nil, dp, rf != nil, rf, limit != nil, limit, preference != nil, preference).Scan(&b)
	return result(b, err)
}

func (r *Repository) GetProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) (json.RawMessage, error) {
	return r.getScoped(ctx, user, slug, project, uuid.Nil(), "project")
}
func (r *Repository) GetCycle(ctx context.Context, user uuid.UUID, slug string, project, cycle uuid.UUID) (json.RawMessage, error) {
	return r.getScoped(ctx, user, slug, project, cycle, "cycle")
}
func (r *Repository) GetModule(ctx context.Context, user uuid.UUID, slug string, project, module uuid.UUID) (json.RawMessage, error) {
	return r.getScoped(ctx, user, slug, project, module, "module")
}

func (r *Repository) getScoped(ctx context.Context, user uuid.UUID, slug string, project, object uuid.UUID, kind string) (json.RawMessage, error) {
	table, objJoin, body := "project_user_properties", "", projectJSON
	if kind == "cycle" {
		table, objJoin, body = "cycle_user_properties", " AND c.id=$4 AND c.project_id=$3", cycleJSON
	}
	if kind == "module" {
		table, objJoin, body = "module_user_properties", " AND m.id=$4 AND m.project_id=$3", moduleJSON
	}
	join := "JOIN workspaces w ON w.id=p.workspace_id JOIN projects x ON x.id=p.project_id"
	if kind == "cycle" {
		join += " JOIN cycles c ON c.id=p.cycle_id"
	}
	if kind == "module" {
		join += " JOIN modules m ON m.id=p.module_id"
	}
	q := `SELECT ` + body + ` FROM ` + table + ` p ` + join + ` WHERE w.slug=$1 AND p.user_id=$2 AND p.project_id=$3 AND p.deleted_at IS NULL` + objJoin
	var b []byte
	args := []any{slug, user, project}
	if kind != "project" {
		args = append(args, object)
	}
	err := r.db.QueryRow(ctx, q, args...).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.createScoped(ctx, user, slug, project, object, kind)
	}
	return result(b, err)
}

func (r *Repository) createScoped(ctx context.Context, user uuid.UUID, slug string, project, object uuid.UUID, kind string) (json.RawMessage, error) {
	table, body, extra, conflict := "project_user_properties", projectJSON, "", "(project_id,user_id) WHERE deleted_at IS NULL"
	if kind == "cycle" {
		table, body = "cycle_user_properties", cycleJSON
		extra = ",cycle_id"
		conflict = "(cycle_id,user_id) WHERE deleted_at IS NULL"
	}
	if kind == "module" {
		table, body = "module_user_properties", moduleJSON
		extra = ",module_id"
		conflict = "(module_id,user_id) WHERE deleted_at IS NULL"
	}
	columns := `(id,created_at,updated_at,workspace_id,project_id` + extra + `,user_id,filters,display_filters,display_properties,rich_filters`
	values := `$1,NOW(),NOW(),w.id,$4` + func() string {
		if extra != "" {
			return ",$5"
		}
		return ""
	}() + `,$3,` + defaults + `,` + displayDefaults + `,` + propertyDefaults + `,'{}'::jsonb`
	if kind == "project" {
		columns += `,preferences,sort_order`
		values += `,` + preferencesDefaults + `,65535`
	}
	columns += `,created_by_id)`
	values += `,$3`
	q := `INSERT INTO ` + table + ` AS p ` + columns + ` SELECT ` + values + ` FROM workspaces w JOIN projects prj ON prj.workspace_id=w.id` + func() string {
		if kind == "cycle" {
			return ` JOIN cycles c ON c.project_id=prj.id AND c.id=$5`
		}
		if kind == "module" {
			return ` JOIN modules m ON m.project_id=prj.id AND m.id=$5`
		}
		return ""
	}() + ` WHERE w.slug=$2 AND prj.id=$4 AND w.deleted_at IS NULL ON CONFLICT ` + conflict + ` DO UPDATE SET id=p.id RETURNING ` + body
	args := []any{uuid.New(), slug, user, project}
	if extra != "" {
		args = append(args, object)
	}
	var b []byte
	err := r.db.QueryRow(ctx, q, args...).Scan(&b)
	return result(b, err)
}

func (r *Repository) PatchProject(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, m map[string]json.RawMessage) (json.RawMessage, error) {
	return r.patchScoped(ctx, user, slug, id, uuid.Nil(), m, "project")
}
func (r *Repository) PatchCycle(ctx context.Context, user uuid.UUID, slug string, project, cycle uuid.UUID, m map[string]json.RawMessage) (json.RawMessage, error) {
	return r.patchScoped(ctx, user, slug, project, cycle, m, "cycle")
}
func (r *Repository) PatchModule(ctx context.Context, user uuid.UUID, slug string, project, module uuid.UUID, m map[string]json.RawMessage) (json.RawMessage, error) {
	return r.patchScoped(ctx, user, slug, project, module, m, "module")
}
func (r *Repository) patchScoped(ctx context.Context, user uuid.UUID, slug string, project, object uuid.UUID, m map[string]json.RawMessage, kind string) (json.RawMessage, error) {
	table, body, join := "project_user_properties", projectJSON, ""
	if kind == "cycle" {
		table, body = "cycle_user_properties", cycleJSON
		join = " AND p.cycle_id=$4"
	}
	if kind == "module" {
		table, body = "module_user_properties", moduleJSON
		join = " AND p.module_id=$4"
	}
	fields := []struct{ name, cast string }{{"filters", "jsonb"}, {"display_filters", "jsonb"}, {"display_properties", "jsonb"}, {"rich_filters", "jsonb"}}
	if kind == "project" {
		fields = append(fields, struct{ name, cast string }{"preferences", "jsonb"}, struct{ name, cast string }{"sort_order", "double precision"})
	}
	next := 4
	if kind != "project" {
		next = 5
	}
	sets := make([]string, 0, len(fields))
	vals := []any{slug, user, project}
	if kind != "project" {
		vals = append(vals, object)
	}
	for _, field := range fields {
		v := m[field.name]
		sets = append(sets, fmt.Sprintf("%s=CASE WHEN $%d THEN $%d::%s ELSE p.%s END", field.name, next, next+1, field.cast, field.name))
		vals = append(vals, v != nil, v)
		next += 2
	}
	set := strings.Join(sets, ",")
	q := `UPDATE ` + table + ` p SET ` + set + `,updated_at=NOW(),updated_by_id=$2 FROM workspaces w WHERE w.id=p.workspace_id AND w.slug=$1 AND p.user_id=$2 AND p.project_id=$3 AND p.deleted_at IS NULL` + join + ` RETURNING ` + body
	var b []byte
	err := r.db.QueryRow(ctx, q, vals...).Scan(&b)
	return result(b, err)
}

// CheckScopedPatch looks up an existing property row without GET's creation side effect.
func (r *Repository) CheckScopedPatch(ctx context.Context, user uuid.UUID, slug string, project, object uuid.UUID, kind string) error {
	var table, column string
	switch kind {
	case "cycle":
		table, column = "cycle_user_properties", "cycle_id"
	case "module":
		table, column = "module_user_properties", "module_id"
	default:
		return fmt.Errorf("unsupported property scope %q", kind)
	}
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` p JOIN workspaces w ON w.id=p.workspace_id WHERE p.user_id=$1 AND w.slug=$2 AND p.project_id=$3 AND p.`+column+`=$4 AND p.deleted_at IS NULL)`, user, slug, project, object).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}
