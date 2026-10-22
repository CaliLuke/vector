package workspaceprojectreads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"uuid"
)

// APIProjects implements the API-key lightweight project picker collection.
func (repository *Repository) APIProjects(ctx context.Context, userID uuid.UUID, slug string, options APIProjectListOptions) (json.RawMessage, error) {
	role, err := repository.apiWorkspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	query := djangoapi.Query{Params: map[string]string{}, Present: map[string]bool{}}
	if options.PerPage != nil {
		query.Params["per_page"], query.Present["per_page"] = *options.PerPage, true
	}
	if options.Cursor != nil {
		query.Params["cursor"], query.Present["cursor"] = *options.Cursor, true
	}
	limit, page, offset, err := djangoapi.PageWindow(query)
	if err != nil {
		return nil, err
	}
	orderSQL := "created_at DESC"
	if options.OrderBy != nil {
		orderSQL = apiProjectOrder(*options.OrderBy, orderSQL)
	}
	return repository.apiProjectLitePage(ctx, userID, slug, role, options, limit, page, offset, orderSQL)
}

// APIProjectList returns the full project serializer used by the API-key
// collection endpoint. Its visibility rules match the source queryset:
// active project members and workspace members may see public projects.
func (repository *Repository) APIProjectList(ctx context.Context, userID uuid.UUID, slug string, options APIProjectListOptions) (json.RawMessage, error) {
	role, err := repository.apiWorkspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	query := djangoapi.Query{Params: map[string]string{}, Present: map[string]bool{}}
	if options.PerPage != nil {
		query.Params["per_page"], query.Present["per_page"] = *options.PerPage, true
	}
	if options.Cursor != nil {
		query.Params["cursor"], query.Present["cursor"] = *options.Cursor, true
	}
	limit, page, offset, err := djangoapi.PageWindow(query)
	if err != nil {
		return nil, err
	}
	order := "sort_order ASC NULLS LAST"
	if options.OrderBy != nil {
		order = apiProjectOrder(*options.OrderBy, order)
	}
	// The source orders only by the requested field. A primary-key tie-breaker
	// below stabilizes rows without inventing a name ordering.
	return repository.apiProjectPage(ctx, userID, slug, role, limit, page, offset, order, options)
}

func (repository *Repository) apiProjectPage(ctx context.Context, userID uuid.UUID, slug string, role, limit, page, offset int, order string, options APIProjectListOptions) (json.RawMessage, error) {
	project := `(` + projectJSON + `)::jsonb - ARRAY['anchor','inbox_view','is_favorite','members','next_work_item_sequence']::text[] || jsonb_build_object(
  'is_member', EXISTS(SELECT 1 FROM project_members current_member WHERE current_member.project_id=project.id AND current_member.workspace_id=project.workspace_id AND current_member.member_id=$1 AND current_member.is_active=true AND current_member.deleted_at IS NULL),
  'total_members', (SELECT count(*) FROM project_members member JOIN users member_user ON member_user.id=member.member_id AND member_user.is_bot=false WHERE member.project_id=project.id AND member.is_active=true AND member.deleted_at IS NULL),
  'total_cycles', (SELECT count(*) FROM cycles cycle WHERE cycle.project_id=project.id AND cycle.deleted_at IS NULL),
  'total_modules', (SELECT count(*) FROM modules module WHERE module.project_id=project.id AND module.deleted_at IS NULL),
  'is_deployed', EXISTS(SELECT 1 FROM deploy_boards board WHERE board.entity_name='project' AND board.entity_identifier=project.id AND board.workspace_id=project.workspace_id AND board.deleted_at IS NULL),
  'sort_order', (SELECT member.sort_order FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1),
  'member_role', (SELECT member.role FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1)
)`
	query := fmt.Sprintf(`
WITH requester AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM users requester_user WHERE requester_user.id=$1 AND requester_user.is_active=true
), rows AS (
  SELECT %s AS item, project.id AS project_id, project.created_at, project.updated_at, project.name,
    (SELECT member.sort_order FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1) AS sort_order
  FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester
  WHERE workspace.slug=$2 AND workspace.deleted_at IS NULL AND requester.user_timezone IS NOT NULL
    AND project.deleted_at IS NULL
    AND $3 IN (5,15,20)
    AND (project.network=2 OR EXISTS (SELECT 1 FROM project_members visible WHERE visible.project_id=project.id AND visible.workspace_id=project.workspace_id AND visible.member_id=$1 AND visible.is_active=true AND visible.deleted_at IS NULL))
), ordered AS (SELECT item, row_number() OVER (ORDER BY %s, project_id ASC) AS ordinal FROM rows), totals AS (SELECT count(*)::int AS total FROM rows)
SELECT json_build_object(
  'grouped_by', NULL, 'sub_grouped_by', NULL, 'total_count', total,
  'next_cursor', $6::int::text || ':' || ($4+1)::text || ':0', 'prev_cursor', $6::int::text || ':' || ($4-1)::text || ':1',
  'next_page_results', total > $5+$6, 'prev_page_results', $4 > 0,
  'count', LEAST(GREATEST(total-$5,0),$6),
  'total_pages', CASE WHEN total=0 THEN 0 ELSE ceil(total::numeric/$6)::int END,
  'total_results', total, 'extra_stats', NULL,
  'results', COALESCE((SELECT json_agg(item ORDER BY ordinal) FROM ordered WHERE ordinal>$5 AND ordinal<=$5+$6),'[]'::json)
) FROM totals`, project, order)
	var body []byte
	if err := repository.database.QueryRow(ctx, query, userID, slug, role, page, offset, limit).Scan(&body); err != nil {
		return nil, fmt.Errorf("list API projects: %w", err)
	}
	return repository.projectProjection(ctx, body, options.Fields, options.Expand, true)
}

// APIProject retrieves one full-serializer project within the requested
// workspace, using the same visibility policy as APIProjectList.
func (repository *Repository) APIProject(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, options APIProjectDetailOptions) (json.RawMessage, error) {
	role, err := repository.apiWorkspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	project := `(` + projectJSON + `)::jsonb - ARRAY['anchor','inbox_view','is_favorite','members','next_work_item_sequence','sort_order']::text[] || jsonb_build_object(
  'is_member', EXISTS(SELECT 1 FROM project_members current_member WHERE current_member.project_id=project.id AND current_member.workspace_id=project.workspace_id AND current_member.member_id=$1 AND current_member.is_active=true AND current_member.deleted_at IS NULL),
  'total_members', (SELECT count(*) FROM project_members member JOIN users member_user ON member_user.id=member.member_id AND member_user.is_bot=false WHERE member.project_id=project.id AND member.is_active=true AND member.deleted_at IS NULL),
  'total_cycles', (SELECT count(*) FROM cycles cycle WHERE cycle.project_id=project.id AND cycle.deleted_at IS NULL),
  'total_modules', (SELECT count(*) FROM modules module WHERE module.project_id=project.id AND module.deleted_at IS NULL),
  'is_deployed', EXISTS(SELECT 1 FROM deploy_boards board WHERE board.entity_name='project' AND board.entity_identifier=project.id AND board.workspace_id=project.workspace_id AND board.deleted_at IS NULL),
  'member_role', (SELECT member.role FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1)
)`
	query := fmt.Sprintf(`WITH requester AS MATERIALIZED (
 SELECT set_config('TimeZone',COALESCE(NULLIF(user_timezone,''),'UTC'),true) FROM users WHERE id=$1 AND is_active=true
)
SELECT %s FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester
WHERE workspace.slug=$2 AND workspace.deleted_at IS NULL AND project.id=$3 AND project.deleted_at IS NULL
 AND requester.set_config IS NOT NULL
 AND $4 IN (5,15,20)
 AND (project.network=2 OR EXISTS (SELECT 1 FROM project_members visible WHERE visible.project_id=project.id AND visible.workspace_id=project.workspace_id AND visible.member_id=$1 AND visible.is_active=true AND visible.deleted_at IS NULL))
LIMIT 1`, project)
	var body []byte
	if err := repository.database.QueryRow(ctx, query, userID, slug, projectID, role).Scan(&body); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get API project: %w", err)
	}
	return repository.projectProjection(ctx, body, options.Fields, options.Expand, false)
}

func apiProjectOrder(value, fallback string) string {
	direction := "ASC"
	field := value
	if strings.HasPrefix(value, "-") {
		direction, field = "DESC", strings.TrimPrefix(value, "-")
	}
	if strings.HasPrefix(field, "-") {
		return fallback
	}
	columns := map[string]string{"created_at": "created_at", "updated_at": "updated_at", "name": "name", "network": "network", "sort_order": "sort_order"}
	column, ok := columns[field]
	if !ok {
		return fallback
	}
	// PostgreSQL's default NULL placement is the same behavior Django gets
	// from QuerySet.order_by: ASC NULLS LAST and DESC NULLS FIRST.
	return column + " " + direction
}

func (repository *Repository) apiProjectLitePage(ctx context.Context, userID uuid.UUID, slug string, _ int, options APIProjectListOptions, limit, page, offset int, order string) (json.RawMessage, error) {
	includeArchived := options.IncludeArchived
	query := fmt.Sprintf(`
WITH requester AS MATERIALIZED (
 SELECT set_config('TimeZone',COALESCE(NULLIF(user_timezone,''),'UTC'),true) AS user_timezone FROM users WHERE id=$1 AND is_active=true
), rows AS (
  SELECT json_build_object(
    'id', project.id, 'name', project.name, 'identifier', project.identifier,
    'description', project.description, 'emoji', project.emoji, 'icon_prop', project.icon_prop,
    'cover_image', project.cover_image, 'cover_image_url', %s,
    'archived_at', %s
  ) AS item, project.created_at, project.updated_at, project.name, project.network,
 (SELECT member.sort_order FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL ORDER BY member.created_at DESC LIMIT 1) AS sort_order
  FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester
  WHERE workspace.slug=$2 AND requester.user_timezone IS NOT NULL AND project.deleted_at IS NULL
    AND ($3 OR project.archived_at IS NULL)
    AND (project.network=2 OR EXISTS (SELECT 1 FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active=true AND member.deleted_at IS NULL))
), ordered AS (SELECT item, row_number() OVER (ORDER BY %s) AS ordinal FROM rows), totals AS (SELECT count(*)::int AS total FROM rows)
SELECT COALESCE((SELECT json_agg(item ORDER BY ordinal) FROM ordered WHERE ordinal > $4 AND ordinal <= $4+$5), '[]'::json), total FROM totals`, apiProjectCoverImageURL(), djangoTimestamp("project.archived_at"), order)
	var body []byte
	var total int
	if err := repository.database.QueryRow(ctx, query, userID, slug, includeArchived, offset, limit).Scan(&body, &total); err != nil {
		return nil, fmt.Errorf("list lite API projects: %w", err)
	}
	var rows []any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode lite API project list: %w", err)
	}
	return json.Marshal(djangoapi.Envelope(rows, total, limit, page))
}

// APIProjectSummary preserves the summary endpoint's distinct permission and
// visibility policy: workspace members and admins may read private projects,
// while guests may not.
func (repository *Repository) APIProjectSummary(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, options APIProjectSummaryOptions) (json.RawMessage, error) {
	role, err := repository.apiWorkspaceRole(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	if role == 5 {
		return nil, ErrForbidden
	}
	allowed := []string{"members", "states", "labels", "cycles", "modules", "issues", "intakes", "pages"}
	wanted := make(map[string]struct{}, len(options.Fields))
	for _, value := range options.Fields {
		for _, field := range strings.Split(value, ",") {
			field = strings.TrimSpace(field)
			if _, ok := summaryCountExpression(field); ok {
				wanted[field] = struct{}{}
			}
		}
	}
	if len(wanted) == 0 {
		for _, field := range allowed {
			wanted[field] = struct{}{}
		}
	}
	values := make([]string, 0, len(wanted)*2)
	for _, field := range allowed {
		if _, ok := wanted[field]; ok {
			values = append(values, fmt.Sprintf("'%s', %s", field, summaryCountExpressionMust(field)))
		}
	}
	query := `SELECT json_build_object('id', project.id, 'name', project.name, 'identifier', project.identifier, 'counts', json_build_object(` + strings.Join(values, ",") + `)) FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id WHERE workspace.slug=$1 AND project.id=$2 AND project.deleted_at IS NULL LIMIT 1`
	var body []byte
	if err := repository.database.QueryRow(ctx, query, slug, projectID).Scan(&body); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get API project summary: %w", err)
	}
	return json.RawMessage(body), nil
}

func (repository *Repository) apiWorkspaceRole(ctx context.Context, userID uuid.UUID, slug string) (int, error) {
	var role int
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(membership.role, 0)
FROM workspaces workspace
LEFT JOIN LATERAL (
  SELECT member.role FROM workspace_members member
  WHERE member.workspace_id=workspace.id AND member.member_id=$1
    AND member.is_active=true AND member.deleted_at IS NULL
    AND member.role IN (5,15,20)
  ORDER BY member.created_at DESC LIMIT 1
) membership ON true
WHERE workspace.slug=$2
LIMIT 1`, userID, slug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("authorize API project workspace: %w", err)
	}
	if role == 0 {
		return 0, ErrForbidden
	}
	return role, nil
}

func summaryCountExpression(field string) (string, bool) {
	values := map[string]string{
		"members": "(SELECT count(*) FROM project_members member WHERE member.project_id=project.id AND member.is_active=true AND member.deleted_at IS NULL)",
		"states":  "(SELECT count(*) FROM states state WHERE state.project_id=project.id AND state.deleted_at IS NULL AND state.\"group\" <> 'triage')",
		"labels":  "(SELECT count(*) FROM labels label WHERE label.project_id=project.id AND label.deleted_at IS NULL)",
		"cycles":  "(SELECT count(*) FROM cycles cycle WHERE cycle.project_id=project.id AND cycle.deleted_at IS NULL)",
		"modules": "(SELECT count(*) FROM modules module WHERE module.project_id=project.id AND module.deleted_at IS NULL)",
		"issues":  "(SELECT count(*) FROM issues issue LEFT JOIN states state ON state.id=issue.state_id WHERE issue.project_id=project.id AND issue.deleted_at IS NULL AND state.\"group\" IS DISTINCT FROM 'triage')",
		"intakes": "(SELECT count(*) FROM intake_issues intake WHERE intake.project_id=project.id AND intake.deleted_at IS NULL)",
		"pages":   "(SELECT count(*) FROM project_pages page WHERE page.project_id=project.id AND page.deleted_at IS NULL)",
	}
	expression, ok := values[field]
	return expression, ok
}

func summaryCountExpressionMust(field string) string {
	expression, ok := summaryCountExpression(field)
	if !ok {
		panic("unreachable summary field")
	}
	return expression
}

// AuthorizeAPIProjectRead runs the DRF permission stage before throttling and
// target lookup. Summary uses WorkSpaceAdminPermission; other reads use
// ProjectBasePermission's safe-method workspace membership policy.
func (repository *Repository) AuthorizeAPIProjectRead(ctx context.Context, userID uuid.UUID, slug string, summary bool) error {
	role, err := repository.apiWorkspaceRole(ctx, userID, slug)
	if errors.Is(err, ErrNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if summary && role != 15 && role != 20 {
		return ErrForbidden
	}
	return nil
}

func apiProjectCoverImageURL() string {
	start := strings.Index(projectJSON, "'cover_image_url', ") + len("'cover_image_url', ")
	end := strings.Index(projectJSON[start:], ",\n  'created_at'")
	return projectJSON[start : start+end]
}

// projectProjection reuses the API BaseSerializer field and relation contract.
func (repository *Repository) projectProjection(ctx context.Context, body json.RawMessage, fields, expand string, collection bool) (json.RawMessage, error) {
	if fields == "" && expand == "" {
		return body, nil
	}
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	project := func(row map[string]any) (map[string]any, error) {
		return djangoapi.Serialize(ctx, repository.database, row, slices.Sorted(maps.Keys(row)), fields, expand, false)
	}
	if collection {
		rows, ok := document["results"].([]any)
		if !ok {
			return nil, fmt.Errorf("API project collection has no results array")
		}
		for index, value := range rows {
			row, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("API project collection contains a non-object")
			}
			projected, err := project(row)
			if err != nil {
				return nil, err
			}
			rows[index] = projected
		}
	} else {
		var err error
		document, err = project(document)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(document)
}
