package workitems

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
)

// PublicIssueList queries only visible issues in a validated public board's
// project. It shares the established filter, order, pagination, and grouping
// helpers with work-item lists without inheriting their member authorization.
func (repository *Repository) PublicIssueList(ctx context.Context, scope PublicFeedbackScope, actorID string, params map[string]string, present map[string]bool) (Result, error) {
	q := Request{Operation: "public_feedback_issue_list", Slug: scope.WorkspaceSlug, ProjectID: scope.ProjectID,
		ActorID: actorID, Params: params, Present: map[string]bool{}, Timezone: "UTC"}
	q.Params = maps.Clone(params)
	delete(q.Params, "filters")
	for key, value := range present {
		q.Present[key] = value
	}
	for _, key := range []string{"per_page", "cursor", "order_by", "group_by", "sub_group_by", "search",
		"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by", "parent", "mentions",
		"name", "created_at", "project", "logged_by", "updated_at", "start_date", "target_date", "completed_at", "cycle",
		"module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date", "type"} {
		if _, exists := params[key]; exists {
			q.Present[key] = true
		}
	}
	options := queryOptions(q)
	if options.GroupBy != "" && options.GroupBy == options.SubGroupBy {
		return Result{}, failure(400, "error", "Group by and sub group by cannot have same parameters")
	}
	limit, page, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return Result{}, err
	}
	if options.GroupBy != "" || options.SubGroupBy != "" {
		if err := issuequery.ValidateGrouping(options.GroupBy, options.SubGroupBy); err != nil {
			return Result{}, failure(400, "detail", err.Error())
		}
	}
	// Django's legacy date filter builder treats malformed/non-ASCII and
	// week-only date values as an empty queryset. Preserve that public behavior
	// before the stricter shared SQL date helper validates its input.
	for _, value := range []string{options.CreatedAt, options.UpdatedAt, options.StartDate, options.TargetDate, options.CompletedAt} {
		if !issuequery.ValidLegacyDateBounds(value) {
			if options.GroupBy != "" {
				return repository.groupedPageWithVisibleRowCounts(ctx, q, []any{}, []map[string]any{}, limit, page, true)
			}
			return Result{Status: 200, Body: djangoapi.Envelope([]any{}, 0, limit, page)}, nil
		}
	}
	actorArg := any(nil)
	if actorID != "" {
		actorArg = actorID
	}
	args := []any{actorArg, scope.WorkspaceSlug, scope.ProjectID}
	where := []string{
		"workspace.slug=$2",
		"issue.project_id=$3",
		"issue.workspace_id=workspace.id",
		"issue.deleted_at IS NULL",
		"issue.archived_at IS NULL",
		"issue.is_draft=FALSE",
		`NOT EXISTS (SELECT 1 FROM states public_state WHERE public_state.id=issue.state_id AND public_state."group"='triage')`,
		"project.deleted_at IS NULL",
		"project.archived_at IS NULL",
		"workspace.deleted_at IS NULL",
	}
	where, args, err = appFilters(options, where, args)
	if err != nil {
		return Result{}, err
	}
	assignees, _ := issuequery.ValidUUIDFilter(options.Assignees, true)
	labels, _ := issuequery.ValidUUIDFilter(options.Labels, true)
	modules, _ := issuequery.ValidUUIDFilter(options.Module, true)
	order, err := issuequery.Order(options, assignees, labels, modules, &args)
	if err != nil {
		return Result{}, err
	}
	moduleGroup := options.GroupBy == "issue_module__module_id" || options.SubGroupBy == "issue_module__module_id"
	subgroupOnlyModule := options.GroupBy == "" && options.SubGroupBy == "issue_module__module_id"
	if moduleGroup {
		// Django applies issue_module__deleted_at__isnull=True to the filtered
		// queryset before grouping and pagination. A missing relation survives
		// the nullable join, while a deleted-only relation does not.
		where = append(where, `(
 NOT EXISTS (SELECT 1 FROM module_issues any_bridge WHERE any_bridge.issue_id=issue.id)
 OR EXISTS (SELECT 1 FROM module_issues live_bridge WHERE live_bridge.issue_id=issue.id AND live_bridge.deleted_at IS NULL)
 )`)
	}
	annotationSQL := issueCollectionAnnotationsSQL("$1")
	if moduleGroup {
		// Grouping's module_ids projection is evaluated after the source's live
		// bridge filter. The ordinary, ungrouped collection keeps its broader
		// archived and deleted bridge annotation above.
		annotationSQL = issueAnnotationsSQLWithModuleConditions("$1", false, " AND m.deleted_at IS NULL")
		annotationSQL += `||jsonb_build_object('_module_group_ids',COALESCE((
 SELECT jsonb_agg(DISTINCT live_group.module_id) FROM module_issues live_group
 WHERE live_group.issue_id=i.id AND live_group.deleted_at IS NULL
 ),'[]'::jsonb))`
	}
	annotations := strings.ReplaceAll(annotationSQL, "i.", "issue.")
	from := ` FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=issue.workspace_id`
	if subgroupOnlyModule {
		from += ` LEFT JOIN module_issues selected_module ON selected_module.issue_id=issue.id AND selected_module.deleted_at IS NULL`
		annotations += `||jsonb_build_object('issue_module__module_id',selected_module.module_id)`
	}
	query := `SELECT ` + djangoapi.IssueRowSQL("issue") + `||` + annotations + `||jsonb_build_object(
 'state__group',(SELECT public_state."group" FROM states public_state WHERE public_state.id=issue.state_id),
 'vote_items',(SELECT jsonb_agg(jsonb_build_object('vote', vote.vote, 'actor_details', jsonb_build_object(
   'id', actor.id, 'first_name', actor.first_name, 'last_name', actor.last_name, 'avatar', actor.avatar,
   'avatar_url', CASE WHEN actor.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || actor.avatar_asset_id::text || '/' ELSE actor.avatar END,
   'display_name', actor.display_name))) FROM issue_votes vote JOIN users actor ON actor.id=vote.actor_id WHERE vote.issue_id=issue.id AND vote.deleted_at IS NULL),
 'reaction_items',(SELECT jsonb_agg(jsonb_build_object('reaction', reaction.reaction, 'actor_details', jsonb_build_object(
   'id', actor.id, 'first_name', actor.first_name, 'last_name', actor.last_name, 'avatar', actor.avatar,
   'avatar_url', CASE WHEN actor.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || actor.avatar_asset_id::text || '/' ELSE actor.avatar END,
   'display_name', actor.display_name))) FROM issue_reactions reaction JOIN users actor ON actor.id=reaction.actor_id WHERE reaction.issue_id=issue.id AND reaction.deleted_at IS NULL)
	 )` + from + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order
	if options.GroupBy == "" {
		var total int
		if err := repository.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+query+`) public_count`, args...).Scan(&total); err != nil {
			return Result{}, err
		}
		args = append(args, limit, offset)
		query += ` LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
		rows, err := repository.rows(ctx, query, args...)
		if err != nil {
			return Result{}, err
		}
		results := make([]any, 0, len(rows))
		for _, row := range rows {
			results = append(results, publicIssueProjection(row, subgroupOnlyModule))
		}
		return Result{Status: 200, Body: djangoapi.Envelope(results, total, limit, page)}, nil
	}
	rows, err := repository.rows(ctx, query, args...)
	if err != nil {
		return Result{}, err
	}
	results := make([]any, 0, len(rows))
	for _, row := range rows {
		results = append(results, publicIssueProjection(row, false))
	}
	return repository.groupedPageWithVisibleRowCounts(ctx, q, results, rows, limit, page, true)
}

func publicIssueProjection(row map[string]any, subgroupOnlyModule bool) map[string]any {
	fields := strings.Fields("id name state_id sort_order estimate_point priority start_date target_date sequence_id project_id parent_id cycle_id created_by state__group assignee_ids label_ids module_ids vote_items reaction_items")
	if subgroupOnlyModule {
		fields = slices.DeleteFunc(fields, func(field string) bool { return field == "module_ids" })
		fields = append(fields, "issue_module__module_id")
	}
	value := projectFields(row, fields)
	if row["vote_items"] == nil {
		value["vote_items"] = nil
	}
	if row["reaction_items"] == nil {
		value["reaction_items"] = nil
	}
	return value
}
