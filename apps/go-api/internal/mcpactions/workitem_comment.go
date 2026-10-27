package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) WorkitemComment(ctx context.Context, input *api.WorkitemCommentRequest) (*api.WorkitemCommentResult, error) {
	return readProjectResource(ctx, input, selectWorkitemCommentAction, collaborationRead{s.collaborationReads, "comments"}, nil, nil, projectWorkitemCommentResult)
}
func selectWorkitemCommentAction(input *api.WorkitemCommentRequest) (djangoapi.ProjectReadRequest, *string, error) {
	return selectItemReadUnion(input, func(input api.WorkitemCommentRequest) itemReadSelection {
		detail, ok := input.AsWorkitemCommentRetrieve()
		if !ok || detail == nil || detail.Action != "retrieve" {
			return itemReadSelection{}
		}
		return itemReadSelection{action: detail.Action, project: detail.ProjectID, issue: detail.WorkitemID, target: detail.CommentID, workspace: detail.WorkspaceSlug}
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

	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := new(api.WorkitemCommentRecord)
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
