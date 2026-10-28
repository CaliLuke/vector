// Package modulereads contains the read-only module boundary.
package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden           = errors.New("active module membership with an allowed role required")
	ErrNotFound            = errors.New("module read model not found")
	ErrInvalidDetail       = errors.New("please provide valid detail")
	ErrInvalidComplexRange = issuequery.ErrInvalidComplexRange
)

type FilterError = issuequery.FilterError

type database interface {
	Begin(context.Context) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type taskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type RecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type Repository struct {
	database        database
	publisher       taskPublisher
	recentVisits    RecentVisitAdmitter
	propertyFilters *modulePropertyFilter
}

type IssueListOptions = issuequery.Options

type IssuePage struct {
	Items          json.RawMessage
	Total          int
	GroupValues    []any
	SubGroupValues []any
	GroupCounts    map[string]int
	SubGroupCounts map[string]int
	GroupedHasNext bool
	PageCount      int
}

// These unexported compatibility wrappers keep modulereads' focused tests and
// call sites stable while the pure implementations live in issuequery.
func buildIssueComplexFilter(raw string, args *[]any) (string, error) {
	return issuequery.BuildComplexFilter(raw, args)
}

func addIssueDateFilters(where []string, args []any, value, column string) ([]string, []any) {
	return issuequery.AddDateFilters(where, args, value, column)
}

func requesterDateSQL(column string) string {
	return issuequery.RequesterDateSQL(column)
}

func validUUIDFilter(value string, allowNone bool) (string, bool) {
	return issuequery.ValidUUIDFilter(value, allowNone)
}

func issueFilterUUIDs(values []string, field string) ([]string, error) {
	return issuequery.ParseUUIDFilterValues(values, field)
}

func filterNullTokens(value string) string {
	return issuequery.FilterNullTokens(value)
}

func escapeLikePattern(value string) string {
	return issuequery.EscapeLikePattern(value)
}

func normalizeLegacyUUIDList(value string) (string, bool) {
	return issuequery.NormalizeLegacyUUIDList(value)
}

func normalizeLegacyIntegerList(value string) (string, bool) {
	return issuequery.NormalizeLegacyIntegerList(value)
}

func validLegacyDateBounds(value string) bool {
	return issuequery.ValidLegacyDateBounds(value)
}

func moduleIssueOrder(value string, present bool) string {
	return issuequery.LegacyOrder(value, present)
}

func moduleIssueOrderWithRelations(options IssueListOptions, assignees, labels, modules string, args *[]any) (string, error) {
	return issuequery.ModuleOrder(options, assignees, labels, modules, args)
}

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("module reads database is required")
	}
	return &Repository{database: database}, nil
}

func NewRepositoryWithRecentVisits(database database, publisher taskPublisher, recentVisits RecentVisitAdmitter) (*Repository, error) {
	if database == nil || publisher == nil {
		return nil, fmt.Errorf("module read database and task publisher are required")
	}
	return &Repository{database: database, publisher: publisher, recentVisits: recentVisits}, nil
}

func (repository *Repository) admitRecentVisit(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID) error {
	if repository.recentVisits != nil {
		return repository.recentVisits.AdmitRecentVisit(ctx, uuid.New().String(), slug, "module", moduleID.String(), userID.String(), projectID.String())
	}
	if repository.publisher == nil {
		return nil
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{
		"slug": slug, "entity_name": "module", "entity_identifier": moduleID.String(), "user_id": userID.String(), "project_id": projectID.String(),
	})
}

func (repository *Repository) WorkspaceList(ctx context.Context, userID uuid.UUID, slug string) (json.RawMessage, error) {
	if err := repository.authorizeWorkspace(ctx, userID, slug); err != nil {
		return nil, err
	}
	return repository.readArray(ctx, `
SELECT COALESCE(json_agg(module_row.item ORDER BY module_row.created_at DESC), '[]'::json)
FROM (
  SELECT `+moduleJSON("module", "workspace")+` AS item, module.created_at
  FROM modules module
  JOIN workspaces workspace ON workspace.id = module.workspace_id
  JOIN projects project ON project.id = module.project_id
  WHERE workspace.slug = $2
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND project.archived_at IS NULL
    AND module.deleted_at IS NULL
    AND module.archived_at IS NULL
    AND EXISTS (
      SELECT 1 FROM project_members project_member
      WHERE project_member.project_id = project.id
        AND project_member.workspace_id = workspace.id
        AND project_member.member_id = $1
        AND project_member.is_active = true
        AND project_member.deleted_at IS NULL
    )
) module_row
`, "list workspace modules", userID, slug)
}

func (repository *Repository) ArchivedList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.moduleList(ctx, userID, slug, projectID, true, false)
}

func (repository *Repository) ArchivedDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	data, err := repository.moduleDetail(ctx, userID, slug, projectID, moduleID, true)
	if errors.Is(err, ErrNotFound) {
		return json.RawMessage(`{"distribution":{"assignees":[],"completion_chart":{},"labels":[]},"estimate_distribution":{},"member_ids":[]}`), nil
	}
	return data, err
}

func (repository *Repository) List(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, serializer bool) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.moduleList(ctx, userID, slug, projectID, false, serializer)
}

// ArchiveDetail is the corrected GET behavior for the active-path archive URL.
// Django's route supplies module_id, so this method deliberately scopes that ID
// against the archived module queryset instead of relying on a pk keyword.
func (repository *Repository) ArchiveDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.moduleDetail(ctx, userID, slug, projectID, moduleID, true)
}

func (repository *Repository) issueResults(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID, options IssueListOptions, typeIDs map[string]bool) (IssuePage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, true); err != nil {
		return IssuePage{}, err
	}
	estimatePoints, validEstimatePoints := normalizeLegacyUUIDList(options.EstimatePoint)
	if !validEstimatePoints {
		return IssuePage{}, ErrInvalidDetail
	}
	for _, filter := range []string{options.CreatedAt, options.UpdatedAt, options.StartDate, options.TargetDate, options.CompletedAt} {
		if !validLegacyDateBounds(filter) {
			return IssuePage{}, ErrInvalidDetail
		}
	}
	intakeStatus, validIntakeStatus := normalizeLegacyIntegerList(options.IntakeStatus)
	inboxStatus, validInboxStatus := normalizeLegacyIntegerList(options.InboxStatus)
	if !validIntakeStatus || !validInboxStatus {
		return IssuePage{}, fmt.Errorf("invalid intake status")
	}
	where := []string{
		"workspace.slug = $2",
		"workspace.deleted_at IS NULL",
		"project.id = $3",
		"project.deleted_at IS NULL",
		"project.archived_at IS NULL",
		"module.id = $4",
		"module.deleted_at IS NULL",
		"module_issue.deleted_at IS NULL",
		"issue.deleted_at IS NULL",
		"issue.archived_at IS NULL",
		"issue.is_draft = false",
		`(issue.state_id IS NULL OR EXISTS (SELECT 1 FROM states base_state WHERE base_state.id = issue.state_id AND base_state."group" <> 'triage'))`,
	}
	args := []any{userID, slug, projectID, moduleID}
	structured := options.Filters
	if repository.propertyFilters != nil {
		structured = ""
	}
	complexFilter, err := issuequery.BuildComplexFilterWithTypes(structured, &args, typeIDs)
	if err != nil {
		return IssuePage{}, err
	}
	if complexFilter != "" {
		where = append(where, complexFilter)
	}
	add := func(value, clause string) {
		if value == "" {
			return
		}
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	states, _ := validUUIDFilter(options.State, false)
	for _, relation := range []struct {
		present bool
		table   string
	}{
		{options.AssigneesSet, "issue_assignees"},
		{options.CycleSet, "cycle_issues"},
		{options.ModuleSet, "module_issues"},
		{options.SubscriberSet, "issue_subscribers"},
	} {
		if relation.present {
			where = append(where, "(NOT EXISTS (SELECT 1 FROM "+relation.table+" presence_relation WHERE presence_relation.issue_id = issue.id) OR EXISTS (SELECT 1 FROM "+relation.table+" presence_relation WHERE presence_relation.issue_id = issue.id AND presence_relation.deleted_at IS NULL))")
		}
	}
	add(states, "issue.state_id::text = ANY(string_to_array($%d, ','))")
	add(filterNullTokens(options.StateGroup), `EXISTS (SELECT 1 FROM states filter_state WHERE filter_state.id = issue.state_id AND filter_state."group" = ANY(string_to_array($%d, ',')))`)
	add(estimatePoints, "issue.estimate_point_id::text = ANY(string_to_array($%d, ','))")
	add(filterNullTokens(options.Priority), "issue.priority = ANY(string_to_array($%d, ','))")
	assignees, unassigned := validUUIDFilter(options.Assignees, true)
	if unassigned {
		where = append(where, "NOT EXISTS (SELECT 1 FROM issue_assignees filter_assignee WHERE filter_assignee.issue_id = issue.id)")
	}
	add(assignees, "EXISTS (SELECT 1 FROM issue_assignees filter_assignee WHERE filter_assignee.issue_id = issue.id AND filter_assignee.deleted_at IS NULL AND filter_assignee.assignee_id::text = ANY(string_to_array($%d, ',')))")
	labels, unlabeled := validUUIDFilter(options.Labels, true)
	if unlabeled {
		where = append(where, "NOT EXISTS (SELECT 1 FROM issue_labels filter_label WHERE "+issuequery.LiveIssueLabelPredicate("issue", "filter_label")+")")
	}
	add(labels, "EXISTS (SELECT 1 FROM issue_labels filter_label WHERE "+issuequery.LiveIssueLabelPredicate("issue", "filter_label")+" AND filter_label.label_id::text = ANY(string_to_array($%d, ',')))")
	projects, _ := validUUIDFilter(options.Project, false)
	add(projects, "issue.project_id::text = ANY(string_to_array($%d, ','))")
	loggedBy, anonymousLogger := validUUIDFilter(options.LoggedBy, true)
	if anonymousLogger {
		where = append(where, "issue.logged_by_id IS NULL")
	}
	add(loggedBy, "issue.logged_by_id::text = ANY(string_to_array($%d, ','))")
	creators, anonymousCreator := validUUIDFilter(options.CreatedBy, true)
	if anonymousCreator {
		where = append(where, "issue.created_by_id IS NULL")
	}
	add(creators, "issue.created_by_id::text = ANY(string_to_array($%d, ','))")
	parents, rootIssues := validUUIDFilter(options.Parent, true)
	if rootIssues {
		where = append(where, "issue.parent_id IS NULL")
	}
	add(parents, "issue.parent_id::text = ANY(string_to_array($%d, ','))")
	mentions, _ := validUUIDFilter(options.Mentions, false)
	add(mentions, "EXISTS (SELECT 1 FROM issue_mentions filter_mention WHERE filter_mention.issue_id = issue.id AND filter_mention.mention_id::text = ANY(string_to_array($%d, ',')))")
	if options.Name != "" {
		add("%"+escapeLikePattern(options.Name)+"%", `issue.name ILIKE $%d ESCAPE '\'`)
	}
	where, args = addIssueDateFilters(where, args, options.CreatedAt, requesterDateSQL("issue.created_at"))
	where, args = addIssueDateFilters(where, args, options.UpdatedAt, requesterDateSQL("issue.updated_at"))
	where, args = addIssueDateFilters(where, args, options.StartDate, "issue.start_date")
	where, args = addIssueDateFilters(where, args, options.TargetDate, "issue.target_date")
	where, args = addIssueDateFilters(where, args, options.CompletedAt, requesterDateSQL("issue.completed_at"))
	cycles, noCycle := validUUIDFilter(options.Cycle, true)
	if noCycle {
		where = append(where, "NOT EXISTS (SELECT 1 FROM cycle_issues filter_cycle WHERE filter_cycle.issue_id = issue.id)")
	}
	add(cycles, "EXISTS (SELECT 1 FROM cycle_issues filter_cycle WHERE filter_cycle.issue_id = issue.id AND filter_cycle.deleted_at IS NULL AND filter_cycle.cycle_id::text = ANY(string_to_array($%d, ',')))")
	modules, noModule := validUUIDFilter(options.Module, true)
	if noModule {
		where = append(where, "NOT EXISTS (SELECT 1 FROM module_issues filter_module_relation WHERE filter_module_relation.issue_id = issue.id)")
	}
	add(modules, "EXISTS (SELECT 1 FROM module_issues filter_module_relation WHERE filter_module_relation.issue_id = issue.id AND filter_module_relation.deleted_at IS NULL AND filter_module_relation.module_id::text = ANY(string_to_array($%d, ',')))")
	if inboxStatus != "" {
		intakeStatus = inboxStatus
	}
	add(intakeStatus, "EXISTS (SELECT 1 FROM intake_issues filter_intake WHERE filter_intake.issue_id = issue.id AND filter_intake.status::text = ANY(string_to_array($%d, ',')))")
	if options.SubIssue == "false" {
		where = append(where, "issue.parent_id IS NULL")
	}
	subscribers, _ := validUUIDFilter(options.Subscriber, false)
	add(subscribers, "EXISTS (SELECT 1 FROM issue_subscribers filter_subscriber WHERE filter_subscriber.issue_id = issue.id AND filter_subscriber.deleted_at IS NULL AND filter_subscriber.subscriber_id::text = ANY(string_to_array($%d, ',')))")
	if options.StartTargetDate == "true" {
		where = append(where, "issue.start_date IS NOT NULL", "issue.target_date IS NOT NULL")
	}
	if options.TypeSet {
		switch options.Type {
		case "backlog":
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" = 'backlog')`)
		case "active":
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" IN ('unstarted', 'started'))`)
		default:
			where = append(where, `EXISTS (SELECT 1 FROM states type_state WHERE type_state.id = issue.state_id AND type_state."group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled'))`)
		}
	}

	if repository.propertyFilters != nil {
		bound, bindErr := repository.bindPropertyFilters(ctx, where, args)
		if bindErr != nil {
			return IssuePage{}, bindErr
		}
		clause, filterErr := bound.SQL(&args)
		if filterErr != nil {
			return IssuePage{}, filterErr
		}
		if clause != "" {
			where = append(where, clause)
		}
		options.PropertyFilter = bound
	}
	order, err := moduleIssueOrderWithRelations(options, assignees, labels, modules, &args)
	if err != nil {
		return IssuePage{}, err
	}
	query, args := moduleIssuePageSQL(options, where, order, args)
	return repository.readIssuePage(ctx, query, args)
}

// moduleIssuePageSQL applies ordering, grouping and pagination to the selected
// issues. Filtering and property evaluation must be complete before this phase.
func moduleIssuePageSQL(options IssueListOptions, where []string, order string, args []any) (string, []any) {
	limit := options.Limit
	if limit > 1000 {
		limit = 1000
	}
	page := options.Page
	if page < 0 {
		page = 0
	}

	issueProjection := issueJSON
	if options.GroupBy != "" && (options.GroupBy == "issue_module__module_id" || options.SubGroupBy == "issue_module__module_id") {
		issueProjection = `(` + issueProjection + `)::jsonb || jsonb_build_object(
		  '_module_group_ids',
		  COALESCE((SELECT jsonb_agg(group_module.module_id) FROM module_issues group_module
		    WHERE group_module.issue_id = issue.id AND group_module.deleted_at IS NULL), '[]'::jsonb)
		)`
	}
	if options.GroupBy != "" {
		issueProjection = `(` + issueProjection + `)::jsonb || jsonb_build_object(
		  '_count_for_pages',
		  (
		    NOT EXISTS (
		      SELECT 1 FROM intake_issues count_intake
		      WHERE count_intake.issue_id = issue.id
		    )
		    OR EXISTS (
		      SELECT 1 FROM intake_issues count_intake
		      WHERE count_intake.issue_id = issue.id
		        AND count_intake.status IN (-1, 1, 2)
		    )
		  )
		)`
	}
	ordinaryAxis := ""
	if options.GroupBy == "" {
		if arrayField := moduleIssueArrayField(options.SubGroupBy); arrayField != "" {
			ordinaryAxis = moduleIssueAxisSQL(options.SubGroupBy, "ungrouped_subgroup_axis", "subgroup_key")
			issueProjection = `((` + issueProjection + `)::jsonb - '` + arrayField + `') || jsonb_build_object(
			  '` + options.SubGroupBy + `', NULLIF(ungrouped_subgroup_axis.subgroup_key, 'None')
			)`
		}
	}
	pageBaseName := "filtered"
	pageBaseCTE := ""
	if ordinaryAxis != "" && options.SubGroupBy != "labels__id" {
		pageBaseName = "subgroup_page_base"
		pageBaseCTE = `,
subgroup_page_base AS (
  SELECT issue.*
  FROM filtered issue
  WHERE NOT EXISTS (
    SELECT 1 FROM ` + moduleIssueRelationTable(options.SubGroupBy) + ` page_relation
    WHERE page_relation.issue_id = issue.id
  ) OR EXISTS (
    SELECT 1 FROM ` + moduleIssueRelationTable(options.SubGroupBy) + ` page_relation
    WHERE page_relation.issue_id = issue.id AND page_relation.deleted_at IS NULL
	  )
)`
	}
	queryLimit := any(limit)
	queryOffset := any(page * limit)
	if options.GroupBy != "" {
		window := options.CursorValue
		if !options.CursorSet {
			window = 0
		}
		groupOffset := float64(page) * window
		if window == 0 {
			window = float64(limit)
		}
		queryOffset = groupOffset
		queryLimit = groupOffset + window + 1
	}
	args = append(args, queryLimit, queryOffset)
	limitParameter := len(args) - 1
	offsetParameter := len(args)
	filteredSQL := `
WITH filtered AS (
  SELECT issue.*
  ` + moduleIssueFromSQL + `
  WHERE ` + strings.Join(where, "\n    AND ") + `
)` + pageBaseCTE
	query := filteredSQL + `,
matching AS (
  SELECT ` + issueProjection + ` AS item,
    row_number() OVER (ORDER BY ` + order + `, issue.created_at DESC) AS position
  FROM filtered issue
  ` + ordinaryAxis + `
  ORDER BY ` + order + `, issue.created_at DESC
  LIMIT $` + fmt.Sprint(limitParameter) + ` OFFSET $` + fmt.Sprint(offsetParameter) + `
)
SELECT COALESCE(json_agg(matching.item ORDER BY matching.position), '[]'::json),
  (SELECT count(*) FROM filtered),
  ` + moduleIssueGroupUniverseSQL(options.GroupBy) + `,
  ` + moduleIssueGroupUniverseSQL(options.SubGroupBy) + `,
  '{}'::json,
  '{}'::json,
  EXISTS (
    SELECT 1 FROM ` + pageBaseName + `
    OFFSET ($` + fmt.Sprint(offsetParameter) + ` + $` + fmt.Sprint(limitParameter) + `) LIMIT 1
  ),
  (SELECT count(*) FROM (
    SELECT 1 FROM ` + pageBaseName + `
    LIMIT $` + fmt.Sprint(limitParameter) + ` OFFSET $` + fmt.Sprint(offsetParameter) + `
  ) page_slice)
FROM matching
`
	if options.GroupBy != "" {
		groupAxis := moduleIssueAxisSQL(options.GroupBy, "group_axis", "group_key")
		subgroupAxis := moduleIssueAxisSQL(options.SubGroupBy, "subgroup_axis", "subgroup_key")
		query = filteredSQL + `,
expanded AS (
  SELECT issue.*, group_axis.group_key, subgroup_axis.subgroup_key,
    (
      NOT EXISTS (SELECT 1 FROM intake_issues count_intake WHERE count_intake.issue_id = issue.id)
      OR EXISTS (
        SELECT 1 FROM intake_issues count_intake
        WHERE count_intake.issue_id = issue.id AND count_intake.status IN (-1, 1, 2)
      )
    ) AS count_for_pages
  FROM filtered issue
  ` + groupAxis + `
  ` + subgroupAxis + `
), ranked AS (
  SELECT issue.*,
    row_number() OVER (
      PARTITION BY issue.group_key, issue.subgroup_key
      ORDER BY ` + order + `, issue.created_at DESC
    ) AS group_position,
    count(*) FILTER (WHERE issue.count_for_pages) OVER (
      PARTITION BY issue.group_key, issue.subgroup_key
    ) AS group_page_count,
    count(*) OVER (
      PARTITION BY issue.group_key, issue.subgroup_key
    ) AS group_row_count
  FROM expanded issue
), group_counts AS (
  SELECT group_key, count(DISTINCT id) FILTER (WHERE count_for_pages) AS total
  FROM expanded
  GROUP BY group_key
), subgroup_counts AS (
  SELECT group_key, subgroup_key, count(DISTINCT id) FILTER (WHERE count_for_pages) AS total
  FROM expanded
  GROUP BY group_key, subgroup_key
), matching AS (
  SELECT (` + issueProjection + `)::jsonb || jsonb_build_object(
      '_selected_group', issue.group_key,
      '_selected_group_by', '` + options.GroupBy + `',
      '_selected_subgroup', issue.subgroup_key,
      '_selected_subgroup_by', '` + options.SubGroupBy + `',
      '_group_page_count', issue.group_page_count,
      '_group_row_count', issue.group_row_count,
      '_group_has_next', issue.group_row_count::double precision >= $` + fmt.Sprint(limitParameter) + `::double precision
    ) AS item,
    issue.group_key, issue.subgroup_key, issue.group_position
  FROM ranked issue
  WHERE issue.group_position::double precision > $` + fmt.Sprint(offsetParameter) + `::double precision
    AND issue.group_position::double precision < $` + fmt.Sprint(limitParameter) + `::double precision
)
SELECT COALESCE(json_agg(matching.item ORDER BY matching.group_key, matching.subgroup_key, matching.group_position), '[]'::json),
  (SELECT count(*) FROM expanded),
  ` + moduleIssueGroupUniverseSQL(options.GroupBy) + `,
  ` + moduleIssueGroupUniverseSQL(options.SubGroupBy) + `,
  COALESCE((SELECT json_object_agg(group_key, total) FROM group_counts), '{}'::json),
  COALESCE((SELECT json_object_agg(group_key || ':::' || subgroup_key, total) FROM subgroup_counts), '{}'::json),
  EXISTS (SELECT 1 FROM ranked WHERE group_position::double precision >= $` + fmt.Sprint(limitParameter) + `::double precision),
  0
FROM matching
`
	}
	return query, args
}

func (repository *Repository) readIssuePage(ctx context.Context, query string, args []any) (IssuePage, error) {
	var data, groupsData, subgroupsData, groupCountsData, subgroupCountsData []byte
	var total int
	var groupedHasNext bool
	var pageCount int
	if err := repository.database.QueryRow(ctx, query, args...).Scan(
		&data, &total, &groupsData, &subgroupsData, &groupCountsData, &subgroupCountsData, &groupedHasNext, &pageCount,
	); err != nil {
		return IssuePage{}, fmt.Errorf("list module issues: %w", err)
	}
	var groups, subgroups []any
	var groupCounts, subgroupCounts map[string]int
	if err := json.Unmarshal(groupsData, &groups); err != nil {
		return IssuePage{}, fmt.Errorf("decode module issue groups: %w", err)
	}
	if err := json.Unmarshal(subgroupsData, &subgroups); err != nil {
		return IssuePage{}, fmt.Errorf("decode module issue subgroups: %w", err)
	}
	if err := json.Unmarshal(groupCountsData, &groupCounts); err != nil {
		return IssuePage{}, fmt.Errorf("decode module issue group counts: %w", err)
	}
	if err := json.Unmarshal(subgroupCountsData, &subgroupCounts); err != nil {
		return IssuePage{}, fmt.Errorf("decode module issue subgroup counts: %w", err)
	}
	return IssuePage{
		Items: data, Total: total, GroupValues: groups, SubGroupValues: subgroups,
		GroupCounts: groupCounts, SubGroupCounts: subgroupCounts, GroupedHasNext: groupedHasNext,
		PageCount: pageCount,
	}, nil
}

func moduleIssueGroupUniverseSQL(group string) string {
	switch group {
	case "":
		return "'[]'::json"
	case "state_id":
		return `(SELECT COALESCE(json_agg(state.id ORDER BY state.sequence), '[]'::json) FROM states state WHERE state.project_id = $3 AND state.is_triage = false AND state.deleted_at IS NULL)`
	case "state__group":
		return `'["backlog","unstarted","started","completed","cancelled"]'::json`
	case "priority":
		return `'["low","medium","high","urgent","none"]'::json`
	case "labels__id":
		return `(SELECT COALESCE(json_agg(value), '[]'::json) FROM (
          SELECT label.id::text AS value
          FROM labels label
          JOIN projects label_project ON label_project.id = label.project_id
          JOIN workspaces label_workspace ON label_workspace.id = label_project.workspace_id
          WHERE label.project_id = $3
            AND label.workspace_id = label_workspace.id
            AND label_workspace.slug = $2
            AND label_workspace.deleted_at IS NULL
            AND label.deleted_at IS NULL
          UNION ALL SELECT 'None'
        ) values)`
	case "assignees__id":
		return `(SELECT COALESCE(json_agg(project_member.member_id), '[]'::json) FROM project_members project_member WHERE project_member.project_id = $3 AND project_member.is_active = true AND project_member.deleted_at IS NULL)`
	case "issue_module__module_id":
		return `(SELECT COALESCE(json_agg(value), '[]'::json) FROM (SELECT related_module.id::text AS value FROM modules related_module WHERE related_module.project_id = $3 AND related_module.deleted_at IS NULL UNION ALL SELECT 'None') values)`
	case "cycle_id":
		return `(SELECT COALESCE(json_agg(value), '[]'::json) FROM (SELECT cycle.id::text AS value FROM cycles cycle WHERE cycle.project_id = $3 AND cycle.deleted_at IS NULL UNION ALL SELECT 'None') values)`
	case "project_id":
		return `(SELECT COALESCE(json_agg(group_project.id ORDER BY group_project.name), '[]'::json)
          FROM projects group_project
          JOIN workspaces group_workspace ON group_workspace.id = group_project.workspace_id
          WHERE group_workspace.slug = $2 AND group_workspace.deleted_at IS NULL
            AND group_project.deleted_at IS NULL)`
	case "created_by":
		return `(SELECT COALESCE(json_agg(DISTINCT created_by_id), '[]'::json) FROM filtered)`
	case "target_date":
		return `(SELECT COALESCE(json_agg(DISTINCT target_date), '[]'::json) FROM filtered)`
	case "start_date":
		return `(SELECT COALESCE(json_agg(DISTINCT start_date), '[]'::json) FROM filtered)`
	default:
		return "'[]'::json"
	}
}
func moduleIssueArrayField(field string) string {
	switch field {
	case "assignees__id":
		return "assignee_ids"
	case "labels__id":
		return "label_ids"
	case "issue_module__module_id":
		return "module_ids"
	default:
		return ""
	}
}
func moduleIssueRelationTable(field string) string {
	switch field {
	case "assignees__id":
		return "issue_assignees"
	case "labels__id":
		return "issue_labels"
	case "issue_module__module_id":
		return "module_issues"
	default:
		return ""
	}
}

func moduleIssueAxisSQL(group, alias, column string) string {
	scalar := map[string]string{
		"":             "''",
		"state_id":     "issue.state_id::text",
		"state__group": `(SELECT grouped_state."group" FROM states grouped_state WHERE grouped_state.id = issue.state_id)`,
		"priority":     "issue.priority",
		"project_id":   "issue.project_id::text",
		"cycle_id": `(SELECT grouped_cycle.cycle_id::text FROM cycle_issues grouped_cycle
      WHERE grouped_cycle.issue_id = issue.id AND grouped_cycle.deleted_at IS NULL
      ORDER BY grouped_cycle.created_at DESC LIMIT 1)`,
		"created_by":  "issue.created_by_id::text",
		"target_date": "issue.target_date::text",
		"start_date":  "issue.start_date::text",
	}
	if expression, exists := scalar[group]; exists {
		return `CROSS JOIN LATERAL (
    SELECT COALESCE(` + expression + `, 'None') AS ` + column + `
  ) ` + alias
	}
	if group == "labels__id" {
		liveRelation := issuequery.LiveIssueLabelPredicate("issue", "grouped_relation")
		return `CROSS JOIN LATERAL (
    SELECT DISTINCT grouped_relation.label_id::text AS ` + column + `
    FROM issue_labels grouped_relation
    WHERE ` + liveRelation + `
    UNION ALL
    SELECT 'None' WHERE NOT EXISTS (
      SELECT 1 FROM issue_labels grouped_relation
      WHERE ` + liveRelation + `
    )
  ) ` + alias
	}
	relation := map[string][2]string{
		"assignees__id":           {"issue_assignees", "assignee_id"},
		"labels__id":              {"issue_labels", "label_id"},
		"issue_module__module_id": {"module_issues", "module_id"},
	}[group]
	return `CROSS JOIN LATERAL (
    SELECT DISTINCT grouped_relation.` + relation[1] + `::text AS ` + column + `
    FROM ` + relation[0] + ` grouped_relation
    WHERE grouped_relation.issue_id = issue.id AND grouped_relation.deleted_at IS NULL
    UNION ALL
    SELECT 'None' WHERE NOT EXISTS (
      SELECT 1 FROM ` + relation[0] + ` grouped_relation
      WHERE grouped_relation.issue_id = issue.id
    )
  ) ` + alias
}

func requesterTimestampSQL(column string) string {
	zone := `COALESCE(
    (SELECT NULLIF(requester_user.user_timezone, '') FROM users requester_user WHERE requester_user.id = $1),
    'UTC'
  )`
	local := "(" + column + " AT TIME ZONE " + zone + ")"
	offset := "(" + local + " - (" + column + " AT TIME ZONE 'UTC'))"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + local + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + local + `)=` + local + ` THEN '' ELSE '.' || to_char(` + local + `,'US') END ||
  CASE WHEN extract(epoch FROM ` + offset + `)=0 THEN 'Z' ELSE
    CASE WHEN extract(epoch FROM ` + offset + `)<0 THEN '-' ELSE '+' END ||
    lpad(floor(abs(extract(epoch FROM ` + offset + `))/3600)::int::text,2,'0') || ':' ||
    lpad(floor(mod(abs(extract(epoch FROM ` + offset + `)),3600)/60)::int::text,2,'0')
  END
END`
}
func requesterJSONTimestampSQL(column string) string {
	zone := `COALESCE(
    (SELECT NULLIF(requester_user.user_timezone, '') FROM users requester_user WHERE requester_user.id = $1),
    'UTC'
  )`
	local := "(" + column + " AT TIME ZONE " + zone + ")"
	offset := "(" + local + " - (" + column + " AT TIME ZONE 'UTC'))"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + local + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + local + `)=` + local + ` THEN '' ELSE '.' || to_char(` + local + `,'MS') END ||
  CASE WHEN extract(epoch FROM ` + offset + `)=0 THEN 'Z' ELSE
    CASE WHEN extract(epoch FROM ` + offset + `)<0 THEN '-' ELSE '+' END ||
    lpad(floor(abs(extract(epoch FROM ` + offset + `))/3600)::int::text,2,'0') || ':' ||
    lpad(floor(mod(abs(extract(epoch FROM ` + offset + `)),3600)/60)::int::text,2,'0')
  END
END`
}

func djangoJSONTimestampSQL(column string) string {
	utc := "(" + column + " AT TIME ZONE 'UTC')"
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE
  to_char(` + utc + `,'YYYY-MM-DD"T"HH24:MI:SS') ||
  CASE WHEN date_trunc('second',` + utc + `)=` + utc + ` THEN '' ELSE '.' || to_char(` + utc + `,'MS') END || 'Z'
END`
}

// IssueDetail intentionally identifies the relationship by issue_id. The
// inherited Django retrieve path incorrectly looks for pk and serializes an
// Issue as a ModuleIssue.
func (repository *Repository) IssueDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID, issueID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.readObject(ctx, `
SELECT `+moduleIssueJSON+`
FROM module_issues module_issue
JOIN modules module
  ON module.id = module_issue.module_id
  AND module.project_id = module_issue.project_id
  AND module.workspace_id = module_issue.workspace_id
JOIN issues issue
  ON issue.id = module_issue.issue_id
  AND issue.project_id = module_issue.project_id
  AND issue.workspace_id = module_issue.workspace_id
JOIN projects project
  ON project.id = module_issue.project_id
  AND project.workspace_id = module_issue.workspace_id
JOIN workspaces workspace ON workspace.id = module_issue.workspace_id
WHERE workspace.slug = $2
  AND workspace.deleted_at IS NULL
  AND project.id = $3
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
  AND module.id = $4
  AND module.deleted_at IS NULL
  AND issue.id = $5
  AND issue.deleted_at IS NULL
  AND issue.archived_at IS NULL
  AND issue.is_draft = false
  AND module_issue.deleted_at IS NULL
LIMIT 1
`, "get module issue relationship", userID, slug, projectID, moduleID, issueID)
}

func (repository *Repository) Links(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.readArray(ctx, `
SELECT COALESCE(json_agg(`+moduleLinkJSON("module_link")+` ORDER BY module_link.created_at DESC), '[]'::json)
FROM module_links module_link
JOIN modules module ON module.id = module_link.module_id
JOIN projects project ON project.id = module_link.project_id AND project.id = module.project_id
JOIN workspaces workspace ON workspace.id = module_link.workspace_id AND workspace.id = module.workspace_id
WHERE workspace.slug = $2
  AND workspace.deleted_at IS NULL
  AND project.id = $3
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
  AND module.id = $4
  AND module.deleted_at IS NULL
  AND module_link.deleted_at IS NULL
`, "list module links", userID, slug, projectID, moduleID)
}

func (repository *Repository) LinkDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID, linkID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, false); err != nil {
		return nil, err
	}
	return repository.readObject(ctx, `
SELECT `+moduleLinkJSON("module_link")+`
FROM module_links module_link
JOIN modules module ON module.id = module_link.module_id
JOIN projects project ON project.id = module_link.project_id AND project.id = module.project_id
JOIN workspaces workspace ON workspace.id = module_link.workspace_id AND workspace.id = module.workspace_id
WHERE workspace.slug = $2
  AND workspace.deleted_at IS NULL
  AND project.id = $3
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
  AND module.id = $4
  AND module.deleted_at IS NULL
  AND module_link.id = $5
  AND module_link.deleted_at IS NULL
LIMIT 1
`, "get module link", userID, slug, projectID, moduleID, linkID)
}

func (repository *Repository) Detail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID, true); err != nil {
		return nil, err
	}
	data, err := repository.moduleDetail(ctx, userID, slug, projectID, moduleID, false)
	if err != nil {
		return nil, err
	}
	if err := repository.admitRecentVisit(ctx, userID, slug, projectID, moduleID); err != nil {
		return nil, fmt.Errorf("record recent module visit: %w", err)
	}
	return data, nil
}

func (repository *Repository) moduleList(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, archived, serializer bool) (json.RawMessage, error) {
	archivePredicate := "module.archived_at IS NULL"
	projection := "active"
	if archived {
		archivePredicate = "module.archived_at IS NOT NULL"
		projection = "archived"
	} else if serializer {
		projection = "active_serializer"
	}
	return repository.readArray(ctx, `
SELECT COALESCE(json_agg(module_row.item ORDER BY module_row.favorite DESC, module_row.created_at DESC), '[]'::json)
FROM (
  SELECT `+moduleJSON("module", projection)+` AS item,
    EXISTS (
      SELECT 1 FROM user_favorites favorite
      WHERE favorite.user_id = $1
        AND favorite.workspace_id = module.workspace_id
        AND favorite.project_id = module.project_id
        AND favorite.entity_type = 'module'
        AND favorite.entity_identifier = module.id
        AND favorite.deleted_at IS NULL
    ) AS favorite,
    module.created_at
  FROM modules module
  JOIN projects project ON project.id = module.project_id
  JOIN workspaces workspace ON workspace.id = module.workspace_id
  WHERE workspace.slug = $2
    AND workspace.deleted_at IS NULL
    AND project.id = $3
    AND project.deleted_at IS NULL
    AND module.deleted_at IS NULL
    AND `+archivePredicate+`
) module_row
`, "list project modules", userID, slug, projectID)
}

func (repository *Repository) moduleDetail(ctx context.Context, userID uuid.UUID, slug string, projectID, moduleID uuid.UUID, archived bool) (json.RawMessage, error) {
	archivePredicate := "module.archived_at IS NULL"
	if archived {
		archivePredicate = "module.archived_at IS NOT NULL"
	}
	return repository.readObject(ctx, `
SELECT `+moduleDetailJSON("module", archived)+`
FROM modules module
JOIN projects project ON project.id = module.project_id
JOIN workspaces workspace ON workspace.id = module.workspace_id
WHERE workspace.slug = $2
  AND workspace.deleted_at IS NULL
  AND project.id = $3
  AND project.deleted_at IS NULL
  AND module.id = $4
  AND module.deleted_at IS NULL
  AND `+archivePredicate+`
LIMIT 1
`, "get module detail", userID, slug, projectID, moduleID)
}

func (repository *Repository) authorizeWorkspace(ctx context.Context, userID uuid.UUID, slug string) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace_member.member_id = $1
    AND workspace.slug = $2
    AND workspace.deleted_at IS NULL
    AND workspace_member.is_active = true
    AND workspace_member.deleted_at IS NULL
)
`, userID, slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize workspace module read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) authorizeProject(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID, elevated bool) error {
	rolePredicate := ""
	if elevated {
		rolePredicate = `AND (
      project_member.role IN (15, 20)
      OR EXISTS (
        SELECT 1 FROM workspace_members workspace_admin
        WHERE workspace_admin.workspace_id = workspace.id
          AND workspace_admin.member_id = $1
          AND workspace_admin.role = 20
          AND workspace_admin.is_active = true
          AND workspace_admin.deleted_at IS NULL
      )
    )`
	}
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN projects project ON project.id = project_member.project_id
  JOIN workspaces workspace ON workspace.id = project_member.workspace_id AND workspace.id = project.workspace_id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND workspace.deleted_at IS NULL
    AND project.id = $3
    AND project.deleted_at IS NULL
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    `+rolePredicate+`
)
`, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize project module read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) readArray(ctx context.Context, query, operation string, args ...any) (json.RawMessage, error) {
	var data []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&data); err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return json.RawMessage(data), nil
}
func (repository *Repository) readObject(ctx context.Context, query, operation string, args ...any) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, query, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return json.RawMessage(data), nil
}

func moduleLinkJSON(alias string) string {
	return `json_build_object(
    'id', ` + alias + `.id,
    'created_at', ` + requesterTimestampSQL(alias+`.created_at`) + `,
    'updated_at', ` + requesterTimestampSQL(alias+`.updated_at`) + `,
    'created_by', ` + alias + `.created_by_id,
    'updated_by', ` + alias + `.updated_by_id,
    'deleted_at', ` + alias + `.deleted_at,
    'workspace', ` + alias + `.workspace_id,
    'project', ` + alias + `.project_id,
    'module', ` + alias + `.module_id,
    'title', ` + alias + `.title,
    'url', ` + alias + `.url,
    'metadata', ` + alias + `.metadata
  )`
}

func moduleJSON(alias, projection string) string {
	archivedField := ""
	createdAt := djangoJSONTimestampSQL(alias + ".created_at")
	updatedAt := djangoJSONTimestampSQL(alias + ".updated_at")
	logoField := "'logo_props', " + alias + ".logo_props,"
	estimateFields := `
    'total_estimate_points', COALESCE((SELECT sum(estimate_point.value::double precision)
      FROM module_issues estimate_module_issue
      JOIN issues estimate_issue ON estimate_issue.id = estimate_module_issue.issue_id
      JOIN estimate_points estimate_point ON ` + issuequery.LiveNumericEstimatePredicate("estimate_issue", "estimate_point") + `
      WHERE estimate_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "estimate_module_issue", "estimate_issue") + ` AND estimate_module_issue.deleted_at IS NULL
        AND estimate_issue.deleted_at IS NULL AND estimate_issue.archived_at IS NULL
        AND estimate_issue.is_draft = false AND ` + issueManagerStateSQL("estimate_issue") + `), 0),
    'completed_estimate_points', COALESCE((SELECT sum(estimate_point.value::double precision)
      FROM module_issues estimate_module_issue
      JOIN issues estimate_issue ON estimate_issue.id = estimate_module_issue.issue_id
      JOIN states estimate_state ON estimate_state.id = estimate_issue.state_id AND estimate_state."group" = 'completed'
      JOIN estimate_points estimate_point ON ` + issuequery.LiveNumericEstimatePredicate("estimate_issue", "estimate_point") + `
      WHERE estimate_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "estimate_module_issue", "estimate_issue") + ` AND estimate_module_issue.deleted_at IS NULL
        AND estimate_issue.deleted_at IS NULL AND estimate_issue.archived_at IS NULL
        AND estimate_issue.is_draft = false AND ` + issueManagerStateSQL("estimate_issue") + `), 0),`
	favoriteField := `
    'is_favorite', EXISTS (SELECT 1 FROM user_favorites favorite
      WHERE favorite.user_id = $1 AND favorite.entity_type = 'module'
        AND favorite.entity_identifier = ` + alias + `.id AND favorite.deleted_at IS NULL),`
	switch projection {
	case "workspace":
		createdAt = requesterTimestampSQL(alias + ".created_at")
		updatedAt = requesterTimestampSQL(alias + ".updated_at")
		archivedField = "'archived_at', " + requesterTimestampSQL(alias+".archived_at") + ","
		estimateFields = ""
		favoriteField = ""
	case "active":
		createdAt = requesterJSONTimestampSQL(alias + ".created_at")
		updatedAt = requesterJSONTimestampSQL(alias + ".updated_at")
	case "active_serializer":
		createdAt = requesterTimestampSQL(alias + ".created_at")
		updatedAt = requesterTimestampSQL(alias + ".updated_at")
		archivedField = "'archived_at', " + requesterTimestampSQL(alias+".archived_at") + ","
	case "archived":
		createdAt = requesterJSONTimestampSQL(alias + ".created_at")
		updatedAt = requesterJSONTimestampSQL(alias + ".updated_at")
		archivedField = "'archived_at', " + djangoJSONTimestampSQL(alias+".archived_at") + ","
		logoField = ""
		estimateFields = ""
	case "detail":
		createdAt = requesterTimestampSQL(alias + ".created_at")
		updatedAt = requesterTimestampSQL(alias + ".updated_at")
		archivedField = "'archived_at', " + requesterTimestampSQL(alias+".archived_at") + ","
	}
	issueScope := issueManagerStateSQL("counted_issue")
	if projection == "workspace" {
		issueScope = "TRUE"
	}
	return `json_build_object(
    'id', ` + alias + `.id,
    'workspace_id', ` + alias + `.workspace_id,
    'project_id', ` + alias + `.project_id,
    'name', ` + alias + `.name,
    'description', ` + alias + `.description,
    'description_text', ` + alias + `.description_text,
    'description_html', ` + alias + `.description_html,
    'start_date', ` + alias + `.start_date,
    'target_date', ` + alias + `.target_date,
    'status', ` + alias + `.status,
    'lead_id', ` + alias + `.lead_id,
    'member_ids', COALESCE((SELECT json_agg(member.member_id ORDER BY member.created_at DESC)
      FROM module_members member WHERE member.module_id = ` + alias + `.id AND member.deleted_at IS NULL), '[]'::json),
    'view_props', ` + alias + `.view_props,
    'sort_order', ` + alias + `.sort_order,
    'external_source', ` + alias + `.external_source,
    'external_id', ` + alias + `.external_id,
    ` + logoField + estimateFields + favoriteField + `
    'total_issues', ` + issueCount(alias, "TRUE", issueScope) + `,
    'cancelled_issues', ` + issueCount(alias, `state."group" = 'cancelled'`, issueScope) + `,
    'completed_issues', ` + issueCount(alias, `state."group" = 'completed'`, issueScope) + `,
    'started_issues', ` + issueCount(alias, `state."group" = 'started'`, issueScope) + `,
    'unstarted_issues', ` + issueCount(alias, `state."group" = 'unstarted'`, issueScope) + `,
    'backlog_issues', ` + issueCount(alias, `state."group" = 'backlog'`, issueScope) + `,
    'created_at', ` + createdAt + `,
    'updated_at', ` + updatedAt + `,
    ` + archivedField + `
    '_end', true
  )::jsonb - '_end'`
}

func issueManagerStateSQL(alias string) string {
	return `(` + alias + `.state_id IS NULL OR EXISTS (
          SELECT 1 FROM states manager_state
          WHERE manager_state.id = ` + alias + `.state_id AND manager_state."group" <> 'triage'
        )) AND EXISTS (
          SELECT 1 FROM projects manager_project
          WHERE manager_project.id = ` + alias + `.project_id
            AND manager_project.deleted_at IS NULL
            AND manager_project.archived_at IS NULL
        )`
}

func moduleAggregateScope(moduleAlias, relationAlias, issueAlias string) string {
	return `
        AND ` + relationAlias + `.workspace_id = ` + moduleAlias + `.workspace_id
        AND ` + relationAlias + `.project_id = ` + moduleAlias + `.project_id
        AND ` + issueAlias + `.workspace_id = ` + moduleAlias + `.workspace_id
        AND ` + issueAlias + `.project_id = ` + moduleAlias + `.project_id`
}

func issueCount(alias, statePredicate, issueScope string) string {
	return `(SELECT count(*) FROM module_issues counted_module_issue
      JOIN issues counted_issue ON counted_issue.id = counted_module_issue.issue_id
      LEFT JOIN states state ON state.id = counted_issue.state_id
      WHERE counted_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "counted_module_issue", "counted_issue") + `
        AND counted_module_issue.deleted_at IS NULL
        AND counted_issue.deleted_at IS NULL
        AND counted_issue.archived_at IS NULL
        AND counted_issue.is_draft = false
        AND ` + issueScope + `
        AND ` + statePredicate + `)`
}

func moduleDetailJSON(alias string, archived bool) string {
	base := moduleJSON(alias, "detail")
	return `(` + base + `)::jsonb || jsonb_build_object(
    'link_module', COALESCE((SELECT jsonb_agg(` + moduleLinkJSON("detail_link") + ` ORDER BY detail_link.created_at DESC)
      FROM module_links detail_link WHERE detail_link.module_id = ` + alias + `.id AND detail_link.deleted_at IS NULL), '[]'::jsonb),
    'sub_issues', (SELECT count(*) FROM module_issues child_module_issue
      JOIN issues child_issue ON child_issue.id = child_module_issue.issue_id
      WHERE child_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "child_module_issue", "child_issue") + ` AND child_module_issue.deleted_at IS NULL
        AND child_issue.deleted_at IS NULL AND child_issue.archived_at IS NULL
        AND child_issue.is_draft = false AND ` + issueManagerStateSQL("child_issue") + `
        AND child_issue.parent_id IS NOT NULL),
    'backlog_estimate_points', ` + estimateCount(alias, "backlog") + `,
    'unstarted_estimate_points', ` + estimateCount(alias, "unstarted") + `,
    'started_estimate_points', ` + estimateCount(alias, "started") + `,
    'cancelled_estimate_points', ` + estimateCount(alias, "cancelled") + `,
    'distribution', jsonb_build_object(
      'assignees', ` + issueAssigneeDistribution(alias, false) + `,
      'labels', ` + issueLabelDistribution(alias, false) + `,
      'completion_chart', ` + moduleBurndown(alias, false, archived) + `
    ),
    'estimate_distribution', CASE WHEN EXISTS (
      SELECT 1 FROM projects estimate_project JOIN estimates estimate ON estimate.id = estimate_project.estimate_id
      WHERE estimate_project.id = ` + alias + `.project_id AND estimate.type = 'points'
    ) THEN jsonb_build_object(
      'assignees', ` + issueAssigneeDistribution(alias, true) + `,
      'labels', ` + issueLabelDistribution(alias, true) + `,
      'completion_chart', ` + moduleBurndown(alias, true, archived) + `
    ) ELSE '{}'::jsonb END
  )`
}

func estimateCount(alias, group string) string {
	return `COALESCE((SELECT sum(estimate_point.value::double precision)
      FROM module_issues estimate_module_issue
      JOIN issues estimate_issue ON estimate_issue.id = estimate_module_issue.issue_id
      JOIN states estimate_state ON estimate_state.id = estimate_issue.state_id AND estimate_state."group" = '` + group + `'
      JOIN estimate_points estimate_point ON ` + issuequery.LiveNumericEstimatePredicate("estimate_issue", "estimate_point") + `
      WHERE estimate_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "estimate_module_issue", "estimate_issue") + ` AND estimate_module_issue.deleted_at IS NULL
        AND estimate_issue.deleted_at IS NULL AND estimate_issue.archived_at IS NULL
        AND estimate_issue.is_draft = false AND ` + issueManagerStateSQL("estimate_issue") + `), 0)`
}

func moduleBurndown(alias string, estimates, archived bool) string {
	total := issueCount(alias, "TRUE", issueManagerStateSQL("counted_issue"))
	completed := `(SELECT count(*) FROM module_issues chart_module_issue
          JOIN issues chart_issue ON chart_issue.id = chart_module_issue.issue_id
          WHERE chart_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "chart_module_issue", "chart_issue") + `
            AND chart_module_issue.deleted_at IS NULL
            AND chart_issue.deleted_at IS NULL AND chart_issue.archived_at IS NULL
            AND chart_issue.is_draft = false AND ` + issueManagerStateSQL("chart_issue") + `
            AND ` + requesterDateSQL("chart_issue.completed_at") + ` <= chart_day::date)`
	emptyCondition := alias + `.start_date IS NULL OR ` + alias + `.target_date IS NULL`
	if !estimates && !archived {
		emptyCondition += ` OR ` + total + ` = 0`
	}
	if estimates {
		total = `COALESCE((SELECT sum(chart_point.value::double precision)
          FROM module_issues chart_module_issue
          JOIN issues chart_issue ON chart_issue.id = chart_module_issue.issue_id
          JOIN estimate_points chart_point ON ` + issuequery.LiveNumericEstimatePredicate("chart_issue", "chart_point") + `
          WHERE chart_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "chart_module_issue", "chart_issue") + `
            AND chart_module_issue.deleted_at IS NULL
            AND chart_issue.deleted_at IS NULL AND chart_issue.archived_at IS NULL
            AND chart_issue.is_draft = false AND ` + issueManagerStateSQL("chart_issue") + `), 0)`
		completed = `COALESCE((SELECT sum(chart_point.value::double precision)
          FROM module_issues chart_module_issue
          JOIN issues chart_issue ON chart_issue.id = chart_module_issue.issue_id
          JOIN estimate_points chart_point ON ` + issuequery.LiveNumericEstimatePredicate("chart_issue", "chart_point") + `
          WHERE chart_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "chart_module_issue", "chart_issue") + `
            AND chart_module_issue.deleted_at IS NULL
            AND chart_issue.deleted_at IS NULL AND chart_issue.archived_at IS NULL
            AND chart_issue.is_draft = false AND ` + issueManagerStateSQL("chart_issue") + `
            AND ` + requesterDateSQL("chart_issue.completed_at") + ` <= chart_day::date), 0)`
		emptyCondition = alias + `.start_date IS NULL OR ` + alias + `.target_date IS NULL`
	}
	return `CASE WHEN ` + emptyCondition + ` THEN '{}'::jsonb ELSE COALESCE((
      SELECT jsonb_object_agg(chart_day::date::text,
        CASE WHEN chart_day::date > CURRENT_DATE THEN NULL ELSE ` + total + ` - ` + completed + ` END
        ORDER BY chart_day)
      FROM generate_series(` + alias + `.start_date, ` + alias + `.target_date, INTERVAL '1 day') chart_day
    ), '{}'::jsonb) END`
}

func issueAssigneeDistribution(alias string, estimates bool) string {
	total := "count(*)"
	completed := "count(*) FILTER (WHERE distribution_issue.completed_at IS NOT NULL)"
	pending := "count(*) FILTER (WHERE distribution_issue.completed_at IS NULL)"
	if estimates {
		total = "sum(distribution_point.value::double precision)"
		completed = "sum(distribution_point.value::double precision) FILTER (WHERE distribution_issue.completed_at IS NOT NULL)"
		pending = "sum(distribution_point.value::double precision) FILTER (WHERE distribution_issue.completed_at IS NULL)"
	}
	return `COALESCE((SELECT jsonb_agg(distribution_row.item ORDER BY distribution_row.first_name, distribution_row.last_name)
      FROM (SELECT distribution_user.first_name, distribution_user.last_name, jsonb_build_object(
        'first_name', distribution_user.first_name,
        'last_name', distribution_user.last_name,
        'assignee_id', distribution_user.id,
        'display_name', distribution_user.display_name,
        'avatar_url', CASE WHEN distribution_user.avatar_asset_id IS NOT NULL
          THEN '/api/assets/v2/static/' || distribution_user.avatar_asset_id::text || '/'
          ELSE distribution_user.avatar END,
        '` + distributionTotalName(estimates) + `', ` + total + `,
        '` + distributionCompletedName(estimates) + `', ` + completed + `,
        '` + distributionPendingName(estimates) + `', ` + pending + `
      ) AS item
      FROM module_issues distribution_module_issue
      JOIN issues distribution_issue ON distribution_issue.id = distribution_module_issue.issue_id
      LEFT JOIN issue_assignees distribution_assignee ON distribution_assignee.issue_id = distribution_issue.id
      LEFT JOIN users distribution_user ON distribution_user.id = distribution_assignee.assignee_id` + estimateDistributionJoin(estimates) + `
      WHERE distribution_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "distribution_module_issue", "distribution_issue") + `
        AND distribution_module_issue.deleted_at IS NULL AND distribution_issue.deleted_at IS NULL
        AND distribution_issue.archived_at IS NULL AND distribution_issue.is_draft = false
        AND ` + issueManagerStateSQL("distribution_issue") + `
      GROUP BY distribution_user.id, distribution_user.first_name, distribution_user.last_name,
        distribution_user.display_name, distribution_user.avatar_asset_id, distribution_user.avatar
      ) distribution_row), '[]'::jsonb)`
}

func issueLabelDistribution(alias string, estimates bool) string {
	total := "count(*)"
	completed := "count(*) FILTER (WHERE distribution_issue.completed_at IS NOT NULL)"
	pending := "count(*) FILTER (WHERE distribution_issue.completed_at IS NULL)"
	if estimates {
		total = "sum(distribution_point.value::double precision)"
		completed = "sum(distribution_point.value::double precision) FILTER (WHERE distribution_issue.completed_at IS NOT NULL)"
		pending = "sum(distribution_point.value::double precision) FILTER (WHERE distribution_issue.completed_at IS NULL)"
	}
	return `COALESCE((SELECT jsonb_agg(distribution_row.item ORDER BY distribution_row.label_name)
      FROM (SELECT distribution_label.name AS label_name, jsonb_build_object(
        'label_name', distribution_label.name,
        'color', distribution_label.color,
        'label_id', distribution_label.id,
        '` + distributionTotalName(estimates) + `', ` + total + `,
        '` + distributionCompletedName(estimates) + `', ` + completed + `,
        '` + distributionPendingName(estimates) + `', ` + pending + `
      ) AS item
      FROM module_issues distribution_module_issue
      JOIN issues distribution_issue ON distribution_issue.id = distribution_module_issue.issue_id
      LEFT JOIN issue_labels distribution_issue_label ON ` + issuequery.LiveIssueLabelPredicate("distribution_issue", "distribution_issue_label") + `
      LEFT JOIN labels distribution_label ON distribution_label.id = distribution_issue_label.label_id
        AND distribution_label.deleted_at IS NULL` + estimateDistributionJoin(estimates) + `
      WHERE distribution_module_issue.module_id = ` + alias + `.id` + moduleAggregateScope(alias, "distribution_module_issue", "distribution_issue") + `
        AND distribution_module_issue.deleted_at IS NULL AND distribution_issue.deleted_at IS NULL
        AND distribution_issue.archived_at IS NULL AND distribution_issue.is_draft = false
        AND ` + issueManagerStateSQL("distribution_issue") + `
      GROUP BY distribution_label.id, distribution_label.name, distribution_label.color
      ) distribution_row), '[]'::jsonb)`
}

func estimateDistributionJoin(estimates bool) string {
	if !estimates {
		return ""
	}
	return ` LEFT JOIN estimate_points distribution_point ON ` + issuequery.LiveNumericEstimatePredicate("distribution_issue", "distribution_point")
}

func distributionTotalName(estimates bool) string {
	if estimates {
		return "total_estimates"
	}
	return "total_issues"
}

func distributionCompletedName(estimates bool) string {
	if estimates {
		return "completed_estimates"
	}
	return "completed_issues"
}

func distributionPendingName(estimates bool) string {
	if estimates {
		return "pending_estimates"
	}
	return "pending_issues"
}

var issueJSON = `json_build_object(
    'id', issue.id,
    'type_id', issue.type_id,
    'name', issue.name,
    'state_id', issue.state_id,
    'state__group', (SELECT state."group" FROM states state WHERE state.id = issue.state_id),
    'sort_order', issue.sort_order,
    'completed_at', ` + djangoJSONTimestampSQL("issue.completed_at") + `,
    'estimate_point', issue.estimate_point_id,
    'priority', issue.priority,
    'start_date', issue.start_date,
    'target_date', issue.target_date,
    'sequence_id', issue.sequence_id,
    'project_id', issue.project_id,
    'parent_id', ` + djangoapi.VisibleParentSQL("issue", "$1::uuid") + `,
    'cycle_id', (SELECT cycle_issue.cycle_id FROM cycle_issues cycle_issue
      WHERE cycle_issue.issue_id = issue.id AND cycle_issue.deleted_at IS NULL ORDER BY cycle_issue.created_at DESC LIMIT 1),
    'sub_issues_count', (SELECT count(*) FROM issues child
      WHERE child.parent_id = issue.id AND ` + djangoapi.IssueVisibleSQL("child", "$1::uuid", "issue.workspace_id") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL AND child.is_draft = false
        AND (child.state_id IS NULL OR EXISTS (
          SELECT 1 FROM states child_state
          WHERE child_state.id = child.state_id AND child_state."group" <> 'triage'
        ))),
    'created_at', ` + djangoJSONTimestampSQL("issue.created_at") + `,
    'updated_at', ` + djangoJSONTimestampSQL("issue.updated_at") + `,
    'created_by', issue.created_by_id,
    'updated_by', issue.updated_by_id,
    'attachment_count', (SELECT count(*) FROM file_assets attachment
      WHERE attachment.issue_id = issue.id AND attachment.deleted_at IS NULL AND attachment.entity_type = 'ISSUE_ATTACHMENT'),
    'link_count', (SELECT count(*) FROM issue_links link WHERE link.issue_id = issue.id AND link.deleted_at IS NULL),
    'is_draft', issue.is_draft,
    'archived_at', ` + djangoJSONTimestampSQL("issue.archived_at") + `,
    'assignee_ids', COALESCE((SELECT json_agg(assignee.assignee_id) FROM issue_assignees assignee
      WHERE assignee.issue_id = issue.id AND assignee.deleted_at IS NULL), '[]'::json),
    'label_ids', COALESCE((SELECT json_agg(label.label_id) FROM issue_labels label
      WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "label") + `), '[]'::json),
    'module_ids', COALESCE((SELECT json_agg(related_module.module_id) FROM module_issues related_module
      JOIN modules active_module ON active_module.id = related_module.module_id
      WHERE related_module.issue_id = issue.id AND related_module.deleted_at IS NULL
        AND active_module.deleted_at IS NULL AND active_module.archived_at IS NULL), '[]'::json)
  )`

var moduleIssueJSON = `json_build_object(
    'id', module_issue.id,
    'created_at', ` + requesterTimestampSQL("module_issue.created_at") + `,
    'updated_at', ` + requesterTimestampSQL("module_issue.updated_at") + `,
    'created_by', module_issue.created_by_id,
    'updated_by', module_issue.updated_by_id,
    'deleted_at', module_issue.deleted_at,
    'workspace', module_issue.workspace_id,
    'project', module_issue.project_id,
    'module', module_issue.module_id,
    'issue', module_issue.issue_id,
    'module_detail', json_build_object(
      'id', module.id,
      'created_at', ` + requesterTimestampSQL("module.created_at") + `,
      'updated_at', ` + requesterTimestampSQL("module.updated_at") + `,
      'created_by', module.created_by_id,
      'updated_by', module.updated_by_id,
      'deleted_at', module.deleted_at,
      'workspace', module.workspace_id,
      'project', module.project_id,
      'name', module.name,
      'description', module.description,
      'description_text', module.description_text,
      'description_html', module.description_html,
      'start_date', module.start_date,
      'target_date', module.target_date,
      'status', module.status,
      'lead', module.lead_id,
      'members', COALESCE((SELECT json_agg(member.member_id) FROM module_members member
        WHERE member.module_id = module.id AND member.deleted_at IS NULL), '[]'::json),
      'view_props', module.view_props,
      'sort_order', module.sort_order,
      'external_source', module.external_source,
      'external_id', module.external_id,
      'logo_props', module.logo_props,
      'archived_at', ` + requesterTimestampSQL("module.archived_at") + `
    ),
    'issue_detail', json_build_object('id', issue.id, 'name', issue.name),
    'sub_issues_count', (SELECT count(*) FROM issues child
      WHERE child.parent_id = issue.id AND ` + djangoapi.IssueVisibleSQL("child", "$1::uuid", "issue.workspace_id") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL AND child.is_draft = false
        AND (child.state_id IS NULL OR EXISTS (
          SELECT 1 FROM states child_state
          WHERE child_state.id = child.state_id AND child_state."group" <> 'triage'
        )))
  )`
