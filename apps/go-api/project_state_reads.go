package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_state_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const projectStateReadsInternalErrorDetail = "Something went wrong please try again later"

type projectStateReadsRepository interface {
	AuthorizeProject(context.Context, uuid.UUID, string, uuid.UUID) error
	List(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Intake(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
}

type projectStateReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type projectStateReadsService struct {
	repository projectStateReadsRepository
	sessions   projectStateReadsSessions
}

type projectStateReadsPrincipal struct{}

type ProjectStateReadsResponseCapture struct{ Data json.RawMessage }

type ProjectStateReadsMetadataAuthorizer interface {
	AuthorizeMetadata(context.Context, string) error
}

type projectStateReadsResponseCaptureKey struct{}

func WithProjectStateReadsResponseCapture(ctx context.Context, capture *ProjectStateReadsResponseCapture) context.Context {
	return context.WithValue(ctx, projectStateReadsResponseCaptureKey{}, capture)
}

func NewProjectStateReads(
	repository projectStateReadsRepository,
	sessions projectStateReadsSessions,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("project state reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &projectStateReadsService{repository: repository, sessions: sessions}, nil
}

func (service *projectStateReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principalID, _, validated := readServiceSessionPrincipal(ctx)
	if !validated {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, projectStateAuthenticationError(ctx)
		}
		principalID = principal.UserID
	}
	userID, err := uuid.Parse(principalID)
	if err != nil {
		return ctx, projectStateAuthenticationError(ctx)
	}
	return context.WithValue(ctx, projectStateReadsPrincipal{}, userID), nil
}

func projectStateAuthenticationError(ctx context.Context) error {
	switch ctx.Value(loom.MethodKey) {
	case "create", "update", "delete", "mark_default":
		return projectStateAppFailure(401, map[string]string{"detail": unauthenticatedDetail})
	default:
		return &api.ProjectStateReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
}

func (service *projectStateReadsService) AuthorizeMetadata(
	ctx context.Context,
	sessionID string,
) error {
	_, err := service.APIKeyAuth(ctx, sessionID, &security.APIKeyScheme{Name: "plane_session_cookie"})
	return err
}

func (service *projectStateReadsService) List(
	ctx context.Context,
	payload *api.ListPayload,
) (loom.JSONValue, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectStateReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	data, err := service.repository.List(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, projectStateReadsRepositoryError(err, false)
	}
	var states []*api.ProjectStateRead
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, &api.ProjectStateReadsInternalError{Message: projectStateReadsInternalErrorDetail}
	}
	if payload.Grouped != nil && *payload.Grouped == "true" {
		grouped := make(map[string][]*api.ProjectStateRead)
		for _, state := range states {
			grouped[state.Group] = append(grouped[state.Group], state)
		}
		encoded, err := marshalGroupedProjectStates(grouped)
		if err != nil {
			return nil, &api.ProjectStateReadsInternalError{Message: projectStateReadsInternalErrorDetail}
		}
		service.capture(ctx, encoded)
		return loom.JSONValue(encoded), nil
	}
	service.capture(ctx, data)
	return loom.JSONValue(data), nil
}

func (service *projectStateReadsService) Detail(
	ctx context.Context,
	payload *api.DetailPayload,
) (*api.ProjectStateRead, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	stateID, stateError := uuid.Parse(payload.Pk)
	if projectError != nil || stateError != nil {
		return nil, projectStateDetailNotFound()
	}
	data, err := service.repository.Detail(ctx, userID, payload.Slug, projectID, stateID)
	if err != nil {
		return nil, projectStateReadsRepositoryError(err, true)
	}
	return service.decodeState(ctx, data)
}

func (service *projectStateReadsService) Intake(
	ctx context.Context,
	payload *api.IntakePayload,
) (*api.ProjectStateRead, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectStateReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	data, err := service.repository.Intake(ctx, userID, payload.Slug, projectID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, &api.ProjectStateReadsNotFound{Message: "Triage state not found"}
	}
	if err != nil {
		return nil, projectStateReadsRepositoryError(err, false)
	}
	return service.decodeState(ctx, data)
}

func (service *projectStateReadsService) decodeState(
	ctx context.Context,
	data json.RawMessage,
) (*api.ProjectStateRead, error) {
	var state api.ProjectStateRead
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, &api.ProjectStateReadsInternalError{Message: projectStateReadsInternalErrorDetail}
	}
	service.capture(ctx, data)
	return &state, nil
}

func (service *projectStateReadsService) capture(ctx context.Context, data json.RawMessage) {
	if capture, ok := ctx.Value(projectStateReadsResponseCaptureKey{}).(*ProjectStateReadsResponseCapture); ok && capture != nil {
		capture.Data = data
	}
}

func (service *projectStateReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(projectStateReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.ProjectStateReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func projectStateReadsRepositoryError(err error, detail bool) error {
	if detail && (errors.Is(err, store.ErrForbidden) || errors.Is(err, store.ErrNotFound)) {
		return projectStateDetailNotFound()
	}
	if errors.Is(err, store.ErrForbidden) {
		return &api.ProjectStateReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	return &api.ProjectStateReadsInternalError{Message: projectStateReadsInternalErrorDetail}
}

func projectStateDetailNotFound() error {
	return &api.ProjectStateReadsDetailNotFound{Detail: "No State matches the given query."}
}

func marshalGroupedProjectStates(grouped map[string][]*api.ProjectStateRead) (json.RawMessage, error) {
	// encoding/json emits map keys in lexical order, matching Django's sorted grouping.
	return json.Marshal(grouped)
}
