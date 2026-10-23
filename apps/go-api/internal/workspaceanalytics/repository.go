// Package workspaceanalytics contains PostgreSQL reads for workspace analytics.
package workspaceanalytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"uuid"

	"github.com/jackc/pgx/v5"
)

var ErrForbidden = errors.New("workspace analytics requires an active admin or member")
var ErrNotFound = errors.New("analytics resource not found")
var ErrInvalidChartXAxis = errors.New("invalid chart x_axis")
var ErrInvalidChartGroupBy = errors.New("invalid chart group_by")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace analytics database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string, project string) (bool, error) {
	var ok bool
	var err error
	if project == "" {
		err = r.db.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id
			WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20)
			AND wm.is_active=true AND wm.deleted_at IS NULL
		)`, user, slug).Scan(&ok)
	} else {
		// allow_permission([ADMIN, MEMBER]) checks project membership first. Its
		// fallback admits any active project member when the same user is an
		// active workspace admin. Workspace membership alone is insufficient.
		err = r.db.QueryRow(ctx, `SELECT
			EXISTS(
				SELECT 1 FROM project_members pm
				JOIN projects p ON p.id=pm.project_id
				JOIN workspaces w ON w.id=p.workspace_id
				WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3::uuid
				AND pm.role IN (15,20) AND pm.is_active=true AND pm.deleted_at IS NULL
			) OR EXISTS(
				SELECT 1 FROM project_members pm
				JOIN projects p ON p.id=pm.project_id
				JOIN workspaces w ON w.id=p.workspace_id
				JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=pm.member_id
				WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3::uuid
				AND pm.is_active=true AND pm.deleted_at IS NULL
				AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
			)`, user, slug, project).Scan(&ok)
	}
	return ok, err
}

func (r *Repository) AuthorizedDefault(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (5,15,20) AND wm.is_active=true AND wm.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

// AuthorizedProjectChart mirrors the chart endpoint's broader project role
// list while retaining the workspace-admin fallback from the permission code.
func (r *Repository) AuthorizedProjectChart(ctx context.Context, user uuid.UUID, slug, project string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT
		EXISTS(
			SELECT 1 FROM project_members pm
			JOIN projects p ON p.id=pm.project_id
			JOIN workspaces w ON w.id=p.workspace_id
			WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3::uuid
			AND pm.role IN (5,15,20) AND pm.is_active=true AND pm.deleted_at IS NULL
		) OR EXISTS(
			SELECT 1 FROM project_members pm
			JOIN projects p ON p.id=pm.project_id
			JOIN workspaces w ON w.id=p.workspace_id
			JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=pm.member_id
			WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3::uuid
			AND pm.is_active=true AND pm.deleted_at IS NULL
			AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
		)`, user, slug, project).Scan(&ok)
	return ok, err
}

func nullableProject(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// JSON runs a query whose single result is a JSON value. The caller supplies a
// fixed query selected from the analytics operation; user values remain bound.
func (r *Repository) JSON(ctx context.Context, query string, args ...any) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, query, args...).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("workspace analytics query: %w", err)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("workspace analytics returned invalid JSON")
	}
	return json.RawMessage(body), nil
}

func (r *Repository) Overview(ctx context.Context, user uuid.UUID, slug string, projectIDs string, dateRange string) (json.RawMessage, error) {
	return r.JSON(ctx, `WITH ws AS (SELECT id FROM workspaces WHERE slug=$2 AND deleted_at IS NULL),
project_scope AS (SELECT p.* FROM projects p JOIN ws ON ws.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL WHERE p.deleted_at IS NULL AND p.archived_at IS NULL AND ($3='' OR p.id=ANY(string_to_array($3,',')::uuid[]))),
visible_issues AS (SELECT i.* FROM issues i JOIN project_scope p ON p.id=i.project_id WHERE i.deleted_at IS NULL AND i.archived_at IS NULL AND (i.is_draft IS NOT TRUE) AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage')),
projects AS (SELECT * FROM project_scope),
members AS (SELECT wm.member_id,wm.role,wm.created_at FROM workspace_members wm JOIN ws ON ws.id=wm.workspace_id WHERE $3='' AND wm.is_active=true AND wm.deleted_at IS NULL UNION ALL SELECT pm.member_id,pm.role,pm.created_at FROM project_members pm JOIN project_scope p ON p.id=pm.project_id WHERE $3<>'' AND pm.is_active=true AND pm.deleted_at IS NULL),
date_filter AS (SELECT $4::text key)
SELECT json_build_object(
 'total_users',json_build_object('count',(SELECT count(*) FROM members m JOIN users u ON u.id=m.member_id WHERE u.is_bot=false AND ($4='' OR CASE $4 WHEN 'yesterday' THEN m.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN m.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN m.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN m.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_admins',json_build_object('count',(SELECT count(*) FROM members m JOIN users u ON u.id=m.member_id WHERE u.is_bot=false AND m.role=20 AND ($4='' OR CASE $4 WHEN 'yesterday' THEN m.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN m.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN m.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN m.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_members',json_build_object('count',(SELECT count(*) FROM members m JOIN users u ON u.id=m.member_id WHERE u.is_bot=false AND m.role=15 AND ($4='' OR CASE $4 WHEN 'yesterday' THEN m.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN m.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN m.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN m.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_guests',json_build_object('count',(SELECT count(*) FROM members m JOIN users u ON u.id=m.member_id WHERE u.is_bot=false AND m.role=5 AND ($4='' OR CASE $4 WHEN 'yesterday' THEN m.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN m.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN m.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN m.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_projects',json_build_object('count',(SELECT count(*) FROM projects p WHERE ($4='' OR CASE $4 WHEN 'yesterday' THEN p.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN p.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN p.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN p.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_work_items',json_build_object('count',(SELECT count(*) FROM visible_issues i WHERE ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_cycles',json_build_object('count',(SELECT count(*) FROM cycles c JOIN project_scope p ON p.id=c.project_id WHERE c.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN c.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN c.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN c.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN c.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END))),
 'total_intake',json_build_object('count',(SELECT count(*) FROM issues i JOIN project_scope p ON p.id=i.project_id JOIN intake_issues intake ON intake.issue_id=i.id WHERE i.deleted_at IS NULL AND intake.status IN (-2,-1,0,1,2) AND ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)))
)`, user, slug, projectIDs, dateRange)
}

func (r *Repository) WorkItemStats(ctx context.Context, user uuid.UUID, slug, projectID, cycleID, moduleID, projectIDs, dateFilter string) (json.RawMessage, error) {
	return r.JSON(ctx, `WITH scoped AS (
 SELECT i.id,s."group" AS state_group FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id LEFT JOIN states s ON s.id=i.state_id
 WHERE w.slug=$2 AND w.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage')
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL)
 AND ($6='' OR p.id=ANY(string_to_array($6,',')::uuid[]))
 AND (($4::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM cycle_issues ci WHERE ci.issue_id=i.id AND ci.cycle_id=$4::uuid AND ci.deleted_at IS NULL)) OR ($4::uuid IS NULL AND $5::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM module_issues mi WHERE mi.issue_id=i.id AND mi.module_id=$5::uuid AND mi.deleted_at IS NULL)) OR ($4::uuid IS NULL AND $5::uuid IS NULL AND ($3::uuid IS NULL OR i.project_id=$3::uuid)))
 AND ($7='' OR CASE $7 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
)
SELECT json_build_object(
 'total_work_items',json_build_object('count',count(*)),
 'started_work_items',json_build_object('count',count(*) FILTER(WHERE state_group='started')),
 'backlog_work_items',json_build_object('count',count(*) FILTER(WHERE state_group='backlog')),
 'un_started_work_items',json_build_object('count',count(*) FILTER(WHERE state_group='unstarted')),
 'completed_work_items',json_build_object('count',count(*) FILTER(WHERE state_group='completed'))
) FROM scoped`, user, slug, nullableProject(projectID), nullableProject(cycleID), nullableProject(moduleID), projectIDs, dateFilter)
}

func (r *Repository) ProjectStats(ctx context.Context, user uuid.UUID, slug, projectIDs, dateFilter string) (json.RawMessage, error) {
	return r.JSON(ctx, `WITH project_stats AS (
 SELECT p.id project_id,p.name project_name,
 count(i.id) FILTER(WHERE s."group"='cancelled') cancelled_work_items,
 count(i.id) FILTER(WHERE s."group"='completed') completed_work_items,
 count(i.id) FILTER(WHERE s."group"='backlog') backlog_work_items,
 count(i.id) FILTER(WHERE s."group"='unstarted') un_started_work_items,
 count(i.id) FILTER(WHERE s."group"='started') started_work_items
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL JOIN issues i ON i.project_id=p.id AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage') LEFT JOIN states s ON s.id=i.state_id
 WHERE w.slug=$2 AND w.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND ($3='' OR p.id=ANY(string_to_array($3,',')::uuid[])) AND ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END) GROUP BY p.id,p.name
 ) SELECT COALESCE(json_agg(json_build_object('project_id',project_id,'project__name',project_name,'cancelled_work_items',cancelled_work_items,'completed_work_items',completed_work_items,'backlog_work_items',backlog_work_items,'un_started_work_items',un_started_work_items,'started_work_items',started_work_items) ORDER BY project_id),'[]'::json) FROM project_stats`, user, slug, projectIDs, dateFilter)
}

func (r *Repository) ProjectAssigneeStats(ctx context.Context, user uuid.UUID, slug, projectID, cycleID, moduleID, projectIDs, dateFilter string) (json.RawMessage, error) {
	return r.JSON(ctx, `WITH assignee_stats AS (
 SELECT u.id assignee_id,u.display_name,
 COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar) avatar_url,
 count(DISTINCT i.id) FILTER(WHERE state."group"='cancelled') cancelled_work_items,
 count(DISTINCT i.id) FILTER(WHERE state."group"='completed') completed_work_items,
 count(DISTINCT i.id) FILTER(WHERE state."group"='backlog') backlog_work_items,
 count(DISTINCT i.id) FILTER(WHERE state."group"='unstarted') un_started_work_items,
 count(DISTINCT i.id) FILTER(WHERE state."group"='started') started_work_items
 FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL LEFT JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.deleted_at IS NULL LEFT JOIN users u ON u.id=ia.assignee_id LEFT JOIN states state ON state.id=i.state_id
 WHERE w.slug=$2 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage') AND (($4::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM cycle_issues ci WHERE ci.issue_id=i.id AND ci.cycle_id=$4::uuid AND ci.deleted_at IS NULL)) OR ($4::uuid IS NULL AND $5::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM module_issues mi WHERE mi.issue_id=i.id AND mi.module_id=$5::uuid AND mi.deleted_at IS NULL)) OR ($4::uuid IS NULL AND $5::uuid IS NULL AND i.project_id=$3::uuid)) AND ($6='' OR p.id=ANY(string_to_array($6,',')::uuid[])) GROUP BY u.id,u.display_name,u.avatar_asset_id,u.avatar
 ) SELECT COALESCE(json_agg(json_build_object('display_name',display_name,'assignee_id',assignee_id,'avatar_url',avatar_url,'cancelled_work_items',cancelled_work_items,'completed_work_items',completed_work_items,'backlog_work_items',backlog_work_items,'un_started_work_items',un_started_work_items,'started_work_items',started_work_items) ORDER BY display_name),'[]'::json) FROM assignee_stats`, user, slug, projectID, nullableProject(cycleID), nullableProject(moduleID), projectIDs)
}

func (r *Repository) ProjectChart(ctx context.Context, user uuid.UUID, slug, projectIDs, dateFilter string) (json.RawMessage, error) {
	return r.JSON(ctx, `WITH projects AS (SELECT p.* FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL WHERE w.slug=$2 AND w.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND ($3='' OR p.id=ANY(string_to_array($3,',')::uuid[]))),
counts AS (
 SELECT 'work_items' key,'Work Items' name,count(*)::int count FROM issues i JOIN projects p ON p.id=i.project_id WHERE i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage') AND ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'cycles','Cycles',count(*)::int FROM cycles c JOIN projects p ON p.id=c.project_id WHERE c.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN c.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN c.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN c.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN c.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'modules','Modules',count(*)::int FROM modules m JOIN projects p ON p.id=m.project_id WHERE m.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN m.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN m.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN m.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN m.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'intake','Intake',count(*)::int FROM intake_issues ii JOIN issues i ON i.id=ii.issue_id JOIN projects p ON p.id=i.project_id WHERE i.deleted_at IS NULL AND ii.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'members','Members',count(*)::int FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN wm.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN wm.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN wm.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN wm.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'pages','Pages',count(*)::int FROM project_pages pp JOIN projects p ON p.id=pp.project_id WHERE pp.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN pp.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN pp.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN pp.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN pp.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
 UNION ALL SELECT 'views','Views',count(*)::int FROM issue_views iv JOIN projects p ON p.id=iv.project_id WHERE iv.deleted_at IS NULL AND ($4='' OR CASE $4 WHEN 'yesterday' THEN iv.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN iv.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN iv.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN iv.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE true END)
) SELECT COALESCE(json_agg(json_build_object('key',key,'name',name,'count',count) ORDER BY ord),'[]'::json) FROM (SELECT *,CASE key WHEN 'work_items' THEN 1 WHEN 'cycles' THEN 2 WHEN 'modules' THEN 3 WHEN 'intake' THEN 4 WHEN 'members' THEN 5 WHEN 'pages' THEN 6 ELSE 7 END ord FROM counts) x`, user, slug, projectIDs, dateFilter)
}

func (r *Repository) SavedView(ctx context.Context, slug string, id uuid.UUID) (json.RawMessage, error) {
	return r.JSON(ctx, `SELECT json_build_object('query',a.query,'query_dict',a.query_dict) FROM analytic_views a JOIN workspaces w ON w.id=a.workspace_id WHERE w.slug=$1 AND w.deleted_at IS NULL AND a.id=$2 AND a.deleted_at IS NULL`, slug, id)
}

type chartDimension struct{ id, name, joins, valid string }

var chartDimensions = map[string]chartDimension{
	"STATES":          {"state.id::text", "state.name", `LEFT JOIN states state ON state.id=i.state_id`, ``},
	"STATE_GROUPS":    {`state."group"`, `state."group"`, `LEFT JOIN states state ON state.id=i.state_id`, ``},
	"LABELS":          {"label.id::text", "label.name", `JOIN label_issues label_link ON label_link.issue_id=i.id AND label_link.deleted_at IS NULL JOIN labels label ON label.id=label_link.label_id`, ``},
	"ASSIGNEES":       {"assignee.id::text", "assignee.display_name", `JOIN issue_assignees assignment ON assignment.issue_id=i.id AND assignment.deleted_at IS NULL JOIN users assignee ON assignee.id=assignment.assignee_id`, ``},
	"ESTIMATE_POINTS": {"estimate_point.key::text", "estimate_point.value::text", `LEFT JOIN estimate_points estimate_point ON estimate_point.id=i.estimate_point_id`, ``},
	"CYCLES":          {"cycle.id::text", "cycle.name", `JOIN cycle_issues cycle_link ON cycle_link.issue_id=i.id AND cycle_link.deleted_at IS NULL JOIN cycles cycle ON cycle.id=cycle_link.cycle_id`, ``},
	"MODULES":         {"module.id::text", "module.name", `JOIN module_issues module_link ON module_link.issue_id=i.id AND module_link.deleted_at IS NULL JOIN modules module ON module.id=module_link.module_id`, ``},
	"PRIORITY":        {"i.priority::text", "i.priority::text", ``, ``},
	"START_DATE":      {"i.start_date::text", "i.start_date::text", ``, ``},
	"TARGET_DATE":     {"i.target_date::text", "i.target_date::text", ``, ``},
	"CREATED_AT":      {"i.created_at::date::text", "i.created_at::date::text", ``, ``},
	"COMPLETED_AT":    {"i.completed_at::date::text", "i.completed_at::date::text", ``, ``},
	"CREATED_BY":      {"creator.id::text", "creator.display_name", `LEFT JOIN users creator ON creator.id=i.created_by_id`, ``},
}

// Chart returns the source build_analytics_chart shape using an allowlisted
// dimension map; request values are used only as keys into this map.
func (r *Repository) Chart(ctx context.Context, user uuid.UUID, slug, projectID, projectIDs, dateFilter, xAxis, groupBy, cycleID, moduleID string) (json.RawMessage, error) {
	axis, ok := chartDimensions[xAxis]
	if !ok {
		return nil, ErrInvalidChartXAxis
	}
	joins := axis.joins
	groupExpr, groupName := "NULL::text", "NULL::text"
	if groupBy != "" {
		group, exists := chartDimensions[groupBy]
		if !exists {
			return nil, ErrInvalidChartGroupBy
		}
		if group.joins != "" && !strings.Contains(joins, group.joins) {
			joins += " " + group.joins
		}
		groupExpr, groupName = group.id, group.name
	}
	base := ` FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id ` + joins + ` WHERE w.slug=$2 AND w.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage') AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL) AND ($3='' OR p.id=ANY(string_to_array($3,',')::uuid[])) AND ($4::uuid IS NULL OR i.project_id=$4::uuid) AND ($5='' OR CASE $5 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE TRUE END) AND ($6::uuid IS NULL OR EXISTS(SELECT 1 FROM cycle_issues cx WHERE cx.issue_id=i.id AND cx.cycle_id=$6::uuid AND cx.deleted_at IS NULL)) AND ($7::uuid IS NULL OR EXISTS(SELECT 1 FROM module_issues mx WHERE mx.issue_id=i.id AND mx.module_id=$7::uuid AND mx.deleted_at IS NULL))`
	var query string
	if groupBy == "" {
		query = `WITH grouped AS (SELECT ` + axis.id + ` k,COALESCE(` + axis.name + `, 'None') n,count(DISTINCT i.id)::int c` + base + ` GROUP BY 1,2) SELECT jsonb_build_object('data',COALESCE(jsonb_agg(jsonb_build_object('key',COALESCE(k,'None'),'name',n,'count',c) ORDER BY k NULLS FIRST),'[]'::jsonb),'schema','{}'::jsonb) FROM grouped`
	} else {
		query = `WITH grouped AS (SELECT ` + axis.id + ` k,COALESCE(` + axis.name + `, 'None') n,` + groupExpr + ` g,` + groupName + ` gn,count(DISTINCT i.id)::int c` + base + ` GROUP BY 1,2,3,4), per_key AS (SELECT k,n,sum(c)::int total,COALESCE(jsonb_object_agg(COALESCE(g,'none'),c),'{}'::jsonb) groups FROM grouped GROUP BY k,n), schema AS (SELECT COALESCE(jsonb_object_agg(COALESCE(g,'none'),COALESCE(gn,'None')),'{}'::jsonb) value FROM grouped) SELECT jsonb_build_object('data',COALESCE((SELECT jsonb_agg(jsonb_build_object('key',COALESCE(k,'none'),'name',n,'count',total)||groups ORDER BY total DESC) FROM per_key),'[]'::jsonb),'schema',(SELECT value FROM schema))`
	}
	return r.JSON(ctx, query, user, slug, projectIDs, nullableProject(projectID), dateFilter, nullableProject(cycleID), nullableProject(moduleID))
}
