package workitems

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

var relationTypes = []string{"blocking", "blocked_by", "duplicate", "relates_to", "start_after", "start_before", "finish_after", "finish_before"}

func actualRelation(value string) string {
	switch value {
	case "blocking":
		return "blocked_by"
	case "start_after":
		return "start_before"
	case "finish_after":
		return "finish_before"
	case "implements":
		return "implemented_by"
	}
	return value
}
func inverseRelation(value string) string {
	switch value {
	case "blocked_by":
		return "blocking"
	case "start_before":
		return "start_after"
	case "finish_before":
		return "finish_after"
	}
	return value
}
func reverseRelation(value string) bool {
	return value == "blocking" || value == "start_after" || value == "finish_after"
}

func (r *Repository) activity(ctx context.Context, q Request, kind string, requested, previous any, defaults bool) error {
	data := map[string]any{"type": kind, "requested_data": requested, "actor_id": q.ActorID, "issue_id": q.IssueID, "project_id": q.ProjectID, "current_instance": previous, "epoch": r.now().Unix()}
	if !defaults {
		data["notification"] = true
		data["origin"] = r.origin
	}
	return r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, data)
}

func parseObject(body io.Reader, contentType string) (djangoinput.Input, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err == nil && input.Kind != "" {
		err = errors.New("request data has no get method")
	}
	return input, err
}
func rawIDs(value any) ([]string, error) {
	values, ok := value.([]any)
	if !ok {
		if value == nil {
			return []string{}, nil
		}
		if s, yes := value.(string); yes {
			values = []any{}
			for _, c := range s {
				values = append(values, string(c))
			}
		} else {
			return nil, errors.New("ids are not iterable")
		}
	}
	ids := []string{}
	for _, v := range values {
		id, e := djangoinput.UUID(v)
		if e != nil {
			return nil, failure(400, "error", "Please provide valid detail")
		}
		ids = append(ids, id.String())
	}
	return ids, nil
}
func (r *Repository) workspaceAnchor(ctx context.Context, q Request, live bool) (map[string]any, error) {
	pred := "i.deleted_at IS NULL"
	if live {
		pred = liveIssue
	}
	return r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND w.slug=$2 AND `+pred, q.IssueID, q.Slug)
}
func (r *Repository) eligibleIssues(ctx context.Context, q Request, ids []string, project bool) ([]map[string]any, error) {
	sql := `SELECT ` + djangoapi.IssueRowSQL("i") + ` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=ANY($1::uuid[]) AND w.slug=$2 AND ` + liveIssue
	args := []any{ids, q.Slug}
	if project {
		sql += " AND i.project_id=$3"
		args = append(args, q.ProjectID)
	}
	return r.rows(ctx, sql+" ORDER BY i.created_at DESC", args...)
}

func (r *Repository) relations(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if q.Method == "GET" {
		return r.listRelations(ctx, q)
	}
	input, err := parseObject(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if strings.HasSuffix(q.Operation, "_remove_relation") {
		return r.removeRelation(ctx, q, input)
	}
	relation, present := input.Fields["relation_type"]
	value := djangoinput.ScalarString(relation)
	if !q.API && (!present || relation == nil) {
		return Result{}, failure(400, "message", "Issue relation type is required")
	}
	raw, hasIssues := input.ListValue("issues", false)
	ids := []string{}
	if q.API {
		fields := map[string]any{}
		if !present {
			fields["relation_type"] = []string{"This field is required."}
		} else if relation == nil {
			fields["relation_type"] = []string{"This field may not be null."}
		} else if !slices.Contains(relationTypes, value) {
			fields["relation_type"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", value)}
		}
		switch list := raw.(type) {
		case []any:
			bad := map[string]any{}
			for i, item := range list {
				id, e := djangoinput.UUID(item)
				if e != nil {
					bad[strconv.Itoa(i)] = []string{"Must be a valid UUID."}
				} else {
					ids = append(ids, id.String())
				}
			}
			if len(bad) > 0 {
				fields["issues"] = bad
			} else if len(list) == 0 {
				fields["issues"] = []string{"Ensure this field has at least 1 elements."}
			}
		default:
			if !hasIssues {
				fields["issues"] = []string{"This field is required."}
			} else if raw == nil {
				fields["issues"] = []string{"This field may not be null."}
			} else {
				fields["issues"] = []string{fmt.Sprintf("Expected a list of items but got type \"%s\".", djangoinput.PythonType(raw))}
			}
		}
		if len(fields) > 0 {
			return Result{}, invalid(fields)
		}
	} else {
		ids, err = rawIDs(raw)
		if err != nil {
			return Result{}, err
		}
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	// The selected target contract closes the source's unscoped relation anchor.
	if _, err = r.workspaceAnchor(ctx, q, false); err != nil {
		return Result{}, err
	}
	targets, err := r.eligibleIssues(ctx, q, ids, false)
	if err != nil {
		return Result{}, err
	}
	actual := actualRelation(value)
	reverse := reverseRelation(value)
	created := []any{}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	for _, target := range targets {
		issue, related := q.IssueID, target["id"].(string)
		if reverse {
			issue, related = related, issue
		}
		now := timestamp(r.now())
		_, err = tx.Exec(ctx, `INSERT INTO issue_relations(id,issue_id,related_issue_id,relation_type,project_id,workspace_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$7,$8,$8) ON CONFLICT DO NOTHING`, uuid.NewString(), issue, related, actual, q.ProjectID, project["workspace_id"], q.ActorID, now)
		if err != nil {
			return Result{}, err
		}
		result := relationProjection(map[string]any{
			"relation_type": actual, "created_by_id": q.ActorID, "updated_by_id": q.ActorID,
			"created_at": now, "updated_at": now,
		}, target)
		created = append(created, result)
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	if err = r.activity(ctx, q, "issue_relation.activity.created", requested, nil, false); err != nil {
		return Result{}, err
	}
	if q.API {
		selected := []string{}
		for _, target := range targets {
			selected = append(selected, target["id"].(string))
		}
		side, other := "related_issue_id", "issue_id"
		if reverse {
			side, other = other, side
		}
		rows, e := r.rows(ctx, `SELECT to_jsonb(rel)||jsonb_build_object('target',to_jsonb(i)) FROM issue_relations rel JOIN issues i ON i.id=rel.`+side+` JOIN workspaces w ON w.id=rel.workspace_id WHERE rel.`+other+`=$1 AND rel.`+side+`=ANY($2::uuid[]) AND rel.relation_type=$3 AND w.slug=$4 AND rel.deleted_at IS NULL ORDER BY rel.created_at DESC`, q.IssueID, selected, actual, q.Slug)
		if e != nil {
			return Result{}, e
		}
		created = []any{}
		for _, row := range rows {
			created = append(created, relationProjection(row, row["target"].(map[string]any)))
		}
	}
	return Result{201, created}, nil
}
func relationProjection(relation, target map[string]any) map[string]any {
	out := projectFields(target, strings.Fields("id project_id sequence_id name state_id priority"))
	// The source field traverses state.id; a missing state omits the field.
	if out["state_id"] == nil {
		delete(out, "state_id")
	}
	for _, key := range []string{"created_at", "updated_at", "relation_type"} {
		out[key] = relation[key]
	}
	out["created_by"], out["updated_by"] = relation["created_by_id"], relation["updated_by_id"]
	return out
}
func (r *Repository) listRelations(ctx context.Context, q Request) (Result, error) {
	rows, err := r.visibleCollaborationRows(ctx, q, "relation", `SELECT to_jsonb(rel)||jsonb_build_object('left_project',i.project_id,'right_project',j.project_id) FROM issue_relations rel JOIN issues i ON i.id=rel.issue_id JOIN issues j ON j.id=rel.related_issue_id JOIN workspaces w ON w.id=rel.workspace_id WHERE (rel.issue_id=$1 OR rel.related_issue_id=$1) AND w.slug=$2 AND rel.deleted_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "w.id")+` AND `+djangoapi.IssueVisibleSQL("j", "$3::uuid", "w.id")+` ORDER BY rel.created_at DESC`, q.IssueID, q.Slug, q.ActorID)
	if err != nil {
		return Result{}, err
	}
	out := map[string]any{}
	selected := map[string][]string{}
	seen := map[string]bool{}
	for _, k := range relationTypes {
		out[k] = []any{}
		selected[k] = []string{}
	}
	for _, row := range rows {
		kind := row["relation_type"].(string)
		if !slices.Contains(relationTypes, kind) {
			continue
		}
		for _, side := range []bool{false, true} {
			anchor, target, project := row["issue_id"], row["related_issue_id"], row["right_project"]
			key := kind
			if side {
				anchor, target, project = row["related_issue_id"], row["issue_id"], row["left_project"]
				key = inverseRelation(kind)
			}
			if anchor != q.IssueID {
				continue
			}
			id := target.(string)
			token := key + id
			if seen[token] {
				continue
			}
			seen[token] = true
			selected[key] = append(selected[key], id)
			out[key] = append(out[key].([]any), map[string]any{"issue_id": id, "project_id": project})
		}
	}
	if q.API {
		return Result{200, out}, nil
	}
	for _, key := range relationTypes {
		values, err := r.eligibleIssues(ctx, q, selected[key], false)
		if err != nil {
			return Result{}, err
		}
		out[key] = []any{}
		for _, row := range values {
			row, err = r.annotateIssue(ctx, row, q.ActorID, false)
			if err != nil {
				return Result{}, err
			}
			item := projectFields(row, strings.Fields("id name state_id sort_order priority sequence_id project_id label_ids assignee_ids created_at updated_at created_by updated_by"))
			item["relation_type"] = key
			out[key] = append(out[key].([]any), item)
		}
	}
	return Result{200, out}, nil
}
func (r *Repository) removeRelation(ctx context.Context, q Request, input djangoinput.Input) (Result, error) {
	related := input.Fields["related_issue"]
	var id any
	if related != nil {
		v, e := djangoinput.UUID(related)
		if e != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
		id = v.String()
	}
	rows, err := r.rows(ctx, `SELECT to_jsonb(rel)||jsonb_build_object('target',to_jsonb(i)) FROM issue_relations rel JOIN issues i ON i.id=rel.related_issue_id JOIN workspaces w ON w.id=rel.workspace_id WHERE w.slug=$1 AND ((rel.issue_id=$2 AND rel.related_issue_id=$3) OR (rel.issue_id=$3 AND rel.related_issue_id=$2)) AND rel.deleted_at IS NULL ORDER BY rel.created_at DESC LIMIT 1`, q.Slug, q.IssueID, id)
	if err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		return Result{}, errors.New("relation missing before instance delete")
	}
	row := rows[0]
	previous, err := orderedDump(relationProjection(row, row["target"].(map[string]any)), strings.Fields("id project_id sequence_id relation_type name state_id priority created_by created_at updated_at updated_by"))
	if err != nil {
		return Result{}, err
	}
	if err = r.softDeleteRow(ctx, "issue_relations", "issuerelation", row["id"].(string), q.ActorID); err != nil {
		return Result{}, err
	}
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	err = r.activity(ctx, q, "issue_relation.activity.deleted", requested, previous, false)
	return Result{Status: 204}, err
}
func (r *Repository) softDeleteRow(ctx context.Context, table, model, id, actor string) error {
	_, err := r.db.Exec(ctx, `UPDATE `+table+` SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1`, id, timestamp(r.now()), actor)
	if err != nil {
		return err
	}
	return r.softDelete(ctx, model, id)
}

func (r *Repository) subissues(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if q.Method == "GET" {
		return r.subissuesTransaction(ctx, q, body, contentType)
	}
	return r.writeProjectTransaction(ctx, q.ProjectID, func(writer *Repository) error {
		if err := writer.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
			return err
		}
		return writer.allowProject(ctx, q, []int{5, 15, 20}, false)
	}, func(writer *Repository) (Result, error) {
		return writer.subissuesTransaction(ctx, q, body, contentType)
	})
}

func (r *Repository) subissuesTransaction(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if err := r.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
		return Result{}, err
	}
	if err := r.allowProject(ctx, q, []int{5, 15, 20}, false); err != nil {
		return Result{}, err
	}
	parent, err := r.issue(ctx, q, q.Method == "POST")
	if err != nil {
		return Result{}, err
	}
	var parentVisible bool
	if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+`)`, q.IssueID, q.ActorID, parent["workspace_id"]).Scan(&parentVisible); err != nil {
		return Result{}, err
	}
	if !parentVisible {
		return Result{}, requiredObject()
	}
	var rows []map[string]any
	if q.Method == "POST" {
		if _, err = r.workspaceAnchor(ctx, q, true); err != nil {
			return Result{}, err
		}
		input, e := parseObject(body, contentType)
		if e != nil {
			return Result{}, e
		}
		raw := input.Fields["sub_issue_ids"]
		if !truthy(raw) {
			return Result{}, failure(400, "error", "Sub Issue IDs are required")
		}
		ids, e := rawIDs(raw)
		if e != nil {
			return Result{}, e
		}
		rows, e = r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE i.id=ANY($1::uuid[]) AND `+liveIssue+` AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+` ORDER BY i.created_at DESC`, ids, q.ActorID, parent["workspace_id"])
		if e != nil {
			return Result{}, e
		}
		requested := map[string]bool{}
		for _, id := range ids {
			requested[id] = true
		}
		if len(rows) != len(requested) {
			return Result{}, failure(400, "sub_issue_ids", "One or more work items are no longer available.")
		}
		eligible := []string{}
		previousParents := make(map[string]any, len(rows))
		for _, row := range rows {
			childRequest := q
			childRequest.ProjectID, childRequest.IssueID = stringValue(row["project_id"]), stringValue(row["id"])
			if err = r.allowProject(ctx, childRequest, []int{15, 20}, false); err != nil {
				return Result{}, err
			}
			eligible = append(eligible, row["id"].(string))
			previousParents[row["id"].(string)] = row["parent_id"]
			row["parent_id"] = q.IssueID
		}
		if _, err = r.db.Exec(ctx, `UPDATE issues SET parent_id=$1 WHERE id=ANY($2::uuid[])`, q.IssueID, eligible); err != nil {
			return Result{}, err
		}
		// Use the same eligible tenant set for tasks and response as for mutation.
		for _, child := range rows {
			id := child["id"].(string)
			local := q
			local.IssueID, local.ProjectID = id, child["project_id"].(string)
			requested, _ := pythoncompat.Dumps(map[string]any{"parent": q.IssueID}, nil)
			previous, _ := pythoncompat.Dumps(map[string]any{"parent": previousParents[id]}, nil)
			if err = r.activity(ctx, local, "issue.activity.updated", requested, previous, false); err != nil {
				return Result{}, err
			}
		}
	} else {
		rows, err = r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id
 WHERE i.parent_id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND EXISTS(
 SELECT 1 FROM project_members m JOIN workspace_members wm ON wm.workspace_id=m.workspace_id AND wm.member_id=m.member_id
 WHERE m.project_id=i.project_id AND m.workspace_id=i.workspace_id AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND (m.role>=15 OR m.role=5 AND (p.guest_view_all_features OR i.created_by_id=$3))) AND `+liveIssue+` ORDER BY `+subissueOrder(q.Params["order_by"]), q.IssueID, q.Slug, q.ActorID)
		if err != nil {
			return Result{}, err
		}
	}
	results := []any{}
	distribution := map[string]any{}
	for _, row := range rows {
		var group *string
		if err = r.db.QueryRow(ctx, `SELECT (SELECT "group" FROM states WHERE id=$1)`, row["state_id"]).Scan(&group); err != nil {
			return Result{}, err
		}
		key := "null"
		if group != nil {
			key = *group
		}
		items, _ := distribution[key].([]any)
		distribution[key] = append(items, row["id"])
		if q.Method == "GET" {
			row, err = r.annotateIssue(ctx, row, q.ActorID, true)
			if err != nil {
				return Result{}, err
			}
		}
		item := projectFields(row, compactFields)
		if q.Method == "GET" {
			item["state_group"] = group
		}
		results = append(results, item)
	}
	var sub any = results
	if group := q.Params["group_by"]; q.Method == "GET" && group != "" {
		sub, err = groupSubissues(results, group)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{200, map[string]any{"sub_issues": sub, "state_distribution": distribution}}, nil
}

// SubIssuesEndpoint evaluates order_issue_queryset directly, without a
// paginator applying its returned ordering key. Priority therefore uses the
// source's ascending rank for both spellings, and PostgreSQL controls nulls.
func subissueOrder(value string) string {
	order := issuequery.LegacyOrder(value, true)
	switch value {
	case "priority", "-priority":
		order = issuequery.LegacyOrder("-priority", true)
	case "-state__group":
		order = strings.TrimSuffix(order, " DESC") + " ASC"
	case "issue_module__module__name", "-issue_module__module__name":
		order = "(SELECT min(m.name) FROM module_issues mi JOIN modules m ON m.id=mi.module_id WHERE mi.issue_id=issue.id) ASC"
		if strings.HasPrefix(value, "-") {
			order = strings.TrimSuffix(order, " ASC") + " DESC"
		}
	}
	order = strings.ReplaceAll(strings.ReplaceAll(order, "issue.", "i."), " NULLS LAST", "")
	if order != "i.created_at ASC" && order != "i.created_at DESC" {
		order += ", i.created_at DESC"
	}
	return order
}

func groupSubissues(results []any, group string) (map[string]any, error) {
	grouped := map[string]any{}
	for _, item := range results {
		row := item.(map[string]any)
		var values []any
		if group == "assignees__ids" {
			values, _ = row["assignee_ids"].([]any)
			if len(values) == 0 {
				values = []any{nil}
			}
		} else {
			v, ok := row[group]
			if !ok {
				return nil, failure(400, "error", "The required key does not exist.")
			}
			values = []any{v}
		}
		for _, v := range values {
			key := djangoinput.ScalarString(v)
			items, _ := grouped[key].([]any)
			grouped[key] = append(items, row)
		}
	}
	return grouped, nil
}
