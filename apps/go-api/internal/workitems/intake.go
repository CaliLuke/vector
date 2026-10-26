package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
)

func (r *Repository) intake(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	return r.projectRequest(ctx, q, body, contentType, (*Repository).authorizeIntake, (*Repository).intakeTransaction)
}

func (r *Repository) intakeTransaction(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	if q.Method == "POST" {
		return r.createIntake(ctx, q, body, contentType)
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	intakeID, err := r.intakeID(ctx, q)
	if err != nil {
		return Result{}, err
	}
	if q.API && q.Method != "GET" && intakeID == nil && !truthy(project["intake_view"]) {
		return Result{}, failure(400, "error", "Intake is not enabled for this project enable it through the project's api")
	}
	if q.Method == "GET" {
		return r.readIntake(ctx, q, project, intakeID)
	}
	var input djangoinput.Input
	if !q.API && q.Method == "PATCH" {
		input, err = djangoinput.Parse(body, contentType)
		if err != nil {
			return Result{}, err
		}
		if input.Form || input.Kind != "" {
			return Result{}, errors.New("immutable intake PATCH body")
		}
	}
	if q.API && intakeID == nil {
		return Result{}, errors.New("missing intake configuration")
	}
	row, err := r.row(ctx, `SELECT to_jsonb(t) FROM intake_issues t JOIN workspaces w ON w.id=t.workspace_id WHERE t.issue_id=$1 AND t.project_id=$2 AND w.slug=$3 AND t.intake_id IS NOT DISTINCT FROM $4::uuid AND t.deleted_at IS NULL`, q.IssueID, q.ProjectID, q.Slug, intakeID)
	if err != nil {
		return Result{}, intakeError(q, err)
	}
	if q.Method == "DELETE" {
		if row["status"] != json.Number("1") {
			issue, err := r.issue(ctx, q, false)
			if err != nil {
				return Result{}, errors.New("missing intake issue")
			}
			if q.API && issue["created_by_id"] != q.ActorID {
				var admin bool
				if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE workspace_id=$1 AND project_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role=20)`, row["workspace_id"], q.ProjectID, q.ActorID).Scan(&admin); err != nil {
					return Result{}, err
				}
				if !admin {
					return Result{}, failure(403, "error", "Only admin or creator can delete the work item")
				}
			}
			if err = r.deleteIssue(ctx, q.ActorID, issue); err != nil {
				return Result{}, err
			}
		}
		return Result{Status: 204}, r.softDeleteRow(ctx, "intake_issues", "intakeissue", row["id"].(string), q.ActorID)
	}
	if q.API {
		input, err = djangoinput.Parse(body, contentType)
		if err != nil {
			return Result{}, err
		}
		if input.Form || input.Kind != "" {
			return Result{}, errors.New("immutable intake PATCH body")
		}
	}
	return r.patchIntake(ctx, q, project, row, input)
}
func intakeError(q Request, err error) error {
	var e *djangoapi.HTTPError
	if q.API && errors.As(err, &e) && e.Status == 404 {
		return failure(404, "error", "The requested resource does not exist.")
	}
	return err
}
func (r *Repository) intakeID(ctx context.Context, q Request) (*string, error) {
	var id *string
	err := r.db.QueryRow(ctx, `SELECT (SELECT t.id::text FROM intakes t JOIN workspaces w ON w.id=t.workspace_id WHERE w.slug=$1 AND t.project_id=$2 AND t.deleted_at IS NULL ORDER BY t.name LIMIT 1)`, q.Slug, q.ProjectID).Scan(&id)
	return id, err
}

func (r *Repository) readIntake(ctx context.Context, q Request, project map[string]any, intakeID *string) (Result, error) {
	where := `t.project_id=$1 AND t.intake_id=$2 AND t.deleted_at IS NULL AND (t.snoozed_till >= $3 OR t.snoozed_till IS NULL)`
	args := []any{q.ProjectID, intakeID, r.now()}
	if intakeID == nil || !truthy(project["intake_view"]) {
		where += " AND false"
	}
	if q.IssueID != "" {
		where += ` AND t.issue_id=$4`
		args = append(args, q.IssueID)
		row, err := r.row(ctx, `SELECT to_jsonb(t) FROM intake_issues t WHERE `+where, args...)
		if err != nil {
			return Result{}, intakeError(q, err)
		}
		result, err := r.intakeProjection(ctx, q, row, true, false)
		return Result{200, result}, err
	}
	limit, page, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return Result{}, err
	}
	var total int
	if err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM intake_issues t WHERE `+where, args...).Scan(&total); err != nil {
		return Result{}, err
	}
	rows, err := r.rows(ctx, `SELECT to_jsonb(t) FROM intake_issues t WHERE `+where+` ORDER BY t.created_at DESC LIMIT $4 OFFSET $5`, append(args, limit, offset)...)
	if err != nil {
		return Result{}, err
	}
	results := []any{}
	for _, row := range rows {
		result, err := r.intakeProjection(ctx, q, row, true, false)
		if err != nil {
			return Result{}, err
		}
		results = append(results, result)
	}
	return Result{200, djangoapi.Envelope(results, total, limit, page)}, nil
}

func (r *Repository) createIntake(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	nested, exists := input.Fields["issue"]
	if !exists {
		nested = map[string]any{}
	}
	data, ok := nested.(map[string]any)
	if !ok {
		return Result{}, errors.New("intake issue data is not an object")
	}
	if !truthy(data["name"]) {
		return Result{}, failure(400, "error", "Name is required")
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	intakeID, err := r.intakeID(ctx, q)
	if err != nil {
		return Result{}, err
	}
	if q.API && intakeID == nil && !truthy(project["intake_view"]) {
		return Result{}, failure(400, "error", "Intake is not enabled for this project enable it through the project's api")
	}
	priority := "none"
	if value, ok := data["priority"]; ok {
		s, ok := value.(string)
		if !ok {
			return Result{}, failure(400, "error", "Invalid priority")
		}
		priority = s
	}
	if !slices.Contains([]string{"low", "medium", "high", "urgent", "none"}, priority) {
		return Result{}, failure(400, "error", "Invalid priority")
	}
	var state *string
	if err = r.db.QueryRow(ctx, `SELECT (SELECT id::text FROM states WHERE project_id=$1 AND workspace_id=$2 AND "group"='triage' AND deleted_at IS NULL ORDER BY sequence LIMIT 1)`, q.ProjectID, project["workspace_id"]).Scan(&state); err != nil {
		return Result{}, err
	}
	if state == nil {
		if !q.API && data["type"] != nil {
			// A foreign effective type must be rejected before creating Triage.
			// Keep ordinary serializer failures in their source position after
			// state creation, and preserve their precedence over tenant errors.
			preflight := maps.Clone(data)
			delete(preflight, "state_id") // The intake view replaces this alias.
			changes, validationErr := r.validateIssue(ctx, djangoinput.Input{Fields: preflight}, issueValidation{ActorID: q.ActorID, Timezone: q.Timezone, AllowTriage: true, ProjectID: q.ProjectID, WorkspaceID: project["workspace_id"].(string)})
			if tenantErr := r.validateTypeTenant(ctx, changes.values, data, project["workspace_id"]); tenantErr != nil {
				if validationErr != nil {
					return Result{}, validationErr
				}
				return Result{}, tenantErr
			}
		}
		id := uuid.NewString()
		_, err = r.db.Exec(ctx, `INSERT INTO states(id,name,description,color,slug,sequence,"group",is_triage,"default",project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,'Triage','','#4E5355','triage',COALESCE((SELECT MAX(sequence)+15000 FROM states WHERE project_id=$2 AND deleted_at IS NULL AND "group"<>'triage'),65000),'triage',false,false,$2,$3,$4,$4,$5)`, id, q.ProjectID, project["workspace_id"], timestamp(r.now()), q.ActorID)
		if err != nil {
			return Result{}, err
		}
		state = &id
	}
	row := defaultIssue(q.ProjectID)
	changes := issueChanges{}
	if q.API {
		row["name"] = djangoinput.ScalarString(data["name"])
		row["priority"] = priority
		row["state_id"] = *state
		description := data["description"]
		if !truthy(description) {
			description = data["description_json"]
		}
		if !truthy(description) {
			description = map[string]any{}
		}
		row["description_json"] = description
		raw := "<p></p>"
		if value, ok := data["description_html"]; ok {
			raw = djangoinput.ScalarString(value)
		}
		clean, _, e := content.ValidateHTML(raw)
		if e != nil {
			clean = "<p></p>"
		}
		row["description_html"] = clean
	} else {
		data["state_id"] = *state
		nestedInput := djangoinput.Input{Fields: data}
		changes, err = r.validateIssue(ctx, nestedInput, issueValidation{ActorID: q.ActorID, Timezone: q.Timezone, AllowTriage: true, ProjectID: q.ProjectID, WorkspaceID: project["workspace_id"].(string)})
		if err != nil {
			return Result{}, err
		}
		if err = r.validateTypeTenant(ctx, changes.values, data, project["workspace_id"]); err != nil {
			return Result{}, err
		}
		maps.Copy(row, changes.values)
	}
	if q.API && hasSubmittedType(data) {
		typeFields := map[string]any{}
		for _, key := range []string{"type_id", "type"} {
			if value, present := data[key]; present {
				typeFields[key] = value
			}
		}
		typeChanges, validationErr := r.validateIssue(ctx, djangoinput.Input{Fields: typeFields}, issueValidation{API: true, Partial: true, ProjectID: q.ProjectID, WorkspaceID: stringValue(project["workspace_id"]), ActorID: q.ActorID})
		if validationErr != nil {
			return Result{}, validationErr
		}
		maps.Copy(row, typeChanges.values)
	}
	row["workspace_id"] = project["workspace_id"]
	submitted, err := submittedCustomValues(djangoinput.Input{Fields: data})
	if err != nil {
		return Result{}, err
	}
	subscribers, err := r.validateCustomValues(ctx, q.ActorID, row, nil, submitted, false, true, hasSubmittedType(data))
	if err != nil {
		return Result{}, err
	}
	row, err = r.saveIssue(ctx, q.ActorID, row, nil)
	if err != nil {
		return Result{}, err
	}
	if err = r.subscribePropertyMembers(ctx, q.ActorID, row, subscribers); err != nil {
		return Result{}, err
	}
	q.IssueID = row["id"].(string)
	if !q.API {
		if len(changes.assignees) == 0 && project["default_assignee_id"] != nil {
			changes.assignees, err = r.filteredIDs(ctx, `SELECT member_id::text FROM project_members WHERE project_id=$1 AND member_id=$2 AND role>=15 AND is_active AND deleted_at IS NULL`, q.ProjectID, project["default_assignee_id"])
			if err != nil {
				return Result{}, err
			}
		}
		if err = r.replaceIssueRelations(ctx, row, changes, true); err != nil {
			return Result{}, err
		}
	}
	if intakeID == nil {
		return Result{}, errors.New("missing intake configuration after issue creation")
	}
	id := uuid.NewString()
	now := timestamp(r.now())
	_, err = r.db.Exec(ctx, `INSERT INTO intake_issues(id,intake_id,issue_id,status,source,extra,project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,$2,$3,-2,'IN_APP','{}',$4,$5,$6,$6,$7)`, id, *intakeID, q.IssueID, q.ProjectID, project["workspace_id"], now, q.ActorID)
	if err != nil {
		return Result{}, err
	}
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	if err = r.intakeActivity(ctx, q, id, "issue.activity.created", requested, nil, true); err != nil {
		return Result{}, err
	}
	if !q.API {
		if err = r.admitIssueDescriptionVersion(ctx, requested, q.IssueID, q.ActorID, true); err != nil {
			return Result{}, err
		}
	}
	intakeRow, err := r.row(ctx, `SELECT to_jsonb(t) FROM intake_issues t WHERE id=$1`, id)
	if err != nil {
		return Result{}, err
	}
	result, err := r.intakeProjection(ctx, q, intakeRow, false, !q.API)
	status := 200
	if q.API {
		status = 201
	}
	return Result{status, result}, err
}

func (r *Repository) intakeActivity(ctx context.Context, q Request, intakeID, kind string, requested, current any, notification bool) error {
	data := map[string]any{"type": kind, "requested_data": requested, "actor_id": q.ActorID, "issue_id": q.IssueID, "project_id": q.ProjectID, "current_instance": current, "epoch": r.now().Unix(), "intake": intakeID}
	if !q.API || strings.HasPrefix(kind, "intake.") {
		data["notification"] = notification
		data["origin"] = r.origin
	}
	return r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, data)
}

func (r *Repository) intakeProjection(ctx context.Context, q Request, row map[string]any, selectFields, detail bool, cached ...map[string]any) (map[string]any, error) {
	var issue map[string]any
	var err error
	if len(cached) > 0 {
		issue = maps.Clone(cached[0])
	} else {
		issue, err = r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE id=$1`, row["issue_id"])
		if err != nil {
			return nil, err
		}
	}
	if !q.API {
		result := map[string]any{"id": row["id"], "status": row["status"], "duplicate_to": row["duplicate_to_id"], "snoozed_till": row["snoozed_till"], "source": row["source"]}
		if detail {
			if err := r.annotateIntakeIssue(ctx, issue); err != nil {
				return nil, err
			}
			result["issue"] = projectFields(issue, detailFields)
			result["duplicate_issue_detail"] = nil
			if row["duplicate_to_id"] != nil {
				duplicate, e := r.row(ctx, `SELECT to_jsonb(i) FROM issues i WHERE id=$1`, row["duplicate_to_id"])
				if e != nil {
					return nil, e
				}
				result["duplicate_issue_detail"] = projectFields(duplicate, strings.Fields("id name priority sequence_id project_id created_at created_by"))
			}
		} else {
			delete(issue, "label_ids")
			result["issue"] = projectFields(issue, strings.Fields("id name priority sequence_id project_id created_at label_ids created_by"))
			result["created_by"] = row["created_by_id"]
		}
		return result, nil
	}
	// IssueExpandSerializer exposes description_json via both the model and its
	// deprecated alias, and omits nonexistent singular cycle/module properties.
	expanded, err := djangoapi.Serialize(ctx, r.db, issue, strings.Fields("id created_at updated_at deleted_at name description_html description_binary description_json description_stripped priority start_date target_date sequence_id sort_order completed_at archived_at is_draft external_source external_id point created_by updated_by workspace project parent state estimate_point type assignees labels"), "", "state", true)
	if err != nil {
		return nil, err
	}
	expanded["description"] = issue["description_json"]
	fields, expand := "", ""
	if selectFields {
		fields, expand = q.Params["fields"], q.Params["expand"]
	}
	prepared := maps.Clone(row)
	prepared["issue_detail"], prepared["inbox"] = expanded, row["intake_id"]
	prepared["duplicate_to"], prepared["intake"] = row["duplicate_to_id"], row["intake_id"]
	return djangoapi.Serialize(ctx, r.db, prepared, strings.Fields("id issue_detail inbox created_at updated_at deleted_at status snoozed_till source source_email external_source external_id extra created_by updated_by project workspace intake issue duplicate_to"), fields, expand, false)
}

func (r *Repository) patchIntake(ctx context.Context, q Request, project, row map[string]any, input djangoinput.Input) (Result, error) {
	cachedIssue, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE id=$1`, row["issue_id"])
	if err != nil {
		return Result{}, err
	}
	var role *int
	var workspaceAdmin bool
	if err := r.db.QueryRow(ctx, `SELECT (SELECT role FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1),EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$2 AND member_id=$3 AND role=20 AND is_active AND deleted_at IS NULL)`, q.ProjectID, row["workspace_id"], q.ActorID).Scan(&role, &workspaceAdmin); err != nil {
		return Result{}, err
	}
	if q.API && role == nil {
		return Result{}, failure(404, "error", "The requested resource does not exist.")
	}
	if role == nil && !workspaceAdmin {
		return Result{}, failure(403, "error", "Only admin or creator can update the intake work items")
	}
	if role != nil && *role <= 5 && (q.API || !workspaceAdmin) && row["created_by_id"] != q.ActorID {
		message := "You cannot edit intake issues"
		if q.API {
			message = "You cannot edit intake work items"
		}
		return Result{}, failure(400, "error", message)
	}
	skip, descriptionUpdate := input.Fields["skip_activity"], input.Fields["description_html"] != nil
	if !q.API {
		delete(input.Fields, "skip_activity")
	}
	nested := input.Fields["issue"]
	delete(input.Fields, "issue")
	var old map[string]any
	var changes issueChanges
	var nestedInput djangoinput.Input
	var previous, requested any
	if truthy(nested) {
		data, ok := nested.(map[string]any)
		if !ok {
			return Result{}, errors.New("nested issue must be an object")
		}
		var err error
		old, err = r.issue(ctx, q, false)
		if err != nil {
			return Result{}, err
		}
		if !q.API {
			if err = r.annotateIntakeIssue(ctx, old); err != nil {
				return Result{}, err
			}
		}
		if role != nil && *role <= 5 {
			get := func(key string) any {
				if value, ok := data[key]; ok {
					return value
				}
				return old[key]
			}
			description := get("description_json")
			if q.API {
				description = data["description"]
				if !truthy(description) {
					description = data["description_json"]
				}
				if !truthy(description) {
					description = map[string]any{}
				}
			}
			data = map[string]any{"name": get("name"), "description_html": get("description_html"), "description_json": description}
		}
		nestedInput = djangoinput.Input{Fields: data}
		options := issueValidation{ActorID: q.ActorID, Timezone: q.Timezone, API: q.API, Partial: true, AllowTriage: !q.API, ProjectID: q.ProjectID, WorkspaceID: project["workspace_id"].(string)}
		if q.API {
			options.ProjectID = ""
		}
		changes, err = r.validateIssue(ctx, nestedInput, options)
		if err != nil {
			return Result{}, err
		}
		if err = r.validateTypeTenant(ctx, changes.values, data, project["workspace_id"]); err != nil {
			return Result{}, err
		}
		projected := projectFields(old, detailFields)
		order := detailFields
		if q.API {
			projected, err = r.serializeAuthorizedIssue(ctx, old, "", "")
			order = apiIssueOrder
		}
		if err != nil {
			return Result{}, err
		}
		previous, err = orderedDump(timestampProjection(projected, q.Timezone).(map[string]any), order)
		if err != nil {
			return Result{}, err
		}
		var rawFields map[string]json.RawMessage
		_ = json.Unmarshal(input.Raw, &rawFields)
		requested, err = pythoncompat.Dumps(data, rawFields["issue"])
		if role != nil && *role <= 5 {
			requested, err = orderedDump(data, []string{"name", "description_html", "description_json"})
		}
		if err != nil {
			return Result{}, err
		}
	}
	canMetadata := role != nil && *role > 15 || !q.API && workspaceAdmin
	var metadata map[string]any
	var intakePrevious any
	previousIntake := func() error {
		projected, err := r.intakeProjection(ctx, q, row, false, false, cachedIssue)
		if err != nil {
			return err
		}
		intakePrevious, err = dumpIntakeSnapshot(timestampProjection(projected, q.Timezone).(map[string]any), q.API)
		return err
	}
	if canMetadata {
		var err error
		metadata, err = r.validateIntake(ctx, q, row, input)
		if err != nil {
			return Result{}, err
		}
		if !q.API {
			if err := previousIntake(); err != nil {
				return Result{}, err
			}
		}
	}
	if old != nil {
		if q.API {
			if err := r.intakeActivity(ctx, q, row["id"].(string), "issue.activity.updated", requested, previous, true); err != nil {
				return Result{}, err
			}
		}
		saved, err := r.saveIntakeIssueEdit(ctx, q.ActorID, old, changes, nestedInput)
		if err != nil {
			return Result{}, err
		}
		// A later acceptance transition must build on this saved edit.
		cachedIssue = saved
		if !q.API && !(truthy(skip) && descriptionUpdate) {
			if err := r.intakeActivity(ctx, q, row["id"].(string), "issue.activity.updated", requested, previous, true); err != nil {
				return Result{}, err
			}
			if err := r.admitIssueDescriptionVersion(ctx, previous, q.IssueID, q.ActorID, false); err != nil {
				return Result{}, err
			}
		}
	}
	if canMetadata {
		if q.API {
			if err := previousIntake(); err != nil {
				return Result{}, err
			}
		}
		maps.Copy(row, metadata)
		row["updated_at"], row["updated_by_id"] = timestamp(r.now()), q.ActorID
		encoded, err := json.Marshal(row)
		if err != nil {
			return Result{}, err
		}
		_, err = r.db.Exec(ctx, `UPDATE intake_issues t SET (status,snoozed_till,duplicate_to_id,source,source_email,created_by_id,updated_by_id,updated_at)=(SELECT x.status,x.snoozed_till,x.duplicate_to_id,x.source,x.source_email,x.created_by_id,x.updated_by_id,x.updated_at FROM jsonb_populate_record(NULL::intake_issues,$1::jsonb) x) WHERE id=($1::jsonb->>'id')::uuid`, encoded)
		if err != nil {
			return Result{}, err
		}
		if metadata["status"] == int64(1) {
			issue := cachedIssue
			var state *string
			if err = r.db.QueryRow(ctx, `SELECT (SELECT s.id::text FROM states s WHERE s.project_id=$1 AND s.workspace_id=$2 AND s."default" AND s."group"<>'triage' AND s.deleted_at IS NULL AND EXISTS(SELECT 1 FROM states current WHERE current.id=$3 AND current."group"='triage') ORDER BY s.sequence LIMIT 1)`, q.ProjectID, row["workspace_id"], issue["state_id"]).Scan(&state); err != nil {
				return Result{}, err
			}
			if state != nil {
				updated := maps.Clone(issue)
				updated["state_id"] = *state
				if _, err = r.saveIssue(ctx, q.ActorID, updated, issue); err != nil {
					return Result{}, err
				}
				cachedIssue = updated
			}
		}
		requested, err := dumpRequest(input)
		if err != nil {
			return Result{}, err
		}
		if err = r.intakeActivity(ctx, q, row["id"].(string), "intake.activity.created", requested, intakePrevious, false); err != nil {
			return Result{}, err
		}
	}
	var result map[string]any
	if q.API {
		result, err = r.intakeProjection(ctx, q, row, false, false, cachedIssue)
	} else {
		result, err = r.intakeProjection(ctx, q, row, false, true)
	}
	return Result{200, result}, err
}

func (r *Repository) annotateIntakeIssue(ctx context.Context, issue map[string]any) error {
	values, err := r.row(ctx, `SELECT jsonb_build_object('label_ids',COALESCE((SELECT jsonb_agg(DISTINCT relation.label_id) FROM issue_labels relation JOIN issues issue ON issue.id=relation.issue_id WHERE issue.id=$1 AND `+issuequery.LiveIssueLabelPredicate("issue", "relation")+`),'[]'::jsonb),'assignee_ids',COALESCE((SELECT jsonb_agg(DISTINCT assignee_id) FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL),'[]'::jsonb))`, issue["id"])
	if err == nil {
		maps.Copy(issue, values)
	}
	return err
}

func dumpIntakeSnapshot(value map[string]any, api bool) (string, error) {
	order := strings.Fields("id status duplicate_to snoozed_till source issue created_by")
	issueOrder := strings.Fields("id name priority sequence_id project_id created_at label_ids created_by")
	nested := "issue"
	if api {
		order = strings.Fields("id issue_detail inbox created_at updated_at deleted_at status snoozed_till source source_email external_source external_id extra created_by updated_by project workspace intake issue duplicate_to")
		issueOrder = strings.Fields("id labels assignees state description created_at updated_at deleted_at point name description_json description_html description_stripped description_binary priority start_date target_date sequence_id sort_order completed_at archived_at is_draft external_source external_id created_by updated_by project workspace parent estimate_point type")
		nested = "issue_detail"
	}
	stateOrder := jsonOrder([]string{"id", "name", "color", "group"}, nil)
	childOrder := jsonOrder(issueOrder, map[string][]byte{"state": stateOrder})
	return pythoncompat.Dumps(value, jsonOrder(order, map[string][]byte{nested: childOrder}))
}

func (r *Repository) validateIntake(ctx context.Context, q Request, row map[string]any, input djangoinput.Input) (map[string]any, error) {
	values, failures := map[string]any{}, map[string]any{}
	if raw, ok := input.Fields["status"]; ok {
		value, valid := map[string]int{"-2": -2, "-1": -1, "0": 0, "1": 1, "2": 2}[djangoinput.ScalarString(raw)]
		if !valid {
			failures["status"] = []string{fmt.Sprintf("%q is not a valid choice.", djangoinput.ScalarString(raw))}
		} else {
			values["status"] = int64(value)
		}
		if raw == nil {
			failures["status"] = []string{"This field may not be null."}
		}
	}
	if raw, ok := input.Fields["snoozed_till"]; ok {
		if raw == nil {
			values["snoozed_till"] = nil
		} else {
			parsed, err := parseQueryDateTime(djangoinput.ScalarString(raw), q.Timezone)
			if err != nil {
				failures["snoozed_till"] = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
			} else {
				values["snoozed_till"] = parsed
			}
		}
	}
	for _, field := range []string{"source", "source_email"} {
		if field == "source_email" && !q.API {
			continue
		}
		if raw, ok := input.Fields[field]; ok {
			if raw == nil {
				values[field] = nil
			} else {
				max := 255
				if field == "source_email" {
					max = 0
				}
				value, messages := djangoinput.TextField(raw, true, max, input.Surrogates[field])
				if len(messages) > 0 {
					failures[field] = messages
				} else {
					values[field] = value
				}
			}
		}
	}
	relations := []string{"duplicate_to"}
	if !q.API {
		relations = append(relations, "created_by")
	}
	for _, field := range relations {
		if raw, ok := input.Fields[field]; ok {
			if raw == nil || raw == "" {
				values[field+"_id"] = nil
				continue
			}
			table := "issues"
			if field == "created_by" {
				table = "users"
			}
			id, messages, err := r.primaryKey(ctx, raw, table, false)
			if err != nil {
				return nil, err
			}
			if len(messages) > 0 {
				failures[field] = messages
			} else {
				values[field+"_id"] = id
			}
		}
	}
	if len(failures) > 0 {
		return nil, invalid(failures)
	}
	if id := values["duplicate_to_id"]; id != nil {
		var foreign bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1 AND workspace_id<>$2)`, id, row["workspace_id"]).Scan(&foreign); err != nil {
			return nil, err
		}
		if foreign {
			return nil, invalid(map[string]any{"duplicate_to": []string{"Invalid duplicate issue."}})
		}
	}
	if values["status"] == int64(1) {
		var missing bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i JOIN states s ON s.id=i.state_id WHERE i.id=$1 AND s."group"='triage') AND NOT EXISTS(SELECT 1 FROM states WHERE project_id=$2 AND workspace_id=$3 AND "default" AND "group"<>'triage' AND deleted_at IS NULL)`, row["issue_id"], q.ProjectID, row["workspace_id"]).Scan(&missing); err != nil {
			return nil, err
		}
		if missing {
			return nil, invalid(map[string]any{"status": []string{"Cannot accept intake issue: No default state found for the project"}})
		}
	}
	return values, nil
}

func (r *Repository) authorizeIntake(ctx context.Context, q Request) error {
	if !q.API {
		roles := []int{20}
		if q.Method == "POST" {
			roles = []int{5, 15, 20}
		}
		if err := r.allowProject(ctx, q, roles, q.Method != "POST"); err != nil {
			return err
		}
	}
	return nil
}
