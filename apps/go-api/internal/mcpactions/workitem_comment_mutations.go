package mcpactions

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) WorkitemComment(ctx context.Context, input *api.WorkitemCommentRequest) (loom.Nullable[api.WorkitemCommentResult], error) {
	return dispatchResource(ctx, input, "invalid work item comment input", workitemCommentIsRead, s.workitemCommentRead, s.mutateWorkitemComment)
}

func workitemCommentIsRead(input *api.WorkitemCommentRequest) bool {
	return input.Kind() == api.WorkitemCommentRequestKindWorkitemReadList || input.Kind() == api.WorkitemCommentRequestKindWorkitemCommentRetrieve
}

func (s *Service) mutateWorkitemComment(ctx context.Context, input *api.WorkitemCommentRequest) (loom.Nullable[api.WorkitemCommentResult], error) {
	mutation, err := selectWorkitemCommentMutation(input)
	return mutateAndProjectResource(ctx, s, "workitem_comment", mutation, err, resourceMutationFunc(s.collaborationReads.MutateComment), workitemCommentMutationProjection)
}

var workitemCommentMutationProjection = resourceProjection[api.WorkitemCommentRecord, api.WorkitemCommentResult]{
	record: projectWorkitemCommentRecord, result: api.NewWorkitemCommentResultWorkitemCommentRecord,
	receipt: func(receipt resourceReceipt[api.WorkitemCommentRecord]) api.WorkitemCommentResult {
		return api.NewWorkitemCommentResultWorkitemCommentMutationReceipt(&api.WorkitemCommentMutationReceipt{
			CommentID: receipt.ID, Revision: receipt.Revision, Result: receipt.Result,
			RequestID: receipt.RequestID, ReplayExpiresAt: receipt.ReplayExpiresAt, Replayed: receipt.Replayed,
		})
	},
}

func selectWorkitemCommentMutation(input *api.WorkitemCommentRequest) (resourceMutationInput, error) {
	var selected resourceMutationInput
	var text map[string]*string
	if value, ok := input.AsWorkitemCommentCreate(); ok && value != nil && value.Action == "create" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, workspace: value.WorkspaceSlug, request: value.RequestID}
		text = map[string]*string{"comment_html": &value.CommentHTML, "access": value.Access, "external_source": value.ExternalSource, "external_id": value.ExternalID}
	} else if value, ok := input.AsWorkitemCommentUpdate(); ok && value != nil && value.Action == "update" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, target: value.CommentID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
		text = map[string]*string{"comment_html": value.CommentHTML, "access": value.Access, "external_source": value.ExternalSource, "external_id": value.ExternalID}
	} else if value, ok := input.AsWorkitemCommentDelete(); ok && value != nil && value.Action == "delete" {
		selected = resourceMutationInput{action: value.Action, project: value.ProjectID, parent: value.WorkitemID, target: value.CommentID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision}
	} else {
		return selected, api.MakeInvalidParams(errors.New("invalid comment mutation"))
	}
	selected.fields = map[string]any{}
	for key, value := range text {
		if value != nil && *value != "" {
			selected.fields[key] = *value
		}
	}
	return selected, nil
}
