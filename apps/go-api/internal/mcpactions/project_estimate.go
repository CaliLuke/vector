package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type estimateApplication interface {
	ReadEstimate(context.Context, projectconfiguration.EstimateRead) (json.RawMessage, error)
	MutateEstimate(context.Context, projectmutations.Command) (projectmutations.Result, error)
}

func (s *Service) ProjectEstimate(ctx context.Context, input *api.ProjectEstimateRequest) (loom.Nullable[api.ProjectEstimateResult], error) {
	if input == nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInvalidParams(errors.New("invalid estimate action"))
	}
	selected, read, includeRevision, err := selectProjectEstimate(input)
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, err
	}
	if read {
		return s.readProjectEstimate(ctx, selected, includeRevision)
	}
	if selected.action == "link" {
		return s.linkProjectEstimate(ctx, selected)
	}
	value, err := s.mutateResource(ctx, "project_estimate", selected, resourceMutationFunc(s.estimates.MutateEstimate))
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, err
	}
	result, err := estimateResult(selected.action, value.Record)
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, err
	}
	if selected.request == nil && selected.revision == nil {
		return result, nil
	}
	receipt := &api.ProjectEstimateMutationReceipt{ProjectID: value.ID, Revision: value.Revision, Estimate: loom.NullValue[api.EstimateRecord](), Point: loom.NullValue[api.EstimatePointRecord](), Points: []*api.EstimatePointRecord{}}
	if record, ok := result.Value(); ok {
		if v, ok := record.AsEstimateRecord(); ok {
			receipt.Estimate = loom.NullableValue(*v)
		}
		if v, ok := record.AsEstimatePointRecord(); ok {
			receipt.Point = loom.NullableValue(*v)
		}
		if v, ok := record.AsEstimatePoints(); ok {
			receipt.Points = v
		}
	}
	if selected.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInternalError(errors.New("missing estimate receipt"))
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewProjectEstimateResultProjectEstimateMutationReceipt(receipt)), nil
}
func (s *Service) readProjectEstimate(ctx context.Context, selected resourceMutationInput, revision bool) (loom.Nullable[api.ProjectEstimateResult], error) {
	ctx, actor, err := actionActor(ctx, selected.workspace, false)
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, err
	}
	in := projectconfiguration.EstimateRead{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, Timezone: actor.Timezone, Points: selected.action == "list_points", IncludeRevision: revision}
	in.ProjectID, err = uuid.Parse(selected.project)
	if err != nil || in.ProjectID == uuid.Nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInvalidParams(errors.New("invalid project identifier"))
	}
	if in.Points {
		in.EstimateID, err = uuid.Parse(selected.target)
		if err != nil || in.EstimateID == uuid.Nil {
			return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInvalidParams(errors.New("invalid estimate identifier"))
		}
	}
	raw, err := s.estimates.ReadEstimate(ctx, in)
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, archiveResourceError(err)
	}
	return estimateResult(selected.action, raw)
}
func (s *Service) linkProjectEstimate(ctx context.Context, selected resourceMutationInput) (loom.Nullable[api.ProjectEstimateResult], error) {
	id, err := uuid.Parse(selected.target)
	if err != nil || id == uuid.Nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInvalidParams(errors.New("invalid estimate identifier"))
	}
	// Linking is exactly the native project update, including its administrator
	// policy, ownership validation, effects, revision and receipt namespace.
	input := &api.ProjectUpdate{Action: "update", ProjectID: selected.project, WorkspaceSlug: selected.workspace, Estimate: ptrEstimateID(id.String()), RequestID: selected.request, ExpectedRevision: selected.revision}
	request := api.NewProjectRequestProjectUpdate(input)
	value, err := s.projectRecordMutation(ctx, &request)
	if err != nil {
		return loom.Nullable[api.ProjectEstimateResult]{}, err
	}
	result, ok := value.Value()
	if !ok {
		return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInternalError(errors.New("missing linked project"))
	}
	if v, ok := result.AsProjectDetail(); ok {
		return loom.NullableValue(api.NewProjectEstimateResultProjectDetail(v)), nil
	}
	if v, ok := result.AsProjectMutationReceipt(); ok {
		return loom.NullableValue(api.NewProjectEstimateResultProjectMutationReceipt(v)), nil
	}
	return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInternalError(errors.New("invalid linked project"))
}
func ptrEstimateID(value string) *string { return &value }
func estimateResult(action string, raw json.RawMessage) (loom.Nullable[api.ProjectEstimateResult], error) {
	if action == "delete" || action == "delete_point" {
		return loom.NullValue[api.ProjectEstimateResult](), nil
	}
	invalid := func() (loom.Nullable[api.ProjectEstimateResult], error) {
		return loom.Nullable[api.ProjectEstimateResult]{}, api.MakeInternalError(errors.New("invalid estimate result"))
	}
	switch action {
	case "list_points", "create_points":
		var points api.EstimatePoints
		if json.Unmarshal(raw, &points) != nil || points == nil {
			return invalid()
		}
		return loom.NullableValue(api.NewProjectEstimateResultEstimatePoints(points)), nil
	case "update_point":
		var point *api.EstimatePointRecord
		if json.Unmarshal(raw, &point) != nil || point == nil {
			return invalid()
		}
		return loom.NullableValue(api.NewProjectEstimateResultEstimatePointRecord(point)), nil
	default:
		var estimate *api.EstimateRecord
		if json.Unmarshal(raw, &estimate) != nil || estimate == nil {
			return invalid()
		}
		return loom.NullableValue(api.NewProjectEstimateResultEstimateRecord(estimate)), nil
	}
}
