package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) validatePublicIntake(ctx context.Context, scope PublicFeedbackScope, intakeID string) error {
	if scope.IntakeID == nil {
		return failure(400, "error", "Intake is not enabled for this Project Board")
	}
	if *scope.IntakeID != intakeID {
		return failure(404, "error", "Not found")
	}
	var exists bool
	if err := repository.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM intakes WHERE id=$1::uuid AND project_id=$2::uuid AND workspace_id=$3::uuid AND deleted_at IS NULL)`, *scope.IntakeID, scope.ProjectID, scope.WorkspaceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return failure(404, "error", "Not found")
	}
	return nil
}

// PublicIntakeList returns the issue serializer used by both public inbox and
// intake collections. The route aliases differ only in their URL binding.
func (repository *Repository) PublicIntakeList(ctx context.Context, scope PublicFeedbackScope, intakeID, _ string, params map[string]string, present map[string]bool) (Result, error) {
	if err := repository.validatePublicIntake(ctx, scope, intakeID); err != nil {
		return Result{}, err
	}
	args := []any{scope.ProjectID, scope.WorkspaceID, intakeID}
	where := []string{
		"issue.project_id=$1", "issue.workspace_id=$2", "intake.intake_id=$3", "issue.deleted_at IS NULL",
		"issue.id=intake.issue_id",
	}
	options := issuequeryOptions(params, present)
	var err error
	options.Filters = ""
	where, args, err = appFilters(options, where, args)
	if err != nil {
		return Result{}, err
	}
	rows, err := repository.rows(ctx, `SELECT `+publicIntakeIssueSelect()+` FROM issues issue JOIN intake_issues intake ON intake.issue_id=issue.id WHERE `+strings.Join(where, " AND ")+` ORDER BY intake.snoozed_till, intake.status`, args...)
	if err != nil {
		return Result{}, err
	}
	results := make([]any, 0, len(rows))
	for _, row := range rows {
		results = append(results, publicIntakeReadProjection(normalizePublicIntakeIssue(row), true))
	}
	return Result{Status: 200, Body: results}, nil
}

func (repository *Repository) PublicIntakeCreate(ctx context.Context, scope PublicFeedbackScope, intakeID, actorID string, body io.Reader, contentType string) (Result, error) {
	return repository.writeProjectTransaction(ctx, scope.ProjectID, func(writer *Repository) error {
		return writer.validatePublicIntake(ctx, scope, intakeID)
	}, func(writer *Repository) (Result, error) {
		return writer.publicIntakeCreate(ctx, scope, intakeID, actorID, body, contentType)
	})
}

func (repository *Repository) publicIntakeCreate(ctx context.Context, scope PublicFeedbackScope, intakeID, actorID string, body io.Reader, contentType string) (Result, error) {
	if err := repository.validatePublicIntake(ctx, scope, intakeID); err != nil {
		return Result{}, err
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if err := publicFeedbackObjectRootError(input); err != nil {
		return Result{}, err
	}
	nested, exists := input.Fields["issue"]
	if !exists {
		nested = map[string]any{}
	}
	fields, ok := nested.(map[string]any)
	if !ok {
		return Result{}, publicFeedbackNestedIssueError()
	}
	if !truthy(fields["name"]) {
		return Result{}, failure(400, "error", "Name is required")
	}
	priority := "low"
	if value, ok := fields["priority"]; ok {
		priority, ok = value.(string)
		if !ok || !containsString([]string{"low", "medium", "high", "urgent", "none"}, priority) {
			return Result{}, failure(400, "error", "Invalid priority")
		}
	}
	var stateID *string
	if err := repository.db.QueryRow(ctx, `SELECT (SELECT id::text FROM states WHERE project_id=$1 AND workspace_id=$2 AND "group"='triage' AND deleted_at IS NULL ORDER BY sequence LIMIT 1)`, scope.ProjectID, scope.WorkspaceID).Scan(&stateID); err != nil {
		return Result{}, err
	}
	if stateID == nil {
		id := uuid.NewString()
		_, err := repository.db.Exec(ctx, `INSERT INTO states(id,name,description,color,slug,sequence,"group",is_triage,"default",project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,'Triage','','#4E5355','triage',COALESCE((SELECT MAX(sequence)+15000 FROM states WHERE project_id=$2 AND deleted_at IS NULL AND "group"<>'triage'),65000),'triage',false,false,$2,$3,$4,$4,$5)`, id, scope.ProjectID, scope.WorkspaceID, timestamp(repository.now()), actorID)
		if err != nil {
			return Result{}, err
		}
		stateID = &id
	}
	row := defaultIssue(scope.ProjectID)
	row["workspace_id"] = scope.WorkspaceID
	row["state_id"] = *stateID
	row["name"] = fields["name"]
	row["priority"] = priority
	descriptionJSON, exists := fields["description_json"]
	if !exists {
		descriptionJSON = map[string]any{}
	}
	row["description_json"] = descriptionJSON
	rawDescription := "<p></p>"
	if value, exists := fields["description_html"]; exists {
		rawDescription = djangoinput.ScalarString(value)
	}
	safeDescription, _, sanitizeErr := content.ValidateHTML(rawDescription)
	if sanitizeErr != nil {
		safeDescription = "<p></p>"
	}
	row["description_html"] = safeDescription
	subscribers, err := repository.validateCustomValues(ctx, actorID, row, nil, nil, false, true, false)
	if err != nil {
		var validation *djangoapi.HTTPError
		if errors.As(err, &validation) {
			return Result{}, failure(400, "error", "This project cannot currently accept new feedback. Contact the project administrator.")
		}
		return Result{}, err
	}
	row, err = repository.saveIssue(ctx, actorID, row, nil)
	if err != nil {
		return Result{}, err
	}
	if err = repository.subscribePropertyMembers(ctx, actorID, row, subscribers); err != nil {
		return Result{}, err
	}
	q := Request{ProjectID: scope.ProjectID, Slug: scope.WorkspaceSlug, ActorID: actorID, IssueID: row["id"].(string)}
	requested, err := pythoncompat.Dumps(input.Fields, input.Raw)
	if err != nil {
		return Result{}, err
	}
	if err := repository.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
		"type": "issue.activity.created", "requested_data": requested, "actor_id": actorID,
		"issue_id": q.IssueID, "project_id": scope.ProjectID, "current_instance": nil,
		"epoch": repository.now().Unix(),
	}); err != nil {
		return Result{}, err
	}
	intakeRowID := uuid.NewString()
	now := timestamp(repository.now())
	_, err = repository.db.Exec(ctx, `INSERT INTO intake_issues(id,intake_id,issue_id,status,source,extra,project_id,workspace_id,created_at,updated_at,created_by_id) VALUES($1,$2,$3,-2,'IN_APP','{}',$4,$5,$6,$6,$7)`, intakeRowID, intakeID, q.IssueID, scope.ProjectID, scope.WorkspaceID, now, actorID)
	if err != nil {
		return Result{}, err
	}
	result, err := repository.publicIntakeIssue(ctx, q.IssueID, scope, intakeRowID)
	result = publicIntakeCreateProjection(result)
	return Result{Status: 200, Body: result}, err
}

func (repository *Repository) PublicIntakeDetail(ctx context.Context, scope PublicFeedbackScope, intakeID, pk string) (Result, error) {
	if err := repository.validatePublicIntake(ctx, scope, intakeID); err != nil {
		return Result{}, err
	}
	var issueID string
	err := repository.db.QueryRow(ctx, `SELECT issue_id::text FROM intake_issues WHERE id=$1::uuid AND intake_id=$2::uuid AND project_id=$3::uuid AND workspace_id=$4::uuid AND deleted_at IS NULL`, pk, intakeID, scope.ProjectID, scope.WorkspaceID).Scan(&issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, requiredObject()
	}
	if err != nil {
		return Result{}, err
	}
	result, err := repository.publicIntakeIssue(ctx, issueID, scope, "")
	if err == nil {
		result = publicIntakeReadProjection(result, false)
	}
	return Result{Status: 200, Body: result}, err
}

func (repository *Repository) PublicIntakeUpdate(ctx context.Context, scope PublicFeedbackScope, intakeID, pk, actorID string, body io.Reader, contentType string) (Result, error) {
	return repository.writeProjectTransaction(ctx, scope.ProjectID, func(writer *Repository) error {
		return writer.validatePublicIntake(ctx, scope, intakeID)
	}, func(writer *Repository) (Result, error) {
		return writer.publicIntakeUpdate(ctx, scope, intakeID, pk, actorID, body, contentType)
	})
}

func (repository *Repository) publicIntakeUpdate(ctx context.Context, scope PublicFeedbackScope, intakeID, pk, actorID string, body io.Reader, contentType string) (Result, error) {
	if err := repository.validatePublicIntake(ctx, scope, intakeID); err != nil {
		return Result{}, err
	}
	var intakeRow map[string]any
	var err error
	intakeRow, err = repository.row(ctx, `SELECT to_jsonb(intake) FROM intake_issues intake WHERE intake.id=$1::uuid AND intake.intake_id=$2::uuid AND intake.project_id=$3::uuid AND intake.workspace_id=$4::uuid AND intake.deleted_at IS NULL`, pk, intakeID, scope.ProjectID, scope.WorkspaceID)
	if err != nil {
		return Result{}, err
	}
	if intakeRow["created_by_id"] != actorID {
		return Result{}, failure(400, "error", "You cannot edit intake issues")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if err := publicFeedbackObjectRootError(input); err != nil {
		return Result{}, err
	}
	nested := input.Fields["issue"]
	var issue map[string]any
	issue, err = repository.row(ctx, `SELECT `+djangoapi.IssueRowSQL("issue")+` FROM issues issue WHERE issue.id=$1::uuid AND issue.workspace_id=$2::uuid AND issue.project_id=$3::uuid AND issue.deleted_at IS NULL`, intakeRow["issue_id"], scope.WorkspaceID, scope.ProjectID)
	if err != nil {
		return Result{}, err
	}
	nestedFields, nestedIsObject := nested.(map[string]any)
	if !nestedIsObject {
		return Result{}, publicFeedbackNestedIssueError()
	}
	old := maps.Clone(issue)
	issueData := map[string]any{}
	for _, field := range []string{"name", "description_html", "description_json"} {
		if value, exists := nestedFields[field]; exists {
			issueData[field] = value
		} else {
			issueData[field] = old[field]
		}
		issue[field] = issueData[field]
	}
	changes, err := repository.validateIssue(ctx, djangoinput.Input{Fields: issueData}, issueValidation{Partial: true, AllowTriage: true, ProjectID: scope.ProjectID, WorkspaceID: scope.WorkspaceID})
	if err != nil {
		return Result{}, err
	}
	maps.Copy(issue, changes.values)
	q := Request{ProjectID: scope.ProjectID, Slug: scope.WorkspaceSlug, ActorID: actorID, IssueID: fmt.Sprint(issue["id"])}
	current, err := orderedDump(publicIssueActivitySnapshot(old), publicIssueActivityOrder)
	if err != nil {
		return Result{}, err
	}
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(input.Raw, &rawFields); err != nil {
		return Result{}, err
	}
	requested, err := pythoncompat.Dumps(issueData, rawFields["issue"])
	if err != nil {
		return Result{}, err
	}
	if err := repository.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
		"type": "issue.activity.updated", "requested_data": requested, "actor_id": actorID,
		"issue_id": q.IssueID, "project_id": scope.ProjectID, "current_instance": current,
		"epoch": repository.now().Unix(),
	}); err != nil {
		return Result{}, err
	}
	issue, err = repository.saveIssue(ctx, actorID, issue, old)
	if err != nil {
		return Result{}, err
	}
	projection, err := repository.publicIntakeIssue(ctx, fmt.Sprint(issue["id"]), scope, "")
	if err == nil {
		projection = publicIntakePatchProjection(projection)
	}
	return Result{Status: 200, Body: projection}, err
}

func (repository *Repository) PublicIntakeDelete(ctx context.Context, scope PublicFeedbackScope, intakeID, pk, actorID string) (Result, error) {
	return repository.writeProjectTransaction(ctx, scope.ProjectID, func(writer *Repository) error {
		return writer.validatePublicIntake(ctx, scope, intakeID)
	}, func(writer *Repository) (Result, error) {
		return writer.publicIntakeDelete(ctx, scope, intakeID, pk, actorID)
	})
}

func (repository *Repository) publicIntakeDelete(ctx context.Context, scope PublicFeedbackScope, intakeID, pk, actorID string) (Result, error) {
	if err := repository.validatePublicIntake(ctx, scope, intakeID); err != nil {
		return Result{}, err
	}
	var createdBy, identifier string
	err := repository.db.QueryRow(ctx, `SELECT id::text,created_by_id::text FROM intake_issues WHERE id=$1::uuid AND intake_id=$2::uuid AND project_id=$3::uuid AND workspace_id=$4::uuid AND deleted_at IS NULL`, pk, intakeID, scope.ProjectID, scope.WorkspaceID).Scan(&identifier, &createdBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, requiredObject()
	}
	if err != nil {
		return Result{}, err
	}
	if createdBy != actorID {
		return Result{}, failure(400, "error", "You cannot delete intake issue")
	}
	return Result{Status: 204}, repository.softDeleteRow(ctx, "intake_issues", "intakeissue", identifier, actorID)
}

func issuequeryOptions(params map[string]string, present map[string]bool) issuequery.Options {
	params = maps.Clone(params)
	delete(params, "filters")
	request := Request{Params: params, Present: present}
	return queryOptions(request)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func publicIntakeIssueSelect() string {
	return djangoapi.IssueRowSQL("issue") + `||jsonb_build_object(
 'project',issue.project_id,'workspace',issue.workspace_id,'state',issue.state_id,
 'created_by',issue.created_by_id,'updated_by',issue.updated_by_id,'parent',` + djangoapi.PublicParentSQL("issue") + `,'parent_id',` + djangoapi.PublicParentSQL("issue") + `,
 'type',issue.type_id,'estimate_point',issue.estimate_point_id,
 'assignees',COALESCE((SELECT jsonb_agg(relation.assignee_id ORDER BY relation.created_at DESC) FROM issue_assignees relation WHERE relation.issue_id=issue.id AND relation.deleted_at IS NULL),'[]'::jsonb),
 'labels',COALESCE((SELECT jsonb_agg(relation.label_id ORDER BY relation.created_at DESC) FROM issue_labels relation WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "relation") + `),'[]'::jsonb),
 'state_detail',(SELECT jsonb_build_object('id',state.id,'name',state.name,'group',state."group",'color',state.color) FROM states state WHERE state.id=issue.state_id),
 'project_detail',(SELECT (jsonb_build_object('id',project.id,'identifier',project.identifier,'name',project.name,'description',project.description,'logo_props',project.logo_props,'cover_image',project.cover_image)||` + djangoapi.FileAssetURLFieldSQL("project.cover_image_asset_id", "project.cover_image", "cover_image_url") + `) FROM projects project WHERE project.id=issue.project_id),
 'label_details',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',label.id,'name',label.name,'color',label.color,'parent',label.parent_id) ORDER BY label.created_at DESC) FROM labels label JOIN issue_labels relation ON relation.label_id=label.id WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "relation") + `),'[]'::jsonb),
 'assignee_details',COALESCE((SELECT jsonb_agg((jsonb_build_object('id',actor.id,'first_name',actor.first_name,'last_name',actor.last_name,'avatar',actor.avatar,'display_name',actor.display_name,'is_bot',actor.is_bot)||` + djangoapi.FileAssetURLFieldSQL("actor.avatar_asset_id", "actor.avatar", "avatar_url") + `) ORDER BY actor.created_at DESC) FROM users actor JOIN issue_assignees relation ON relation.assignee_id=actor.id WHERE relation.issue_id=issue.id AND relation.deleted_at IS NULL),'[]'::jsonb),
 'issue_intake',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',related_intake.id,'duplicate_to',related_intake.duplicate_to_id,'snoozed_till',related_intake.snoozed_till,'source',related_intake.source,'status',related_intake.status)) FROM intake_issues related_intake WHERE related_intake.issue_id=issue.id AND related_intake.deleted_at IS NULL),'[]'::jsonb),
 'sub_issues_count',(SELECT COUNT(*) FROM issues child WHERE child.parent_id=issue.id AND ` + djangoapi.PublicRelatedIssueSQL("child", "issue") + ` AND child.deleted_at IS NULL AND child.archived_at IS NULL AND NOT child.is_draft)
 )`
}

var publicIssueActivityOrder = strings.Fields("id name state_id sort_order completed_at estimate_point priority start_date target_date sequence_id project_id parent_id created_at updated_at created_by updated_by is_draft archived_at")

func publicIssueActivitySnapshot(row map[string]any) map[string]any {
	sortOrder := row["sort_order"]
	if number, ok := sortOrder.(json.Number); ok {
		if value, err := number.Float64(); err == nil {
			sortOrder = json.Number(pythoncompat.FloatString(value))
		}
	}
	return map[string]any{
		"id": row["id"], "name": row["name"], "state_id": row["state_id"], "sort_order": sortOrder,
		"completed_at": row["completed_at"], "estimate_point": row["estimate_point_id"], "priority": row["priority"],
		"start_date": row["start_date"], "target_date": row["target_date"], "sequence_id": row["sequence_id"],
		"project_id": row["project_id"], "parent_id": row["parent_id"], "created_at": row["created_at"],
		"updated_at": row["updated_at"], "created_by": row["created_by_id"], "updated_by": row["updated_by_id"],
		"is_draft": row["is_draft"], "archived_at": row["archived_at"],
	}
}

func (repository *Repository) publicIntakeIssue(ctx context.Context, issueID string, scope PublicFeedbackScope, intakeID string) (map[string]any, error) {
	query := `SELECT ` + publicIntakeIssueSelect() + ` FROM issues issue WHERE issue.id=$1::uuid AND issue.workspace_id=$2::uuid AND issue.project_id=$3::uuid AND issue.deleted_at IS NULL`
	row, err := repository.row(ctx, query, issueID, scope.WorkspaceID, scope.ProjectID)
	if err != nil {
		return nil, err
	}
	if intakeID != "" {
		row["bridge_id"] = intakeID
	}
	return normalizePublicIntakeIssue(row), nil
}

func normalizePublicIntakeIssue(row map[string]any) map[string]any {
	fields := strings.Fields("id name description_html description_json description_stripped description_binary priority point estimate_point sequence_id sort_order start_date target_date completed_at archived_at created_at updated_at deleted_at created_by updated_by project project_id workspace workspace_id state state_id parent parent_id type external_id external_source is_draft assignee_ids label_ids assignees labels assignee_details label_details issue_intake project_detail state_detail sub_issues_count")
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, exists := row[field]; exists {
			out[field] = value
			continue
		}
		if alias := map[string]string{"estimate_point": "estimate_point_id", "created_by": "created_by_id", "updated_by": "updated_by_id", "project": "project_id", "workspace": "workspace_id", "state": "state_id", "parent": "parent_id", "type": "type_id", "assignee_ids": "assignees", "label_ids": "labels"}[field]; alias != "" {
			out[field] = row[alias]
			continue
		}
		out[field] = nil
	}
	return out
}

func publicIntakeCreateProjection(issue map[string]any) map[string]any {
	delete(issue, "sub_issues_count")
	return publicIntakeReadProjection(issue, false)
}

func publicIntakeReadProjection(issue map[string]any, includeSubIssueCount bool) map[string]any {
	for _, field := range []string{"project_id", "workspace_id", "state_id", "parent_id", "type_id", "assignee_ids", "label_ids"} {
		delete(issue, field)
	}
	if !includeSubIssueCount {
		delete(issue, "sub_issues_count")
	}
	return issue
}

func publicIntakePatchProjection(issue map[string]any) map[string]any {
	for _, field := range []string{"issue_intake", "sub_issues_count", "assignee_details", "label_details", "project_detail", "state_detail"} {
		delete(issue, field)
	}
	return issue
}
