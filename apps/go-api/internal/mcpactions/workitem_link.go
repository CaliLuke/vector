package mcpactions

import (
	"context"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) workitemLinkRead(ctx context.Context, input *api.WorkitemLinkRequest) (*api.WorkitemLinkResult, error) {
	return readProjectResource(ctx, input, selectWorkitemLinkAction, collaborationRead{s.collaborationReads, "links"}, nil, nil, projectWorkitemLinkResult)
}
func selectWorkitemLinkAction(input *api.WorkitemLinkRequest) (djangoapi.ProjectReadRequest, *string, error) {
	return selectItemReadUnion(input, func(input api.WorkitemLinkRequest) itemReadSelection {
		detail, ok := input.AsWorkitemLinkRetrieve()
		if !ok || detail == nil {
			return itemReadSelection{}
		}
		return retrieveItemSelection(detail.Action, detail.ProjectID, detail.WorkitemID, detail.LinkID, detail.WorkspaceSlug, detail.IncludeRevision)
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

	if revision, ok := row["revision"]; ok {
		wire["revision"] = revision
	}
	return decodeRecord[api.WorkitemLinkRecord](wire)
}
