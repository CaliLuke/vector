package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func queryOptions(q Request) issuequery.Options {
	o := issuequery.Options{}
	value := reflect.ValueOf(&o).Elem()
	names := map[string]string{"OrderBy": "order_by", "GroupBy": "group_by", "SubGroupBy": "sub_group_by", "StateGroup": "state_group", "EstimatePoint": "estimate_point", "LoggedBy": "logged_by", "CreatedBy": "created_by", "CreatedAt": "created_at", "UpdatedAt": "updated_at", "StartDate": "start_date", "TargetDate": "target_date", "CompletedAt": "completed_at", "IntakeStatus": "intake_status", "InboxStatus": "inbox_status", "SubIssue": "sub_issue", "StartTargetDate": "start_target_date"}
	for i := 0; i < value.NumField(); i++ {
		name := value.Type().Field(i).Name
		field := value.Field(i)
		if field.Kind() == reflect.String {
			key := names[name]
			if key == "" {
				key = strings.ToLower(name)
			}
			field.SetString(q.Params[key])
		} else if field.Kind() == reflect.Bool && strings.HasSuffix(name, "Set") {
			key := strings.TrimSuffix(name, "Set")
			mapped := names[key]
			if mapped == "" {
				mapped = strings.ToLower(key)
			}
			field.SetBool(q.Present[mapped])
		}
	}
	if !o.OrderBySet {
		o.OrderBy = "-created_at"
	}
	return o
}

type issueListShape struct {
	archive, v2, byIDs, details, user, workspace bool
}

type issueListFilterPlan struct {
	where         []string
	args          []any
	options       issuequery.Options
	deferredError error
}

type issueListPage struct {
	rows               []map[string]any
	total, limit, page int
}

func (r *Repository) listIssueResults(ctx context.Context, q Request, typeIDs map[string]bool) (result Result, resultErr error) {
	shape := issueListShapeFor(q)
	defer func() {
		if resultErr == nil && !q.API && !shape.v2 && !shape.workspace && !shape.archive && !shape.details {
			r.recordRecentVisit(ctx, q, "project", q.ProjectID)
		}
	}()
	if err := r.authorizeIssueList(ctx, q, shape); err != nil {
		return Result{}, err
	}
	if q.API {
		if err := validateAPIFilterMode(q); err != nil {
			return Result{}, err
		}
		if q.ExpectedWorkspaceID == nil && q.Params["external_id"] != "" && q.Params["external_source"] != "" {
			return r.externalIssueListResult(ctx, q)
		}
	}
	where, args, err := r.issueListScope(q, shape)
	if err != nil {
		return Result{}, err
	}
	plan, err := r.issueListFilters(ctx, q, shape, typeIDs, where, args)
	if err != nil {
		return Result{}, err
	}
	args = plan.args
	order, err := issueListOrder(q, shape, plan.options, &args)
	if err != nil {
		return Result{}, err
	}
	if err = r.validateIssueListGrouping(q, shape, plan.options); err != nil {
		return Result{}, err
	}
	if plan.deferredError != nil {
		return Result{}, plan.deferredError
	}
	from := ` FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=issue.workspace_id WHERE ` + strings.Join(plan.where, " AND ")
	if q.API && q.ExpectedWorkspaceID != nil && q.Operation == "count" {
		return r.countIssueResults(ctx, q, from, args)
	}
	if !q.API && !shape.v2 && !shape.byIDs && plan.options.GroupBy != "" {
		return r.readGroupedListPage(ctx, q, shape, plan, from, order, args)
	}
	page, err := r.readIssueListPage(ctx, q, shape, from, order, args)
	if err != nil {
		return Result{}, err
	}
	if shape.v2 {
		if !q.API && q.Params["include_properties"] == "true" {
			if err = r.projectDiscoveryRows(ctx, page.rows); err != nil {
				return Result{}, err
			}
		}
		return v2Envelope(q, page.rows, page.total, page.limit, page.page)
	}
	results, err := r.projectIssueListRows(ctx, q, shape, page.rows)
	if err != nil {
		return Result{}, err
	}
	if shape.byIDs {
		return Result{200, results}, nil
	}
	return Result{200, apicontract.Envelope(results, page.total, page.limit, page.page)}, nil
}

func issueListShapeFor(q Request) issueListShape {
	op := q.Operation
	archive := strings.HasSuffix(op, "_archived_issues")
	v2 := strings.HasSuffix(op, "_v2_issues")
	byIDs := strings.HasSuffix(op, "_issues_list")
	details := strings.HasSuffix(op, "_issues_detail")
	user := strings.Contains(op, "_user_issues_by_user_id")
	workspace := q.ProjectID == ""
	return issueListShape{archive: archive, v2: v2, byIDs: byIDs, details: details, user: user, workspace: workspace}
}

func (shape issueListShape) hasDetailProjection() bool {
	return shape.details || shape.workspace && !shape.user
}

func (shape issueListShape) projectsStateGroup() bool {
	if shape.hasDetailProjection() {
		return shape.workspace && !shape.user
	}
	return !shape.byIDs || shape.user
}

func (r *Repository) authorizeIssueList(ctx context.Context, q Request, shape issueListShape) error {
	if !q.API {
		if shape.workspace && !shape.user {
			if err := r.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
				return err
			}
		} else if !shape.workspace {
			roles := []int{5, 15, 20}
			if shape.archive {
				roles = []int{15, 20}
			}
			if err := r.allowProject(ctx, q, roles, false); err != nil {
				return err
			}
		}
	}
	if !q.API && !shape.workspace && !shape.v2 && !shape.byIDs && !shape.details && !shape.archive {
		if _, err := r.project(ctx, q.ProjectID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) externalIssueListResult(ctx context.Context, q Request) (Result, error) {

	row, err := r.row(ctx, `SELECT `+apicontract.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.external_id=$3 AND i.external_source=$4 AND i.deleted_at IS NULL`, q.Slug, q.ProjectID, q.Params["external_id"], q.Params["external_source"])
	if err != nil {
		var missing *apicontract.HTTPError
		if errors.As(err, &missing) && missing.Status == 404 {
			return Result{}, failure(404, "error", "The requested resource does not exist.")
		}
		return Result{}, err
	}
	if err = r.authorizeGuestItemRead(ctx, q, row); err != nil {
		return Result{}, err
	}
	result, err := r.serializeVisibleIssue(ctx, row, q, q.Params["fields"], q.Params["expand"])
	return Result{200, result}, err
}

func (r *Repository) issueListScope(q Request, shape issueListShape) ([]string, []any, error) {
	args := []any{q.ActorID, q.Slug}
	where := []string{`workspace.slug=$2`, strings.ReplaceAll(liveIssue, "i.", "issue.")}
	if shape.archive {
		where = []string{`workspace.slug=$2`, `issue.deleted_at IS NULL`, `issue.archived_at IS NOT NULL`, `NOT EXISTS(SELECT 1 FROM issue_types t WHERE t.id=issue.type_id AND t.is_epic)`}
		if r.parentGrouping == nil && q.Present["show_sub_issues"] && q.Params["show_sub_issues"] != "true" {
			where = append(where, `issue.parent_id IS NULL`)
		}
	}
	where = append(where, "project.deleted_at IS NULL", "workspace.deleted_at IS NULL")
	if !shape.workspace {
		args = append(args, q.ProjectID)
		where = append(where, "issue.project_id=$3")
	}
	add := func(value any, format string) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(format, len(args)))
	}
	where, args = appendBoundWorkitemScope(q, where, args)
	if q.ExpectedWorkspaceID != nil {
		for _, key := range []string{"external_id", "external_source"} {
			if q.Params[key] != "" {
				add(q.Params[key], "issue."+key+"=$%d")
			}
		}
	}
	if shape.workspace || shape.details {
		where = append(where, `EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=issue.project_id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL)`)
	}
	if shape.user {
		add(q.Params["user_id"], `(issue.created_by_id=$%[1]d OR EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=issue.id AND a.assignee_id=$%[1]d) OR EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=issue.id AND s.subscriber_id=$%[1]d))`)
	}
	where = append(where, `(project.guest_view_all_features OR issue.created_by_id=$1 OR NOT EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=issue.project_id AND pm.member_id=$1 AND pm.is_active AND pm.role=5 AND pm.deleted_at IS NULL))`)
	if shape.byIDs {
		raw := q.Params["issues"]
		if raw == "" {
			return nil, nil, failure(400, "error", "Issues are required")
		}
		values := []any{}
		for _, id := range strings.Split(raw, ",") {
			if id != "" {
				values = append(values, id)
			}
		}
		ids, err := rawIDs(values)
		if err != nil {
			return nil, nil, err
		}
		add(ids, `issue.id=ANY($%d::uuid[])`)
		where = append(where, `NOT EXISTS(SELECT 1 FROM states s WHERE s.id=issue.state_id AND s.deleted_at IS NOT NULL)`)
	}
	return where, args, nil
}

func (r *Repository) issueListFilters(ctx context.Context, q Request, shape issueListShape, typeIDs map[string]bool, where []string, args []any) (issueListFilterPlan, error) {
	options := queryOptions(q)
	add := func(value any, format string) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(format, len(args)))
	}
	var err error
	where, args, err = r.appendPQL(q, where, args)
	if err != nil {
		return issueListFilterPlan{}, err
	}
	var deferredFilterError error
	if r.parentGrouping != nil {
		options.SubIssue = "true"
	}
	if shape.workspace && !shape.user && !usesDiscoveryGrouping(q) {
		options.GroupBy, options.SubGroupBy = "", ""
	}
	if !q.API {
		flatOptions := listFilterOptions(options, shape.v2)
		if r.propertyFilter != nil {
			flatOptions.Filters = ""
		}
		where, args, err = appFiltersWithTypes(flatOptions, where, args, typeIDs)
		if err != nil {
			if !errors.Is(err, issuequery.ErrInvalidComplexRange) {
				return issueListFilterPlan{}, err
			}
			deferredFilterError = err
		}
	}
	where, args, err = r.appendDiscoveryGroupSelectors(ctx, q, where, args)
	if err != nil {
		return issueListFilterPlan{}, err
	}
	if !q.API && q.Present["updated_at__gt"] {
		value, e := parseQueryDateTime(q.Params["updated_at__gt"], q.Timezone)
		if e != nil {
			return issueListFilterPlan{}, failure(400, "error", "Please provide valid detail")
		}
		add(value, `issue.updated_at>$%d`)
	}
	if r.propertyFilter != nil && deferredFilterError == nil {
		options.PropertyFilter, err = r.bindPropertyFilters(ctx, where, args)
		if err != nil {
			return issueListFilterPlan{}, err
		}
		predicate, filterErr := options.PropertyFilter.SQL(&args)
		if filterErr != nil {
			return issueListFilterPlan{}, filterErr
		}
		where = append(where, predicate)
	}
	where, args, err = r.appendPropertyPQL(ctx, where, args)
	if err != nil {
		return issueListFilterPlan{}, err
	}
	return issueListFilterPlan{where: where, args: args, options: options, deferredError: deferredFilterError}, nil
}

func issueListOrder(q Request, shape issueListShape, options issuequery.Options, args *[]any) (string, error) {
	if shape.v2 {
		return "issue.updated_at ASC", nil
	}
	assignees, _ := issuequery.ValidUUIDFilter(options.Assignees, true)
	labels, _ := issuequery.ValidUUIDFilter(options.Labels, true)
	modules, _ := issuequery.ValidUUIDFilter(options.Module, true)
	order, err := issuequery.Order(options, assignees, labels, modules, args)
	if err != nil {
		return "", err
	}
	if q.API {
		order = apiIssueOrderSQL(q.Params["order_by"])
	}
	return order, nil
}

func (r *Repository) validateIssueListGrouping(q Request, shape issueListShape, options issuequery.Options) error {
	if !q.API && !shape.v2 && !shape.byIDs && options.GroupBy != "" && options.GroupBy == options.SubGroupBy {
		return failure(400, "error", "Group by and sub group by cannot have same parameters")
	}
	if !q.API && !shape.v2 && !shape.byIDs {
		if err := r.validateDiscoveryGrouping(options.GroupBy, options.SubGroupBy); err != nil {
			return failure(400, "detail", err.Error())
		}
	}
	return nil
}

func (r *Repository) readIssueListPage(ctx context.Context, q Request, shape issueListShape, from, order string, args []any) (issueListPage, error) {
	var err error
	if !q.API {
		order += ",issue.created_at DESC"
	}
	sql := `SELECT ` + issueListProjectionSQL(shape, !q.API) + from + ` ORDER BY ` + order
	limit, page, offset, total := 0, 0, 0, 0
	if !shape.byIDs {
		if shape.v2 {
			limit, page, err = v2Window(q)
			offset = max(0, page) * limit
		} else {
			limit, page, offset, err = apicontract.PageWindow(apicontract.Query{Params: q.Params, Present: q.Present})
		}
		if err != nil {
			return issueListPage{}, err
		}
		// Wrap the ordered query so positional parameters and invalid ordering
		// retain their source behavior, while PostgreSQL bounds returned rows.
		if err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+sql+`) counted`, args...).Scan(&total); err != nil {
			return issueListPage{}, err
		}
		args = append(args, limit, offset)
		sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := r.rows(ctx, sql, args...)
	if err != nil {
		return issueListPage{}, err
	}
	return issueListPage{rows: rows, total: total, limit: limit, page: page}, nil
}

func issueListProjectionSQL(shape issueListShape, browser bool) string {
	projection := `(` + apicontract.IssueRowSQL("issue") + `||` + strings.ReplaceAll(issueAnnotationsSQL("$1", shape.v2), "i.", "issue.") + `||jsonb_build_object('state__group',(SELECT "group" FROM states s WHERE s.id=issue.state_id))`
	if browser && shape.hasDetailProjection() {
		projection += `||jsonb_build_object('module_ids', ARRAY(SELECT module_id FROM module_issues m WHERE m.issue_id=issue.id AND m.deleted_at IS NULL ORDER BY m.created_at DESC))`
	}
	return projection + ")"
}

func (r *Repository) projectIssueListRows(ctx context.Context, q Request, shape issueListShape, rows []map[string]any) ([]any, error) {
	if !q.API && (q.Params["include_properties"] == "true" || len(r.propertyGroups) > 0) {
		if err := r.projectDiscoveryRows(ctx, rows); err != nil {
			return nil, err
		}
	}
	if !q.API && shape.hasDetailProjection() {
		if err := r.loadListDetailRelations(ctx, q, rows); err != nil {
			return nil, err
		}
	}
	results := make([]any, 0, len(rows))
	for _, row := range rows {
		value, err := r.projectIssueListRow(ctx, q, shape, row)
		if err != nil {
			return nil, err
		}
		results = append(results, value)
	}
	return results, nil
}

func (r *Repository) projectIssueListRow(ctx context.Context, q Request, shape issueListShape, row map[string]any) (map[string]any, error) {
	var err error
	if q.API {
		value, e := r.serializeVisibleIssue(ctx, row, q, q.Params["fields"], q.Params["expand"])
		if e != nil {
			return nil, e
		}
		return value, nil
	}
	if !shape.byIDs && !shape.details && !shape.workspace {
		for _, field := range []string{"sub_issues_count", "attachment_count", "link_count"} {
			if row[field] == json.Number("0") {
				row[field] = nil
			}
		}
	}
	fields := compactFields
	detail := shape.hasDetailProjection()
	if detail {
		row["label_ids"] = row["labels"]
		row["assignee_ids"] = row["assignees"]
	}
	if !detail && shape.byIDs && q.Params["expand"] == "" && q.Params["fields"] == "" {
		fields = append(append([]string{}, compactFields...), "deleted_at")
	}
	if shape.projectsStateGroup() {
		fields = append(append([]string{}, compactFields...), "state__group")
	}
	value, projectionErr := projectFields(row, fields)
	if projectionErr != nil {
		return nil, projectionErr
	}
	if !detail && shape.byIDs && (q.Params["expand"] != "" || q.Params["fields"] != "") {
		value, err = r.expandedCompact(ctx, row, q, value)
		if err != nil {
			return nil, err
		}
	}
	if detail {
		for _, field := range []string{"issue_relation", "issue_related"} {
			if related, expanded := row[field]; expanded {
				value[field] = related
			}
		}
	}
	if r.propertyDiscovery != nil && q.Params["include_properties"] == "true" {
		value["property_values"], value["properties_evaluated_at"] = row["property_values"], row["properties_evaluated_at"]
		value["properties_revision"] = row["properties_revision"]
	}
	r.projectPropertyGroups(row, value)
	r.projectParentGroup(row, value)
	return value, nil
}

func apiIssueOrderSQL(raw string) string {
	name := strings.TrimPrefix(raw, "-")
	direction := " ASC"
	if strings.HasPrefix(raw, "-") {
		direction = " DESC"
	}
	switch name {
	case "priority":
		return `array_position(ARRAY['urgent','high','medium','low','none'],issue.priority)` + direction
	case "state__name", "state__group":
		return `(SELECT array_position(ARRAY['backlog','unstarted','started','completed','cancelled'],s."group") FROM states s WHERE s.id=issue.state_id)` + direction
	case "assignees__first_name":
		return `(SELECT MAX(u.first_name) FROM issue_assignees a JOIN users u ON u.id=a.assignee_id WHERE a.issue_id=issue.id)` + direction
	case "labels__name":
		return `(SELECT MAX(l.name) FROM issue_labels a JOIN labels l ON l.id=a.label_id WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "a") + `)` + direction
	case "issue_module__module__name":
		return `(SELECT MAX(m.name) FROM module_issues a JOIN modules m ON m.id=a.module_id WHERE a.issue_id=issue.id)` + direction
	case "created_at", "updated_at", "sequence_id", "sort_order", "target_date", "start_date", "completed_at", "archived_at":
		return "issue." + name + direction
	default:
		return "issue.created_at DESC"
	}
}
func (r *Repository) expandedCompact(ctx context.Context, row map[string]any, q Request, value map[string]any) (map[string]any, error) {
	return r.expandAppIssue(ctx, row, q, value)
}
func (r *Repository) loadListDetailRelations(ctx context.Context, q Request, issues []map[string]any) error {
	if len(issues) == 0 {
		return nil
	}
	ids := make([]string, 0, len(issues))
	byID := make(map[string]map[string]any, len(issues))
	for _, issue := range issues {
		id := stringValue(issue["id"])
		ids = append(ids, id)
		byID[id] = issue
	}
	for _, field := range []string{"issue_relation", "issue_related"} {
		if !strings.Contains(","+q.Params["expand"]+",", ","+field+",") {
			continue
		}
		side, anchor := "related_issue_id", "issue_id"
		if field == "issue_related" {
			side, anchor = anchor, side
		}
		rows, err := r.rows(ctx, `SELECT to_jsonb(i)||jsonb_build_object('relation_type',rel.relation_type,'anchor',rel.`+anchor+`) FROM issue_relations rel JOIN issues i ON i.id=rel.`+side+` WHERE rel.`+anchor+`=ANY($1::uuid[]) AND rel.deleted_at IS NULL ORDER BY rel.created_at DESC`, ids)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			issue[field] = []any{}
		}
		for _, row := range rows {
			projected, err := projectFields(row, strings.Fields("id project_id sequence_id name relation_type state_id priority created_by created_at updated_at updated_by"))
			if err != nil {
				return err
			}
			issue := byID[stringValue(row["anchor"])]
			issue[field] = append(issue[field].([]any), projected)
		}
	}
	return nil
}

func (r *Repository) v2Page(q Request, rows []map[string]any) (Result, error) {
	limit, page, err := v2Window(q)
	if err != nil {
		return Result{}, err
	}
	total := len(rows)
	start := min(total, page*limit)
	end := min(total, start+limit)
	return v2Envelope(q, rows[start:end], total, limit, page)
}

func v2Window(q Request) (int, int, error) {
	var cursor *string
	if q.Present["cursor"] {
		value := q.Params["cursor"]
		cursor = &value
	}
	limit, page, _, err := apicontract.CursorWindow(cursor)
	return limit, page, err
}

func v2Envelope(q Request, rows []map[string]any, total, limit, page int) (Result, error) {
	start := min(total, page*limit)
	end := min(total, start+limit)
	results := []any{}
	fields := append(append([]string{}, compactFields...), "state__group")
	if !q.API && q.Params["include_properties"] == "true" {
		fields = append(fields, "property_values", "properties_evaluated_at", "properties_revision")
	}
	if strings.ToLower(q.Params["description"]) == "true" {
		fields = append(fields, "description_html")
	}
	for _, row := range rows {
		for _, field := range []string{"sub_issues_count", "attachment_count", "link_count"} {
			if row[field] == json.Number("0") {
				row[field] = nil
			}
		}
		projected, projectionErr := projectFields(row, fields)
		if projectionErr != nil {
			return Result{}, projectionErr
		}
		results = append(results, projected)
	}
	var next any
	if end < total {
		next = fmt.Sprintf("%d:%d:0", limit, page+1)
	}
	return Result{200, map[string]any{"results": results, "cursor": fmt.Sprintf("%d:%d:0", limit, page), "next_cursor": next, "prev_cursor": fmt.Sprintf("%d:%d:0", limit, page-1), "next_page_results": end < total, "prev_page_results": page > 0, "page_count": len(results), "total_pages": int(math.Ceil(float64(total) / float64(limit))), "total_results": total}}, nil
}
