package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
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
func (r *Repository) listIssueResults(ctx context.Context, q Request, typeIDs map[string]bool) (Result, error) {
	op := q.Operation
	archive := strings.HasSuffix(op, "_archived_issues")
	v2 := strings.HasSuffix(op, "_v2_issues")
	byIDs := strings.HasSuffix(op, "_issues_list")
	details := strings.HasSuffix(op, "_issues_detail")
	user := strings.Contains(op, "_user_issues_by_user_id")
	workspace := q.ProjectID == ""
	if !q.API {
		if workspace && !user {
			if err := r.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
				return Result{}, err
			}
		} else if !workspace {
			roles := []int{5, 15, 20}
			if archive {
				roles = []int{15, 20}
			}
			if err := r.allowProject(ctx, q, roles, false); err != nil {
				return Result{}, err
			}
		}
	}
	if !q.API && !workspace && !v2 && !byIDs && !details && !archive {
		if _, err := r.project(ctx, q.ProjectID); err != nil {
			return Result{}, err
		}
	}
	if q.API {
		unsupported := []string{}
		for _, key := range []string{"pql", "filters"} {
			if q.Params[key] != "" {
				unsupported = append(unsupported, key)
			}
		}
		if len(unsupported) > 0 {
			return Result{}, invalid(map[string]any{"pql": "PQL and structured filters are not supported in Vector. Remove the pql/filters parameter and filter results client-side.", "unsupported_parameters": unsupported})
		}
		if q.Params["external_id"] != "" && q.Params["external_source"] != "" {
			row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.external_id=$3 AND i.external_source=$4 AND i.deleted_at IS NULL`, q.Slug, q.ProjectID, q.Params["external_id"], q.Params["external_source"])
			if err != nil {
				var missing *djangoapi.HTTPError
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
	}
	args := []any{q.ActorID, q.Slug}
	where := []string{`workspace.slug=$2`, strings.ReplaceAll(liveIssue, "i.", "issue.")}
	if archive {
		where = []string{`workspace.slug=$2`, `issue.deleted_at IS NULL`, `issue.archived_at IS NOT NULL`, `NOT EXISTS(SELECT 1 FROM issue_types t WHERE t.id=issue.type_id AND t.is_epic)`}
		if r.parentGrouping == nil && q.Present["show_sub_issues"] && q.Params["show_sub_issues"] != "true" {
			where = append(where, `issue.parent_id IS NULL`)
		}
	}
	where = append(where, "project.deleted_at IS NULL", "workspace.deleted_at IS NULL")
	if !workspace {
		args = append(args, q.ProjectID)
		where = append(where, "issue.project_id=$3")
	}
	add := func(value any, format string) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(format, len(args)))
	}
	if workspace || details {
		where = append(where, `EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=issue.project_id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL)`)
	}
	if user {
		add(q.Params["user_id"], `(issue.created_by_id=$%[1]d OR EXISTS(SELECT 1 FROM issue_assignees a WHERE a.issue_id=issue.id AND a.assignee_id=$%[1]d) OR EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=issue.id AND s.subscriber_id=$%[1]d))`)
	}
	where = append(where, `(project.guest_view_all_features OR issue.created_by_id=$1 OR NOT EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=issue.project_id AND pm.member_id=$1 AND pm.is_active AND pm.role=5 AND pm.deleted_at IS NULL))`)
	if byIDs {
		raw := q.Params["issues"]
		if raw == "" {
			return Result{}, failure(400, "error", "Issues are required")
		}
		values := []any{}
		for _, id := range strings.Split(raw, ",") {
			if id != "" {
				values = append(values, id)
			}
		}
		ids, err := rawIDs(values)
		if err != nil {
			return Result{}, err
		}
		add(ids, `issue.id=ANY($%d::uuid[])`)
		where = append(where, `NOT EXISTS(SELECT 1 FROM states s WHERE s.id=issue.state_id AND s.deleted_at IS NOT NULL)`)
	}
	var err error
	var deferredFilterError error
	options := queryOptions(q)
	if r.parentGrouping != nil {
		options.SubIssue = "true"
	}
	if workspace && !user && !usesDiscoveryGrouping(q) {
		options.GroupBy, options.SubGroupBy = "", ""
	}
	if !q.API {
		flatOptions := listFilterOptions(options, v2)
		if r.propertyFilter != nil {
			flatOptions.Filters = ""
		}
		where, args, err = appFiltersWithTypes(flatOptions, where, args, typeIDs)
		if err != nil {
			if !errors.Is(err, issuequery.ErrInvalidComplexRange) {
				return Result{}, err
			}
			// Django constructs this invalid range lazily, then queues the visit
			// before evaluating the queryset.
			deferredFilterError = err
		}
	}
	where, args, err = r.appendDiscoveryGroupSelectors(ctx, q, where, args)
	if err != nil {
		return Result{}, err
	}
	if !q.API && q.Present["updated_at__gt"] {
		value, e := parseQueryDateTime(q.Params["updated_at__gt"], q.Timezone)
		if e != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
		add(value, `issue.updated_at>$%d`)
	}
	if r.propertyFilter != nil && deferredFilterError == nil {
		options.PropertyFilter, err = r.bindPropertyFilters(ctx, where, args)
		if err != nil {
			return Result{}, err
		}
		predicate, filterErr := options.PropertyFilter.SQL(&args)
		if filterErr != nil {
			return Result{}, filterErr
		}
		where = append(where, predicate)
	}
	order := "issue.created_at DESC"
	if v2 {
		order = "issue.updated_at ASC"
	} else {
		assignees, _ := issuequery.ValidUUIDFilter(options.Assignees, true)
		labels, _ := issuequery.ValidUUIDFilter(options.Labels, true)
		modules, _ := issuequery.ValidUUIDFilter(options.Module, true)
		order, err = issuequery.Order(options, assignees, labels, modules, &args)
		if err != nil {
			return Result{}, err
		}
		if q.API {
			order = apiIssueOrderSQL(q.Params["order_by"])
		}
	}
	if !q.API && !v2 && !workspace && !archive && !details {
		if err = r.recentVisit(ctx, q, "project", q.ProjectID); err != nil {
			return Result{}, err
		}
	}
	if !q.API && !v2 && !byIDs && options.GroupBy != "" && options.GroupBy == options.SubGroupBy {
		return Result{}, failure(400, "error", "Group by and sub group by cannot have same parameters")
	}
	if !q.API && !v2 && !byIDs {
		if err = r.validateDiscoveryGrouping(options.GroupBy, options.SubGroupBy); err != nil {
			return Result{}, failure(400, "detail", err.Error())
		}
	}
	if deferredFilterError != nil {
		return Result{}, deferredFilterError
	}
	from := ` FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=issue.workspace_id WHERE ` + strings.Join(where, " AND ")
	if !q.API {
		order += ",issue.created_at DESC"
	}
	sql := `SELECT ` + djangoapi.IssueRowSQL("issue") + `||` + strings.ReplaceAll(issueAnnotationsSQL("$1", v2), "i.", "issue.") + `||jsonb_build_object('state__group',(SELECT "group" FROM states s WHERE s.id=issue.state_id))` + from + ` ORDER BY ` + order
	limit, page, offset, total := 0, 0, 0, 0
	pageInSQL := v2 || !byIDs && (q.API || options.GroupBy == "")
	if pageInSQL {
		if v2 {
			limit, page, err = v2Window(q)
			offset = max(0, page) * limit
		} else {
			limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
		}
		if err != nil {
			return Result{}, err
		}
		// Wrap the ordered query so positional parameters and invalid ordering
		// retain their source behavior, while PostgreSQL bounds returned rows.
		if err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+sql+`) counted`, args...).Scan(&total); err != nil {
			return Result{}, err
		}
		args = append(args, limit, offset)
		sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	rows, err := r.rows(ctx, sql, args...)
	if err != nil {
		return Result{}, err
	}
	if q.Params["include_properties"] == "true" || len(r.propertyGroups) > 0 {
		if err = r.projectDiscoveryRows(ctx, rows); err != nil {
			return Result{}, err
		}
	}
	if v2 {
		return v2Envelope(q, rows, total, limit, page), nil
	}
	results := []any{}
	for _, row := range rows {
		if q.API {
			value, e := r.serializeVisibleIssue(ctx, row, q, q.Params["fields"], q.Params["expand"])
			if e != nil {
				return Result{}, e
			}
			results = append(results, value)
			continue
		}
		if !byIDs && !details && !workspace {
			for _, field := range []string{"sub_issues_count", "attachment_count", "link_count"} {
				if row[field] == json.Number("0") {
					row[field] = nil
				}
			}
		}
		fields := compactFields
		if byIDs && q.Params["expand"] == "" && q.Params["fields"] == "" {
			fields = append(append([]string{}, compactFields...), "deleted_at")
		}
		if !byIDs && !details || user {
			fields = append(append([]string{}, compactFields...), "state__group")
		}
		value := projectFields(row, fields)
		if byIDs && (q.Params["expand"] != "" || q.Params["fields"] != "") {
			value, err = r.expandedCompact(ctx, row, q, value)
			if err != nil {
				return Result{}, err
			}
		}
		if details || workspace && !user {
			value, err = r.listDetailProjection(ctx, row, q)
			if err != nil {
				return Result{}, err
			}
			if workspace && !user {
				value["state__group"] = row["state__group"]
			}
		}
		if r.propertyDiscovery != nil && q.Params["include_properties"] == "true" {
			value["property_values"], value["properties_evaluated_at"] = row["property_values"], row["properties_evaluated_at"]
			value["properties_revision"] = row["properties_revision"]
		}
		r.projectPropertyGroups(row, value)
		r.projectParentGroup(row, value)
		results = append(results, value)
	}
	if byIDs {
		return Result{200, results}, nil
	}
	if pageInSQL {
		return Result{200, djangoapi.Envelope(results, total, limit, page)}, nil
	}
	limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return Result{}, err
	}
	if !q.API && options.GroupBy != "" {
		return r.groupedPage(ctx, q, results, rows, limit, page)
	}
	total = len(results)
	end := min(total, offset+limit)
	offset = min(offset, total)
	return Result{200, djangoapi.Envelope(results[offset:end], total, limit, page)}, nil
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
func (r *Repository) listDetailProjection(ctx context.Context, row map[string]any, q Request) (map[string]any, error) {
	value := projectFields(row, compactFields)
	for _, rel := range []struct{ field, table, column string }{{"module_ids", "module_issues", "module_id"}, {"label_ids", "issue_labels", "label_id"}, {"assignee_ids", "issue_assignees", "assignee_id"}} {
		query := `SELECT jsonb_build_object('id',` + rel.column + `) FROM ` + rel.table + ` WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`
		if rel.field == "label_ids" {
			query = `SELECT jsonb_build_object('id',label.label_id) FROM issue_labels label JOIN issues issue ON issue.id=label.issue_id WHERE issue.id=$1 AND ` + issuequery.LiveIssueLabelPredicate("issue", "label") + ` ORDER BY label.created_at DESC`
		}
		items, err := r.rows(ctx, query, row["id"])
		if err != nil {
			return nil, err
		}
		ids := []any{}
		for _, item := range items {
			ids = append(ids, item["id"])
		}
		value[rel.field] = ids
	}
	for _, field := range []string{"issue_relation", "issue_related"} {
		if !strings.Contains(","+q.Params["expand"]+",", ","+field+",") {
			continue
		}
		side, anchor := "related_issue_id", "issue_id"
		if field == "issue_related" {
			side, anchor = anchor, side
		}
		rows, err := r.rows(ctx, `SELECT to_jsonb(i)||jsonb_build_object('relation_type',rel.relation_type) FROM issue_relations rel JOIN issues i ON i.id=rel.`+side+` WHERE rel.`+anchor+`=$1 AND rel.deleted_at IS NULL ORDER BY rel.created_at DESC`, row["id"])
		if err != nil {
			return nil, err
		}
		items := []any{}
		for _, item := range rows {
			items = append(items, projectFields(item, strings.Fields("id project_id sequence_id name relation_type state_id priority created_by created_at updated_at updated_by")))
		}
		value[field] = items
	}
	return value, nil
}
func (r *Repository) v2Page(q Request, rows []map[string]any) (Result, error) {
	limit, page, err := v2Window(q)
	if err != nil {
		return Result{}, err
	}
	total := len(rows)
	start := min(total, max(0, page)*limit)
	end := min(total, start+limit)
	return v2Envelope(q, rows[start:end], total, limit, page), nil
}

func v2Window(q Request) (int, int, error) {
	limit, page := 1000, 0
	if q.Present["cursor"] {
		parts := strings.Split(q.Params["cursor"], ":")
		if len(parts) != 3 {
			return 0, 0, errors.New("invalid global cursor")
		}
		numbers := []int{}
		for _, part := range parts {
			n, e := pythoncompat.DecimalInteger(part)
			if e != nil || !n.IsInt64() {
				return 0, 0, errors.New("invalid global cursor")
			}
			numbers = append(numbers, int(n.Int64()))
		}
		limit, page = min(numbers[0], 1000), numbers[1]
	}
	if limit <= 0 || page > math.MaxInt/limit {
		return 0, 0, errors.New("invalid global pagination")
	}
	return limit, page, nil
}

func v2Envelope(q Request, rows []map[string]any, total, limit, page int) Result {
	start := min(total, max(0, page)*limit)
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
		results = append(results, projectFields(row, fields))
	}
	var next any
	if end < total {
		next = fmt.Sprintf("%d:%d:0", limit, page+1)
	}
	return Result{200, map[string]any{"results": results, "cursor": fmt.Sprintf("%d:%d:0", limit, page), "next_cursor": next, "prev_cursor": fmt.Sprintf("%d:%d:0", limit, page-1), "next_page_results": end < total, "prev_page_results": page > 0, "page_count": len(results), "total_pages": int(math.Ceil(float64(total) / float64(limit))), "total_results": total}}
}
