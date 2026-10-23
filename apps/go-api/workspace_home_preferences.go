package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacehomepreferences"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type workspaceHomePreferencesRepo interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, string, string) (json.RawMessage, error)
	Update(context.Context, uuid.UUID, string, string, *string, *bool, *float64) (json.RawMessage, error)
}
type workspaceHomePreferencesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceHomePreferencesService struct {
	repo     workspaceHomePreferencesRepo
	sessions workspaceHomePreferencesSessions
}

func NewWorkspaceHomePreferences(r workspaceHomePreferencesRepo, s workspaceHomePreferencesSessions) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("workspace home preferences dependencies are required")
	}
	return &workspaceHomePreferencesService{r, s}, nil
}
func (s *workspaceHomePreferencesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.WorkspaceHomePreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.WorkspaceHomePreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, homePreferencesPrincipal{}, id), nil
}
func (s *workspaceHomePreferencesService) auth(ctx context.Context, slug string) (uuid.UUID, error) {
	id, ok := ctx.Value(homePreferencesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceHomePreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	yes, e := s.repo.Authorized(ctx, id, slug)
	if e != nil {
		return uuid.Nil(), homePreferenceInternal()
	}
	if !yes {
		return uuid.Nil(), &api.WorkspaceHomePreferenceForbidden{Detail: "You do not have the required permissions."}
	}
	return id, nil
}
func (s *workspaceHomePreferencesService) List(ctx context.Context, p *api.ListPayload) ([]*api.WorkspaceHomePreferenceList, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	b, e := s.repo.List(ctx, id, p.Slug)
	if e != nil {
		return nil, homePreferenceInternal()
	}
	var out []*api.WorkspaceHomePreferenceList
	if json.Unmarshal(b, &out) != nil {
		return nil, homePreferenceInternal()
	}
	return out, nil
}
func (s *workspaceHomePreferencesService) PatchCollection(ctx context.Context, p *api.PatchCollectionPayload) ([]*api.WorkspaceHomePreferenceList, error) {
	if _, err := s.auth(ctx, p.Slug); err != nil {
		return nil, err
	}
	return nil, homePreferenceInternal()
}
func (s *workspaceHomePreferencesService) Retrieve(ctx context.Context, p *api.RetrievePayload) (*api.WorkspaceHomePreference, error) {
	if _, err := s.auth(ctx, p.Slug); err != nil {
		return nil, err
	}
	return nil, homePreferenceInternal()
}
func (s *workspaceHomePreferencesService) Patch(ctx context.Context, p *api.PatchPayload) (*api.WorkspaceHomePreference, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	newKey, enabled, sort := (*string)(nil), (*bool)(nil), (*float64)(nil)
	if p.Body != nil && p.Body.Key.Present() {
		v, ok := p.Body.Key.Value()
		if !ok || strings.TrimSpace(v) == "" {
			return nil, &api.WorkspaceHomePreferenceInvalid{Key: []string{"This field may not be blank."}}
		}
		newKey = &v
	}
	if p.Body != nil && p.Body.IsEnabled.Present() {
		v, ok := p.Body.IsEnabled.Value()
		if !ok {
			return nil, &api.WorkspaceHomePreferenceInvalid{IsEnabled: []string{"This field may not be null."}}
		}
		enabled = &v
	}
	if p.Body != nil && p.Body.SortOrder.Present() {
		v, ok := p.Body.SortOrder.Value()
		if !ok {
			return nil, &api.WorkspaceHomePreferenceInvalid{SortOrder: []string{"This field may not be null."}}
		}
		sort = &v
	}
	b, e := s.repo.Update(ctx, id, p.Slug, p.Key, newKey, enabled, sort)
	if errors.Is(e, store.ErrNotFound) {
		return nil, &api.WorkspaceHomePreferenceNotFound{Message: "Preference not found"}
	}
	if e != nil {
		return nil, homePreferenceInternal()
	}
	var out api.WorkspaceHomePreference
	if json.Unmarshal(b, &out) != nil {
		return nil, homePreferenceInternal()
	}
	return &out, nil
}
func homePreferenceInternal() error {
	return &api.WorkspaceHomePreferenceInternal{Message: "Something went wrong please try again later"}
}

type homePreferencesPrincipal struct{}

var _ workspaceHomePreferencesRepo = (*store.Repository)(nil)
var _ = loom.NullableValue[int]
