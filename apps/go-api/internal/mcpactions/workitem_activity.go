package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) WorkitemActivity(ctx context.Context, input *api.WorkitemActivityRequest) (*api.WorkitemActivityResult, error) {
	return readProjectResource(ctx, input, selectWorkitemActivityAction, activityRead{s.activityReads}, activityreads.ErrForbidden, activityreads.ErrNotFound, projectWorkitemActivityResult)
}
func selectWorkitemActivityAction(input *api.WorkitemActivityRequest) (djangoapi.ProjectReadRequest, *string, error) {
	return selectItemReadUnion(input, func(input api.WorkitemActivityRequest) itemReadSelection {
		detail, ok := input.AsWorkitemActivityRetrieve()
		if !ok || detail == nil || detail.Action != "retrieve" {
			return itemReadSelection{}
		}
		return itemReadSelection{action: detail.Action, project: detail.ProjectID, issue: detail.WorkitemID, target: detail.ActivityID, workspace: detail.WorkspaceSlug}
	})
}

func projectWorkitemActivityResult(value any, operation string) (*api.WorkitemActivityResult, error) {
	return projectReadResult(value, operation == "detail", projectWorkitemActivityRecord, api.NewWorkitemActivityResultWorkitemActivityRecord, func(page *api.WorkitemActivityPage, records []*api.WorkitemActivityRecord) api.WorkitemActivityResult {
		page.Results = records
		return api.NewWorkitemActivityResultWorkitemActivityPage(page)
	})
}
func projectWorkitemActivityRecord(row map[string]any) (*api.WorkitemActivityRecord, error) {
	wire := map[string]any{}
	for _, field := range strings.Fields("id created_at updated_at deleted_at verb field old_value new_value comment attachments old_identifier new_identifier epoch project workspace issue issue_comment actor") {
		wire[field] = row[field]
	}
	for _, field := range strings.Fields("project workspace") {
		if _, ok := wire[field].(string); !ok {
			return nil, errors.New("required work-item field missing")
		}
	}
	epoch, err := sdkInteger(wire["epoch"])
	if err != nil {
		return nil, err
	}
	wire["epoch"] = epoch

	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := new(api.WorkitemActivityRecord)
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
