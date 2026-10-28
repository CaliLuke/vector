package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/worklogs"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type workLogApplication interface {
	Summary(context.Context, worklogs.Query) (json.RawMessage, error)
	List(context.Context, worklogs.Query) (json.RawMessage, error)
	MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error)
}

func (s *Service) WorkLog(ctx context.Context, input *api.WorkLogRequest) (loom.Nullable[api.WorkLogResult], error) {
	if input == nil {
		return loom.Nullable[api.WorkLogResult]{}, api.MakeInvalidParams(errors.New("invalid work log action"))
	}
	if value, ok := input.AsWorkLogList(); ok && value != nil && value.Action == "list" {
		return s.listWorkLogs(ctx, value)
	}
	selected, err := selectWorkLogMutation(input)
	if err != nil {
		return loom.Nullable[api.WorkLogResult]{}, err
	}
	result, err := s.mutateResource(ctx, "work_log", selected, s.workLogs)
	if err != nil {
		return loom.Nullable[api.WorkLogResult]{}, err
	}
	var record *api.WorkLogRecord
	if selected.action != "delete" {
		actor, _ := mcpauth.PrincipalFromContext(ctx)
		record, err = projectWorkLogRecord(result.Record, actor.Timezone)
		if err != nil {
			return loom.Nullable[api.WorkLogResult]{}, err
		}
	}
	if selected.request == nil && selected.revision == nil {
		if record == nil {
			return loom.NullValue[api.WorkLogResult](), nil
		}
		return loom.NullableValue(api.NewWorkLogResultWorkLogRecord(record)), nil
	}
	receipt := &api.WorkLogMutationReceipt{WorkLogID: result.ID, Revision: result.Revision, Result: loom.NullValue[api.WorkLogRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if selected.request != nil {
		if result.RequestID == nil || result.ReplayExpiresAt == nil || result.Replayed == nil {
			return loom.Nullable[api.WorkLogResult]{}, workitemResultError()
		}
		id, expires := result.RequestID.String(), result.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, result.Replayed
	}
	return loom.NullableValue(api.NewWorkLogResultWorkLogMutationReceipt(receipt)), nil
}
func selectWorkLogMutation(input *api.WorkLogRequest) (resourceMutationInput, error) {
	selected := resourceMutationInput{fields: map[string]any{}}
	var description *string
	if value, ok := input.AsWorkLogCreate(); ok && value != nil && value.Action == "create" {
		selected.action, selected.project, selected.parent, selected.workspace, selected.request = value.Action, value.ProjectID, value.WorkitemID, value.WorkspaceSlug, value.RequestID
		selected.fields["duration"] = value.Duration
		description = value.Description
	} else if value, ok := input.AsWorkLogUpdate(); ok && value != nil && value.Action == "update" {
		selected.action, selected.project, selected.parent, selected.target = value.Action, value.ProjectID, value.WorkitemID, value.WorkLogID
		selected.workspace, selected.request, selected.revision = value.WorkspaceSlug, value.RequestID, value.ExpectedRevision
		if value.Duration != nil && *value.Duration != 0 {
			selected.fields["duration"] = *value.Duration
		}
		description = value.Description
	} else if value, ok := input.AsWorkLogDelete(); ok && value != nil && value.Action == "delete" {
		selected.action, selected.project, selected.parent, selected.target = value.Action, value.ProjectID, value.WorkitemID, value.WorkLogID
		selected.workspace, selected.request, selected.revision = value.WorkspaceSlug, value.RequestID, value.ExpectedRevision
	} else {
		return selected, api.MakeInvalidParams(errors.New("invalid work log mutation"))
	}
	if description != nil && *description != "" {
		selected.fields["description"] = *description
	}
	return selected, nil
}
func (s *Service) listWorkLogs(ctx context.Context, input *api.WorkLogList) (loom.Nullable[api.WorkLogResult], error) {
	project, err := uuid.Parse(input.ProjectID)
	if err != nil || project == uuid.Nil {
		return loom.Nullable[api.WorkLogResult]{}, api.MakeInvalidParams(errors.New("invalid work log project"))
	}
	item, err := uuid.Parse(input.WorkitemID)
	if err != nil || item == uuid.Nil {
		return loom.Nullable[api.WorkLogResult]{}, api.MakeInvalidParams(errors.New("invalid work log item"))
	}
	ctx, actor, err := readActor(ctx, input.WorkspaceSlug)
	if err != nil {
		return loom.Nullable[api.WorkLogResult]{}, err
	}
	q := worklogs.Query{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, ProjectID: project, IssueID: item, Params: map[string]string{}, Present: map[string]bool{}, IncludeRevision: input.IncludeRevision != nil && *input.IncludeRevision}
	if input.Cursor != nil && *input.Cursor != "" {
		q.Params["cursor"], q.Present["cursor"] = *input.Cursor, true
	}
	if input.PerPage != nil && *input.PerPage != 0 {
		if *input.PerPage < 0 || *input.PerPage > 1000 {
			return loom.Nullable[api.WorkLogResult]{}, api.MakeInvalidParams(errors.New("invalid work log page size"))
		}
		q.Params["per_page"], q.Present["per_page"] = strconv.Itoa(*input.PerPage), true
	}
	raw, err := s.workLogs.List(ctx, q)
	if err != nil {
		return loom.Nullable[api.WorkLogResult]{}, resourceReadError(err, projectmutations.ErrForbidden, projectmutations.ErrNotFound)
	}
	var rows []json.RawMessage
	if json.Unmarshal(raw, &rows) != nil {
		return loom.Nullable[api.WorkLogResult]{}, workitemResultError()
	}
	records := api.WorkLogRecords{}
	for _, row := range rows {
		record, err := projectWorkLogRecord(row, actor.Timezone)
		if err != nil {
			return loom.Nullable[api.WorkLogResult]{}, err
		}
		records = append(records, record)
	}
	return loom.NullableValue(api.NewWorkLogResultWorkLogRecords(records)), nil
}
func projectWorkLogRecord(raw json.RawMessage, timezone string) (*api.WorkLogRecord, error) {
	var value map[string]any
	if json.Unmarshal(raw, &value) != nil || value == nil {
		return nil, workitemResultError()
	}
	result := &api.WorkLogRecord{}
	return result, decodeWorkitemProjection(djangoapi.Timestamps(value, timezone), result)
}
