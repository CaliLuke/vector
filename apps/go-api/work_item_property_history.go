package systemapi

import (
	"context"
	"io"

	store "github.com/CaliLuke/go-plane/internal/workitemhistory"
	api "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	"github.com/google/uuid"
)

func (service *workItemHistoryService) PropertyHistory(ctx context.Context, payload *api.PropertyHistoryPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	request, err := service.propertyHistoryRequest("property_history", payload.Slug, payload.ProjectID, payload.WorkItemID, "", "")
	if err != nil {
		return nil, nil, err
	}
	return service.read(ctx, request)
}

func (service *workItemHistoryService) PropertyVersions(ctx context.Context, payload *api.PropertyVersionsPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	request, err := service.propertyHistoryRequest("property_versions", payload.Slug, payload.ProjectID, payload.WorkItemID, payload.PropertyID, "")
	if err != nil {
		return nil, nil, err
	}
	if payload.Before != nil {
		before, parseErr := uuid.Parse(*payload.Before)
		if parseErr != nil {
			return nil, nil, service.error(400, map[string]any{"detail": "Invalid revision cursor."})
		}
		request.BeforeID = &before
	}
	return service.read(ctx, request)
}

func (service *workItemHistoryService) PropertyVersionDetail(ctx context.Context, payload *api.PropertyVersionDetailPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	request, err := service.propertyHistoryRequest("property_version_detail", payload.Slug, payload.ProjectID, payload.WorkItemID, payload.PropertyID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	return service.read(ctx, request)
}

func (service *workItemHistoryService) propertyHistoryRequest(operation, slug, project, item, property, version string) (store.Request, error) {
	request := store.Request{Operation: operation, Slug: slug}
	values := []string{project, item, property, version}
	targets := []*uuid.UUID{&request.ProjectID, &request.ResourceID, &request.PropertyID, &request.VersionID}
	for index, value := range values {
		if index >= 2 && value == "" {
			continue
		}
		parsed, err := uuid.Parse(value)
		if err != nil {
			return request, service.error(404, map[string]any{"error": "The required object does not exist."})
		}
		*targets[index] = parsed
	}
	return request, nil
}
