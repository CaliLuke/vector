package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) WorkitemLink(ctx context.Context, input *api.WorkitemLinkRequest) (*api.WorkitemLinkResult, error) {
	return readProjectResource(ctx, input, selectWorkitemLinkAction, collaborationRead{s.collaborationReads, "links"}, nil, nil, projectWorkitemLinkResult)
}
func selectWorkitemLinkAction(input *api.WorkitemLinkRequest) (djangoapi.ProjectReadRequest, *string, error) {
	return selectItemReadUnion(input, func(input api.WorkitemLinkRequest) itemReadSelection {
		detail, ok := input.AsWorkitemLinkRetrieve()
		if !ok || detail == nil || detail.Action != "retrieve" {
			return itemReadSelection{}
		}
		return itemReadSelection{action: detail.Action, project: detail.ProjectID, issue: detail.WorkitemID, target: detail.LinkID, workspace: detail.WorkspaceSlug}
	})
}

func projectWorkitemLinkResult(value any, operation string) (*api.WorkitemLinkResult, error) {
	return projectReadResult(value, operation == "detail", projectWorkitemLinkRecord, api.NewWorkitemLinkResultWorkitemLinkRecord, func(page *api.WorkitemLinkPage, records []*api.WorkitemLinkRecord) api.WorkitemLinkResult {
		page.Results = records
		return api.NewWorkitemLinkResultWorkitemLinkPage(page)
	})
}
func projectWorkitemLinkRecord(row map[string]any) (*api.WorkitemLinkRecord, error) {
	wire := map[string]any{}
	for _, field := range strings.Fields("id created_at updated_at deleted_at title url metadata created_by updated_by project workspace issue") {
		wire[field] = row[field]
	}
	for _, field := range strings.Fields("url") {
		if _, ok := wire[field].(string); !ok {
			return nil, errors.New("required work-item field missing")
		}
	}

	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := new(api.WorkitemLinkRecord)
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
