package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/platform"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	api "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/jackc/pgx/v5"
)

const workspaceLifecycleInternalMessage = "Something went wrong please try again later"

type workspaceLifecycleSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceLifecycleRepository interface {
	WorkspaceRole(context.Context, uuid.UUID, string) (int, error)
	WorkspaceCreationDisabled(context.Context) (bool, error)
	CreateWorkspace(context.Context, uuid.UUID, map[string]json.RawMessage, string) (json.RawMessage, error)
	UpdateWorkspace(context.Context, uuid.UUID, string, map[string]json.RawMessage) (json.RawMessage, error)
	PrepareWorkspaceDeletion(context.Context, uuid.UUID, string) (instancemanagement.WorkspaceDeletion, error)
	SoftDeleteWorkspace(context.Context, uuid.UUID, instancemanagement.WorkspaceDeletion, time.Time) error
}

type workspaceLifecyclePublisher interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

type workspaceLifecycleService struct {
	repository workspaceLifecycleRepository
	sessions   workspaceLifecycleSessions
	publisher  workspaceLifecyclePublisher
}

type workspaceLifecyclePrincipal struct {
	userID   uuid.UUID
	timezone string
}

type workspaceLifecyclePrincipalKey struct{}

func NewWorkspaceLifecycle(repository workspaceLifecycleRepository, sessions workspaceLifecycleSessions, publisher workspaceLifecyclePublisher) (api.Service, error) {
	if repository == nil || sessions == nil || publisher == nil {
		return nil, errors.New("workspace lifecycle repository, session validator, and task publisher are required")
	}
	return &workspaceLifecycleService{repository: repository, sessions: sessions, publisher: publisher}, nil
}

func (service *workspaceLifecycleService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if key == "" {
			key = sessionCredentialFromContext(ctx)
		}
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, service.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
		}
		userID, timezone = principal.UserID, principal.UserTimezone
	}
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return ctx, service.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, workspaceLifecyclePrincipalKey{}, workspaceLifecyclePrincipal{userID: parsed, timezone: timezone}), nil
}

func (service *workspaceLifecycleService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := service.APIKeyAuth(ctx, credential, nil)
	return workspaceLifecycleProtocolError(err)
}

func (service *workspaceLifecycleService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	_, err := service.APIKeyAuth(ctx, credential, nil)
	if status, body := workspaceLifecycleProtocolError(err); status != 0 {
		return status, body
	}

	switch resource.Pattern {
	case "/api/workspaces/":
		switch resource.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost:
			return 0, nil
		default:
			return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
		}
	case "/api/workspaces/{slug}/":
		var authorizationErr error
		switch resource.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost:
			return 0, nil
		case http.MethodPut, http.MethodPatch:
			_, authorizationErr = service.workspaceAdmin(ctx, resource.Slug, false)
		case http.MethodDelete:
			_, authorizationErr = service.workspaceAdmin(ctx, resource.Slug, true)
		default:
			return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
		}
		return workspaceLifecycleProtocolError(authorizationErr)
	default:
		return 0, nil
	}
}

func workspaceLifecycleProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkspaceLifecycleError
	if !errors.As(err, &failure) {
		return http.StatusInternalServerError, map[string]any{"error": workspaceLifecycleInternalMessage}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	encoded, _ := json.Marshal(failure.Body)
	if json.Unmarshal(encoded, &body) != nil || body == nil {
		body = map[string]any{}
	}
	return status, body
}

func (service *workspaceLifecycleService) error(status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkspaceLifecycleError{Name: "status_" + strconv.Itoa(status), Body: value}
}

func (service *workspaceLifecycleService) principal(ctx context.Context) (workspaceLifecyclePrincipal, error) {
	principal, ok := ctx.Value(workspaceLifecyclePrincipalKey{}).(workspaceLifecyclePrincipal)
	if !ok {
		return principal, service.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	return principal, nil
}

func (service *workspaceLifecycleService) workspaceAdmin(ctx context.Context, slug string, deleting bool) (workspaceLifecyclePrincipal, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		return principal, err
	}
	role, err := service.repository.WorkspaceRole(ctx, principal.userID, slug)
	if err != nil {
		return principal, service.internal()
	}
	if role != 20 {
		if role == 0 {
			return principal, service.error(http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		if deleting {
			return principal, service.error(http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		return principal, service.error(http.StatusForbidden, map[string]any{"error": "You don't have the required permissions."})
	}
	return principal, nil
}

func (service *workspaceLifecycleService) internal() error {
	return service.error(http.StatusInternalServerError, map[string]any{"error": workspaceLifecycleInternalMessage})
}

func workspaceLifecycleRequestBody(ctx context.Context, body io.ReadCloser) (map[string]json.RawMessage, error) {
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if input.Kind != "" {
		return nil, fmt.Errorf("invalid data: expected dictionary, got %s", input.Kind)
	}
	encoded, err := json.Marshal(input.Fields)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return fields, nil
}

func workspaceLifecycleRaw(body json.RawMessage) (*api.WorkspaceLifecycleRawJSON, io.ReadCloser, error) {
	if !json.Valid(body) {
		return nil, nil, errors.New("workspace lifecycle repository returned invalid JSON")
	}
	return &api.WorkspaceLifecycleRawJSON{ContentType: "application/json"}, rawJSONReadCloser(body), nil
}

func (service *workspaceLifecycleService) CreateWorkspace(ctx context.Context, _ *api.CreateWorkspacePayload, body io.ReadCloser) (*api.WorkspaceLifecycleRawJSON, io.ReadCloser, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		closeWorkspaceLifecycleBody(body)
		return nil, nil, err
	}
	fields, err := workspaceLifecycleRequestBody(ctx, body)
	if err != nil {
		return nil, nil, service.error(http.StatusBadRequest, map[string]any{"detail": err.Error()})
	}
	disabled, err := service.repository.WorkspaceCreationDisabled(ctx)
	if err != nil {
		return nil, nil, service.internal()
	}
	if disabled {
		return nil, nil, service.error(http.StatusForbidden, map[string]any{"error": "Workspace creation is not allowed"})
	}
	var timezone string
	if value, ok := ReadServiceSessionTimezone(ctx); ok {
		timezone = value
	} else {
		timezone = principal.timezone
	}
	result, err := service.repository.CreateWorkspace(ctx, principal.userID, fields, timezone)
	if err != nil {
		var validation *instancemanagement.WorkspaceValidationError
		if errors.As(err, &validation) {
			if validation.Field != "" {
				return nil, nil, service.error(http.StatusBadRequest, map[string][]string{validation.Field: {validation.Message}})
			}
			if len(validation.Fields) != 0 {
				return nil, nil, service.error(http.StatusBadRequest, validation.Fields)
			}
			return nil, nil, service.error(http.StatusBadRequest, map[string]string{"error": validation.Message})
		}
		return nil, nil, service.internal()
	}
	var workspace map[string]any
	if json.Unmarshal(result, &workspace) != nil {
		return nil, nil, service.internal()
	}
	workspace["owner"] = principal.userID.String()
	workspace["total_members"] = 1
	workspace["role"] = 20
	result, err = json.Marshal(workspace)
	if err != nil {
		return nil, nil, service.internal()
	}
	workspaceID, _ := workspace["id"].(string)
	createdAt, _ := workspace["created_at"].(string)
	if err = service.publisher.Task(ctx, "plane.bgtasks.workspace_seed_task.workspace_seed", []any{workspaceID}, nil); err != nil {
		return nil, nil, service.internal()
	}
	properties := map[string]any{
		"user_id": principal.userID.String(), "workspace_id": workspaceID, "workspace_slug": workspace["slug"],
		"role": "owner", "workspace_name": workspace["name"], "created_at": createdAt,
	}
	if err = service.publisher.Task(ctx, "plane.bgtasks.event_tracking_task.track_event", nil, map[string]any{
		"user_id": principal.userID.String(), "event_name": "workspace_created", "slug": workspace["slug"], "event_properties": properties,
	}); err != nil {
		return nil, nil, service.internal()
	}
	return workspaceLifecycleRaw(result)
}

func (service *workspaceLifecycleService) updateWorkspace(ctx context.Context, slug string, body io.ReadCloser) (*api.WorkspaceLifecycleRawJSON, io.ReadCloser, error) {
	principal, err := service.workspaceAdmin(ctx, slug, false)
	if err != nil {
		closeWorkspaceLifecycleBody(body)
		return nil, nil, err
	}
	fields, err := workspaceLifecycleRequestBody(ctx, body)
	if err != nil {
		return nil, nil, service.error(http.StatusBadRequest, map[string]any{"detail": err.Error()})
	}
	result, err := service.repository.UpdateWorkspace(ctx, principal.userID, slug, fields)
	if err != nil {
		var validation *instancemanagement.WorkspaceValidationError
		if errors.As(err, &validation) {
			return nil, nil, service.error(http.StatusBadRequest, validation.Fields)
		}
		var httpError *djangoapi.HTTPError
		if errors.As(err, &httpError) {
			return nil, nil, service.error(httpError.Status, httpError.Body)
		}
		return nil, nil, service.internal()
	}
	return workspaceLifecycleRaw(result)
}

func (service *workspaceLifecycleService) PatchWorkspace(ctx context.Context, payload *api.PatchWorkspacePayload, body io.ReadCloser) (*api.WorkspaceLifecycleRawJSON, io.ReadCloser, error) {
	return service.updateWorkspace(ctx, payload.Slug, body)
}

func (service *workspaceLifecycleService) PutWorkspace(ctx context.Context, payload *api.PutWorkspacePayload, body io.ReadCloser) (*api.WorkspaceLifecycleRawJSON, io.ReadCloser, error) {
	return service.updateWorkspace(ctx, payload.Slug, body)
}

func (service *workspaceLifecycleService) DeleteWorkspace(ctx context.Context, payload *api.DeleteWorkspacePayload) error {
	principal, err := service.workspaceAdmin(ctx, payload.Slug, true)
	if err != nil {
		return err
	}
	deleted, err := service.repository.PrepareWorkspaceDeletion(ctx, principal.userID, payload.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.error(http.StatusNotFound, map[string]any{"error": "The required object does not exist."})
	}
	if err != nil {
		return service.internal()
	}
	deletedAt := time.Now().UTC()
	properties := map[string]any{
		"user_id": principal.userID.String(), "workspace_id": deleted.ID, "workspace_slug": deleted.Slug,
		"role": "owner", "workspace_name": deleted.Name, "deleted_at": deletedAt.Format(time.RFC3339Nano),
	}
	if err := service.publisher.Task(ctx, "plane.bgtasks.event_tracking_task.track_event", nil, map[string]any{
		"user_id": principal.userID.String(), "event_name": "workspace_deleted", "slug": deleted.Slug, "event_properties": properties,
	}); err != nil {
		return service.internal()
	}
	if err := service.repository.SoftDeleteWorkspace(ctx, principal.userID, deleted, deletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.error(http.StatusNotFound, map[string]any{"error": "The required object does not exist."})
		}
		return service.internal()
	}
	if err := service.publisher.SoftDelete(ctx, "db", "workspace", deleted.ID); err != nil {
		return service.internal()
	}
	return nil
}

func closeWorkspaceLifecycleBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}
