package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	api "github.com/CaliLuke/go-plane/gen/estimate_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/estimatereads"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const estimateReadsInternalErrorDetail = "Something went wrong please try again later"

type estimateReadsRepository interface {
	AuthorizeWorkspace(context.Context, uuid.UUID, string) error
	WorkspaceSelected(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ProjectList(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ProjectDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type estimateReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type estimateReadsService struct {
	repository estimateReadsRepository
	sessions   estimateReadsSessions
}

type estimateReadsPrincipal struct{}

func NewEstimateReads(repository estimateReadsRepository, sessions estimateReadsSessions) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("estimate reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &estimateReadsService{repository: repository, sessions: sessions}, nil
}

func (service *estimateReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principalID, _, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, estimateReadsUnauthenticated()
		}
		principalID = principal.UserID
	}
	userID, err := uuid.Parse(principalID)
	if err != nil {
		return ctx, estimateReadsUnauthenticated()
	}
	return context.WithValue(ctx, estimateReadsPrincipal{}, userID), nil
}

func (service *estimateReadsService) WorkspaceSelected(ctx context.Context,
	payload *api.WorkspaceSelectedPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.WorkspaceSelected(ctx, userID, payload.Slug)
	if err != nil {
		return nil, nil, estimateReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *estimateReadsService) WorkspaceSelectedOptions(ctx context.Context,
	payload *api.WorkspaceSelectedOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := service.repository.AuthorizeWorkspace(ctx, userID, payload.Slug); err != nil {
		return nil, nil, estimateReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser([]byte(`{"name":"Workspace Estimates Endpoint","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`)), nil
}

func (service *estimateReadsService) ProjectList(ctx context.Context,
	payload *api.ProjectListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, nil, estimateReadsForbidden()
	}
	data, err := service.repository.ProjectList(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, estimateReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *estimateReadsService) ProjectDetail(ctx context.Context,
	payload *api.ProjectDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	estimateID, estimateError := uuid.Parse(payload.EstimateID)
	if projectError != nil || estimateError != nil {
		return nil, nil, estimateReadsNotFound()
	}
	data, err := service.repository.ProjectDetail(ctx, userID, payload.Slug, projectID, estimateID)
	if err != nil {
		return nil, nil, estimateReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *estimateReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(estimateReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), estimateReadsUnauthenticated()
	}
	return userID, nil
}

func estimateReadsRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return estimateReadsForbidden()
	}
	if errors.Is(err, store.ErrNotFound) {
		return estimateReadsNotFound()
	}
	return estimateReadsInternalError()
}

func estimateReadsUnauthenticated() error {
	return &api.EstimateReadsUnauthenticated{Detail: unauthenticatedDetail}
}

func estimateReadsForbidden() error {
	return &api.EstimateReadsForbidden{Detail: "You do not have permission to perform this action."}
}

func estimateReadsNotFound() error {
	return &api.EstimateReadsNotFound{Message: "The required object does not exist."}
}

func estimateReadsInternalError() error {
	return &api.EstimateReadsInternalError{Message: estimateReadsInternalErrorDetail}
}
