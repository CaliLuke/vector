package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/workitems"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

// Closed generated action variants own the wire validation; this structure
// translates their common selectors into native item application requests.
type intakeInput struct {
	Action          string  `json:"action"`
	Project         string  `json:"project_id"`
	Item            string  `json:"workitem_id"`
	Workspace       *string `json:"workspace_slug"`
	Name            *string `json:"name"`
	Description     *string `json:"description_html"`
	Priority        *string `json:"priority"`
	Status          *int    `json:"status"`
	Snoozed         *string `json:"snoozed_till"`
	Duplicate       *string `json:"duplicate_to"`
	Source          *string `json:"source"`
	Email           *string `json:"source_email"`
	Cursor          *string `json:"cursor"`
	PerPage         *int    `json:"per_page"`
	IncludeRevision bool    `json:"include_revision"`
	Request         *string `json:"request_id"`
	Revision        *string `json:"expected_revision"`
}

func (s *Service) Intake(ctx context.Context, input *api.IntakeRequest) (loom.Nullable[api.IntakeResult], error) {
	var selected intakeInput
	if input == nil || input.Validate() != nil {
		return loom.Nullable[api.IntakeResult]{}, api.MakeInvalidParams(errors.New("invalid intake input"))
	}
	raw, err := json.Marshal(input)
	if err != nil || json.Unmarshal(raw, &selected) != nil {
		return loom.Nullable[api.IntakeResult]{}, workitemResultError()
	}
	if selected.Action == "list" || selected.Action == "retrieve" {
		return s.readIntake(ctx, selected)
	}
	fields := map[string]any{}
	for key, value := range map[string]*string{"name": selected.Name, "description_html": selected.Description, "priority": selected.Priority, "snoozed_till": selected.Snoozed, "duplicate_to": selected.Duplicate, "source": selected.Source, "source_email": selected.Email} {
		if value != nil && *value != "" {
			fields[key] = *value
		}
	}
	if selected.Status != nil {
		fields["status"] = *selected.Status
	}
	mutation := resourceMutationInput{action: selected.Action, project: selected.Project, target: selected.Item, workspace: selected.Workspace, request: selected.Request, revision: selected.Revision, fields: fields}
	value, err := s.mutateResource(ctx, "intake", mutation, resourceMutationFunc(s.collaborationReads.MutateIntake))
	if err != nil {
		return loom.Nullable[api.IntakeResult]{}, err
	}
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	var record *api.IntakeRecord
	if selected.Action != "delete" {
		var row map[string]any
		if json.Unmarshal(value.Record, &row) != nil {
			return loom.Nullable[api.IntakeResult]{}, workitemResultError()
		}
		record, err = projectIntakeRecord(row, actor.Timezone)
		if err != nil {
			return loom.Nullable[api.IntakeResult]{}, err
		}
	}
	if selected.Request == nil && selected.Revision == nil {
		if record == nil {
			return loom.NullValue[api.IntakeResult](), nil
		}
		return loom.NullableValue(api.NewIntakeResultIntakeRecord(record)), nil
	}
	receipt := &api.IntakeMutationReceipt{WorkitemID: value.ID, Revision: value.Revision, Result: loom.NullValue[api.IntakeRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if selected.Request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return loom.Nullable[api.IntakeResult]{}, workitemResultError()
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewIntakeResultIntakeMutationReceipt(receipt)), nil
}

func (s *Service) readIntake(ctx context.Context, in intakeInput) (loom.Nullable[api.IntakeResult], error) {
	project, err := typeUUID(in.Project)
	if err != nil {
		return loom.Nullable[api.IntakeResult]{}, err
	}
	if in.Action == "retrieve" {
		if _, err = typeUUID(in.Item); err != nil {
			return loom.Nullable[api.IntakeResult]{}, err
		}
	}
	ctx, actor, err := readActor(ctx, in.Workspace)
	if err != nil {
		return loom.Nullable[api.IntakeResult]{}, err
	}
	params, present := map[string]string{}, map[string]bool{}
	if in.Cursor != nil && *in.Cursor != "" {
		params["cursor"], present["cursor"] = *in.Cursor, true
	}
	if in.PerPage != nil && *in.PerPage != 0 {
		params["per_page"], present["per_page"] = strconv.Itoa(*in.PerPage), true
	}
	if in.IncludeRevision {
		params["include_revision"] = "true"
	}
	result, err := s.collaborationReads.ReadIntake(ctx, workitems.Request{ActorID: actor.UserID.String(), ExpectedWorkspaceID: &actor.WorkspaceID, Slug: actor.WorkspaceSlug, ProjectID: project.String(), IssueID: in.Item, API: true, Method: "GET", Operation: in.Action, Timezone: actor.Timezone, Params: params, Present: present})
	if err != nil {
		return loom.Nullable[api.IntakeResult]{}, archiveResourceError(err)
	}
	if in.Action == "retrieve" {
		row, ok := result.Body.(map[string]any)
		if !ok {
			return loom.Nullable[api.IntakeResult]{}, workitemResultError()
		}
		record, projectIntakeRecordErr := projectIntakeRecord(row, actor.Timezone)
		return loom.NullableValue(api.NewIntakeResultIntakeRecord(record)), projectIntakeRecordErr
	}
	var page map[string]any
	raw, err := json.Marshal(result.Body)
	if err != nil || json.Unmarshal(raw, &page) != nil {
		return loom.Nullable[api.IntakeResult]{}, workitemResultError()
	}
	rows, ok := page["results"].([]any)
	if !ok {
		return loom.Nullable[api.IntakeResult]{}, workitemResultError()
	}
	records := []*api.IntakeRecord{}
	for _, value := range rows {
		row, ok := value.(map[string]any)
		if !ok {
			return loom.Nullable[api.IntakeResult]{}, workitemResultError()
		}
		record, projectIntakeRecordErr2 := projectIntakeRecord(row, actor.Timezone)
		if projectIntakeRecordErr2 != nil {
			return loom.Nullable[api.IntakeResult]{}, projectIntakeRecordErr2
		}
		records = append(records, record)
	}
	page["results"] = records
	projected := &api.IntakePage{}
	if err = decodeWorkitemProjection(page, projected); err != nil {
		return loom.Nullable[api.IntakeResult]{}, err
	}
	return loom.NullableValue(api.NewIntakeResultIntakePage(projected)), nil
}

func projectIntakeRecord(row map[string]any, timezone string) (*api.IntakeRecord, error) {
	if row == nil {
		return nil, workitemResultError()
	}
	if nested, ok := row["issue_detail"].(map[string]any); ok {
		record, err := workitemReadProjection(projectNestedTimestamps(nested, timezone), false)
		if err != nil {
			return nil, err
		}
		row["issue_detail"] = record
	}
	record := &api.IntakeRecord{}
	return record, decodeWorkitemProjection(projectNestedTimestamps(row, timezone), record)
}
