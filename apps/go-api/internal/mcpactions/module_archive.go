package mcpactions

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) Module(ctx context.Context, input *api.ModuleRequest) (loom.Nullable[api.ModuleResult], error) {
	return dispatchResource(ctx, input, "invalid module input", moduleIsRead, s.moduleRead, s.mutateModule)
}

func moduleIsRead(input *api.ModuleRequest) bool {
	return input.Kind() == api.ModuleRequestKindModuleList || input.Kind() == api.ModuleRequestKindModuleRetrieve || input.Kind() == api.ModuleRequestKindModuleListWorkitems
}

func (s *Service) mutateModule(ctx context.Context, input *api.ModuleRequest) (loom.Nullable[api.ModuleResult], error) {
	mutation, err := selectModuleMutation(input)
	return mutateAndProjectResource(ctx, s, "module", mutation, err, s.moduleWrites, moduleMutationProjection)
}

var moduleMutationProjection = resourceProjection[api.ModuleRecord, api.ModuleResult]{
	record: func(row map[string]any) (*api.ModuleRecord, error) { return projectModuleRecord(row, false) }, result: api.NewModuleResultModuleRecord,
	receipt: func(receipt resourceReceipt[api.ModuleRecord]) api.ModuleResult {
		return api.NewModuleResultModuleMutationReceipt(&api.ModuleMutationReceipt{
			ModuleID: receipt.ID, Revision: receipt.Revision, Result: receipt.Result,
			RequestID: receipt.RequestID, ReplayExpiresAt: receipt.ReplayExpiresAt, Replayed: receipt.Replayed,
		})
	},
}

func selectModuleMutation(input *api.ModuleRequest) (resourceMutationInput, error) {
	var selected resourceMutationInput
	var fields map[string]*string
	if value, ok := input.AsModuleCreate(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID}
		fields = map[string]*string{"name": &value.Name, "description": value.Description, "start_date": value.StartDate, "external_source": value.ExternalSource, "external_id": value.ExternalID, "target_date": value.TargetDate, "status": value.Status, "lead": value.Lead}
		if err := moduleMembersInput(&selected, value.Members); err != nil {
			return selected, err
		}
	} else if value, ok := input.AsModuleUpdate(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID, target: value.ModuleID, revision: value.ExpectedRevision}
		fields = map[string]*string{"name": value.Name, "description": value.Description, "start_date": value.StartDate, "external_source": value.ExternalSource, "external_id": value.ExternalID, "target_date": value.TargetDate, "status": value.Status, "lead": value.Lead}
		if err := moduleMembersInput(&selected, value.Members); err != nil {
			return selected, err
		}
	} else if value, ok := input.AsModuleManageWorkitems(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, target: value.ModuleID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
		var err error
		selected.fields, err = planningFields(value.AddIds, value.RemoveIds)
		if err != nil {
			return selected, err
		}
	} else if value, ok := input.AsModuleArchive(); ok && value != nil {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, workspace: value.WorkspaceSlug, request: value.RequestID, target: value.ModuleID, revision: value.ExpectedRevision}
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

func moduleMembersInput(selected *resourceMutationInput, input *string) error {
	ids, err := milestoneIDs(input)
	if err != nil {
		return api.MakeInvalidParams(errors.New("invalid module members"))
	}
	if ids == nil {
		return nil
	}
	members := make([]any, len(ids))
	for i, id := range ids {
		members[i] = id.String()
	}
	selected.fields = map[string]any{"members": members}
	return nil
}
