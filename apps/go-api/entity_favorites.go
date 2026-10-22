package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	api "github.com/CaliLuke/go-plane/gen/entity_favorites"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/entityfavorites"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type entityFavoritesRepository interface {
	Authorized(context.Context, uuid.UUID, string, *uuid.UUID) (bool, error)
	ListProjects(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ListCycles(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ListModules(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ListViews(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	CreateProject(context.Context, uuid.UUID, string, uuid.UUID) error
	CreateCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	CreateModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	CreateView(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	CreatePage(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	DeleteProject(context.Context, uuid.UUID, string, uuid.UUID) error
	DeleteCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	DeleteModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	DeleteView(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	DeletePage(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error
}
type entityFavoritesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type entityFavoritesPrincipal struct{}
type entityFavoritesService struct {
	repository entityFavoritesRepository
	sessions   entityFavoritesSessions
}

func NewEntityFavorites(repository entityFavoritesRepository, sessions entityFavoritesSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("entity favorites dependencies are required")
	}
	return &entityFavoritesService{repository: repository, sessions: sessions}, nil
}
func (s *entityFavoritesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.EntityFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, err := uuid.Parse(p.UserID)
	if err != nil {
		return ctx, &api.EntityFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, entityFavoritesPrincipal{}, id), nil
}
func (s *entityFavoritesService) auth(ctx context.Context, slug string, project *string) (uuid.UUID, error) {
	id, ok := ctx.Value(entityFavoritesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.EntityFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	var pid *uuid.UUID
	if project != nil {
		parsed, err := uuid.Parse(*project)
		if err != nil {
			return uuid.Nil(), &api.EntityFavoriteNotFound{Message: "Not found."}
		}
		pid = &parsed
	}
	allowed, err := s.repository.Authorized(ctx, id, slug, pid)
	if err != nil {
		return uuid.Nil(), &api.EntityFavoriteInternal{Message: "Internal server error."}
	}
	if !allowed {
		return uuid.Nil(), &api.EntityFavoriteNotFound{Message: "Not found."}
	}
	return id, nil
}
func parseID(v *string) (uuid.UUID, error) {
	if v == nil {
		return uuid.Nil(), errors.New("missing id")
	}
	return uuid.Parse(*v)
}
func decodeFavorites(body json.RawMessage) ([]*api.EntityFavorite, error) {
	var out []*api.EntityFavorite
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
func (s *entityFavoritesService) list(ctx context.Context, slug string, project *string, fn func(uuid.UUID) (json.RawMessage, error)) ([]*api.EntityFavorite, error) {
	id, err := s.auth(ctx, slug, project)
	if err != nil {
		return nil, err
	}
	b, err := fn(id)
	if err != nil {
		return nil, &api.EntityFavoriteInternal{Message: "Internal server error."}
	}
	return decodeFavorites(b)
}
func (s *entityFavoritesService) write(ctx context.Context, slug string, project, entity *string, fn func(uuid.UUID, uuid.UUID) error) error {
	id, err := s.auth(ctx, slug, project)
	if err != nil {
		return err
	}
	eid, err := parseID(entity)
	if err != nil {
		return &api.EntityFavoriteNotFound{Message: "Not found."}
	}
	if err := fn(id, eid); errors.Is(err, store.ErrNotFound) {
		return &api.EntityFavoriteNotFound{Message: "Not found."}
	} else if err != nil {
		return &api.EntityFavoriteInternal{Message: "Internal server error."}
	}
	return nil
}
func (s *entityFavoritesService) ProjectList(c context.Context, p *api.ProjectListPayload) ([]*api.EntityFavorite, error) {
	return s.list(c, p.Slug, nil, func(u uuid.UUID) (json.RawMessage, error) { return s.repository.ListProjects(c, u, p.Slug) })
}
func (s *entityFavoritesService) CycleList(c context.Context, p *api.CycleListPayload) ([]*api.EntityFavorite, error) {
	return s.list(c, p.Slug, p.ProjectID, func(u uuid.UUID) (json.RawMessage, error) {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.ListCycles(c, u, p.Slug, id)
	})
}
func (s *entityFavoritesService) ModuleList(c context.Context, p *api.ModuleListPayload) ([]*api.EntityFavorite, error) {
	return s.list(c, p.Slug, p.ProjectID, func(u uuid.UUID) (json.RawMessage, error) {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.ListModules(c, u, p.Slug, id)
	})
}
func (s *entityFavoritesService) ViewList(c context.Context, p *api.ViewListPayload) ([]*api.EntityFavorite, error) {
	return s.list(c, p.Slug, p.ProjectID, func(u uuid.UUID) (json.RawMessage, error) {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.ListViews(c, u, p.Slug, id)
	})
}
func (s *entityFavoritesService) ProjectCreate(c context.Context, p *api.ProjectCreatePayload) error {
	return s.write(c, p.Slug, nil, p.Project, func(u, e uuid.UUID) error { return s.repository.CreateProject(c, u, p.Slug, e) })
}
func (s *entityFavoritesService) CycleCreate(c context.Context, p *api.CycleCreatePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.Cycle, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.CreateCycle(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) ModuleCreate(c context.Context, p *api.ModuleCreatePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.Module, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.CreateModule(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) ViewCreate(c context.Context, p *api.ViewCreatePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.View, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.CreateView(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) PageCreate(c context.Context, p *api.PageCreatePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.PageID, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.CreatePage(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) ProjectDelete(c context.Context, p *api.ProjectDeletePayload) error {
	return s.write(c, p.Slug, nil, p.ProjectID, func(u, e uuid.UUID) error { return s.repository.DeleteProject(c, u, p.Slug, e) })
}
func (s *entityFavoritesService) CycleDelete(c context.Context, p *api.CycleDeletePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.CycleID, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.DeleteCycle(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) ModuleDelete(c context.Context, p *api.ModuleDeletePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.ModuleID, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.DeleteModule(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) ViewDelete(c context.Context, p *api.ViewDeletePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.ViewID, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.DeleteView(c, u, p.Slug, id, e)
	})
}
func (s *entityFavoritesService) PageDelete(c context.Context, p *api.PageDeletePayload) error {
	return s.write(c, p.Slug, p.ProjectID, p.PageID, func(u, e uuid.UUID) error {
		id, _ := uuid.Parse(*p.ProjectID)
		return s.repository.DeletePage(c, u, p.Slug, id, e)
	})
}
