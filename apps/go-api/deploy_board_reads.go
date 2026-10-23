package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/deployboardreads"
	"github.com/CaliLuke/loom/security"
)

const deployBoardReadsInternalErrorDetail = "Something went wrong please try again later"

type deployBoardReadsRepository interface {
	Collection(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type deployBoardReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type deployBoardReadsService struct {
	repository deployBoardReadsRepository
	sessions   deployBoardReadsSessions
}

type deployBoardReadsPrincipal struct{}

func NewDeployBoardReads(
	repository deployBoardReadsRepository,
	sessions deployBoardReadsSessions,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("deploy board reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &deployBoardReadsService{repository: repository, sessions: sessions}, nil
}

func (service *deployBoardReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principalID, _, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, &api.DeployBoardReadsUnauthenticated{Detail: unauthenticatedDetail}
		}
		principalID = principal.UserID
	}
	userID, err := uuid.Parse(principalID)
	if err != nil {
		return ctx, &api.DeployBoardReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, deployBoardReadsPrincipal{}, userID), nil
}

func (service *deployBoardReadsService) Collection(ctx context.Context,
	payload *api.CollectionPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, nil, deployBoardForbidden()
	}
	data, err := service.repository.Collection(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, deployBoardReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *deployBoardReadsService) Detail(ctx context.Context,
	payload *api.DetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	boardID, boardError := uuid.Parse(payload.Pk)
	if projectError != nil || boardError != nil {
		return nil, nil, deployBoardNotFound()
	}
	data, err := service.repository.Detail(ctx, userID, payload.Slug, projectID, boardID)
	if err != nil {
		return nil, nil, deployBoardReadsRepositoryError(err)
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *deployBoardReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(deployBoardReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.DeployBoardReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func deployBoardReadsRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return deployBoardForbidden()
	}
	if errors.Is(err, store.ErrNotFound) {
		return deployBoardNotFound()
	}
	return deployBoardInternalError()
}

func deployBoardForbidden() error {
	return &api.DeployBoardReadsForbidden{Detail: "You do not have permission to perform this action."}
}

func deployBoardNotFound() error {
	return &api.DeployBoardReadsNotFound{Detail: "No DeployBoard matches the given query."}
}

func deployBoardInternalError() error {
	return &api.DeployBoardReadsInternalError{Message: deployBoardReadsInternalErrorDetail}
}
