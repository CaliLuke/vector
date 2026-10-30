package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_label_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/loom/security"
)

const projectLabelReadsInternalErrorDetail = "Something went wrong please try again later"

type projectLabelReadsRepository interface {
	List(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type projectLabelReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type projectLabelReadsService struct {
	repository projectLabelReadsRepository
	sessions   projectLabelReadsSessions
}

type projectLabelReadsPrincipal struct{}

type ProjectLabelReadsResponseCapture struct{ Data json.RawMessage }

type projectLabelReadsResponseCaptureKey struct{}

func WithProjectLabelReadsResponseCapture(ctx context.Context, capture *ProjectLabelReadsResponseCapture) context.Context {
	return context.WithValue(ctx, projectLabelReadsResponseCaptureKey{}, capture)
}

func NewProjectLabelReads(
	repository projectLabelReadsRepository,
	sessions projectLabelReadsSessions,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("project label reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &projectLabelReadsService{repository: repository, sessions: sessions}, nil
}

func (service *projectLabelReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.ProjectLabelReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.ProjectLabelReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, projectLabelReadsPrincipal{}, userID), nil
}

func (service *projectLabelReadsService) List(
	ctx context.Context,
	payload *api.ListPayload,
) ([]*api.ProjectLabelRead, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.ProjectLabelReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	data, err := service.repository.List(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, projectLabelReadsRepositoryError(err)
	}
	var labels []*api.ProjectLabelRead
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, &api.ProjectLabelReadsInternalError{Message: projectLabelReadsInternalErrorDetail}
	}
	service.capture(ctx, data)
	return labels, nil
}

func (service *projectLabelReadsService) Detail(
	ctx context.Context,
	payload *api.DetailPayload,
) (*api.ProjectLabelRead, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	labelID, labelError := uuid.Parse(payload.Pk)
	if projectError != nil || labelError != nil {
		return nil, projectLabelNotFound()
	}
	data, err := service.repository.Detail(ctx, userID, payload.Slug, projectID, labelID)
	if err != nil {
		return nil, projectLabelReadsRepositoryError(err)
	}
	var label api.ProjectLabelRead
	if err := json.Unmarshal(data, &label); err != nil {
		return nil, &api.ProjectLabelReadsInternalError{Message: projectLabelReadsInternalErrorDetail}
	}
	service.capture(ctx, data)
	return &label, nil
}

func (service *projectLabelReadsService) capture(ctx context.Context, data json.RawMessage) {
	if capture, ok := ctx.Value(projectLabelReadsResponseCaptureKey{}).(*ProjectLabelReadsResponseCapture); ok && capture != nil {
		capture.Data = data
	}
}

func (service *projectLabelReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(projectLabelReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.ProjectLabelReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func projectLabelReadsRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return &api.ProjectLabelReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	if errors.Is(err, store.ErrNotFound) {
		return projectLabelNotFound()
	}
	return &api.ProjectLabelReadsInternalError{Message: projectLabelReadsInternalErrorDetail}
}

func projectLabelNotFound() error {
	return &api.ProjectLabelReadsNotFound{Detail: "No Label matches the given query."}
}
