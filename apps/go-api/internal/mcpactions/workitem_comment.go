package mcpactions

import (
	"context"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) workitemCommentRead(ctx context.Context, input *api.WorkitemCommentRequest) (*api.WorkitemCommentResult, error) {
	return readProjectResource(ctx, input, selectWorkitemCommentAction, collaborationRead{s.collaborationReads, "comments"}, nil, nil, projectWorkitemCommentResult)
}
func selectWorkitemCommentAction(input *api.WorkitemCommentRequest) (djangoapi.ProjectReadRequest, *string, error) {
	return selectItemReadUnion(input, func(input api.WorkitemCommentRequest) itemReadSelection {
		detail, ok := input.AsWorkitemCommentRetrieve()
		if !ok || detail == nil {
			return itemReadSelection{}
		}
		return retrieveItemSelection(detail.Action, detail.ProjectID, detail.WorkitemID, detail.CommentID, detail.WorkspaceSlug, detail.IncludeRevision)
	})
}

func projectWorkitemCommentResult(value any, operation string) (*api.WorkitemCommentResult, error) {
	return projectReadResult(value, operation == "detail", projectWorkitemCommentRecord, api.NewWorkitemCommentResultWorkitemCommentRecord, func(page *api.WorkitemCommentPage, records []*api.WorkitemCommentRecord) api.WorkitemCommentResult {
		page.Results = records
		return api.NewWorkitemCommentResultWorkitemCommentPage(page)
	})
}
func projectWorkitemCommentRecord(row map[string]any) (*api.WorkitemCommentRecord, error) {
	wire := map[string]any{}
	for _, field := range strings.Fields("id is_member created_at updated_at deleted_at comment_stripped comment_html attachments access external_source external_id edited_at created_by updated_by project workspace issue actor parent description") {
		wire[field] = row[field]
	}

	if value, ok := wire["comment_html"].(string); ok {
		wire["comment_html"] = tokenizeMentions(value)
	}
	if access := wire["access"]; access != nil && access != "INTERNAL" && access != "EXTERNAL" {
		return nil, errors.New("invalid comment access")
	}

	if revision, ok := row["revision"]; ok {
		wire["revision"] = revision
	}
	return decodeRecord[api.WorkitemCommentRecord](wire)
}
