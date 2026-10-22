// Package workspaceuserinsights serves workspace member profiles, activity,
// and per-user issue statistics.
package workspaceuserinsights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("workspace user insight not found")
var ErrForbidden = errors.New("workspace user activity requires workspace membership")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("workspace user insights database required")
	}
	return &Repository{db: db}, nil
}

// AuthorizeWorkspace mirrors WorkspaceEntityPermission before the request's
// timezone mixin runs. Safe methods accept any active workspace member; other
// methods require the source Admin or Member roles (15 and 20).
func (r *Repository) AuthorizeWorkspace(ctx context.Context, requester, slug string, requireWriteRole bool) error {
	query := `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1::uuid AND wm.is_active=true)`
	if requireWriteRole {
		query = `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1::uuid AND wm.is_active=true AND wm.role IN (15,20))`
	}
	var allowed bool
	if err := r.db.QueryRow(ctx, query, requester, slug).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (r *Repository) Profile(ctx context.Context, requester, slug, target string) (json.RawMessage, error) {
	var raw []byte
	err := r.db.QueryRow(ctx, `WITH requester AS (SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1::uuid AND wm.is_active=true), target AS (SELECT u.* FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id JOIN users u ON u.id=wm.member_id WHERE w.slug=$2 AND wm.member_id=$3::uuid AND wm.is_active=true), project_data AS (SELECT p.id,p.logo_props,
count(DISTINCT i.id) FILTER(WHERE i.created_by_id=$3::uuid AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE)::int created_issues,
count(DISTINCT i.id) FILTER(WHERE ia.assignee_id=$3::uuid AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE)::int assigned_issues,
count(DISTINCT i.id) FILTER(WHERE ia.assignee_id=$3::uuid AND i.completed_at IS NOT NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE)::int completed_issues,
count(DISTINCT i.id) FILTER(WHERE ia.assignee_id=$3::uuid AND st."group" IN ('backlog','unstarted','started') AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE)::int pending_issues
FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$1::uuid AND pm.is_active=true LEFT JOIN issues i ON i.project_id=p.id LEFT JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.deleted_at IS NULL LEFT JOIN states st ON st.id=i.state_id WHERE w.slug=$2 AND p.archived_at IS NULL AND EXISTS(SELECT 1 FROM requester WHERE role>=15) GROUP BY p.id),
project_json AS (SELECT COALESCE(json_agg(json_build_object('id',id,'logo_props',logo_props,'created_issues',created_issues,'assigned_issues',assigned_issues,'completed_issues',completed_issues,'pending_issues',pending_issues)),'[]'::json) value FROM project_data),
user_json AS (SELECT json_build_object('email',target.email,'first_name',target.first_name,'last_name',target.last_name,'avatar_url',CASE WHEN target.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||target.avatar_asset_id::text||'/' ELSE NULLIF(target.avatar,'') END,'cover_image_url',CASE WHEN target.cover_image_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||target.cover_image_asset_id::text||'/' ELSE NULLIF(target.cover_image,'') END,'date_joined',`+djangoapi.DjangoTimestampSQL("target.date_joined")+`,'user_timezone',target.user_timezone,'display_name',target.display_name) value FROM target)
SELECT json_build_object('project_data',(SELECT value FROM project_json),'user_data',(SELECT value FROM user_json)) WHERE EXISTS(SELECT 1 FROM requester) AND EXISTS(SELECT 1 FROM target)`, requester, slug, target).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return validJSON(raw)
}

func (r *Repository) Activity(ctx context.Context, requester, slug, target string, projects []string, query djangoapi.Query) (json.RawMessage, error) {
	var allowed bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1::uuid AND wm.is_active=true)`, requester, slug).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	limit, err := djangoapi.PageSizeWithMinimum(query, 1)
	if err != nil {
		return nil, err
	}
	page, err := djangoapi.CursorPage(query, limit)
	if err != nil {
		return nil, err
	}
	if page > math.MaxInt/limit {
		return nil, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	offset := page * limit
	orderValue := query.Params["order_by"]
	orderDescending := strings.HasPrefix(orderValue, "-")
	orderField := strings.TrimPrefix(orderValue, "-")
	if strings.HasPrefix(orderField, "-") || (orderField != "created_at" && orderField != "updated_at") {
		orderField, orderDescending = "created_at", true
	}
	orderDirection := "ASC"
	if orderDescending {
		orderDirection = "DESC"
	}
	order := orderField + " " + orderDirection
	args := []any{requester, slug, target, limit, offset}
	projectClause := ""
	if len(projects) > 0 {
		projectClause = " AND a.project_id::text=ANY($6::text[])"
		args = append(args, projects)
	}
	var raw []byte
	err = r.db.QueryRow(ctx, `WITH filtered AS (SELECT a.*,u.first_name,u.last_name,u.avatar,u.avatar_asset_id,u.is_bot,u.display_name,p.identifier,p.name project_name,p.cover_image,p.cover_image_asset_id,p.description,p.logo_props,w.name workspace_name,w.slug workspace_slug,w.id workspace_detail_id,w.logo_asset_id,w.logo,i.name issue_name,i.description_json,i.description_html,i.priority,i.start_date,i.target_date,i.sequence_id,i.sort_order,i.is_draft FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects p ON p.id=a.project_id LEFT JOIN users u ON u.id=a.actor_id LEFT JOIN issues i ON i.id=a.issue_id WHERE w.slug=$2 AND a.actor_id=$3::uuid AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft')) AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$1::uuid AND pm.is_active=true) AND p.archived_at IS NULL`+projectClause+`), totals AS (SELECT count(*)::int total FROM filtered), rows AS (SELECT * FROM filtered ORDER BY `+order+` LIMIT $4 OFFSET $5), result AS (SELECT COALESCE(json_agg(json_build_object('id',id,'created_at',`+djangoapi.DjangoTimestampSQL("created_at")+`,'updated_at',`+djangoapi.DjangoTimestampSQL("updated_at")+`,'created_by',created_by_id,'updated_by',updated_by_id,'deleted_at',`+djangoapi.DjangoTimestampSQL("deleted_at")+`,'workspace',workspace_id,'project',project_id,'issue',issue_id,'verb',verb,'field',field,'old_value',old_value,'new_value',new_value,'comment',comment,'attachments',attachments,'issue_comment',issue_comment_id,'actor',actor_id,'old_identifier',old_identifier,'new_identifier',new_identifier,'epoch',epoch,'actor_detail',CASE WHEN actor_id IS NULL THEN NULL ELSE json_build_object('id',actor_id,'first_name',first_name,'last_name',last_name,'avatar',avatar,'avatar_url',CASE WHEN avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||avatar_asset_id::text||'/' ELSE NULLIF(avatar,'') END,'is_bot',is_bot,'display_name',display_name) END,'issue_detail',CASE WHEN issue_id IS NULL THEN NULL ELSE json_build_object('id',issue_id,'name',issue_name,'description_json',description_json,'description_html',description_html,'priority',priority,'start_date',start_date,'target_date',target_date,'sequence_id',sequence_id,'sort_order',`+djangoFloatSQL("sort_order")+`,'is_draft',is_draft) END,'project_detail',json_build_object('id',project_id,'identifier',identifier,'name',project_name,'cover_image',cover_image,'cover_image_url',CASE WHEN cover_image_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||cover_image_asset_id::text||'/' ELSE NULLIF(cover_image,'') END,'logo_props',logo_props,'description',description),'workspace_detail',json_build_object('name',workspace_name,'slug',workspace_slug,'id',workspace_detail_id,'logo_url',`+djangoapi.FileAssetURLSQL("logo_asset_id", "logo")+`),'source_data',NULL) ORDER BY `+order+`),'[]'::json) results FROM rows) SELECT json_build_object('grouped_by',NULL,'sub_grouped_by',NULL,'total_count',totals.total,'next_cursor',concat($4,':',($5/$4)+1,':0'),'prev_cursor',concat($4,':',($5/$4)-1,':1'),'next_page_results',($5+$4)<totals.total,'prev_page_results',$5>0,'count',json_array_length(result.results),'total_pages',CASE WHEN totals.total=0 THEN 0 ELSE CEIL(totals.total::numeric/$4)::int END,'total_results',totals.total,'extra_stats',NULL,'results',result.results) FROM totals CROSS JOIN result`, args...).Scan(&raw)
	if err != nil {
		return nil, err
	}
	return validJSON(raw)
}

func (r *Repository) Stats(ctx context.Context, requester, target, slug string, filters map[string]string) (json.RawMessage, error) {
	present := make(map[string]bool, len(filters))
	for name := range filters {
		present[name] = true
	}
	where := []string{"issue.workspace_id=(SELECT id FROM workspaces WHERE slug=$2)", "issue.deleted_at IS NULL", "issue.archived_at IS NULL", "issue.is_draft IS NOT TRUE", "COALESCE(state.\"group\", '') <> 'triage'", "project.archived_at IS NULL", "EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=project.id AND pm.member_id=$1::uuid AND pm.is_active=true)"}
	args := []any{requester, slug, target}
	where, args, err := workitems.BuildAppIssueFilters(filters, present, where, args)
	if err != nil {
		return nil, err
	}
	subscriberWhere, subscriberArgs, err := workitems.BuildAppIssueFilters(filters, present, []string{
		"issue.workspace_id=(SELECT id FROM workspaces WHERE slug=$2)",
		"issue.deleted_at IS NULL",
		"issue.subscriber_id=$3::uuid",
		"project.archived_at IS NULL",
		"EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=project.id AND pm.member_id=$1::uuid AND pm.is_active=true)",
	}, []any{requester, slug, target})
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(args, subscriberArgs) {
		return nil, errors.New("workspace user stats filter parameter mismatch")
	}
	var raw []byte
	query := `WITH scoped AS (
SELECT issue.*,state."group" AS state_group
FROM issues issue JOIN projects project ON project.id=issue.project_id
LEFT JOIN states state ON state.id=issue.state_id
WHERE ` + strings.Join(where, " AND ") + `
), assigned AS (
SELECT scoped.* FROM scoped WHERE EXISTS(SELECT 1 FROM issue_assignees ia WHERE ia.issue_id=scoped.id AND ia.assignee_id=$3::uuid AND ia.deleted_at IS NULL)
), state_distribution AS (
SELECT COALESCE(json_agg(json_build_object('state_group',state_group,'state_count',n) ORDER BY state_group),'[]'::json) v
FROM (SELECT state_group,count(state_group)::int n FROM assigned GROUP BY state_group)x
), priority_distribution AS (
SELECT COALESCE(json_agg(json_build_object('priority',priority,'priority_count',n,'priority_order',CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 WHEN 'none' THEN 4 ELSE 5 END) ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 WHEN 'none' THEN 4 ELSE 5 END),'[]'::json) v
FROM (SELECT priority,count(priority)::int n FROM assigned GROUP BY priority HAVING count(priority)>=1)x
), subscribed_count AS (
SELECT count(*)::int n FROM issue_subscribers issue JOIN projects project ON project.id=issue.project_id WHERE ` + strings.Join(subscriberWhere, " AND ") + `
), present AS (
SELECT COALESCE(json_agg(json_build_object('cycle__name',c.name,'cycle__id',c.id,'cycle__project_id',c.project_id)),'[]'::json) v
FROM cycle_issues ci JOIN cycles c ON c.id=ci.cycle_id JOIN workspaces w ON w.id=ci.workspace_id JOIN issue_assignees ia ON ia.issue_id=ci.issue_id AND ia.assignee_id=$3::uuid AND ia.deleted_at IS NULL WHERE w.slug=$2 AND c.start_date<CURRENT_TIMESTAMP AND c.end_date>CURRENT_TIMESTAMP
), upcoming AS (
SELECT COALESCE(json_agg(json_build_object('cycle__name',c.name,'cycle__id',c.id,'cycle__project_id',c.project_id)),'[]'::json) v
FROM cycle_issues ci JOIN cycles c ON c.id=ci.cycle_id JOIN workspaces w ON w.id=ci.workspace_id JOIN issue_assignees ia ON ia.issue_id=ci.issue_id AND ia.assignee_id=$3::uuid AND ia.deleted_at IS NULL WHERE w.slug=$2 AND c.start_date>CURRENT_TIMESTAMP
)
SELECT json_build_object('state_distribution',(SELECT v FROM state_distribution),'priority_distribution',(SELECT v FROM priority_distribution),'created_issues',(SELECT count(*)::int FROM scoped WHERE created_by_id=$3::uuid),'assigned_issues',(SELECT count(*)::int FROM assigned),'completed_issues',(SELECT count(*)::int FROM assigned WHERE state_group='completed'),'pending_issues',(SELECT count(*)::int FROM assigned WHERE state_group NOT IN ('completed','cancelled')),'subscribed_issues',(SELECT n FROM subscribed_count),'present_cycles',(SELECT v FROM present),'upcoming_cycles',(SELECT v FROM upcoming))`
	if err := r.db.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	return validJSON(raw)
}

type ActivityCSVRow struct {
	Actor, IssueID, Project         string
	CreatedAt, UpdatedAt            any
	Verb, Field, OldValue, NewValue string
}

func (r *Repository) ExportActivity(ctx context.Context, requester, slug, target, date string) ([]ActivityCSVRow, error) {
	var allowed bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1::uuid AND wm.role IN (15,20) AND wm.is_active=true)`, requester, slug).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	rows, err := r.db.Query(ctx, `SELECT COALESCE(actor.display_name,''),concat(project.identifier,' - ',COALESCE(issue.sequence_id::text,'')),project.name,a.created_at,a.updated_at,a.verb,COALESCE(a.field,''),COALESCE(a.old_value,''),COALESCE(a.new_value,'') FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects project ON project.id=a.project_id LEFT JOIN users actor ON actor.id=a.actor_id LEFT JOIN issues issue ON issue.id=a.issue_id WHERE w.slug=$1 AND a.created_at::date=$2::date AND a.actor_id=$3::uuid AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft')) AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=project.id AND pm.member_id=$3::uuid AND pm.is_active=true) ORDER BY a.created_at LIMIT 10000`, slug, date, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]ActivityCSVRow, 0)
	for rows.Next() {
		var row ActivityCSVRow
		if err := rows.Scan(&row.Actor, &row.IssueID, &row.Project, &row.CreatedAt, &row.UpdatedAt, &row.Verb, &row.Field, &row.OldValue, &row.NewValue); err != nil {
			return nil, err
		}
		values = append(values, row)
	}
	return values, rows.Err()
}

func (r *Repository) CanExportActivity(ctx context.Context, requester, slug string) (bool, error) {
	err := r.AuthorizeWorkspace(ctx, requester, slug, true)
	if errors.Is(err, ErrForbidden) {
		return false, nil
	}
	return err == nil, err
}

func validJSON(raw []byte) (json.RawMessage, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("workspace user insights returned invalid JSON")
	}
	return json.RawMessage(raw), nil
}

// djangoFloatSQL preserves Python's float-shaped JSON number spelling.
// PostgreSQL emits integral float8 values as `2`, while DRF's Python float
// encoder emits `2.0`; non-integral and exponent forms already match.
func djangoFloatSQL(column string) string {
	value := "to_json(" + column + "::double precision)::text"
	return "CASE WHEN " + column + " IS NULL THEN 'null'::json " +
		"WHEN position('.' in " + value + ") > 0 OR position('e' in lower(" + value + ")) > 0 THEN (" + value + ")::json " +
		"ELSE (" + value + " || '.0')::json END"
}
