package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/standard_views"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/standardviews"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type standardViewsRepository interface {
	CheckMutation(context.Context, uuid.UUID, string, *uuid.UUID, uuid.UUID, bool) error
	AuthorizedWorkspace(context.Context, uuid.UUID, string) (bool, error)
	AuthorizedProject(context.Context, uuid.UUID, string, uuid.UUID) (bool, error)
	ListProject(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	GetProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	CreateProject(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error)
	UpdateProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.ViewInput) (json.RawMessage, error)
	PatchProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.ViewInput) (json.RawMessage, error)
	DeleteProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	ListWorkspace(context.Context, uuid.UUID, string) (json.RawMessage, error)
	GetWorkspace(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	CreateWorkspace(context.Context, uuid.UUID, string, store.ViewInput) (json.RawMessage, error)
	UpdateWorkspace(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error)
	PatchWorkspace(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error)
	DeleteWorkspace(context.Context, uuid.UUID, string, uuid.UUID) error
	RecordRecentVisit(context.Context, uuid.UUID, string, *uuid.UUID, uuid.UUID) error
}

type standardViewsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type standardViewsRecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type standardViewsService struct {
	repository   standardViewsRepository
	sessions     standardViewsSessions
	recentVisits standardViewsRecentVisitAdmitter
}

type standardViewsPrincipal struct{}

// NewStandardViews constructs the project and workspace saved-view service.
func NewStandardViews(repository standardViewsRepository, sessions standardViewsSessions, recentVisits ...standardViewsRecentVisitAdmitter) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("standard views dependencies are required")
	}
	if len(recentVisits) > 1 {
		return nil, fmt.Errorf("at most one recent visit admitter is supported")
	}
	service := &standardViewsService{repository: repository, sessions: sessions}
	if len(recentVisits) == 1 {
		service.recentVisits = recentVisits[0]
	}
	return service, nil
}

// recordRecentVisit keeps this read-side effect best-effort, as it was before
// native task admission. When the durable runtime is enabled, it uses the
// shared recent-visit workflow; request-parity mode keeps the direct SQL path.
func (s *standardViewsService) recordRecentVisit(ctx context.Context, user uuid.UUID, slug string, project *uuid.UUID, id uuid.UUID) {
	var err error
	if s.recentVisits == nil {
		err = s.repository.RecordRecentVisit(ctx, user, slug, project, id)
	} else {
		projectID := ""
		if project != nil {
			projectID = project.String()
		}
		err = s.recentVisits.AdmitRecentVisit(ctx, uuid.New().String(), slug, "view", id.String(), user.String(), projectID)
	}
	if err != nil {
		slog.ErrorContext(ctx, "record standard view recent visit", "error", err)
	}
}

func (s *standardViewsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.StandardViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.StandardViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, standardViewsPrincipal{}, userID), nil
}

func (s *standardViewsService) user(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(standardViewsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.StandardViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func (s *standardViewsService) authorizeWorkspace(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, err := s.user(ctx)
	if err != nil {
		return uuid.Nil(), err
	}
	allowed, err := s.repository.AuthorizedWorkspace(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), standardViewInternalError()
	}
	if !allowed {
		return uuid.Nil(), &api.StandardViewForbidden{Message: forbiddenDetail}
	}
	return userID, nil
}

func (s *standardViewsService) authorizeProject(ctx context.Context, slug, projectID string) (uuid.UUID, uuid.UUID, error) {
	userID, err := s.user(ctx)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), err
	}
	project, err := uuid.Parse(projectID)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), standardViewNotFoundError()
	}
	allowed, err := s.repository.AuthorizedProject(ctx, userID, slug, project)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), standardViewInternalError()
	}
	if !allowed {
		return uuid.Nil(), uuid.Nil(), &api.StandardViewForbidden{Message: forbiddenDetail}
	}
	return userID, project, nil
}

func standardViewID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil(), standardViewNotFoundError()
	}
	return id, nil
}

func standardViewInput(
	name, description *string,
	filters, displayFilters, displayProperties, richFilters, logoProps loom.Nullable[loom.JSONValue],
	archivedAt loom.Nullable[string],
) (store.ViewInput, error) {
	fields := map[string]json.RawMessage{
		"filters": rawValue(filters), "display_filters": rawValue(displayFilters), "display_properties": rawValue(displayProperties), "rich_filters": rawValue(richFilters), "logo_props": rawValue(logoProps),
	}
	if invalid := normalizeJSONFields(fields, "filters", "display_filters", "display_properties", "rich_filters", "logo_props"); invalid != nil {
		return store.ViewInput{}, api.JSONFieldValidation(invalid)
	}
	return store.ViewInput{
		Name:              name,
		Description:       description,
		Filters:           fields["filters"],
		DisplayFilters:    fields["display_filters"],
		DisplayProperties: fields["display_properties"],
		RichFilters:       fields["rich_filters"],
		LogoProps:         fields["logo_props"],
		ArchivedAt:        rawValue(archivedAt),
	}, nil
}

func decodeStandardView(body json.RawMessage) (*api.StandardView, error) {
	var view api.StandardView
	if err := json.Unmarshal(body, &view); err != nil {
		return nil, standardViewInternalError()
	}
	return &view, nil
}

func decodeStandardViews(body json.RawMessage) ([]*api.StandardView, error) {
	var views []*api.StandardView
	if err := json.Unmarshal(body, &views); err != nil {
		return nil, standardViewInternalError()
	}
	return views, nil
}

func standardViewError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return standardViewNotFoundError()
	case errors.Is(err, store.ErrLocked):
		return &api.StandardViewInvalid{Message: "view is locked"}
	case errors.Is(err, store.ErrForbidden):
		return &api.StandardViewForbidden{Message: forbiddenDetail}
	case err != nil:
		return standardViewInternalError()
	default:
		return nil
	}
}

func standardViewMutationError(err error, message string) error {
	if errors.Is(err, store.ErrForbidden) {
		return &api.StandardViewInvalid{Message: message}
	}
	return standardViewError(err)
}

func standardViewNotFoundError() error {
	return &api.StandardViewNotFound{Message: "View not found"}
}

func standardViewInternalError() error {
	return &api.StandardViewInternal{Message: "Something went wrong please try again later"}
}

func (s *standardViewsService) ListProject(ctx context.Context, payload *api.ListProjectPayload) ([]*api.StandardView, error) {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.ListProject(ctx, userID, payload.Slug, project)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardViews(body)
}

func (s *standardViewsService) CreateProject(ctx context.Context, payload *api.CreateProjectPayload) (*api.StandardView, error) {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(&payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.CreateProject(ctx, userID, payload.Slug, project, input)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) RetrieveProject(ctx context.Context, payload *api.RetrieveProjectPayload) (*api.StandardView, error) {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.GetProject(ctx, userID, payload.Slug, project, id)
	if err != nil {
		return nil, standardViewError(err)
	}
	s.recordRecentVisit(ctx, userID, payload.Slug, &project, id)
	return decodeStandardView(body)
}

func (s *standardViewsService) UpdateProject(ctx context.Context, payload *api.UpdateProjectPayload) (*api.StandardView, error) {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		if precedence := s.repository.CheckMutation(ctx, userID, payload.Slug, &project, id, false); precedence != nil {
			if errors.Is(precedence, store.ErrPermission) {
				return nil, &api.StandardViewForbidden{Message: forbiddenDetail}
			}
			return nil, standardViewMutationError(precedence, "Only the owner of the view can update the view")
		}
		return nil, err
	}
	body, err := s.repository.UpdateProject(ctx, userID, payload.Slug, project, id, input)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) PatchProject(ctx context.Context, payload *api.PatchProjectPayload) (*api.StandardView, error) {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		if precedence := s.repository.CheckMutation(ctx, userID, payload.Slug, &project, id, true); precedence != nil {
			if errors.Is(precedence, store.ErrPermission) {
				return nil, &api.StandardViewForbidden{Message: forbiddenDetail}
			}
			return nil, standardViewMutationError(precedence, "Only the owner of the view can update the view")
		}
		return nil, err
	}
	body, err := s.repository.PatchProject(ctx, userID, payload.Slug, project, id, input)
	if err != nil {
		return nil, standardViewMutationError(err, "Only the owner of the view can update the view")
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) DeleteProject(ctx context.Context, payload *api.DeleteProjectPayload) error {
	userID, project, err := s.authorizeProject(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return err
	}
	return standardViewMutationError(s.repository.DeleteProject(ctx, userID, payload.Slug, project, id), "Only admin or owner can delete the view")
}

func (s *standardViewsService) ListWorkspace(ctx context.Context, payload *api.ListWorkspacePayload) ([]*api.StandardView, error) {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.ListWorkspace(ctx, userID, payload.Slug)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardViews(body)
}

func (s *standardViewsService) CreateWorkspace(ctx context.Context, payload *api.CreateWorkspacePayload) (*api.StandardView, error) {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(&payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.CreateWorkspace(ctx, userID, payload.Slug, input)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) RetrieveWorkspace(ctx context.Context, payload *api.RetrieveWorkspacePayload) (*api.StandardView, error) {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.GetWorkspace(ctx, userID, payload.Slug, id)
	if err != nil {
		return nil, standardViewError(err)
	}
	s.recordRecentVisit(ctx, userID, payload.Slug, nil, id)
	return decodeStandardView(body)
}

func (s *standardViewsService) UpdateWorkspace(ctx context.Context, payload *api.UpdateWorkspacePayload) (*api.StandardView, error) {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		if precedence := s.repository.CheckMutation(ctx, userID, payload.Slug, nil, id, false); precedence != nil {
			if errors.Is(precedence, store.ErrPermission) {
				return nil, &api.StandardViewForbidden{Message: forbiddenDetail}
			}
			return nil, standardViewMutationError(precedence, "Only the owner of the view can update the view")
		}
		return nil, err
	}
	body, err := s.repository.UpdateWorkspace(ctx, userID, payload.Slug, id, input)
	if err != nil {
		return nil, standardViewError(err)
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) PatchWorkspace(ctx context.Context, payload *api.PatchWorkspacePayload) (*api.StandardView, error) {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return nil, err
	}
	input, err := standardViewInput(payload.Name, payload.Description, payload.Filters, payload.DisplayFilters, payload.DisplayProperties, payload.RichFilters, payload.LogoProps, payload.ArchivedAt)
	if err != nil {
		if precedence := s.repository.CheckMutation(ctx, userID, payload.Slug, nil, id, true); precedence != nil {
			if errors.Is(precedence, store.ErrPermission) {
				return nil, &api.StandardViewForbidden{Message: forbiddenDetail}
			}
			return nil, standardViewMutationError(precedence, "Only the owner of the view can update the view")
		}
		return nil, err
	}
	body, err := s.repository.PatchWorkspace(ctx, userID, payload.Slug, id, input)
	if err != nil {
		return nil, standardViewMutationError(err, "Only the owner of the view can update the view")
	}
	return decodeStandardView(body)
}

func (s *standardViewsService) DeleteWorkspace(ctx context.Context, payload *api.DeleteWorkspacePayload) error {
	userID, err := s.authorizeWorkspace(ctx, payload.Slug)
	if err != nil {
		return err
	}
	id, err := standardViewID(payload.Pk)
	if err != nil {
		return err
	}
	return standardViewMutationError(s.repository.DeleteWorkspace(ctx, userID, payload.Slug, id), "Only admin or owner can delete the view")
}

var _ standardViewsRepository = (*store.Repository)(nil)
