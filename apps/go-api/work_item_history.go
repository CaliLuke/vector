package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/workitemhistory"
	api "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type workItemHistorySessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workItemHistoryRepository interface {
	Read(context.Context, store.Request) (json.RawMessage, error)
	ProjectEntityAllowed(context.Context, string, string, uuid.UUID) (bool, error)
}

type workItemHistoryPrincipal struct {
	userID, timezone string
}

type workItemHistoryPrincipalKey struct{}

type workItemHistoryService struct {
	repository workItemHistoryRepository
	sessions   workItemHistorySessions
}

func NewWorkItemHistory(repository workItemHistoryRepository, sessions workItemHistorySessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("work item history dependencies are required")
	}
	return &workItemHistoryService{repository: repository, sessions: sessions}, nil
}

func (service *workItemHistoryService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if credential == "" {
			credential = sessionCredentialFromContext(ctx)
		}
		principal, err := service.sessions.Validate(ctx, credential)
		if err == nil {
			userID, timezone = principal.UserID, principal.UserTimezone
		}
	}
	if _, err := uuid.Parse(userID); userID == "" || err != nil {
		return ctx, service.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, workItemHistoryPrincipalKey{}, workItemHistoryPrincipal{userID: userID, timezone: timezone}), nil
}

func (service *workItemHistoryService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	ctx, err := service.APIKeyAuth(ctx, credential, nil)
	if status, body := workItemHistoryProtocolError(err); status != 0 {
		return status, body
	}
	principal, err := service.principal(ctx)
	if err != nil {
		return workItemHistoryProtocolError(err)
	}
	return workItemHistoryTimezoneError(djangosession.ValidateUserTimezone(principal.timezone))
}

func (service *workItemHistoryService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := service.APIKeyAuth(ctx, credential, nil)
	if status, body := workItemHistoryProtocolError(err); status != 0 {
		return status, body
	}
	if resource.Pattern != "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/" || resource.Method == "GET" || resource.Method == "HEAD" {
		principal, err := service.principal(ctx)
		if err != nil {
			return workItemHistoryProtocolError(err)
		}
		return workItemHistoryTimezoneError(djangosession.ValidateUserTimezone(principal.timezone))
	}
	projectID, parseErr := uuid.Parse(resource.ProjectID)
	principal, principalErr := service.principal(ctx)
	if parseErr != nil || principalErr != nil {
		return 404, map[string]any{"detail": "Not found."}
	}
	allowed, err := service.repository.ProjectEntityAllowed(ctx, principal.userID, resource.Slug, projectID)
	if err != nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	if !allowed {
		return 403, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	return workItemHistoryTimezoneError(djangosession.ValidateUserTimezone(principal.timezone))
}

func workItemHistoryTimezoneError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	if errors.Is(err, djangosession.ErrTimezoneNotFound) {
		return 400, map[string]any{"error": "The required key does not exist."}
	}
	return 500, map[string]any{"error": "Something went wrong please try again later"}
}

func workItemHistoryProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkItemHistoryErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	encoded, _ := json.Marshal(failure.Body)
	if json.Unmarshal(encoded, &body) != nil || body == nil {
		body = map[string]any{}
	}
	return status, body
}

func (service *workItemHistoryService) error(status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkItemHistoryErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}

func (service *workItemHistoryService) principal(ctx context.Context) (workItemHistoryPrincipal, error) {
	principal, ok := ctx.Value(workItemHistoryPrincipalKey{}).(workItemHistoryPrincipal)
	if !ok {
		return principal, service.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return principal, nil
}

func (service *workItemHistoryService) read(ctx context.Context, request store.Request) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		return nil, nil, err
	}
	if request.Operation == "history" {
		allowed, checkErr := service.repository.ProjectEntityAllowed(ctx, principal.userID, request.Slug, request.ProjectID)
		if checkErr != nil {
			return nil, nil, service.internal()
		}
		if !allowed {
			return nil, nil, service.error(403, map[string]any{"detail": "You do not have permission to perform this action."})
		}
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		status, body := workItemHistoryTimezoneError(err)
		return nil, nil, service.error(status, body)
	}
	request.UserID = uuid.MustParse(principal.userID)
	request.Timezone = principal.timezone
	body, err := service.repository.Read(ctx, request)
	if err != nil {
		var httpError *djangoapi.HTTPError
		if errors.As(err, &httpError) {
			return nil, nil, service.error(httpError.Status, httpError.Body)
		}
		if errors.Is(err, store.ErrForbidden) {
			return nil, nil, service.error(403, map[string]any{"error": "You don't have the required permissions."})
		}
		if errors.Is(err, store.ErrProjectEntity) {
			return nil, nil, service.error(403, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		if errors.Is(err, store.ErrGuestRestricted) {
			return nil, nil, service.error(403, map[string]any{"error": "You are not allowed to view this issue"})
		}
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
		}
		return nil, nil, service.internal()
	}
	if !json.Valid(body) {
		return nil, nil, service.internal()
	}
	return &api.WorkItemHistoryResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(body)), nil
}

func (service *workItemHistoryService) internal() error {
	return service.error(500, map[string]any{"error": "Something went wrong please try again later"})
}

func (service *workItemHistoryService) History(ctx context.Context, payload *api.HistoryPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, err := workItemHistoryIDs(payload.ProjectID, payload.IssueID)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "history", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID, CreatedAtGT: payload.CreatedAtGt, ActivityType: payload.ActivityType})
}

func (service *workItemHistoryService) Meta(ctx context.Context, payload *api.MetaPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, err := workItemHistoryIDs(payload.ProjectID, payload.IssueID)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "meta", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID})
}

func (service *workItemHistoryService) Versions(ctx context.Context, payload *api.VersionsPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, err := workItemHistoryIDs(payload.ProjectID, payload.IssueID)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "versions", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID, Cursor: payload.Cursor})
}

func (service *workItemHistoryService) VersionDetail(ctx context.Context, payload *api.VersionDetailPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, versionID, err := workItemHistoryDetailIDs(payload.ProjectID, payload.IssueID, payload.Pk)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "version_detail", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID, VersionID: versionID})
}

func (service *workItemHistoryService) DescriptionVersions(ctx context.Context, payload *api.DescriptionVersionsPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, err := workItemHistoryIDs(payload.ProjectID, payload.WorkItemID)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "description_versions", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID, Cursor: payload.Cursor})
}

func (service *workItemHistoryService) DescriptionVersionDetail(ctx context.Context, payload *api.DescriptionVersionDetailPayload) (*api.WorkItemHistoryResponseMetadata, io.ReadCloser, error) {
	projectID, issueID, versionID, err := workItemHistoryDetailIDs(payload.ProjectID, payload.WorkItemID, payload.Pk)
	if err != nil {
		return nil, nil, service.error(404, map[string]any{"error": "The required object does not exist."})
	}
	return service.read(ctx, store.Request{Operation: "description_version_detail", Slug: payload.Slug, ProjectID: projectID, ResourceID: issueID, VersionID: versionID})
}

func workItemHistoryIDs(projectID, resourceID string) (uuid.UUID, uuid.UUID, error) {
	project, err := uuid.Parse(projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	resource, err := uuid.Parse(resourceID)
	return project, resource, err
}

func workItemHistoryDetailIDs(projectID, resourceID, versionID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	project, resource, err := workItemHistoryIDs(projectID, resourceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	version, err := uuid.Parse(versionID)
	return project, resource, version, err
}
