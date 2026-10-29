package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	labels "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type labelWriter interface {
	MutateLabel(context.Context, labels.LabelMutation) (labels.LabelMutationResult, error)
	VisibleLabelWrites(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

func (s *Service) Label(ctx context.Context, input *api.LabelRequest) (loom.Nullable[api.LabelResult], error) {
	var result *api.LabelResult
	var err error
	if input == nil {
		return loom.Nullable[api.LabelResult]{}, labelInvalid()
	}
	if input.Kind() == api.LabelRequestKindLabelList || input.Kind() == api.LabelRequestKindLabelRetrieve {
		result, err = s.labelRead(ctx, input)
	} else {
		result, err = s.labelMutation(ctx, input)
	}
	if err != nil {
		return loom.Nullable[api.LabelResult]{}, err
	}
	if result == nil {
		return loom.NullValue[api.LabelResult](), nil
	}
	return loom.NullableValue(*result), nil
}
func labelInvalid() error { return api.MakeInvalidParams(errors.New("invalid label input")) }
func labelUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, labelInvalid()
	}
	return id, nil
}
func selectLabelMutation(input *api.LabelRequest) (labels.LabelMutation, *string, *string, error) {
	var cmd labels.LabelMutation
	var project, target string
	var workspace, request *string
	switch input.Kind() {
	case api.LabelRequestKindLabelCreate:
		v, ok := input.AsLabelCreate()
		if !ok || v == nil || v.Action != "create" {
			return cmd, nil, nil, labelInvalid()
		}
		cmd.Action, project, workspace, request = v.Action, v.ProjectID, v.WorkspaceSlug, v.RequestID
		cmd.Changes = labels.LabelChanges{Name: &v.Name, Color: v.Color, Description: v.Description, Parent: v.Parent, ExternalSource: v.ExternalSource, ExternalID: v.ExternalID, SortOrder: nullableStateInput(v.SortOrder)}
	case api.LabelRequestKindLabelUpdate:
		v, ok := input.AsLabelUpdate()
		if !ok || v == nil || v.Action != "update" {
			return cmd, nil, nil, labelInvalid()
		}
		cmd.Action, project, target, workspace, request, cmd.ExpectedRevision = v.Action, v.ProjectID, v.LabelID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
		cmd.Changes = labels.LabelChanges{Name: v.Name, Color: v.Color, Description: v.Description, Parent: v.Parent, ExternalSource: v.ExternalSource, ExternalID: v.ExternalID, SortOrder: nullableStateInput(v.SortOrder)}
	case api.LabelRequestKindLabelDelete:
		v, ok := input.AsLabelDelete()
		if !ok || v == nil || v.Action != "delete" {
			return cmd, nil, nil, labelInvalid()
		}
		cmd.Action, project, target, workspace, request, cmd.ExpectedRevision = v.Action, v.ProjectID, v.LabelID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
	default:
		return cmd, nil, nil, labelInvalid()
	}
	var err error
	cmd.ProjectID, err = labelUUID(project)
	if err != nil {
		return cmd, nil, nil, err
	}
	if cmd.Action != "create" {
		cmd.LabelID, err = labelUUID(target)
		if err != nil {
			return cmd, nil, nil, err
		}
	}
	if request != nil {
		id, parseErr := labelUUID(*request)
		if parseErr != nil {
			return cmd, nil, nil, parseErr
		}
		cmd.RequestID = &id
	}
	return cmd, workspace, request, nil
}
func (s *Service) labelMutation(ctx context.Context, input *api.LabelRequest) (result *api.LabelResult, err error) {
	cmd, workspace, _, err := selectLabelMutation(input)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.label."+cmd.Action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	var value labels.LabelMutationResult
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "label", cmd.Action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_project_id"] = cmd.ProjectID.String()
			fields["committed_label_id"] = value.LabelID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, workspace, true)
	if authErr != nil {
		return nil, authErr
	}
	actor = current
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	cmd.Revalidate = mutationRevalidator(actor, workspace, false)
	value, err = s.labelWrites.MutateLabel(verified, cmd)
	if err != nil {
		return nil, labelMutationError(err)
	}
	var record *api.LabelRecord
	if cmd.Action != "delete" {
		var row map[string]any
		if json.Unmarshal(value.Result, &row) != nil || row == nil {
			return nil, labelMutationError(nil)
		}
		record, err = projectLabelRecord(djangoapi.Timestamps(row, actor.Timezone).(map[string]any))
		if err != nil {
			return nil, labelMutationError(err)
		}
	}
	if cmd.RequestID == nil && cmd.ExpectedRevision == nil {
		if record == nil {
			return nil, nil
		}
		response := api.NewLabelResultLabelRecord(record)
		return &response, nil
	}
	receipt := &api.LabelMutationReceipt{LabelID: value.LabelID, Revision: value.Revision, Result: loom.NullValue[api.LabelRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if cmd.RequestID != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return nil, labelMutationError(nil)
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	response := api.NewLabelResultLabelMutationReceipt(receipt)
	return &response, nil
}
func labelMutationError(err error) error {
	switch {
	case errors.Is(err, labels.ErrLabelRevision):
		return api.MakeRevisionConflict(errors.New("label revision changed"))
	case errors.Is(err, mutationreceipts.ErrInvalidRequest):
		return labelInvalid()
	case errors.Is(err, mutationreceipts.ErrConflict):
		return api.MakeRequestConflict(errors.New("request input changed"))
	case errors.Is(err, mutationreceipts.ErrIndeterminate):
		return api.MakeIndeterminate(errors.New("unacknowledged commit"))
	}
	var failure *labels.HTTPError
	if errors.As(err, &failure) {
		switch failure.Status {
		case 400:
			return labelInvalid()
		case 403:
			return api.MakeForbidden(errors.New("label access denied"))
		case 404:
			return api.MakeNotFound(errors.New("label unavailable"))
		case 409:
			return api.MakeConflict(errors.New("label conflict"))
		}
	}
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) {
		return err
	}
	return api.MakeInternalError(errors.New("label operation failed"))
}
