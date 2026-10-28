package workitems

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

func (r *Repository) planning(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	// Preserve the unsupported inherited-update contract before database access.
	if !q.API && (q.Method == "PATCH" || q.Method == "PUT") {
		return Result{}, errors.New("planning update expected pk route keyword")
	}
	return r.projectRequest(ctx, q, body, contentType, (*Repository).authorizePlanning, (*Repository).planningTransaction)
}

func (r *Repository) authorizePlanning(ctx context.Context, q Request) error {
	if err := r.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
		return err
	}
	if !q.API {
		return r.allowProject(ctx, q, []int{15, 20}, false)
	}
	return nil
}

func (r *Repository) planningTransaction(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if strings.HasSuffix(q.Operation, "_transfer_issues") {
		return r.transferCycle(ctx, q, body, contentType)
	}
	kind := "module"
	if q.Params["cycle_id"] != "" {
		kind = "cycle"
	}
	if q.Method == "DELETE" {
		return r.removePlanning(ctx, q, kind, false)
	}
	input, err := parseObject(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if strings.HasSuffix(q.Operation, "_issues_by_issue_id_modules") {
		return r.issueModules(ctx, q, input)
	}
	raw := input.Fields["issues"]
	if !truthy(raw) {
		if q.API && kind == "cycle" {
			return Result{}, invalid(map[string]any{"error": "Work items are required", "code": "MISSING_WORK_ITEMS"})
		}
		return Result{}, failure(400, "error", "Issues are required")
	}
	ids, err := rawIDs(raw)
	if err != nil {
		return Result{}, err
	}
	container, err := r.planningContainer(ctx, q, kind)
	if err != nil {
		return Result{}, err
	}
	targets, err := r.eligibleIssues(ctx, q, ids, true)
	if err != nil {
		return Result{}, err
	}
	if q.API && kind == "module" {
		targets, err = r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=ANY($1::uuid[]) AND w.slug=$2 AND i.project_id=$3 AND i.deleted_at IS NULL ORDER BY i.created_at DESC`, ids, q.Slug, q.ProjectID)
		if err != nil {
			return Result{}, err
		}
	}
	existing := []map[string]any{}
	updated := []any{}
	if kind == "cycle" {
		// Both the relation and the underlying Issue must belong to this project.
		existing, err = r.rows(ctx, `SELECT to_jsonb(link) FROM cycle_issues link JOIN cycles c ON c.id=link.cycle_id AND c.project_id=link.project_id AND c.workspace_id=link.workspace_id AND c.deleted_at IS NULL JOIN issues i ON i.id=link.issue_id JOIN workspaces w ON w.id=link.workspace_id WHERE link.issue_id=ANY($1::uuid[]) AND link.cycle_id<>$2 AND link.project_id=$3 AND w.slug=$4 AND i.project_id=$3 AND i.workspace_id=link.workspace_id AND link.deleted_at IS NULL ORDER BY link.created_at DESC`, ids, q.Params["cycle_id"], q.ProjectID, q.Slug)
		if err != nil {
			return Result{}, err
		}
		for _, link := range existing {
			updated = append(updated, map[string]any{"old_cycle_id": link["cycle_id"], "new_cycle_id": q.Params["cycle_id"], "issue_id": link["issue_id"]})
		}
	}
	selected := unassignedPlanningIssues(targets, existing)
	audit := any(q.ActorID)
	if q.API && kind == "cycle" {
		audit = nil
	}
	created, err := r.insertPlanning(ctx, kind, q, container["workspace_id"], selected, q.Params[kind+"_id"], audit, q.API || kind == "module")
	if err != nil {
		return Result{}, err
	}
	if len(existing) > 0 {
		links := []string{}
		for _, link := range existing {
			links = append(links, link["id"].(string))
		}
		if _, err = r.db.Exec(ctx, `UPDATE cycle_issues SET cycle_id=$1 WHERE id=ANY($2::uuid[])`, q.Params["cycle_id"], links); err != nil {
			return Result{}, err
		}
	}
	if err = r.publishPlanningCreation(ctx, q, kind, input, selected, created, updated); err != nil {
		return Result{}, err
	}
	if !q.API {
		return Result{201, map[string]any{"message": "success"}}, nil
	}
	rows, err := r.rows(ctx, `SELECT to_jsonb(link)||jsonb_build_object('sub_issues_count',(SELECT COUNT(*) FROM issues child WHERE child.parent_id=link.issue_id AND `+djangoapi.IssueVisibleSQL("child", "$4::uuid", "link.workspace_id")+` AND child.deleted_at IS NULL AND child.archived_at IS NULL AND NOT child.is_draft AND NOT EXISTS(SELECT 1 FROM states s WHERE s.id=child.state_id AND s."group"='triage'))) FROM `+kind+`_issues link JOIN projects p ON p.id=link.project_id JOIN workspaces w ON w.id=link.workspace_id WHERE link.`+kind+`_id=$1 AND link.project_id=$2 AND w.slug=$3 AND p.archived_at IS NULL AND link.deleted_at IS NULL ORDER BY link.created_at DESC`, q.Params[kind+"_id"], q.ProjectID, q.Slug, q.ActorID)
	if err != nil {
		return Result{}, err
	}
	results := []any{}
	for _, row := range rows {
		results = append(results, planningProjection(row, kind))
	}
	return Result{200, results}, nil
}
func (r *Repository) planningContainer(ctx context.Context, q Request, kind string) (map[string]any, error) {
	var err error
	var container map[string]any
	if kind == "cycle" || q.API {
		container, err = r.row(ctx, `SELECT to_jsonb(p) FROM `+kind+`s p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND p.project_id=$2 AND w.slug=$3 AND p.deleted_at IS NULL`, q.Params[kind+"_id"], q.ProjectID, q.Slug)
	} else {
		container, err = r.project(ctx, q.ProjectID)
	}
	if err != nil {
		return nil, err
	}
	if kind == "cycle" && beforeNow(container["end_date"], r.now()) {
		if q.API {
			return nil, invalid(map[string]any{"code": "CYCLE_COMPLETED", "message": "The Cycle has already been completed so no new issues can be added"})
		}
		return nil, failure(400, "error", "The Cycle has already been completed so no new issues can be added")
	}
	return container, nil
}

func unassignedPlanningIssues(targets, existing []map[string]any) []string {
	assigned := make(map[any]struct{}, len(existing))
	for _, link := range existing {
		assigned[link["issue_id"]] = struct{}{}
	}
	selected := []string{}
	for _, row := range targets {
		_, found := assigned[row["id"]]
		if !found {
			selected = append(selected, row["id"].(string))
		}
	}
	return selected
}

func (r *Repository) publishPlanningCreation(ctx context.Context, q Request, kind string, input djangoinput.Input, selected []string, created []map[string]any, updated []any) error {
	var err error
	raw := input.Fields["issues"]
	if !q.API && kind == "module" {
		requested, _ := pythoncompat.Dumps(map[string]any{"module_id": q.Params["module_id"]}, nil)
		for _, id := range selected {
			local := q
			local.IssueID = id
			if err = r.activity(ctx, local, "module.activity.created", requested, nil, false); err != nil {
				return err
			}
		}
	} else {
		serialized, err := serializePlanningRecords(kind, created)
		if err != nil {
			return err
		}
		previous, err := planningCurrent(kind, updated, serialized)
		if err != nil {
			return err
		}
		var requested string
		if kind == "cycle" {
			requested, err = pythoncompat.Dumps(map[string]any{"cycles_list": raw}, input.Raw)
		} else {
			names := []string{}
			for i, id := range selected {
				if i >= 20 {
					names = append(names, "'...(remaining elements truncated)...'")
					break
				}
				names = append(names, "UUID('"+id+"')")
			}
			requested, err = pythoncompat.Dumps(map[string]any{"modules_list": "<SoftDeletionQuerySet [" + strings.Join(names, ", ") + "]>"}, nil)
		}
		if err != nil {
			return err
		}
		args := map[string]any{"type": kind + ".activity.created", "requested_data": requested, "actor_id": q.ActorID, "issue_id": nil, "project_id": q.ProjectID, "current_instance": previous, "epoch": r.now().Unix(), "origin": r.origin}
		if kind == "cycle" {
			args["notification"] = true
		}
		if err = r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, args); err != nil {
			return err
		}
	}
	return nil
}

func beforeNow(raw any, now time.Time) bool {
	s, ok := raw.(string)
	if !ok {
		return false
	}
	value, e := time.Parse(time.RFC3339Nano, s)
	return e == nil && value.Before(now)
}
func (r *Repository) insertPlanning(ctx context.Context, kind string, q Request, workspace any, issues []string, target string, audit any, ignore bool) ([]map[string]any, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows := []map[string]any{}
	conflict := ""
	if ignore {
		conflict = " ON CONFLICT DO NOTHING"
	}
	for _, id := range issues {
		now := timestamp(r.now())
		row := map[string]any{"id": uuid.NewString(), "issue_id": id, kind + "_id": target, "project_id": q.ProjectID, "workspace_id": workspace, "created_at": now, "updated_at": now, "created_by_id": audit, "updated_by_id": audit, "deleted_at": nil}
		_, err = tx.Exec(ctx, `INSERT INTO `+kind+`_issues(id,issue_id,`+kind+`_id,project_id,workspace_id,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$7)`+conflict, row["id"], id, target, q.ProjectID, workspace, now, audit)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return rows, nil
}
func planningProjection(row map[string]any, kind string) map[string]any {
	result := map[string]any{}
	for _, key := range []string{"id", "created_at", "updated_at", "deleted_at", "sub_issues_count"} {
		if value, ok := row[key]; ok {
			result[key] = value
		}
	}
	for _, key := range []string{"created_by", "updated_by", "workspace", "project", "issue", kind} {
		result[key] = row[key+"_id"]
	}
	return result
}
func serializePlanningRecords(kind string, rows []map[string]any) (string, error) {
	results := []string{}
	order := []string{"created_at", "updated_at", "created_by", "updated_by", "deleted_at", "project", "workspace"}
	if kind == "cycle" {
		order = append(order, "issue", "cycle")
	} else {
		order = append(order, "module", "issue")
	}
	for _, row := range rows {
		fields := planningProjection(row, kind)
		delete(fields, "id")
		delete(fields, "sub_issues_count")
		for _, key := range []string{"created_at", "updated_at"} {
			if raw, ok := fields[key].(string); ok {
				t, _ := time.Parse(time.RFC3339Nano, raw)
				fields[key] = t.UTC().Format("2006-01-02T15:04:05.000Z")
			}
		}
		data, err := orderedDump(fields, order)
		if err != nil {
			return "", err
		}
		model, _ := pythoncompat.Dumps("db."+kind+"issue", nil)
		id, _ := pythoncompat.Dumps(row["id"], nil)
		results = append(results, `{"model": `+model+`, "pk": `+id+`, "fields": `+data+`}`)
	}
	return "[" + strings.Join(results, ", ") + "]", nil
}
func planningCurrent(kind string, updated []any, created any) (string, error) {
	serialized := []string{}
	for _, value := range updated {
		data, err := orderedDump(value.(map[string]any), []string{"old_" + kind + "_id", "new_" + kind + "_id", "issue_id"})
		if err != nil {
			return "", err
		}
		serialized = append(serialized, data)
	}
	data, err := pythoncompat.Dumps(created, nil)
	if err != nil {
		return "", err
	}
	return `{"updated_` + kind + `_issues": [` + strings.Join(serialized, ", ") + `], "created_` + kind + `_issues": ` + data + `}`, nil
}
func (r *Repository) removePlanning(ctx context.Context, q Request, kind string, allowMissing bool) (Result, error) {
	rows, err := r.rows(ctx, `SELECT to_jsonb(link)||jsonb_build_object('container_name',container.name) FROM `+kind+`_issues link JOIN `+kind+`s container ON container.id=link.`+kind+`_id AND container.project_id=link.project_id AND container.workspace_id=link.workspace_id JOIN issues i ON i.id=link.issue_id AND i.project_id=link.project_id AND i.workspace_id=link.workspace_id JOIN workspaces w ON w.id=link.workspace_id WHERE w.slug=$1 AND link.project_id=$2 AND link.issue_id=$3 AND link.`+kind+`_id=$4 AND link.deleted_at IS NULL ORDER BY link.created_at DESC`, q.Slug, q.ProjectID, q.IssueID, q.Params[kind+"_id"])
	if err != nil {
		return Result{}, err
	}
	if len(rows) == 0 {
		if q.API {
			return Result{}, failure(404, "error", "The requested resource does not exist.")
		}
		if kind == "module" && !allowMissing {
			return Result{}, errors.New("module link missing")
		}
	}
	if q.API {
		if err = r.softDeleteRow(ctx, kind+"_issues", kind+"issue", rows[0]["id"].(string), q.ActorID); err != nil {
			return Result{}, err
		}
	}
	requested := map[string]any{kind + "_id": q.Params[kind+"_id"]}
	keys := []string{kind + "_id"}
	var previous any
	if kind == "cycle" || q.API {
		requested["issues"] = []string{q.IssueID}
		keys = append(keys, "issues")
	}
	if kind == "module" {
		name := any(nil)
		if len(rows) > 0 {
			name = rows[0]["container_name"]
		}
		previous, _ = pythoncompat.Dumps(map[string]any{"module_name": name}, nil)
	}
	data, _ := orderedDump(requested, keys)
	if err = r.activity(ctx, q, kind+".activity.deleted", data, previous, q.API); err != nil {
		return Result{}, err
	}
	if !q.API {
		ids := []string{}
		for _, row := range rows {
			ids = append(ids, row["id"].(string))
		}
		_, err = r.db.Exec(ctx, `UPDATE `+kind+`_issues SET deleted_at=$2 WHERE id=ANY($1::uuid[])`, ids, timestamp(r.now()))
	}
	return Result{Status: 204}, err
}
func (r *Repository) issueModules(ctx context.Context, q Request, input djangoinput.Input) (Result, error) {
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	var foreignAnchor bool
	if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1 AND workspace_id<>$2)`, q.IssueID, project["workspace_id"]).Scan(&foreignAnchor); err != nil {
		return Result{}, err
	}
	if foreignAnchor {
		return Result{}, requiredObject()
	}
	// Reject existing foreign references before either creation or removal.
	fieldErrors := map[string]any{}
	for _, field := range []string{"modules", "removed_modules"} {
		// Ownership checks precede writes, but malformed/missing references keep
		// their source execution point and its earlier durable effects.
		ids := []string{}
		if values, ok := input.Fields[field].([]any); ok {
			for _, value := range values {
				if id, e := djangoinput.UUID(value); e == nil {
					ids = append(ids, id.String())
				}
			}
		}
		var foreign bool
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM modules WHERE id=ANY($1::uuid[]) AND workspace_id<>$2)`, ids, project["workspace_id"]).Scan(&foreign)
		if err != nil {
			return Result{}, err
		}
		if foreign {
			fieldErrors[field] = []string{"Invalid module."}
		}
	}
	if len(fieldErrors) > 0 {
		return Result{}, invalid(fieldErrors)
	}
	modules, err := rawIDs(input.Fields["modules"])
	if err != nil {
		return Result{}, err
	}
	// One bulk insert means a late missing FK rolls back this insertion batch.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	for _, id := range modules {
		_, err = tx.Exec(ctx, `INSERT INTO module_issues(id,issue_id,module_id,project_id,workspace_id,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$7) ON CONFLICT DO NOTHING`, uuid.NewString(), q.IssueID, id, q.ProjectID, project["workspace_id"], timestamp(r.now()), q.ActorID)
		if err != nil {
			return Result{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	for _, id := range modules {
		requested, _ := pythoncompat.Dumps(map[string]any{"module_id": id}, nil)
		if err = r.activity(ctx, q, "module.activity.created", requested, nil, false); err != nil {
			return Result{}, err
		}
	}
	removed, ok := input.Fields["removed_modules"].([]any)
	if !ok && input.Fields["removed_modules"] != nil {
		if text, yes := input.Fields["removed_modules"].(string); yes {
			for _, char := range text {
				removed = append(removed, string(char))
			}
		} else {
			return Result{}, errors.New("removed modules are not iterable")
		}
	}
	for _, value := range removed {
		ids, e := rawIDs([]any{value})
		if e != nil {
			return Result{}, e
		}
		id := ids[0]
		local := q
		local.Params = map[string]string{"module_id": id}
		if _, err = r.removePlanning(ctx, local, "module", true); err != nil {
			return Result{}, err
		}
	}
	return Result{201, map[string]any{"message": "success"}}, nil
}
