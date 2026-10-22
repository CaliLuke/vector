package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	api "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	store "github.com/CaliLuke/go-plane/internal/workspacethemes"
	"github.com/CaliLuke/loom/security"
	"strings"
	"uuid"
)

type workspaceThemesRepo interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, uuid.UUID, string, string, json.RawMessage) (json.RawMessage, error)
	Update(context.Context, uuid.UUID, string, uuid.UUID, *string, json.RawMessage) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, string, uuid.UUID) error
}
type workspaceThemesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceThemesService struct {
	repo     workspaceThemesRepo
	sessions workspaceThemesSessions
}

func NewWorkspaceThemes(r workspaceThemesRepo, s workspaceThemesSessions) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("workspace themes dependencies are required")
	}
	return &workspaceThemesService{r, s}, nil
}
func (s *workspaceThemesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.WorkspaceThemeUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.WorkspaceThemeUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceThemesPrincipal{}, id), nil
}
func (s *workspaceThemesService) auth(ctx context.Context, slug string) (uuid.UUID, error) {
	id, ok := ctx.Value(workspaceThemesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceThemeUnauthenticated{Detail: unauthenticatedDetail}
	}
	yes, e := s.repo.Authorized(ctx, id, slug)
	if e != nil {
		return uuid.Nil(), &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	if !yes {
		return uuid.Nil(), &api.WorkspaceThemeForbidden{Detail: "You do not have the required permissions."}
	}
	return id, nil
}
func (s *workspaceThemesService) List(ctx context.Context, p *api.ListPayload) ([]*api.WorkspaceTheme, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	b, e := s.repo.List(ctx, id, p.Slug)
	if e != nil {
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	var out []*api.WorkspaceTheme
	if json.Unmarshal(b, &out) != nil {
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	return out, nil
}
func (s *workspaceThemesService) Create(ctx context.Context, p *api.CreatePayload) (*api.WorkspaceTheme, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	if p.Name == nil {
		return nil, &api.WorkspaceThemeInvalid{Name: []string{"This field is required."}}
	}
	if strings.TrimSpace(*p.Name) == "" {
		return nil, &api.WorkspaceThemeInvalid{Name: []string{"This field may not be blank."}}
	}
	colors := json.RawMessage("{}")
	if p.Colors.Kind() == 'n' {
		return nil, &api.WorkspaceThemeInvalid{Colors: []string{"This field may not be null."}}
	}
	if len(p.Colors) > 0 {
		colors, e = pythoncompat.NormalizeJSONNumbers(p.Colors)
		if e != nil {
			return nil, &api.WorkspaceThemeInvalid{Colors: []string{"Value must be valid JSON."}}
		}
	}
	b, e := s.repo.Create(ctx, id, p.Slug, *p.Name, colors)
	if e != nil {
		return nil, &api.WorkspaceThemeInvalid{Name: []string{"workspace theme with this workspace and name already exists."}}
	}
	var out api.WorkspaceTheme
	if e = json.Unmarshal(b, &out); e != nil {
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	return &out, nil
}
func (s *workspaceThemesService) Retrieve(ctx context.Context, p *api.RetrievePayload) (*api.WorkspaceTheme, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	tid, e := uuid.Parse(p.Pk)
	if e != nil {
		return nil, &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
	}
	b, e := s.repo.Get(ctx, id, p.Slug, tid)
	if e != nil {
		return nil, &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
	}
	var out api.WorkspaceTheme
	if e = json.Unmarshal(b, &out); e != nil {
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	return &out, nil
}
func (s *workspaceThemesService) Patch(ctx context.Context, p *api.PatchPayload) (*api.WorkspaceTheme, error) {
	id, err := s.auth(ctx, p.Slug)
	if err != nil {
		return nil, err
	}
	themeID, err := uuid.Parse(p.Pk)
	if err != nil {
		return nil, &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
	}
	name, nameOK := p.Name.Value()
	colors, colorsOK := p.Colors.Value()
	if p.Name.Present() && p.Name.IsNull() {
		return nil, &api.WorkspaceThemeInvalid{Name: []string{"This field may not be null."}}
	}
	if nameOK && strings.TrimSpace(name) == "" {
		return nil, &api.WorkspaceThemeInvalid{Name: []string{"This field may not be blank."}}
	}
	if p.Colors.Present() && p.Colors.IsNull() {
		return nil, &api.WorkspaceThemeInvalid{Colors: []string{"This field may not be null."}}
	}
	var colorsJSON json.RawMessage
	if colorsOK {
		colorsJSON, err = pythoncompat.NormalizeJSONNumbers(colors)
		if err != nil {
			if _, lookupErr := s.repo.Get(ctx, id, p.Slug, themeID); lookupErr != nil {
				if errors.Is(lookupErr, store.ErrNotFound) {
					return nil, &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
				}
				return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
			}
			return nil, &api.WorkspaceThemeInvalid{Colors: []string{"Value must be valid JSON."}}
		}
	}
	var namePtr *string
	if nameOK {
		namePtr = &name
	}
	b, err := s.repo.Update(ctx, id, p.Slug, themeID, namePtr, colorsJSON)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
		}
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	var out api.WorkspaceTheme
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, &api.WorkspaceThemeInternal{Message: "Something went wrong please try again later"}
	}
	return &out, nil
}
func (s *workspaceThemesService) Delete(ctx context.Context, p *api.DeletePayload) error {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return e
	}
	tid, e := uuid.Parse(p.Pk)
	if e != nil {
		return &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
	}
	if e = s.repo.Delete(ctx, id, p.Slug, tid); e != nil {
		return &api.WorkspaceThemeNotFound{Message: "The required object does not exist."}
	}
	return nil
}

type workspaceThemesPrincipal struct{}

var _ = store.ErrNotFound
var _ workspaceThemesRepo = (*store.Repository)(nil)
