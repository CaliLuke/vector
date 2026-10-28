package mcpactions

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) WorkitemLink(ctx context.Context, input *api.WorkitemLinkRequest) (loom.Nullable[api.WorkitemLinkResult], error) {
	return dispatchResource(ctx, input, "invalid work item link input", workitemLinkIsRead, s.workitemLinkRead, s.mutateWorkitemLink)
}

func workitemLinkIsRead(input *api.WorkitemLinkRequest) bool {
	return input.Kind() == api.WorkitemLinkRequestKindWorkitemReadList || input.Kind() == api.WorkitemLinkRequestKindWorkitemLinkRetrieve
}

func (s *Service) mutateWorkitemLink(ctx context.Context, input *api.WorkitemLinkRequest) (loom.Nullable[api.WorkitemLinkResult], error) {
	mutation, err := selectWorkitemLinkMutation(input)
	return mutateAndProjectResource(ctx, s, "workitem_link", mutation, err, resourceMutationFunc(s.collaborationReads.MutateLink), workitemLinkMutationProjection)
}

var workitemLinkMutationProjection = resourceProjection[api.WorkitemLinkRecord, api.WorkitemLinkResult]{
	record: projectWorkitemLinkRecord, result: api.NewWorkitemLinkResultWorkitemLinkRecord,
	receipt: func(receipt resourceReceipt[api.WorkitemLinkRecord]) api.WorkitemLinkResult {
		return api.NewWorkitemLinkResultWorkitemLinkMutationReceipt(&api.WorkitemLinkMutationReceipt{
			LinkID: receipt.ID, Revision: receipt.Revision, Result: receipt.Result,
			RequestID: receipt.RequestID, ReplayExpiresAt: receipt.ReplayExpiresAt, Replayed: receipt.Replayed,
		})
	},
}

func selectWorkitemLinkMutation(input *api.WorkitemLinkRequest) (resourceMutationInput, error) {
	var selected resourceMutationInput
	var url, title *string
	if value, ok := input.AsWorkitemLinkCreate(); ok && value != nil && value.Action == "create" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, workspace: value.WorkspaceSlug, request: value.RequestID}
		url, title = &value.URL, value.Title
	} else if value, ok := input.AsWorkitemLinkUpdate(); ok && value != nil && value.Action == "update" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, target: value.LinkID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
		url, title = value.URL, value.Title
	} else if value, ok := input.AsWorkitemLinkDelete(); ok && value != nil && value.Action == "delete" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, target: value.LinkID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
	} else {
		return selected, api.MakeInvalidParams(errors.New("invalid link mutation"))
	}
	selected.fields = map[string]any{}
	for key, value := range map[string]*string{"url": url, "title": title} {
		if value != nil && *value != "" {
			selected.fields[key] = *value
		}
	}
	return selected, nil
}
