// Package workspaceprojectreads contains the workspace and project core read boundary.
package workspaceprojectreads

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden        = errors.New("active workspace membership with an allowed role required")
	ErrNotFound         = errors.New("workspace or project not found")
	ErrProjectForbidden = errors.New("project membership required for secret project")
	ErrProjectConflict  = errors.New("project membership required for public project")
	ErrBadRequest       = errors.New("invalid query parameter")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type taskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type RecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type Repository struct {
	database     database
	publisher    taskPublisher
	recentVisits RecentVisitAdmitter
}

type WorkspaceOptions struct {
	Owner       *uuid.UUID
	SearchTerms []string
}

type ProjectDetailsOptions struct {
	PerPage *string
	Cursor  *string
	OrderBy *string
}

type APIProjectDetailOptions struct {
	Fields, Expand string
	// ExpectedWorkspaceID binds a previously selected workspace through slug changes.
	ExpectedWorkspaceID *uuid.UUID
}

type APIProjectListOptions struct {
	// ExpectedWorkspaceID binds a previously selected workspace through slug changes.
	ExpectedWorkspaceID *uuid.UUID
	Fields, Expand      string
	IncludeArchived     bool
	PerPage             *string
	Cursor              *string
	OrderBy             *string
}

type APIProjectSummaryOptions struct{ Fields []string }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("workspace project reads database is required")
	}
	return &Repository{database: database}, nil
}

func NewRepositoryWithRecentVisits(database database, publisher taskPublisher, recentVisits RecentVisitAdmitter) (*Repository, error) {
	if database == nil || publisher == nil {
		return nil, fmt.Errorf("workspace project read database and task publisher are required")
	}
	return &Repository{database: database, publisher: publisher, recentVisits: recentVisits}, nil
}

func (repository *Repository) admitRecentVisit(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) error {
	if repository.recentVisits != nil {
		return repository.recentVisits.AdmitRecentVisit(ctx, uuid.New().String(), slug, "project", projectID.String(), userID.String(), projectID.String())
	}
	if repository.publisher == nil {
		return nil
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{
		"slug": slug, "entity_name": "project", "entity_identifier": projectID.String(), "user_id": userID.String(), "project_id": projectID.String(),
	})
}

func (repository *Repository) UserWorkspaces(ctx context.Context, userID uuid.UUID, options WorkspaceOptions) (jsontext.Value, error) {
	return repository.workspaceList(ctx, userID, options, true, "created_at DESC")
}

func (repository *Repository) Workspaces(ctx context.Context, userID uuid.UUID, options WorkspaceOptions) (jsontext.Value, error) {
	return repository.workspaceList(ctx, userID, options, false, "name ASC")
}

func (repository *Repository) workspaceList(ctx context.Context, userID uuid.UUID, options WorkspaceOptions, includeRole bool, order string) (jsontext.Value, error) {
	owner := optionalUUID(options.Owner)
	searchTerms := options.SearchTerms
	workspaceProjection := workspaceJSON(false)
	if includeRole {
		workspaceProjection = workspaceJSON(true)
	}
	query := fmt.Sprintf(`
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
)
SELECT COALESCE(json_agg(item ORDER BY %s), '[]'::json)
FROM (
  SELECT %s AS item, workspace.created_at, workspace.name
  FROM workspaces workspace
  JOIN workspace_members membership
    ON membership.workspace_id = workspace.id
   AND membership.member_id = $1
   AND membership.is_active = true
   AND membership.deleted_at IS NULL
  CROSS JOIN requester
  WHERE workspace.deleted_at IS NULL
    AND requester.user_timezone IS NOT NULL
    AND ($2::uuid IS NULL OR workspace.owner_id = $2)
    AND NOT EXISTS (SELECT 1 FROM unnest($3::text[]) AS search_term WHERE NOT (UPPER(workspace.name::text) LIKE UPPER('%%' || search_term || '%%') ESCAPE '\'))
  GROUP BY workspace.id
) workspace_rows
`, order, workspaceProjection)
	var body []byte
	if err := repository.database.QueryRow(ctx, query, userID, owner, searchTerms).Scan(&body); err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) Workspace(ctx context.Context, userID uuid.UUID, slug string, options WorkspaceOptions) (jsontext.Value, error) {
	var body []byte
	err := repository.database.QueryRow(ctx, `
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
)
SELECT `+workspaceJSON(false)+`
FROM requester
JOIN workspaces workspace ON true
JOIN workspace_members membership
  ON membership.workspace_id = workspace.id
 AND membership.member_id = $1
 AND membership.is_active = true
 AND membership.deleted_at IS NULL
WHERE workspace.slug = $2
  AND requester.user_timezone IS NOT NULL
  AND workspace.deleted_at IS NULL
  AND ($3::uuid IS NULL OR workspace.owner_id = $3)
  AND NOT EXISTS (SELECT 1 FROM unnest($4::text[]) AS search_term WHERE NOT (UPPER(workspace.name::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\'))
GROUP BY workspace.id
LIMIT 1
`, userID, slug, optionalUUID(options.Owner), options.SearchTerms).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) ProjectStats(ctx context.Context, userID uuid.UUID, slug string, fields []string, projectIDsValue *string) (jsontext.Value, error) {
	if _, err := repository.workspaceRole(ctx, userID, slug); err != nil {
		return nil, err
	}
	projectIDs, err := parseProjectIDs(projectIDsValue)
	if err != nil {
		return nil, fmt.Errorf("%w: Please provide valid detail", ErrBadRequest)
	}
	selected := selectedStatFields(fields)
	properties := []string{"'id', project.id"}
	for _, field := range selected {
		properties = append(properties, fmt.Sprintf("'%s', %s", field, projectStatExpression(field)))
	}
	filterIDs := len(projectIDs) > 0
	query := `
SELECT COALESCE(json_agg(json_build_object(` + strings.Join(properties, ", ") + `) ORDER BY project.created_at DESC), '[]'::json)
FROM projects project
JOIN workspaces workspace ON workspace.id = project.workspace_id
WHERE workspace.slug = $1
  AND workspace.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND (NOT $2 OR project.id = ANY($3::uuid[]))
`
	var body []byte
	if err := repository.database.QueryRow(ctx, query, slug, filterIDs, projectIDs).Scan(&body); err != nil {
		return nil, fmt.Errorf("get project stats: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) Projects(ctx context.Context, userID uuid.UUID, slug string) (jsontext.Value, error) {
	role, err := repository.workspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	var body []byte
	err = repository.database.QueryRow(ctx, `
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
)
SELECT COALESCE(json_agg(item ORDER BY created_at ASC), '[]'::json)
FROM (
  SELECT json_build_object(
    'id', project.id,
    'name', project.name,
    'identifier', project.identifier,
    'sort_order', (SELECT property.sort_order FROM project_user_properties property WHERE property.project_id=project.id AND property.workspace_id=project.workspace_id AND property.user_id=$1 AND property.deleted_at IS NULL ORDER BY property.created_at DESC LIMIT 1),
    'logo_props', project.logo_props,
    'member_role', (SELECT member.role FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1),
    'intake_count', (SELECT count(*) FROM intake_issues intake WHERE intake.project_id=project.id AND intake.status=-2 AND intake.deleted_at IS NULL),
    'archived_at', `+djangoTimestamp("project.archived_at")+`,
    'workspace', project.workspace_id,
    'cycle_view', project.cycle_view,
    'issue_views_view', project.issue_views_view,
    'module_view', project.module_view,
    'page_view', project.page_view,
    'inbox_view', project.intake_view,
    'guest_view_all_features', project.guest_view_all_features,
    'project_lead', project.project_lead_id,
    'network', project.network,
    'created_at', `+djangoTimestamp("project.created_at")+`,
    'updated_at', `+djangoTimestamp("project.updated_at")+`,
    'created_by', project.created_by_id,
    'updated_by', project.updated_by_id
  ) AS item, project.created_at
  FROM projects project
  JOIN workspaces workspace ON workspace.id=project.workspace_id
  CROSS JOIN requester
  WHERE workspace.slug=$2
    AND requester.user_timezone IS NOT NULL
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND (
      $3=20
      OR ($3=15 AND (project.network=2 OR EXISTS (SELECT 1 FROM project_members visible_member WHERE visible_member.project_id=project.id AND visible_member.workspace_id=project.workspace_id AND visible_member.member_id=$1 AND visible_member.is_active=true AND visible_member.deleted_at IS NULL)))
      OR ($3=5 AND EXISTS (SELECT 1 FROM project_members visible_guest WHERE visible_guest.project_id=project.id AND visible_guest.workspace_id=project.workspace_id AND visible_guest.member_id=$1 AND visible_guest.is_active=true AND visible_guest.deleted_at IS NULL))
    )
) project_rows
`, userID, slug, role).Scan(&body)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) ProjectDetails(ctx context.Context, userID uuid.UUID, slug string, options ProjectDetailsOptions) (jsontext.Value, error) {
	role, err := repository.workspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	paginate := options.PerPage != nil && *options.PerPage != "" && options.Cursor != nil && *options.Cursor != ""
	if !paginate {
		return repository.plainProjectDetails(ctx, userID, slug, role)
	}
	perPage, cursorPage, err := parsePagination(*options.PerPage, *options.Cursor)
	if err != nil {
		return nil, err
	}
	orderBy := "-created_at"
	if options.OrderBy != nil {
		orderBy = *options.OrderBy
	}
	orderSQL := sanitizedProjectOrder(orderBy)
	return repository.paginatedProjectDetails(ctx, userID, slug, role, perPage, cursorPage, orderSQL)
}

func (repository *Repository) plainProjectDetails(ctx context.Context, userID uuid.UUID, slug string, role int) (jsontext.Value, error) {
	query := projectDetailRowsSQL + `
SELECT COALESCE(json_agg(item ORDER BY sort_order ASC NULLS LAST, name ASC), '[]'::json)
FROM project_rows
`
	var body []byte
	if err := repository.database.QueryRow(ctx, query, userID, slug, role).Scan(&body); err != nil {
		return nil, fmt.Errorf("list detailed projects: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) paginatedProjectDetails(ctx context.Context, userID uuid.UUID, slug string, role, perPage, page int, orderSQL string) (jsontext.Value, error) {
	offset := page * perPage
	query := projectDetailRowsSQL + fmt.Sprintf(`,
ordered_rows AS (
  SELECT item, row_number() OVER (ORDER BY %s, created_at DESC) AS ordinal
  FROM project_rows
),
page_rows AS (
  SELECT item, ordinal FROM ordered_rows ORDER BY ordinal LIMIT $4 OFFSET $5
),
totals AS (SELECT count(*)::int AS total FROM project_rows)
SELECT json_build_object(
  'grouped_by', NULL,
  'sub_grouped_by', NULL,
  'total_count', total,
  'next_cursor', $4::text || ':' || ($6 + 1)::text || ':0',
  'prev_cursor', $4::text || ':' || ($6 - 1)::text || ':1',
  'next_page_results', total > $5 + $4,
  'prev_page_results', $6 > 0,
  'count', (SELECT count(*)::int FROM page_rows),
  'total_pages', CASE WHEN total=0 THEN 0 ELSE ceil(total::numeric / $4)::int END,
  'total_results', total,
  'extra_stats', NULL,
  'results', COALESCE((SELECT json_agg(item ORDER BY ordinal) FROM page_rows), '[]'::json)
)
FROM totals
`, orderSQL)
	var body []byte
	if err := repository.database.QueryRow(ctx, query, userID, slug, role, perPage, offset, page).Scan(&body); err != nil {
		return nil, fmt.Errorf("paginate detailed projects: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) Project(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (jsontext.Value, error) {
	if _, err := repository.workspaceRole(ctx, userID, slug); err != nil {
		return nil, err
	}
	var body []byte
	var network int
	var member bool
	err := repository.database.QueryRow(ctx, `
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
)
SELECT `+projectJSON+`, project.network,
  EXISTS(SELECT 1 FROM project_members membership WHERE membership.project_id=project.id AND membership.workspace_id=project.workspace_id AND membership.member_id=$1 AND membership.is_active=true AND membership.deleted_at IS NULL)
FROM requester
JOIN projects project ON true
JOIN workspaces workspace ON workspace.id=project.workspace_id
WHERE workspace.slug=$2
  AND requester.user_timezone IS NOT NULL
  AND workspace.deleted_at IS NULL
  AND project.id=$3
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
LIMIT 1
`, userID, slug, projectID).Scan(&body, &network, &member)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if !member {
		if network == 0 {
			return nil, ErrProjectForbidden
		}
		return nil, ErrProjectConflict
	}
	if err := repository.admitRecentVisit(ctx, userID, slug, projectID); err != nil {
		return nil, fmt.Errorf("record recent project visit: %w", err)
	}
	return jsontext.Value(body), nil
}

func (repository *Repository) workspaceRole(ctx context.Context, userID uuid.UUID, slug string) (int, error) {
	var role int
	err := repository.database.QueryRow(ctx, `
SELECT membership.role
FROM workspace_members membership
JOIN workspaces workspace ON workspace.id=membership.workspace_id
WHERE membership.member_id=$1
  AND workspace.slug=$2
  AND membership.role IN (5,15,20)
  AND membership.is_active=true
  AND membership.deleted_at IS NULL
  AND workspace.deleted_at IS NULL
ORDER BY membership.created_at DESC
LIMIT 1
`, userID, slug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	if err != nil {
		return 0, fmt.Errorf("authorize workspace project read: %w", err)
	}
	return role, nil
}

func djangoTimestamp(column string) string {
	return djangoapi.DjangoTimestampSQL(column)
}

func workspaceJSON(includeRole bool) string {
	role := ""
	if includeRole {
		role = ", 'role', (SELECT current_membership.role FROM workspace_members current_membership WHERE current_membership.workspace_id=workspace.id AND current_membership.member_id=$1 AND current_membership.is_active=true AND current_membership.deleted_at IS NULL ORDER BY current_membership.created_at DESC LIMIT 1)"
	}
	return `json_build_object(
  'background_color', workspace.background_color,
  'created_at', ` + djangoTimestamp("workspace.created_at") + `,
  'created_by', workspace.created_by_id,
  'deleted_at', ` + djangoTimestamp("workspace.deleted_at") + `,
  'id', workspace.id,
  'logo', workspace.logo,
  'logo_asset', workspace.logo_asset_id,
  'logo_url', CASE
    WHEN workspace.logo_asset_id IS NULL THEN NULLIF(workspace.logo,'')
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=workspace.logo_asset_id) IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER')
      THEN '/api/assets/v2/static/' || workspace.logo_asset_id::text || '/'
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=workspace.logo_asset_id)='ISSUE_ATTACHMENT'
      THEN (SELECT '/api/assets/v2/workspaces/' || asset_workspace.slug || '/projects/' || COALESCE(asset.project_id::text,'None') || '/issues/' || COALESCE(asset.issue_id::text,'None') || '/attachments/' || asset.id::text || '/' FROM file_assets asset LEFT JOIN workspaces asset_workspace ON asset_workspace.id=asset.workspace_id WHERE asset.id=workspace.logo_asset_id)
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=workspace.logo_asset_id) IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION')
      THEN (SELECT '/api/assets/v2/workspaces/' || asset_workspace.slug || '/projects/' || COALESCE(asset.project_id::text,'None') || '/' || asset.id::text || '/' FROM file_assets asset LEFT JOIN workspaces asset_workspace ON asset_workspace.id=asset.workspace_id WHERE asset.id=workspace.logo_asset_id)
    ELSE NULL
  END,
  'name', workspace.name,
  'organization_size', workspace.organization_size,
  'owner', workspace.owner_id` + role + `,
  'slug', workspace.slug,
  'timezone', workspace.timezone,
  'total_members', (SELECT count(*) FROM workspace_members counted_membership JOIN users counted_user ON counted_user.id=counted_membership.member_id WHERE counted_membership.workspace_id=workspace.id AND counted_membership.is_active=true AND counted_membership.deleted_at IS NULL AND counted_user.is_bot=false),
  'updated_at', ` + djangoTimestamp("workspace.updated_at") + `,
  'updated_by', workspace.updated_by_id
)`
}

var projectJSON = `json_build_object(
  'anchor', (SELECT board.anchor FROM deploy_boards board WHERE board.entity_name='project' AND board.entity_identifier=project.id AND board.workspace_id=project.workspace_id AND board.deleted_at IS NULL ORDER BY board.created_at DESC LIMIT 1),
  'archive_in', project.archive_in,
  'archived_at', ` + djangoTimestamp("project.archived_at") + `,
  'close_in', project.close_in,
  'cover_image', project.cover_image,
  'cover_image_asset', project.cover_image_asset_id,
  'cover_image_url', CASE
    WHEN project.cover_image_asset_id IS NULL THEN NULLIF(project.cover_image,'')
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=project.cover_image_asset_id) IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER')
      THEN '/api/assets/v2/static/' || project.cover_image_asset_id::text || '/'
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=project.cover_image_asset_id)='ISSUE_ATTACHMENT'
      THEN (SELECT '/api/assets/v2/workspaces/' || asset_workspace.slug || '/projects/' || COALESCE(asset.project_id::text,'None') || '/issues/' || COALESCE(asset.issue_id::text,'None') || '/attachments/' || asset.id::text || '/' FROM file_assets asset LEFT JOIN workspaces asset_workspace ON asset_workspace.id=asset.workspace_id WHERE asset.id=project.cover_image_asset_id)
    WHEN (SELECT asset.entity_type FROM file_assets asset WHERE asset.id=project.cover_image_asset_id) IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION')
      THEN (SELECT '/api/assets/v2/workspaces/' || asset_workspace.slug || '/projects/' || COALESCE(asset.project_id::text,'None') || '/' || asset.id::text || '/' FROM file_assets asset LEFT JOIN workspaces asset_workspace ON asset_workspace.id=asset.workspace_id WHERE asset.id=project.cover_image_asset_id)
    ELSE NULL
  END,
  'created_at', ` + djangoTimestamp("project.created_at") + `,
  'created_by', project.created_by_id,
  'cycle_view', project.cycle_view,
  'default_assignee', project.default_assignee_id,
  'default_state', project.default_state_id,
  'deleted_at', ` + djangoTimestamp("project.deleted_at") + `,
  'description', project.description,
  'description_html', project.description_html,
  'description_text', project.description_text,
  'emoji', project.emoji,
  'estimate', project.estimate_id,
  'external_id', project.external_id,
  'external_source', project.external_source,
  'guest_view_all_features', project.guest_view_all_features,
  'icon_prop', project.icon_prop,
  'id', project.id,
  'identifier', project.identifier,
  'inbox_view', project.intake_view,
  'intake_view', project.intake_view,
  'is_favorite', EXISTS(SELECT 1 FROM user_favorites favorite WHERE favorite.user_id=$1 AND favorite.entity_identifier=project.id AND favorite.entity_type='project' AND favorite.project_id=project.id AND favorite.deleted_at IS NULL),
  'is_issue_type_enabled', project.is_issue_type_enabled,
  'is_time_tracking_enabled', project.is_time_tracking_enabled,
  'issue_views_view', project.issue_views_view,
  'logo_props', project.logo_props,
  'member_role', (SELECT membership.role FROM project_members membership WHERE membership.project_id=project.id AND membership.workspace_id=project.workspace_id AND membership.member_id=$1 AND membership.is_active=true AND membership.deleted_at IS NULL ORDER BY membership.created_at DESC LIMIT 1),
  'members', (SELECT COALESCE(json_agg(membership.member_id ORDER BY membership.created_at DESC), '[]'::json) FROM project_members membership JOIN users member_user ON member_user.id=membership.member_id WHERE membership.project_id=project.id AND membership.workspace_id=project.workspace_id AND membership.is_active=true AND membership.deleted_at IS NULL AND member_user.is_bot=false),
  'module_view', project.module_view,
  'name', project.name,
  'network', project.network,
  'next_work_item_sequence', COALESCE((SELECT max(issue_sequence.sequence)+1 FROM issue_sequences issue_sequence WHERE issue_sequence.project_id=project.id AND issue_sequence.deleted_at IS NULL), 1),
  'page_view', project.page_view,
  'project_lead', project.project_lead_id,
  'sort_order', (SELECT property.sort_order FROM project_user_properties property WHERE property.project_id=project.id AND property.workspace_id=project.workspace_id AND property.user_id=$1 AND property.deleted_at IS NULL ORDER BY property.created_at DESC LIMIT 1),
  'timezone', project.timezone,
  'updated_at', ` + djangoTimestamp("project.updated_at") + `,
  'updated_by', project.updated_by_id,
  'workspace', project.workspace_id
)`

var projectDetailRowsSQL = `
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
),
project_rows AS (
  SELECT ` + projectJSON + ` AS item,
    project.created_at, project.updated_at, project.name, project.network,
    (SELECT property.sort_order FROM project_user_properties property WHERE property.project_id=project.id AND property.workspace_id=project.workspace_id AND property.user_id=$1 AND property.deleted_at IS NULL ORDER BY property.created_at DESC LIMIT 1) AS sort_order
  FROM projects project
  JOIN workspaces workspace ON workspace.id=project.workspace_id
  CROSS JOIN requester
  WHERE workspace.slug=$2
    AND requester.user_timezone IS NOT NULL
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND (
      $3=20
      OR ($3=15 AND (project.network=2 OR EXISTS (SELECT 1 FROM project_members visible_member WHERE visible_member.project_id=project.id AND visible_member.workspace_id=project.workspace_id AND visible_member.member_id=$1 AND visible_member.is_active=true AND visible_member.deleted_at IS NULL)))
      OR ($3=5 AND EXISTS (SELECT 1 FROM project_members visible_guest WHERE visible_guest.project_id=project.id AND visible_guest.workspace_id=project.workspace_id AND visible_guest.member_id=$1 AND visible_guest.is_active=true AND visible_guest.deleted_at IS NULL))
    )
)`

func parseProjectIDs(value *string) ([]uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parts := strings.Split(*value, ",")
	values := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		projectID, err := uuid.Parse(part)
		if err != nil {
			return nil, err
		}
		values = append(values, projectID)
	}
	return values, nil
}

func selectedStatFields(fields []string) []string {
	allowed := []string{"total_issues", "completed_issues", "total_members", "total_cycles", "total_modules"}
	requested := make(map[string]bool, len(fields))
	for _, field := range fields {
		requested[field] = true
	}
	selected := make([]string, 0, len(allowed))
	for _, field := range allowed {
		if requested[field] {
			selected = append(selected, field)
		}
	}
	if len(selected) == 0 {
		return allowed
	}
	return selected
}

func projectStatExpression(field string) string {
	switch field {
	case "total_issues":
		return "(SELECT count(*) FROM issues issue LEFT JOIN states issue_state ON issue_state.id=issue.state_id WHERE issue.project_id=project.id AND issue.deleted_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false AND project.archived_at IS NULL AND (issue_state.\"group\" IS NULL OR issue_state.\"group\" <> 'triage'))"
	case "completed_issues":
		return "(SELECT count(*) FROM issues issue LEFT JOIN states issue_state ON issue_state.id=issue.state_id WHERE issue.project_id=project.id AND issue.deleted_at IS NULL AND issue.archived_at IS NULL AND issue.is_draft=false AND project.archived_at IS NULL AND (issue_state.\"group\" IS NULL OR issue_state.\"group\" <> 'triage') AND issue_state.\"group\" IN ('completed','cancelled'))"
	case "total_members":
		return "(SELECT count(*) FROM project_members member JOIN users member_user ON member_user.id=member.member_id WHERE member.project_id=project.id AND member.is_active=true AND member.deleted_at IS NULL AND member_user.is_bot=false)"
	case "total_cycles":
		return "(SELECT count(*) FROM cycles cycle WHERE cycle.project_id=project.id AND cycle.deleted_at IS NULL)"
	case "total_modules":
		return "(SELECT count(*) FROM modules module WHERE module.project_id=project.id AND module.deleted_at IS NULL)"
	default:
		panic("unreachable project stat field")
	}
}

func parsePagination(perPageValue, cursorValue string) (int, int, error) {
	perPage, err := strconv.Atoi(perPageValue)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: Invalid per_page parameter.", ErrBadRequest)
	}
	if perPage <= 0 {
		return 0, 0, fmt.Errorf("%w: Invalid per_page parameter.", ErrBadRequest)
	}
	if perPage > 1000 {
		return 0, 0, fmt.Errorf("%w: Invalid per_page value. Cannot exceed 1000.", ErrBadRequest)
	}
	parts := strings.Split(cursorValue, ":")
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("%w: Invalid cursor parameter.", ErrBadRequest)
	}
	_, limitError := strconv.ParseFloat(parts[0], 64)
	page, pageError := strconv.Atoi(parts[1])
	_, previousError := strconv.Atoi(parts[2])
	if limitError != nil || pageError != nil || previousError != nil {
		return 0, 0, fmt.Errorf("%w: Invalid cursor parameter.", ErrBadRequest)
	}
	if page < 0 {
		return 0, 0, fmt.Errorf("%w: Error in parsing", ErrBadRequest)
	}
	return perPage, page, nil
}

func sanitizedProjectOrder(value string) string {
	allowed := map[string]string{
		"created_at": "created_at ASC NULLS LAST", "-created_at": "created_at DESC NULLS LAST",
		"updated_at": "updated_at ASC NULLS LAST", "-updated_at": "updated_at DESC NULLS LAST",
		"name": "name ASC NULLS LAST", "-name": "name DESC NULLS LAST",
		"network": "network ASC NULLS LAST", "-network": "network DESC NULLS LAST",
		"sort_order": "sort_order ASC NULLS LAST", "-sort_order": "sort_order DESC NULLS LAST",
	}
	if order, ok := allowed[value]; ok {
		return order
	}
	return allowed["-created_at"]
}

func optionalUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}
