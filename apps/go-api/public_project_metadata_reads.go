package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	store "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/security"
)

const publicProjectMetadataInternalErrorDetail = "Something went wrong please try again later"

type publicProjectMetadataReadsRepository interface {
	Cycles(context.Context, string) (json.RawMessage, error)
	Labels(context.Context, string) (json.RawMessage, error)
	Members(context.Context, string) (json.RawMessage, error)
	Meta(context.Context, string) (json.RawMessage, error)
	Modules(context.Context, string) (json.RawMessage, error)
	ProjectAnchor(context.Context, string, uuid.UUID) (json.RawMessage, error)
	Settings(context.Context, string) (json.RawMessage, error)
	States(context.Context, string) (json.RawMessage, error)
}

// APIKeyAuth satisfies Loom's optional session-cookie security scheme (the
// generated callback name is generic). Django's space BaseAPIView uses
// BaseSessionAuthentication, not APIKeyAuthentication, so an X-Api-Key header
// does not authenticate these public endpoints. Session validation and its
// anonymous-throttle exemption run in the HTTP compatibility middleware.
func (service *publicProjectMetadataReadsService) APIKeyAuth(
	ctx context.Context,
	_ string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	return ctx, nil
}

type publicProjectMetadataReadsService struct {
	repository publicProjectMetadataReadsRepository
}

type PublicProjectMetadataReadsResponseCapture struct {
	Data        json.RawMessage
	RenderError bool
}

type publicProjectMetadataReadsResponseCaptureKey struct{}

func WithPublicProjectMetadataReadsResponseCapture(ctx context.Context, capture *PublicProjectMetadataReadsResponseCapture) context.Context {
	return context.WithValue(ctx, publicProjectMetadataReadsResponseCaptureKey{}, capture)
}

func NewPublicProjectMetadataReads(repository publicProjectMetadataReadsRepository) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("public project metadata reads repository is required")
	}
	return &publicProjectMetadataReadsService{repository: repository}, nil
}

func (service *publicProjectMetadataReadsService) Cycles(
	ctx context.Context,
	payload *api.CyclesPayload,
) ([]*api.PublicProjectCollectionItem, error) {
	data, err := service.repository.Cycles(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var cycles []*api.PublicProjectCollectionItem
	if err := json.Unmarshal(data, &cycles); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return cycles, nil
}

func (service *publicProjectMetadataReadsService) Labels(
	ctx context.Context,
	payload *api.LabelsPayload,
) ([]*api.PublicProjectLabel, error) {
	data, err := service.repository.Labels(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var labels []*api.PublicProjectLabel
	if err := json.Unmarshal(data, &labels); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return labels, nil
}

func (service *publicProjectMetadataReadsService) Modules(
	ctx context.Context,
	payload *api.ModulesPayload,
) ([]*api.PublicProjectCollectionItem, error) {
	data, err := service.repository.Modules(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var modules []*api.PublicProjectCollectionItem
	if err := json.Unmarshal(data, &modules); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return modules, nil
}

func (service *publicProjectMetadataReadsService) Members(
	ctx context.Context,
	payload *api.MembersPayload,
) ([]*api.PublicProjectMember, error) {
	data, err := service.repository.Members(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var members []*api.PublicProjectMember
	if err := json.Unmarshal(data, &members); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return members, nil
}

func (service *publicProjectMetadataReadsService) Meta(
	ctx context.Context,
	payload *api.MetaPayload,
) (*api.PublicProjectMeta, error) {
	data, err := service.repository.Meta(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var meta api.PublicProjectMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return &meta, nil
}

func (service *publicProjectMetadataReadsService) Settings(
	ctx context.Context,
	payload *api.SettingsPayload,
) (*api.PublicProjectSettings, error) {
	data, err := service.repository.Settings(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var settings api.PublicProjectSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return &settings, nil
}

func (service *publicProjectMetadataReadsService) ProjectAnchor(
	ctx context.Context,
	payload *api.ProjectAnchorPayload,
) (*api.PublicProjectSettings, error) {
	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	data, err := service.repository.ProjectAnchor(ctx, payload.Slug, projectID)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var anchor api.PublicProjectSettings
	if err := json.Unmarshal(data, &anchor); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return &anchor, nil
}

func (service *publicProjectMetadataReadsService) States(
	ctx context.Context,
	payload *api.StatesPayload,
) ([]*api.PublicProjectState, error) {
	data, err := service.repository.States(ctx, payload.Anchor)
	if err != nil {
		service.captureRenderError(ctx, err)
		return nil, publicProjectMetadataReadsRepositoryError(err)
	}
	var states []*api.PublicProjectState
	if err := json.Unmarshal(data, &states); err != nil {
		service.markRenderError(ctx)
		return nil, &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return states, nil
}

func (service *publicProjectMetadataReadsService) captureRenderError(ctx context.Context, err error) {
	if errors.Is(err, store.ErrRender) {
		service.markRenderError(ctx)
	}
}

func (service *publicProjectMetadataReadsService) markRenderError(ctx context.Context) {
	if capture, ok := ctx.Value(publicProjectMetadataReadsResponseCaptureKey{}).(*PublicProjectMetadataReadsResponseCapture); ok && capture != nil {
		capture.RenderError = true
	}
}

func (service *publicProjectMetadataReadsService) capture(ctx context.Context, data json.RawMessage) {
	if capture, ok := ctx.Value(publicProjectMetadataReadsResponseCaptureKey{}).(*PublicProjectMetadataReadsResponseCapture); ok && capture != nil {
		capture.Data = data
	}
}

func publicProjectMetadataReadsRepositoryError(err error) error {
	if errors.Is(err, store.ErrRequiredObject) {
		return &api.PublicProjectMetadataNotFound{Message: "The required object does not exist."}
	}
	if errors.Is(err, store.ErrProjectNotPublished) {
		return &api.PublicProjectMetadataNotFound{Message: "Project is not published"}
	}
	if errors.Is(err, store.ErrInvalidAnchor) {
		return &api.PublicProjectMetadataNotFound{Message: "Invalid anchor"}
	}
	return &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
}
