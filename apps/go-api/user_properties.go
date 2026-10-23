package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_properties"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/userproperties"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type userPropertiesService struct {
	repo     userPropertiesRepository
	sessions interface {
		Validate(context.Context, string) (djangosession.Principal, error)
	}
}

type userPropertiesRepository interface {
	CheckScopedPatch(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, string) error
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	AuthorizedProject(context.Context, uuid.UUID, string, uuid.UUID) (bool, error)
	GetWorkspace(context.Context, uuid.UUID, string) (json.RawMessage, error)
	PatchWorkspace(context.Context, uuid.UUID, string, map[string]json.RawMessage) (json.RawMessage, error)
	GetProject(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	PatchProject(context.Context, uuid.UUID, string, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
	GetCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	PatchCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
	GetModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	PatchModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
}

func NewUserProperties(r userPropertiesRepository, s interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("user properties dependencies are required")
	}
	return &userPropertiesService{r, s}, nil
}

var _ userPropertiesRepository = (*store.Repository)(nil)

type userPropertiesPrincipal struct{}

func (s *userPropertiesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.UserPropertiesUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.UserPropertiesUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userPropertiesPrincipal{}, id), nil
}
func (s *userPropertiesService) auth(ctx context.Context, slug string, project *uuid.UUID) (uuid.UUID, error) {
	id, ok := ctx.Value(userPropertiesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.UserPropertiesUnauthenticated{Detail: unauthenticatedDetail}
	}
	var yes bool
	var e error
	if project == nil {
		yes, e = s.repo.Authorized(ctx, id, slug)
	} else {
		yes, e = s.repo.AuthorizedProject(ctx, id, slug, *project)
	}
	if e != nil {
		return uuid.Nil(), userPropertiesInternalError()
	}
	if !yes {
		return uuid.Nil(), &api.UserPropertiesForbidden{Message: forbiddenDetail}
	}
	return id, nil
}

func (s *userPropertiesService) projectIdentity(ctx context.Context, slug, projectID string) (uuid.UUID, uuid.UUID, error) {
	project, err := uuid.Parse(projectID)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), userPropertiesNotFoundError()
	}
	user, err := s.auth(ctx, slug, &project)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), err
	}
	return user, project, nil
}

func (s *userPropertiesService) scopedIdentity(ctx context.Context, slug, projectID, entityID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	user, project, err := s.projectIdentity(ctx, slug, projectID)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), err
	}
	entity, err := uuid.Parse(entityID)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), userPropertiesNotFoundError()
	}
	return user, project, entity, nil
}

func addUserPropertyNullable[T any](m map[string]json.RawMessage, key string, value loom.Nullable[T]) {
	if raw := rawValue(value); raw != nil {
		m[key] = raw
	}
}
func rawValue[T any](n loom.Nullable[T]) json.RawMessage {
	if !n.Present() {
		return nil
	}
	if n.IsNull() {
		return json.RawMessage("null")
	}
	v, ok := n.Value()
	if !ok {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}
func userPropertyFilterFields(filters, displayFilters, displayProperties, richFilters loom.Nullable[loom.JSONValue]) map[string]json.RawMessage {
	fields := make(map[string]json.RawMessage)
	addUserPropertyNullable(fields, "filters", filters)
	addUserPropertyNullable(fields, "display_filters", displayFilters)
	addUserPropertyNullable(fields, "display_properties", displayProperties)
	addUserPropertyNullable(fields, "rich_filters", richFilters)
	return fields
}

func userPropertiesResult(data json.RawMessage, err error) (*api.UserProperties, error) {
	if err != nil {
		return nil, userPropertiesError(err)
	}
	var properties api.UserProperties
	if err := json.Unmarshal(data, &properties); err != nil {
		return nil, userPropertiesInternalError()
	}
	return &properties, nil
}

func (s *userPropertiesService) WorkspaceGet(ctx context.Context, p *api.WorkspaceGetPayload) (*api.UserProperties, error) {
	id, e := s.auth(ctx, p.Slug, nil)
	if e != nil {
		return nil, e
	}
	return userPropertiesResult(s.repo.GetWorkspace(ctx, id, p.Slug))
}
func (s *userPropertiesService) WorkspacePatch(ctx context.Context, p *api.WorkspacePatchPayload) (*api.UserProperties, error) {
	id, e := s.auth(ctx, p.Slug, nil)
	if e != nil {
		return nil, e
	}
	fields := userPropertyFilterFields(p.Filters, p.DisplayFilters, p.DisplayProperties, p.RichFilters)
	addUserPropertyNullable(fields, "navigation_project_limit", p.NavigationProjectLimit)
	addUserPropertyNullable(fields, "navigation_control_preference", p.NavigationControlPreference)
	if invalid := normalizeJSONFields(fields, "filters", "display_filters", "display_properties", "rich_filters", "preferences"); invalid != nil {
		// Django creates the default property row before serializer validation.
		if _, lookupErr := s.repo.GetWorkspace(ctx, id, p.Slug); lookupErr != nil {
			return nil, userPropertiesError(lookupErr)
		}
		return nil, api.JSONFieldValidation(invalid)
	}
	return userPropertiesResult(s.repo.PatchWorkspace(ctx, id, p.Slug, fields))
}
func (s *userPropertiesService) ProjectGet(ctx context.Context, p *api.ProjectGetPayload) (*api.UserProperties, error) {
	id, x, e := s.projectIdentity(ctx, p.Slug, p.ProjectID)
	if e != nil {
		return nil, e
	}
	return userPropertiesResult(s.repo.GetProject(ctx, id, p.Slug, x))
}
func (s *userPropertiesService) ProjectPatch(ctx context.Context, p *api.ProjectPatchPayload) (*api.UserProperties, error) {
	id, x, e := s.projectIdentity(ctx, p.Slug, p.ProjectID)
	if e != nil {
		return nil, e
	}
	fields := userPropertyFilterFields(p.Filters, p.DisplayFilters, p.DisplayProperties, p.RichFilters)
	addUserPropertyNullable(fields, "preferences", p.Preferences)
	addUserPropertyNullable(fields, "sort_order", p.SortOrder)
	if invalid := normalizeJSONFields(fields, "filters", "display_filters", "display_properties", "rich_filters", "preferences"); invalid != nil {
		// Django creates the default property row before serializer validation.
		if _, lookupErr := s.repo.GetProject(ctx, id, p.Slug, x); lookupErr != nil {
			return nil, userPropertiesError(lookupErr)
		}
		return nil, api.JSONFieldValidation(invalid)
	}
	return userPropertiesResult(s.repo.PatchProject(ctx, id, p.Slug, x, fields))
}
func (s *userPropertiesService) CycleGet(ctx context.Context, p *api.CycleGetPayload) (*api.UserProperties, error) {
	id, x, y, e := s.scopedIdentity(ctx, p.Slug, p.ProjectID, p.CycleID)
	if e != nil {
		return nil, e
	}
	return userPropertiesResult(s.repo.GetCycle(ctx, id, p.Slug, x, y))
}
func (s *userPropertiesService) CyclePatch(ctx context.Context, p *api.CyclePatchPayload) (*api.UserProperties, error) {
	id, x, y, e := s.scopedIdentity(ctx, p.Slug, p.ProjectID, p.CycleID)
	if e != nil {
		return nil, e
	}
	fields := userPropertyFilterFields(p.Filters, p.DisplayFilters, p.DisplayProperties, p.RichFilters)
	if invalid := normalizeJSONFields(fields, "filters", "display_filters", "display_properties", "rich_filters", "preferences"); invalid != nil {
		if lookupErr := s.repo.CheckScopedPatch(ctx, id, p.Slug, x, y, "cycle"); lookupErr != nil {
			return nil, userPropertiesError(lookupErr)
		}
		return nil, userPropertiesInternalError()
	}
	return userPropertiesResult(s.repo.PatchCycle(ctx, id, p.Slug, x, y, fields))
}
func (s *userPropertiesService) ModuleGet(ctx context.Context, p *api.ModuleGetPayload) (*api.UserProperties, error) {
	id, x, y, e := s.scopedIdentity(ctx, p.Slug, p.ProjectID, p.ModuleID)
	if e != nil {
		return nil, e
	}
	return userPropertiesResult(s.repo.GetModule(ctx, id, p.Slug, x, y))
}
func (s *userPropertiesService) ModulePatch(ctx context.Context, p *api.ModulePatchPayload) (*api.UserProperties, error) {
	id, x, y, e := s.scopedIdentity(ctx, p.Slug, p.ProjectID, p.ModuleID)
	if e != nil {
		return nil, e
	}
	fields := userPropertyFilterFields(p.Filters, p.DisplayFilters, p.DisplayProperties, p.RichFilters)
	if invalid := normalizeJSONFields(fields, "filters", "display_filters", "display_properties", "rich_filters", "preferences"); invalid != nil {
		if lookupErr := s.repo.CheckScopedPatch(ctx, id, p.Slug, x, y, "module"); lookupErr != nil {
			return nil, userPropertiesError(lookupErr)
		}
		return nil, userPropertiesInternalError()
	}
	return userPropertiesResult(s.repo.PatchModule(ctx, id, p.Slug, x, y, fields))
}

func userPropertiesNotFoundError() error {
	return &api.UserPropertiesNotFound{Message: "Not found."}
}

func userPropertiesInternalError() error {
	return &api.UserPropertiesInternal{Message: "Something went wrong please try again later"}
}

func userPropertiesError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return userPropertiesNotFoundError()
	}
	return userPropertiesInternalError()
}
