package mcpactions

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) Cycle(ctx context.Context, input *api.CycleRequest) (loom.Nullable[api.CycleResult], error) {
	return dispatchResource(ctx, input, "invalid cycle input", cycleIsRead, s.cycleRead, s.mutateCycle)
}

func cycleIsRead(input *api.CycleRequest) bool {
	return input.Kind() == api.CycleRequestKindCycleList || input.Kind() == api.CycleRequestKindCycleRetrieve || input.Kind() == api.CycleRequestKindCycleListWorkitems
}

func (s *Service) mutateCycle(ctx context.Context, input *api.CycleRequest) (loom.Nullable[api.CycleResult], error) {
	mutation, err := selectCycleMutation(input)
	return mutateAndProjectResource(ctx, s, "cycle", mutation, err, s.cycleWrites, cycleMutationProjection)
}

var cycleMutationProjection = resourceProjection[api.CycleRecord, api.CycleResult]{
	record: func(row map[string]any) (*api.CycleRecord, error) { return projectCycleRecord(row, false) }, result: api.NewCycleResultCycleRecord,
	receipt: func(receipt resourceReceipt[api.CycleRecord]) api.CycleResult {
		return api.NewCycleResultCycleMutationReceipt(&api.CycleMutationReceipt{
			CycleID: receipt.ID, Revision: receipt.Revision, Result: receipt.Result,
			RequestID: receipt.RequestID, ReplayExpiresAt: receipt.ReplayExpiresAt, Replayed: receipt.Replayed,
		})
	},
}

func selectCycleMutation(input *api.CycleRequest) (resourceMutationInput, error) {
	var selected resourceMutationInput
	var fields map[string]*string
	if value, ok := input.AsCycleCreate(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID}
		fields = map[string]*string{"name": &value.Name, "description": value.Description, "start_date": value.StartDate, "external_source": value.ExternalSource, "external_id": value.ExternalID, "end_date": value.EndDate, "owned_by": &value.OwnedBy, "timezone": value.Timezone}
	} else if value, ok := input.AsCycleUpdate(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID, target: value.CycleID, revision: value.ExpectedRevision}
		fields = map[string]*string{"name": value.Name, "description": value.Description, "start_date": value.StartDate, "external_source": value.ExternalSource, "external_id": value.ExternalID, "end_date": value.EndDate, "owned_by": value.OwnedBy, "timezone": value.Timezone}
	} else if value, ok := input.AsCycleManageWorkitems(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, target: value.CycleID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
		var err error
		selected.fields, err = planningFields(value.AddIds, value.RemoveIds)
		if err != nil {
			return selected, err
		}
	} else if value, ok := input.AsCycleTransferWorkitems(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, target: value.CycleID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision, fields: map[string]any{"new_cycle_id": value.NewCycleID}}
	} else if value, ok := input.AsCycleArchive(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID, target: value.CycleID, revision: value.ExpectedRevision}
	} else {
		return selected, api.MakeInvalidParams(errors.New("invalid mutation"))
	}
	if selected.fields == nil {
		selected.fields = map[string]any{}
	}
	for key, value := range fields {
		if value != nil && *value != "" {
			selected.fields[key] = *value
		}
	}
	return selected, nil
}
