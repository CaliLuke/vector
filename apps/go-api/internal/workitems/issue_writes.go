package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const apiDefaultIssueTypeSQL = `SELECT (SELECT t.id::text FROM issue_types t JOIN project_issue_types p ON p.issue_type_id=t.id WHERE p.project_id=$1 AND NOT EXISTS(SELECT 1 FROM work_item_type_settings s WHERE s.workspace_id=t.workspace_id AND s.enabled) AND t.is_default AND t.deleted_at IS NULL ORDER BY t.id ASC LIMIT 1)`

func (r *Repository) writeIssue(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if !q.API && q.Method == "PUT" {
		args := []any{q.ActorID, q.Slug, q.ProjectID, q.IssueID}
		if _, err := issuequery.BuildComplexFilter(q.Params["filters"], &args); err != nil {
			return Result{}, err
		}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := *r
	writer.db, writer.transactionalEffects = tx, true
	result, err := writer.writeIssueTransaction(ctx, q, body, contentType)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

type issueEdit struct {
	input                   djangoinput.Input
	changes                 issueChanges
	old, project, submitted map[string]any
	skipActivity            any
}

func (r *Repository) writeIssueTransaction(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	project, err := r.lockIssueWriteScope(ctx, q)
	if err != nil {
		return Result{}, err
	}
	if q.Method == "DELETE" {
		return r.destroyIssue(ctx, q)
	}
	edit, err := r.prepareIssueEdit(ctx, q, body, contentType, project)
	if err != nil {
		return Result{}, err
	}
	previous, err := r.previousIssueWriteProjection(ctx, q, edit.old)
	if err != nil {
		return Result{}, err
	}
	row, err := r.persistIssueEdit(ctx, q, &edit)
	if err != nil {
		return Result{}, err
	}
	return r.finishIssueWrite(ctx, q, edit.input, edit.old, row, previous, edit.skipActivity)
}

func (r *Repository) lockIssueWriteScope(ctx context.Context, q Request) (map[string]any, error) {
	if !q.API && q.Method != "PUT" {
		roles, creator := []int{15, 20}, q.Method != "POST"
		if q.Method == "DELETE" {
			roles = []int{20}
		}
		if err := r.allowProject(ctx, q, roles, creator); err != nil {
			return nil, err
		}
	}
	project, err := r.row(ctx, `SELECT to_jsonb(p) FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, q.ProjectID, q.Slug)
	if err != nil {
		return nil, err
	}
	if _, err = r.db.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, project["workspace_id"]); err != nil {
		return nil, err
	}
	if err = r.AuthorizeInitial(ctx, q); err != nil {
		return nil, err
	}
	if err = r.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
		return nil, err
	}
	if !q.API && q.Method != "PUT" {
		roles := []int{15, 20}
		if q.Method == "DELETE" {
			roles = []int{20}
		}
		if err = r.allowProject(ctx, q, roles, q.Method != "POST"); err != nil {
			return nil, err
		}
	}
	if q.Method != "POST" {
		// Lock before every old-value read. A partial update merges only its
		// submitted fields into the latest committed row, never a stale snapshot.
		if _, err = r.db.Exec(ctx, `SELECT id FROM issues WHERE id=$1 AND project_id=$2 AND workspace_id=$3 FOR UPDATE`, q.IssueID, q.ProjectID, project["workspace_id"]); err != nil {
			return nil, err
		}
	}
	return project, nil
}

func (r *Repository) prepareIssueEdit(ctx context.Context, q Request, body io.Reader, contentType string, project map[string]any) (issueEdit, error) {
	var err error
	// API PATCH performs lookup before touching the request data.
	var old map[string]any
	if q.API && q.Method == "PATCH" {
		old, err = r.issue(ctx, q, false)
		if err != nil {
			return issueEdit{}, err
		}
	}
	if !q.API && q.Method == "PUT" {
		// DRF's inherited update resolves the filtered object before parsing
		// serializer input. The custom PATCH action does not apply this filter.
		old, err = r.putIssue(ctx, q)
		if err != nil {
			return issueEdit{}, err
		}
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return issueEdit{}, err
	}
	submitted, err := submittedCustomValues(input)
	if err != nil {
		return issueEdit{}, err
	}
	if !q.API && q.Method == "PATCH" && (input.Form || input.Kind != "") {
		return issueEdit{}, errors.New("Django PATCH requires mutable request.data")
	}
	skip := input.Fields["skip_activity"]
	if !q.API && q.Method == "PATCH" {
		delete(input.Fields, "skip_activity")
	}
	if q.Method != "POST" && old == nil {
		old, err = r.issue(ctx, q, true)
		if err != nil {
			var e *djangoapi.HTTPError
			if q.Method == "PATCH" && errors.As(err, &e) && e.Status == 404 {
				return issueEdit{}, failure(404, "error", "Issue not found")
			}
			return issueEdit{}, err
		}
	}
	options := issueValidation{ActorID: q.ActorID, Timezone: q.Timezone, API: q.API, Partial: q.Method == "PATCH", ProjectID: q.ProjectID, WorkspaceID: project["workspace_id"].(string)}
	if q.Method == "PUT" {
		options.ProjectID = ""
	}
	validationInput := input
	validationInput.Fields = maps.Clone(input.Fields)
	for _, key := range []string{"assignee_ids", "label_ids", "assignees", "labels"} {
		if value, exists := input.ListValue(key, options.Partial); exists {
			validationInput.Fields[key] = value
		}
	}
	changes, err := r.validateIssue(ctx, validationInput, options)
	if err != nil {
		return issueEdit{}, err
	}
	if err = r.validateTypeTenant(ctx, changes.values, input.Fields, project["workspace_id"]); err != nil {
		return issueEdit{}, err
	}
	if err := r.validateIssueExternalIdentity(ctx, q, input, old); err != nil {
		return issueEdit{}, err
	}
	return issueEdit{input: input, changes: changes, old: old, project: project, submitted: submitted, skipActivity: skip}, nil
}

func (r *Repository) validateIssueExternalIdentity(ctx context.Context, q Request, input djangoinput.Input, old map[string]any) error {
	if q.API && truthy(input.Fields["external_id"]) && (q.Method == "POST" && truthy(input.Fields["external_source"]) || q.Method == "PATCH" && old["external_id"] != djangoinput.ScalarString(input.Fields["external_id"])) {
		source := input.Fields["external_source"]
		if _, present := input.Fields["external_source"]; !present && old != nil {
			source = old["external_source"]
		}
		var id *string
		err := r.db.QueryRow(ctx, `SELECT (SELECT i.id::text FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.project_id=$1 AND w.slug=$2 AND i.deleted_at IS NULL AND i.external_id=$3 AND i.external_source=$4 ORDER BY i.created_at DESC LIMIT 1)`, q.ProjectID, q.Slug, djangoinput.ScalarString(input.Fields["external_id"]), source).Scan(&id)
		if err != nil {
			return err
		}
		if id != nil {
			conflict := *id
			if old != nil {
				conflict = q.IssueID
			}
			return &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": "Issue with the same external id and external source already exists", "id": conflict}}
		}
	}
	return nil
}

func (r *Repository) previousIssueWriteProjection(ctx context.Context, q Request, old map[string]any) (any, error) {
	var err error
	var previous any
	if old != nil && q.Method != "PUT" {
		var projection map[string]any
		keys := detailFields
		if q.API {
			projection, err = r.serializeAuthorizedIssue(ctx, old, "", "")
			keys = apiIssueOrder
		} else {
			detailQ := q
			detailQ.Params = map[string]string{}
			projection, err = r.detail(ctx, maps.Clone(old), detailQ, false, false)
		}
		if err != nil {
			return nil, err
		}
		previous, err = orderedDump(timestampProjection(projection, q.Timezone).(map[string]any), keys)
		if err != nil {
			return nil, err
		}
	}
	return previous, nil
}

func (r *Repository) persistIssueEdit(ctx context.Context, q Request, edit *issueEdit) (map[string]any, error) {
	var err error
	row := maps.Clone(edit.old)
	if edit.old == nil {
		row = defaultIssue(q.ProjectID)
	}
	maps.Copy(row, edit.changes.values)
	if edit.old == nil && q.API && row["type_id"] == nil {
		var defaultType *string
		err = r.db.QueryRow(ctx, apiDefaultIssueTypeSQL, q.ProjectID).Scan(&defaultType)
		if err != nil {
			return nil, err
		}
		if defaultType != nil {
			row["type_id"] = *defaultType
		}
	}
	row["workspace_id"] = edit.project["workspace_id"]
	creator := q.ActorID
	if q.API && edit.old == nil {
		if value, exists := edit.input.Fields["created_by"]; exists {
			creator = stringValue(value)
		}
	}
	subscribers, err := r.validateCustomValues(ctx, creator, row, edit.old, edit.submitted, false, edit.old == nil, hasSubmittedType(edit.input.Fields))
	if err != nil {
		return nil, err
	}
	edit.input = r.composedItemEditInput(edit.input, row)
	if edit.old != nil {
		row, err = r.updateIssueWithRelations(ctx, q.ActorID, row, edit.old, edit.changes)
	} else {
		row, err = r.saveIssue(ctx, q.ActorID, row, edit.old)
	}
	if err != nil {
		return nil, err
	}
	if err = r.subscribePropertyMembers(ctx, q.ActorID, row, subscribers); err != nil {
		return nil, err
	}
	if edit.old == nil {
		if len(edit.changes.assignees) == 0 && !edit.changes.assigneesSet && edit.project["default_assignee_id"] != nil {
			ids, e := r.filteredIDs(ctx, `SELECT member_id::text FROM project_members WHERE project_id=$1 AND member_id=$2 AND role>=15 AND is_active AND deleted_at IS NULL`, q.ProjectID, edit.project["default_assignee_id"])
			if e != nil {
				return nil, e
			}
			edit.changes.assignees = ids
		}
		if err = r.replaceIssueRelations(ctx, row, edit.changes, true); err != nil {
			return nil, err
		}
		if q.API {
			row, err = r.issue(ctx, Request{IssueID: row["id"].(string), ProjectID: q.ProjectID, Slug: q.Slug}, false)
			if err != nil {
				return nil, errors.New("created API issue cannot be refetched")
			}
		}
	}
	return row, nil
}

func (r *Repository) finishIssueWrite(ctx context.Context, q Request, input djangoinput.Input, old, row map[string]any, previous, skip any) (Result, error) {
	var err error
	q.IssueID = row["id"].(string)
	if old == nil && !q.API {
		return r.createdAppIssueResponse(ctx, q, input)
	}
	if q.API && old == nil {
		if err = r.saveAPIIssueCreationAudit(ctx, q, input); err != nil {
			return Result{}, err
		}
	}
	if q.Method != "PUT" && !(q.Method == "PATCH" && !q.API && truthy(skip) && input.Fields["description_html"] != nil) {
		if err = r.issueWriteEffects(ctx, q, input, previous, old == nil); err != nil {
			return Result{}, err
		}
	}
	if q.API {
		projection, err := r.serializeVisibleIssue(ctx, row, q, "", "")
		status := 200
		if old == nil {
			status = 201
		}
		return Result{status, projection}, err
	}
	if q.Method == "PATCH" {
		return Result{Status: 204}, nil
	}
	if q.Method == "PUT" {
		for _, relation := range []string{"labels", "assignees"} {
			value, e := r.appIssueRelationIDs(ctx, row["id"], relation)
			if e != nil {
				return Result{}, e
			}
			row[relation] = value
		}
		body, projectionErr := r.visibleAppWriteProjection(ctx, q, row, input)
		return Result{200, body}, projectionErr
	}
	annotated, err := r.annotateIssue(ctx, row, q.ActorID, false)
	if err != nil {
		return Result{}, err
	}
	return Result{201, projectFields(annotated, append(append([]string{}, compactFields...), "deleted_at"))}, nil
}

func (r *Repository) createdAppIssueResponse(ctx context.Context, q Request, input djangoinput.Input) (Result, error) {
	requested, e := r.issueWriteActivity(ctx, q, input, nil, true)
	if e != nil {
		return Result{}, e
	}
	// The source queues activity before resolving its live response queryset.
	// Deleted or draft creates therefore persist and queue activity, then fail.
	live, e := r.issue(ctx, q, true)
	if e != nil {
		return Result{}, errors.New("created issue missing from live response queryset")
	}
	annotated, e := r.annotateIssue(ctx, live, q.ActorID, false)
	if e != nil {
		return Result{}, e
	}
	if e = r.issueWriteFollowups(ctx, q, input, nil, requested, true); e != nil {
		return Result{}, e
	}
	return Result{201, projectFields(annotated, append(append([]string{}, compactFields...), "deleted_at"))}, nil
}

func (r *Repository) saveAPIIssueCreationAudit(ctx context.Context, q Request, input djangoinput.Input) error {
	createdAt := input.Fields["created_at"]
	if createdAt == nil {
		createdAt = timestamp(r.now())
	}
	createdBy, exists := input.Fields["created_by"]
	if !exists {
		createdBy = q.ActorID
	}
	if _, err := r.db.Exec(ctx, `UPDATE issues SET created_at=$2,created_by_id=$3 WHERE id=$1`, q.IssueID, createdAt, createdBy); err != nil {
		return err
	}
	return nil
}

// ProtocolPutAction mirrors DRF SimpleMetadata.determine_actions: PUT metadata
// resolves the filtered object, suppressing handled lookup/validation failures.
// Unexpected failures still reach the ordinary exception boundary.
func (r *Repository) ProtocolPutAction(ctx context.Context, q Request) (bool, error) {
	_, err := r.putIssue(ctx, q)
	var missing *djangoapi.HTTPError
	var filter *issuequery.FilterError
	if errors.As(err, &missing) || errors.As(err, &filter) {
		return false, nil
	}
	return err == nil, err
}

func (r *Repository) putIssue(ctx context.Context, q Request) (map[string]any, error) {
	args := []any{q.ActorID, q.Slug, q.ProjectID, q.IssueID}
	filter, err := issuequery.BuildComplexFilter(q.Params["filters"], &args)
	if err != nil {
		return nil, err
	}
	where := `workspace.slug=$2 AND issue.project_id=$3 AND issue.id=$4 AND ` + strings.ReplaceAll(liveIssue, "i.", "issue.")
	if filter != "" {
		where += " AND (" + filter + ")"
	}
	// $1 supplies the request timezone to date filter expressions. Keep it
	// typed even when a filter does not use that expression.
	row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("issue")+` FROM issues issue JOIN workspaces workspace ON workspace.id=issue.workspace_id WHERE $1::uuid IS NOT NULL AND `+where, args...)
	var missing *djangoapi.HTTPError
	if errors.As(err, &missing) && missing.Status == 404 {
		return nil, failure(404, "detail", "No Issue matches the given query.")
	}
	return row, err
}

func (r *Repository) validateTypeTenant(ctx context.Context, values, input map[string]any, workspace any) error {
	id := values["type_id"]
	if id == nil {
		return nil
	}
	var foreign bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_types WHERE id=$1 AND workspace_id<>$2)`, id, workspace).Scan(&foreign); err != nil {
		return err
	}
	if foreign {
		field := "type_id"
		if _, ok := input["type"]; ok {
			field = "type"
		}
		return invalid(map[string]any{field: []string{"Invalid type."}})
	}
	return nil
}

func (r *Repository) replaceIssueRelations(ctx context.Context, row map[string]any, changes issueChanges, create bool) error {
	for _, relation := range []struct {
		table, column string
		ids           []string
		set           bool
	}{{"issue_assignees", "assignee_id", changes.assignees, changes.assigneesSet}, {"issue_labels", "label_id", changes.labels, changes.labelsSet}} {
		if !create && !relation.set {
			continue
		}
		if !create {
			if _, err := r.db.Exec(ctx, `UPDATE `+relation.table+` SET deleted_at=$2 WHERE issue_id=$1 AND deleted_at IS NULL`, row["id"], timestamp(r.now())); err != nil {
				return err
			}
		}
		if len(relation.ids) == 0 {
			continue
		}
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return err
		}
		conflicts := ""
		if !create {
			conflicts = " ON CONFLICT DO NOTHING"
		}
		for _, id := range relation.ids {
			_, err = tx.Exec(ctx, `INSERT INTO `+relation.table+`(id,issue_id,`+relation.column+`,project_id,workspace_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`+conflicts, uuid.NewString(), row["id"], id, row["project_id"], row["workspace_id"], row["created_by_id"], row["updated_by_id"], timestamp(r.now()))
			if err != nil {
				break
			}
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				continue
			}
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) issueWriteEffects(ctx context.Context, q Request, input djangoinput.Input, previous any, create bool) error {
	requested, err := r.issueWriteActivity(ctx, q, input, previous, create)
	if err != nil {
		return err
	}
	return r.issueWriteFollowups(ctx, q, input, previous, requested, create)
}

func (r *Repository) issueWriteActivity(ctx context.Context, q Request, input djangoinput.Input, previous any, create bool) (string, error) {
	requested, err := dumpRequest(input)
	if err != nil {
		return "", err
	}
	verb := "updated"
	if create {
		verb = "created"
	}
	activity := map[string]any{"type": "issue.activity." + verb, "requested_data": requested, "actor_id": q.ActorID, "issue_id": q.IssueID, "project_id": q.ProjectID, "current_instance": previous, "epoch": r.now().Unix(), "notification": true, "origin": r.origin}
	if err = r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, activity); err != nil {
		return "", err
	}
	return requested, nil
}

func dumpRequest(input djangoinput.Input) (string, error) {
	if input.Form {
		return orderedDump(input.Fields, input.FormKeys)
	}
	return pythoncompat.Dumps(input.Fields, input.Raw)
}

func (r *Repository) issueWriteFollowups(ctx context.Context, q Request, input djangoinput.Input, previous any, requested string, create bool) error {
	if err := r.publishTask(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "issue", "model_id": q.IssueID, "requested_data": input.Fields, "current_instance": previous, "actor_id": q.ActorID, "slug": q.Slug, "origin": r.origin}); err != nil {
		return err
	}
	if !q.API {
		updated := previous
		if create {
			updated = requested
		}
		return r.admitIssueDescriptionVersion(ctx, updated, q.IssueID, q.ActorID, create)
	}
	return nil
}

func (r *Repository) destroyIssue(ctx context.Context, q Request) (Result, error) {
	row, err := r.issue(ctx, q, false)
	if err != nil {
		return Result{}, err
	}
	previous := any(map[string]any{})
	if q.API {
		var admin bool
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.project_id=$2 AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL AND m.role=20)`, q.Slug, q.ProjectID, q.ActorID).Scan(&admin)
		if err != nil {
			return Result{}, err
		}
		if !admin && row["created_by_id"] != q.ActorID {
			return Result{}, failure(403, "error", "Only admin or creator can delete the work item")
		}
		projection, err := r.serializeVisibleIssue(ctx, row, q, "", "")
		if err != nil {
			return Result{}, err
		}
		previous, err = orderedDump(timestampProjection(projection, q.Timezone).(map[string]any), apiIssueOrder)
		if err != nil {
			return Result{}, err
		}
	}
	if err = r.deleteIssue(ctx, q.ActorID, row); err != nil {
		return Result{}, err
	}
	if !q.API {
		if _, err = r.db.Exec(ctx, `DELETE FROM user_recent_visits v USING workspaces w WHERE v.workspace_id=w.id AND w.slug=$1 AND v.project_id=$2 AND v.entity_identifier=$3 AND v.entity_name='issue'`, q.Slug, q.ProjectID, q.IssueID); err != nil {
			return Result{}, err
		}
	}
	requested, _ := pythoncompat.Dumps(map[string]any{"issue_id": q.IssueID}, nil)
	kwargs := map[string]any{"type": "issue.activity.deleted", "requested_data": requested, "actor_id": q.ActorID, "issue_id": q.IssueID, "project_id": q.ProjectID, "current_instance": previous, "epoch": r.now().Unix()}
	if !q.API {
		kwargs["notification"], kwargs["origin"], kwargs["subscriber"] = true, r.origin, false
	}
	if err = r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, kwargs); err != nil {
		return Result{}, err
	}
	return Result{Status: 204}, nil
}

func (r *Repository) visibleAppWriteProjection(ctx context.Context, q Request, row map[string]any, input djangoinput.Input) (map[string]any, error) {
	row, err := djangoapi.MaskIssueParent(ctx, r.db, row, q.ActorID)
	if err != nil {
		return nil, err
	}
	return appWriteProjection(row, input), nil
}

func appWriteProjection(row map[string]any, input djangoinput.Input) map[string]any {
	result := maps.Clone(row)
	for _, field := range []string{"created_by", "updated_by", "workspace", "project", "state", "parent", "estimate_point", "type"} {
		result[field] = row[field+"_id"]
		if field != "state" && field != "parent" && field != "workspace" && field != "project" {
			delete(result, field+"_id")
		}
	}
	for _, field := range []string{"assignee_ids", "label_ids"} {
		result[field] = input.Fields[field]
		if !truthy(result[field]) {
			result[field] = []any{}
		}
	}
	if number, ok := result["sort_order"].(json.Number); ok {
		f, _ := number.Float64()
		result["sort_order"] = json.Number(pythoncompat.FloatString(f))
	}
	return result
}
