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
	var cycles []*api.PublicProjectCollectionItem
	if decodeErr := service.decodeResponse(ctx, data, err, &cycles); decodeErr != nil {
		return nil, decodeErr
	}
	return cycles, nil
}

func (service *publicProjectMetadataReadsService) Labels(
	ctx context.Context,
	payload *api.LabelsPayload,
) ([]*api.PublicProjectLabel, error) {
	data, err := service.repository.Labels(ctx, payload.Anchor)
	var labels []*api.PublicProjectLabel
	if decodeErr := service.decodeResponse(ctx, data, err, &labels); decodeErr != nil {
		return nil, decodeErr
	}
	return labels, nil
}

func (service *publicProjectMetadataReadsService) Modules(
	ctx context.Context,
	payload *api.ModulesPayload,
) ([]*api.PublicProjectCollectionItem, error) {
	data, err := service.repository.Modules(ctx, payload.Anchor)
	var modules []*api.PublicProjectCollectionItem
	if decodeErr := service.decodeResponse(ctx, data, err, &modules); decodeErr != nil {
		return nil, decodeErr
	}
	return modules, nil
}

func (service *publicProjectMetadataReadsService) Members(
	ctx context.Context,
	payload *api.MembersPayload,
) ([]*api.PublicProjectMember, error) {
	data, err := service.repository.Members(ctx, payload.Anchor)
	var members []*api.PublicProjectMember
	if decodeErr := service.decodeResponse(ctx, data, err, &members); decodeErr != nil {
		return nil, decodeErr
	}
	return members, nil
}

func (service *publicProjectMetadataReadsService) Meta(
	ctx context.Context,
	payload *api.MetaPayload,
) (*api.PublicProjectMeta, error) {
	data, err := service.repository.Meta(ctx, payload.Anchor)
	var meta api.PublicProjectMeta
	if decodeErr := service.decodeResponse(ctx, data, err, &meta); decodeErr != nil {
		return nil, decodeErr
	}
	return &meta, nil
}

func (service *publicProjectMetadataReadsService) Settings(
	ctx context.Context,
	payload *api.SettingsPayload,
) (*api.PublicProjectSettings, error) {
	data, err := service.repository.Settings(ctx, payload.Anchor)
	var settings api.PublicProjectSettings
	if decodeErr := service.decodeResponse(ctx, data, err, &settings); decodeErr != nil {
		return nil, decodeErr
	}
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
	var anchor api.PublicProjectSettings
	if decodeErr := service.decodeResponse(ctx, data, err, &anchor); decodeErr != nil {
		return nil, decodeErr
	}
	return &anchor, nil
}

func (service *publicProjectMetadataReadsService) States(
	ctx context.Context,
	payload *api.StatesPayload,
) ([]*api.PublicProjectState, error) {
	data, err := service.repository.States(ctx, payload.Anchor)
	var states []*api.PublicProjectState
	if decodeErr := service.decodeResponse(ctx, data, err, &states); decodeErr != nil {
		return nil, decodeErr
	}
	return states, nil
}

func (service *publicProjectMetadataReadsService) decodeResponse(ctx context.Context, data json.RawMessage, readErr error, destination any) error {
	if readErr != nil {
		service.captureRenderError(ctx, readErr)
		return publicProjectMetadataReadsRepositoryError(readErr)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		service.markRenderError(ctx)
		return &api.PublicProjectMetadataInternalError{Message: publicProjectMetadataInternalErrorDetail}
	}
	service.capture(ctx, data)
	return nil
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
