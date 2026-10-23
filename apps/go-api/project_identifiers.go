package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_identifiers"
	store "github.com/CaliLuke/go-plane/internal/projectidentifiers"
	"github.com/CaliLuke/loom/security"
)

type projectIdentifiersRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	Get(context.Context, uuid.UUID, string, string) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, string, string) error
}

type projectIdentifiersService struct {
	repository projectIdentifiersRepository
	sessions   serviceSessionValidator
}

type projectIdentifiersPrincipal struct{}

// NewProjectIdentifiers constructs the project identifier service.
func NewProjectIdentifiers(repository projectIdentifiersRepository, sessions serviceSessionValidator) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("project identifiers dependencies are required")
	}
	return &projectIdentifiersService{repository: repository, sessions: sessions}, nil
}

func (s *projectIdentifiersService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, err := validateSessionUserUUID(ctx, key, s.sessions)
	if err != nil {
		return ctx, &api.ProjectIdentifierUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, projectIdentifiersPrincipal{}, userID), nil
}

func (s *projectIdentifiersService) authorize(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, ok := ctx.Value(projectIdentifiersPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.ProjectIdentifierUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := s.repository.Authorized(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), projectIdentifiersInternalError()
	}
	if !authorized {
		return uuid.Nil(), &api.ProjectIdentifierForbidden{Detail: "You do not have the required permissions."}
	}
	return userID, nil
}

func (s *projectIdentifiersService) Get(ctx context.Context, payload *api.GetPayload) (*api.ProjectIdentifierCollection, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.Get(ctx, userID, payload.Slug, payload.Name)
	if errors.Is(err, store.ErrNameRequired) {
		return nil, &api.ProjectIdentifierInvalid{Message: "Name is required"}
	}
	if err != nil {
		return nil, projectIdentifiersInternalError()
	}
	var result api.ProjectIdentifierCollection
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, projectIdentifiersInternalError()
	}
	return &result, nil
}

func (s *projectIdentifiersService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return err
	}
	if payload.Body == nil {
		return &api.ProjectIdentifierInvalid{Message: "Name is required"}
	}
	err = s.repository.Delete(ctx, userID, payload.Slug, payload.Body.Name)
	if errors.Is(err, store.ErrNameRequired) {
		return &api.ProjectIdentifierInvalid{Message: "Name is required"}
	}
	if errors.Is(err, store.ErrProjectExists) {
		return &api.ProjectIdentifierConflict{Message: "Cannot delete an identifier of an existing project"}
	}
	if err != nil {
		return projectIdentifiersInternalError()
	}
	return nil
}

func projectIdentifiersInternalError() error {
	return &api.ProjectIdentifierInternal{Message: "Something went wrong please try again later"}
}

var _ projectIdentifiersRepository = (*store.Repository)(nil)
