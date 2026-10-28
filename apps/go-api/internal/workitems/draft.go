package workitems

import (
	"context"
	"errors"
	"io"
	"maps"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

func (r *Repository) drafts(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := *r
	writer.db, writer.transactionalEffects = tx, true
	result, err := writer.writeDraft(ctx, q, body, contentType)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) writeDraft(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	publish := strings.Contains(q.Operation, "_draft_to_issue")
	workspace, err := r.lockDraftWriteScope(ctx, q, publish)
	if err != nil {
		return Result{}, err
	}
	if publish {
		if result, found, publicationErr := r.publishedDraft(ctx, q, workspace["id"]); found || publicationErr != nil {
			return result, publicationErr
		}
	}
	old, err := r.draftForWrite(ctx, q, workspace["id"], publish)
	if err != nil {
		return Result{}, err
	}
	if q.Method == "DELETE" {
		return Result{Status: 204}, r.deleteDraft(ctx, q, old)
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	submitted, err := submittedCustomValues(input)
	if err != nil {
		return Result{}, err
	}
	if publish && !input.Null && input.Kind == "" {
		q, err = r.publicationDestination(ctx, q, old, input)
		if err != nil {
			return Result{}, err
		}
		input, err = r.publicationInput(ctx, old, input)
		if err != nil {
			return Result{}, err
		}
		if err = r.validatePublicationRelations(ctx, q, input.Fields); err != nil {
			return Result{}, err
		}
	}
	edit, err := r.validateDraftEdit(ctx, q, input, old, workspace, publish)
	if err != nil {
		return Result{}, err
	}
	if publish {
		return r.publishDraft(ctx, q, old, input, edit.changes, submitted)
	}
	return r.persistDraftEdit(ctx, q, input, old, workspace, submitted, edit)
}

type draftEdit struct {
	project   any
	projectID string
	changes   issueChanges
}

func (r *Repository) lockDraftWriteScope(ctx context.Context, q Request, publish bool) (map[string]any, error) {
	roles := []int{5, 15, 20}
	if publish {
		roles = []int{15, 20}
	}
	if err := r.allowWorkspace(ctx, q, roles); err != nil {
		return nil, err
	}
	workspace, err := r.row(ctx, `SELECT to_jsonb(w) FROM workspaces w WHERE slug=$1 AND deleted_at IS NULL`, q.Slug)
	if err != nil {
		return nil, err
	}
	if _, err = r.db.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace["id"]); err != nil {
		return nil, err
	}
	if err = r.allowWorkspace(ctx, q, roles); err != nil {
		return nil, err
	}
	return workspace, nil
}

func (r *Repository) draftForWrite(ctx context.Context, q Request, workspace any, publish bool) (map[string]any, error) {
	var err error
	var old map[string]any
	if q.Method != "POST" || publish {
		id := q.Params["pk"]
		if publish {
			id = q.Params["draft_id"]
		}
		// Drafts remain private even to workspace administrators. Publication
		// and deletion use the same owner boundary as editing and listing.
		old, err = r.row(ctx, `SELECT to_jsonb(d) FROM draft_issues d WHERE d.id=$1 AND d.workspace_id=$2 AND d.created_by_id=$3 AND d.deleted_at IS NULL FOR UPDATE`, id, workspace, q.ActorID)
		if err != nil {
			if q.Method == "PATCH" {
				var e *djangoapi.HTTPError
				if errors.As(err, &e) && e.Status == 404 {
					return nil, failure(404, "error", "Issue not found")
				}
			}
			return nil, err
		}
	}
	return old, nil
}

func (r *Repository) validateDraftEdit(ctx context.Context, q Request, input djangoinput.Input, old, workspace map[string]any, publish bool) (draftEdit, error) {
	project := any(nil)
	if old != nil {
		project = old["project_id"]
	}
	if v, ok := input.Fields["project_id"]; ok && !publish {
		project = v
	}
	if publish {
		project = q.ProjectID
	}
	optionsProject := ""
	if project != nil {
		optionsProject = djangoinput.ScalarString(project)
	}
	validation := input
	validation.Fields = maps.Clone(input.Fields)
	if !publish {
		for _, f := range []string{"archived_at", "point", "sequence_id", "is_draft"} {
			delete(validation.Fields, f)
		}
	}
	for _, f := range []string{"assignee_ids", "label_ids"} {
		if value, ok := input.ListValue(f, q.Method == "PATCH"); ok {
			validation.Fields[f] = value
		}
	}
	changes, err := r.validateIssue(ctx, validation, issueValidation{ActorID: q.ActorID, Timezone: q.Timezone, Partial: q.Method == "PATCH", Draft: !publish, ProjectID: optionsProject, WorkspaceID: workspace["id"].(string)})
	if err != nil {
		return draftEdit{}, err
	}
	if !publish {
		if err = r.validateDraftProjectAndCompletion(ctx, q, input, &changes); err != nil {
			return draftEdit{}, err
		}
	}
	effective := project
	if v, ok := changes.values["project_id"]; ok {
		effective = v
	}
	if err = r.validateDraftReferences(ctx, input.Fields, effective, workspace["id"]); err != nil {
		return draftEdit{}, err
	}
	if err = r.validateTypeTenant(ctx, changes.values, input.Fields, workspace["id"]); err != nil {
		return draftEdit{}, err
	}
	return draftEdit{project: project, projectID: optionsProject, changes: changes}, nil
}

func (r *Repository) validateDraftProjectAndCompletion(ctx context.Context, q Request, input djangoinput.Input, changes *issueChanges) error {
	for _, field := range []string{"project_id", "project"} {
		value, ok := input.Fields[field]
		if !ok {
			continue
		}
		if value == nil || value == "" {
			changes.values["project_id"] = nil
		} else {
			id, messages, e := r.primaryKey(ctx, value, "projects", false)
			if e != nil {
				return e
			}
			if len(messages) > 0 {
				return invalid(map[string]any{field: messages})
			}
			changes.values["project_id"] = id
		}
	}
	if value, ok := input.Fields["completed_at"]; ok {
		if value == nil {
			changes.values["completed_at"] = nil
		} else {
			parsed, e := parseQueryDateTime(djangoinput.ScalarString(value), q.Timezone)
			if e != nil {
				return invalid(map[string]any{"completed_at": []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}})
			}
			changes.values["completed_at"] = parsed
		}
	}
	return nil
}

func (r *Repository) persistDraftEdit(ctx context.Context, q Request, input djangoinput.Input, old, workspace, submitted map[string]any, edit draftEdit) (Result, error) {
	project, optionsProject, changes := edit.project, edit.projectID, edit.changes
	var err error
	row := maps.Clone(old)
	if old == nil {
		row = defaultIssue(optionsProject)
		for _, f := range []string{"archived_at", "point", "sequence_id", "is_draft"} {
			delete(row, f)
		}
		row["name"] = nil
		row["project_id"] = project
		row["workspace_id"] = workspace["id"]
	}
	maps.Copy(row, changes.values)
	if row["project_id"] != nil && (old == nil || old["project_id"] != row["project_id"]) {
		projectRequest := q
		projectRequest.ProjectID = stringValue(row["project_id"])
		if err = r.allowProject(ctx, projectRequest, []int{5, 15, 20}, false); err != nil {
			return Result{}, err
		}
	}
	if _, err = r.validateCustomValues(ctx, q.ActorID, row, old, submitted, true, old == nil, hasSubmittedType(input.Fields)); err != nil {
		return Result{}, err
	}
	if old != nil {
		if err = r.replaceDraftRelations(ctx, q, row, changes, input, false); err != nil {
			return Result{}, err
		}
	}
	row, err = r.saveDraft(ctx, q.ActorID, row, old == nil)
	if err != nil {
		return Result{}, err
	}
	if old != nil {
		return Result{Status: 204}, nil
	}
	if err = r.replaceDraftRelations(ctx, q, row, changes, input, true); err != nil {
		return Result{}, err
	}
	if row["deleted_at"] != nil {
		return Result{201, nil}, nil
	}
	result, err := r.draftProjection(ctx, row, q.ActorID)
	return Result{201, result}, err
}

func (r *Repository) validateDraftReferences(ctx context.Context, input map[string]any, project, workspace any) error {
	if project != nil && project != "" {
		if id, e := djangoinput.UUID(project); e == nil {
			var foreign bool
			if e = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND workspace_id<>$2)`, id.String(), workspace).Scan(&foreign); e != nil {
				return e
			}
			if foreign {
				return invalid(map[string]any{"project": []string{"Invalid project."}})
			}
		}
	}
	failures := map[string]any{}
	for _, entry := range []struct{ field, table, message string }{{"cycle_id", "cycles", "Invalid cycle."}, {"module_ids", "modules", "Invalid module."}} {
		values := []any{input[entry.field]}
		if entry.field == "module_ids" {
			values, _ = input[entry.field].([]any)
		}
		for _, value := range values {
			if id, e := djangoinput.UUID(value); e == nil {
				var foreign bool
				if e = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+entry.table+` WHERE id=$1 AND workspace_id<>$2)`, id.String(), workspace).Scan(&foreign); e != nil {
					return e
				}
				if foreign {
					failures[entry.field] = []string{entry.message}
					break
				}
			}
		}
	}
	if len(failures) > 0 {
		return invalid(failures)
	}
	return nil
}

func (r *Repository) saveDraft(ctx context.Context, actor string, row map[string]any, create bool) (map[string]any, error) {
	if row["project_id"] != nil {
		project, err := r.project(ctx, djangoinput.ScalarString(row["project_id"]))
		if err != nil {
			return nil, err
		}
		row["workspace_id"] = project["workspace_id"]
	}
	now := timestamp(r.now())
	if row["state_id"] == nil {
		var state *string
		if err := r.db.QueryRow(ctx, `SELECT (SELECT id::text FROM states WHERE project_id IS NOT DISTINCT FROM $1::uuid AND deleted_at IS NULL AND NOT is_triage ORDER BY "default" DESC,sequence LIMIT 1)`, row["project_id"]).Scan(&state); err != nil {
			return nil, err
		}
		if state != nil {
			row["state_id"] = *state
		}
	} else {
		var completed bool
		if err := r.db.QueryRow(ctx, `SELECT "group"='completed' FROM states WHERE id=$1`, row["state_id"]).Scan(&completed); err != nil {
			return nil, err
		}
		row["completed_at"] = nil
		if completed {
			row["completed_at"] = now
		}
	}
	row["description_stripped"] = stripTags(row["description_html"].(string))
	row["updated_at"] = now
	if create {
		row["created_at"], row["created_by_id"], row["updated_by_id"] = now, actor, nil
		var order *float64
		if err := r.db.QueryRow(ctx, `SELECT MAX(sort_order) FROM draft_issues WHERE project_id IS NOT DISTINCT FROM $1::uuid AND state_id IS NOT DISTINCT FROM $2::uuid AND deleted_at IS NULL`, row["project_id"], row["state_id"]).Scan(&order); err != nil {
			return nil, err
		}
		if order != nil {
			row["sort_order"] = *order + 10000
		}
	} else {
		row["updated_by_id"] = actor
	}
	data, err := issueRecordJSON(row)
	if err != nil {
		return nil, err
	}
	query := `INSERT INTO draft_issues SELECT (jsonb_populate_record(NULL::draft_issues,$1::jsonb)).*`
	if !create {
		query = `UPDATE draft_issues d SET (name,project_id,workspace_id,state_id,parent_id,type_id,estimate_point_id,description_json,description_html,description_stripped,priority,start_date,target_date,sort_order,completed_at,external_id,external_source,updated_at,updated_by_id,deleted_at,custom_properties)=(SELECT x.name,x.project_id,x.workspace_id,x.state_id,x.parent_id,x.type_id,x.estimate_point_id,x.description_json,x.description_html,x.description_stripped,x.priority,x.start_date,x.target_date,x.sort_order,x.completed_at,x.external_id,x.external_source,x.updated_at,x.updated_by_id,x.deleted_at,x.custom_properties FROM jsonb_populate_record(NULL::draft_issues,$1::jsonb) x) WHERE d.id=($1::jsonb->>'id')::uuid`
	}
	if _, err = r.db.Exec(ctx, query, data); err != nil {
		return nil, err
	}
	return r.row(ctx, `SELECT to_jsonb(d) FROM draft_issues d WHERE id=$1`, row["id"])
}

func (r *Repository) deleteDraft(ctx context.Context, q Request, row map[string]any) error {
	if _, err := r.db.Exec(ctx, `UPDATE draft_issues SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1`, row["id"], timestamp(r.now()), q.ActorID); err != nil {
		return err
	}
	return r.softDelete(ctx, "draftissue", row["id"].(string))
}

func (r *Repository) replaceDraftRelations(ctx context.Context, q Request, row map[string]any, changes issueChanges, input djangoinput.Input, create bool) error {
	cycle, cycleSet := input.Fields["cycle_id"]
	module, moduleSet := input.Fields["module_ids"]
	cycleIDs := []string{}
	if cycle != nil && (create || truthy(cycle)) {
		cycleIDs = append(cycleIDs, djangoinput.ScalarString(cycle))
	}
	for _, rel := range []struct {
		kind, column  string
		ids           []string
		set, instance bool
	}{{"assignees", "assignee_id", changes.assignees, changes.assigneesSet, false}, {"labels", "label_id", changes.labels, changes.labelsSet, false}, {"cycles", "cycle_id", cycleIDs, cycleSet, true}, {"modules", "module_id", nil, moduleSet && module != nil, false}} {
		if !create && !rel.set {
			continue
		}
		table := "draft_issue_" + rel.kind
		if !create {
			if _, err := r.db.Exec(ctx, `UPDATE `+table+` SET deleted_at=$2 WHERE draft_issue_id=$1 AND deleted_at IS NULL`, row["id"], timestamp(r.now())); err != nil {
				return err
			}
		}
		if rel.kind == "modules" {
			// The source constructs raw module associations after replacing
			// prior relationships and soft-deleting existing module links.
			var err error
			rel.ids, err = rawIDs(module)
			if module != nil && err != nil {
				return err
			}
		}
		if len(rel.ids) == 0 {
			continue
		}
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return err
		}
		for _, id := range rel.ids {
			creator, updater := row["created_by_id"], row["updated_by_id"]
			workspace := row["workspace_id"]
			if rel.instance {
				creator, updater = q.ActorID, nil
				if row["project_id"] != nil {
					project, e := r.project(ctx, djangoinput.ScalarString(row["project_id"]))
					if e != nil {
						err = e
						break
					}
					workspace = project["workspace_id"]
				}
			}
			_, err = tx.Exec(ctx, `INSERT INTO `+table+`(id,draft_issue_id,`+rel.column+`,workspace_id,project_id,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$8)`, uuid.NewString(), row["id"], id, workspace, row["project_id"], timestamp(r.now()), creator, updater)
			if err != nil {
				break
			}
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) draftProjection(ctx context.Context, row map[string]any, actor string) (map[string]any, error) {
	row, err := djangoapi.MaskIssueParent(ctx, r.db, row, actor)
	if err != nil {
		return nil, err
	}
	result := projectFields(row, []string{"id", "name", "state_id", "sort_order", "completed_at", "estimate_point", "priority", "start_date", "target_date", "project_id", "parent_id", "created_at", "updated_at", "created_by", "updated_by", "type_id", "description_html", "custom_properties"})
	var cycle *string
	if err := r.db.QueryRow(ctx, `SELECT (SELECT cycle_id::text FROM draft_issue_cycles WHERE draft_issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1)`, row["id"]).Scan(&cycle); err != nil {
		return nil, err
	}
	result["cycle_id"] = cycle
	for _, rel := range []struct{ table, column, field, predicate string }{{"draft_issue_labels", "label_id", "label_ids", ""}, {"draft_issue_assignees", "assignee_id", "assignee_ids", ` AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.member_id=d.assignee_id AND pm.is_active)`}, {"draft_issue_modules", "module_id", "module_ids", ` AND EXISTS(SELECT 1 FROM modules m WHERE m.id=d.module_id AND m.archived_at IS NULL)`}} {
		query := `SELECT DISTINCT ` + rel.column + `::text FROM ` + rel.table + ` d WHERE draft_issue_id=$1 AND deleted_at IS NULL` + rel.predicate + ` ORDER BY 1`
		if rel.field == "label_ids" {
			query = `SELECT DISTINCT relation.label_id::text FROM draft_issue_labels relation JOIN draft_issues draft ON draft.id=relation.draft_issue_id WHERE draft.id=$1 AND ` + issuequery.LiveDraftIssueLabelPredicate("draft", "relation") + ` ORDER BY 1`
		}
		ids, err := r.filteredIDs(ctx, query, row["id"])
		if err != nil {
			return nil, err
		}
		result[rel.field] = ids
	}
	return result, nil
}

func (r *Repository) publishDraft(ctx context.Context, q Request, draft map[string]any, input djangoinput.Input, changes issueChanges, submitted map[string]any) (Result, error) {
	if err := r.allowProject(ctx, q, []int{15, 20}, false); err != nil {
		return Result{}, err
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	row := defaultIssue(q.ProjectID)
	row["workspace_id"] = draft["workspace_id"]
	maps.Copy(row, changes.values)
	subscribers, err := r.validateCustomValues(ctx, q.ActorID, row, draft, submitted, false, true, hasSubmittedType(input.Fields))
	if err != nil {
		return Result{}, err
	}
	input = r.composedItemEditInput(input, row)
	if err = r.validateDraftUploads(ctx, draft, row); err != nil {
		return Result{}, err
	}
	row, err = r.saveIssue(ctx, q.ActorID, row, nil)
	if err != nil {
		return Result{}, err
	}
	if len(changes.assignees) == 0 && !changes.assigneesSet && project["default_assignee_id"] != nil {
		changes.assignees, err = r.filteredIDs(ctx, `SELECT member_id::text FROM project_members WHERE project_id=$1 AND member_id=$2 AND role>=15 AND is_active AND deleted_at IS NULL`, q.ProjectID, project["default_assignee_id"])
		if err != nil {
			return Result{}, err
		}
	}
	if err = r.replaceIssueRelations(ctx, row, changes, true); err != nil {
		return Result{}, err
	}
	if err = r.subscribePropertyMembers(ctx, q.ActorID, row, subscribers); err != nil {
		return Result{}, err
	}
	q.IssueID = row["id"].(string)
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	if err = r.activity(ctx, q, "issue.activity.created", requested, nil, false); err != nil {
		return Result{}, err
	}
	if cycle := input.Fields["cycle_id"]; truthy(cycle) {
		now := timestamp(r.now())
		record := map[string]any{"id": uuid.NewString(), "issue_id": q.IssueID, "cycle_id": cycle, "project_id": q.ProjectID, "workspace_id": row["workspace_id"], "created_at": now, "updated_at": now, "deleted_at": nil, "created_by_id": q.ActorID, "updated_by_id": nil}
		_, err = r.db.Exec(ctx, `INSERT INTO cycle_issues(id,issue_id,cycle_id,project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7)`, record["id"], q.IssueID, cycle, q.ProjectID, row["workspace_id"], now, q.ActorID)
		if err != nil {
			return Result{}, err
		}
		serialized, err := serializePlanningRecords("cycle", []map[string]any{record})
		if err != nil {
			return Result{}, err
		}
		current, _ := orderedDump(map[string]any{"updated_cycle_issues": nil, "created_cycle_issues": serialized}, []string{"updated_cycle_issues", "created_cycle_issues"})
		if err = r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{"type": "cycle.activity.created", "requested_data": nil, "actor_id": q.ActorID, "issue_id": nil, "project_id": q.ProjectID, "current_instance": current, "epoch": r.now().Unix(), "notification": true, "origin": r.origin}); err != nil {
			return Result{}, err
		}
	}
	if truthy(input.Fields["module_ids"]) {
		ids, err := rawIDs(input.Fields["module_ids"])
		if err != nil {
			return Result{}, err
		}
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return Result{}, err
		}
		for _, id := range ids {
			_, err = tx.Exec(ctx, `INSERT INTO module_issues(id,issue_id,module_id,project_id,workspace_id,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$8)`, uuid.NewString(), q.IssueID, id, q.ProjectID, draft["workspace_id"], timestamp(r.now()), draft["created_by_id"], draft["updated_by_id"])
			if err != nil {
				break
			}
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return Result{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return Result{}, err
		}
		for _, id := range ids {
			requested, _ := pythoncompat.Dumps(map[string]any{"module_id": id}, nil)
			if err = r.activity(ctx, q, "module.activity.created", requested, nil, false); err != nil {
				return Result{}, err
			}
		}
	}
	var reviewedAssets []string
	if r.itemEdit != nil {
		reviewedAssets = append([]string{}, r.itemEdit.publicationAssets...)
	}
	if _, err = r.db.Exec(ctx, `UPDATE file_assets SET issue_id=$2::uuid,entity_identifier=$2::uuid::text,project_id=$3,entity_type=CASE WHEN entity_type='DRAFT_ISSUE_ATTACHMENT' THEN 'ISSUE_ATTACHMENT' ELSE 'ISSUE_DESCRIPTION' END,draft_issue_id=NULL WHERE draft_issue_id=$1 AND deleted_at IS NULL AND NOT is_deleted AND NOT is_archived AND is_uploaded AND workspace_id=$4 AND created_by_id=$5 AND issue_id IS NULL AND comment_id IS NULL AND page_id IS NULL AND user_id IS NULL AND ($6::uuid[] IS NULL OR id=ANY($6::uuid[]))`, draft["id"], q.IssueID, q.ProjectID, draft["workspace_id"], q.ActorID, reviewedAssets); err != nil {
		return Result{}, err
	}
	if err = r.deleteDraft(ctx, q, draft); err != nil {
		return Result{}, err
	}
	if _, err = r.db.Exec(ctx, `INSERT INTO work_item_draft_publications(draft_id,workspace_id,actor_id,project_id,issue_id) VALUES($1,$2,$3,$4,$5)`, draft["id"], draft["workspace_id"], q.ActorID, q.ProjectID, q.IssueID); err != nil {
		return Result{}, err
	}
	body, projectionErr := r.visibleAppWriteProjection(ctx, q, row, input)
	return Result{201, body}, projectionErr
}
