package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	api "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacefavorites"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type workspaceFavoritesRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Group(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, uuid.UUID, string, store.CreateInput) (json.RawMessage, error)
	Update(context.Context, uuid.UUID, string, uuid.UUID, store.UpdateInput) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, string, uuid.UUID) error
}

type workspaceFavoritesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceFavoritesService struct {
	repository workspaceFavoritesRepository
	sessions   workspaceFavoritesSessions
}

type workspaceFavoritesPrincipal struct{}

func NewWorkspaceFavorites(repository workspaceFavoritesRepository, sessions workspaceFavoritesSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace favorites dependencies are required")
	}
	return &workspaceFavoritesService{repository: repository, sessions: sessions}, nil
}

func (s *workspaceFavoritesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.WorkspaceFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.WorkspaceFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceFavoritesPrincipal{}, userID), nil
}

func (s *workspaceFavoritesService) authorize(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceFavoritesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceFavoriteUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := s.repository.Authorized(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), workspaceFavoriteInternalError()
	}
	if !authorized {
		return uuid.Nil(), &api.WorkspaceFavoriteForbidden{Message: "You don't have the required permissions."}
	}
	return userID, nil
}

func (s *workspaceFavoritesService) List(ctx context.Context, payload *api.ListPayload) ([]*api.WorkspaceFavorite, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.List(ctx, userID, payload.Slug)
	if err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return decodeWorkspaceFavorites(body)
}

func (s *workspaceFavoritesService) Create(ctx context.Context, payload *api.CreatePayload) (*api.WorkspaceFavorite, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	input, validation := workspaceFavoriteCreateInput(payload)
	if validation != nil {
		return nil, validation
	}
	body, err := s.repository.Create(ctx, userID, payload.Slug, input)
	if errors.Is(err, store.ErrAlreadyExists) {
		message := "Favorite already exists"
		return nil, &api.WorkspaceFavoriteInvalid{Message: &message}
	}
	if err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return decodeWorkspaceFavorite(body)
}

func (s *workspaceFavoritesService) PatchCollection(ctx context.Context, payload *api.PatchCollectionPayload) (*api.WorkspaceFavorite, error) {
	if _, err := s.authorize(ctx, payload.Slug); err != nil {
		return nil, err
	}
	return nil, workspaceFavoriteInternalError()
}

func (s *workspaceFavoritesService) DeleteCollection(ctx context.Context, payload *api.DeleteCollectionPayload) error {
	if _, err := s.authorize(ctx, payload.Slug); err != nil {
		return err
	}
	return workspaceFavoriteInternalError()
}

func (s *workspaceFavoritesService) Retrieve(ctx context.Context, payload *api.RetrievePayload) (*api.WorkspaceFavorite, error) {
	if _, err := s.authorize(ctx, payload.Slug); err != nil {
		return nil, err
	}
	return nil, workspaceFavoriteInternalError()
}

func (s *workspaceFavoritesService) CreateDetail(ctx context.Context, payload *api.CreateDetailPayload) (*api.WorkspaceFavorite, error) {
	if _, err := s.authorize(ctx, payload.Slug); err != nil {
		return nil, err
	}
	return nil, workspaceFavoriteInternalError()
}

func (s *workspaceFavoritesService) Patch(ctx context.Context, payload *api.PatchPayload) (*api.WorkspaceFavorite, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	favoriteID, err := uuid.Parse(payload.FavoriteID)
	if err != nil {
		return nil, workspaceFavoriteNotFoundError()
	}
	input, validation := workspaceFavoriteUpdateInput(payload)
	if validation != nil {
		return nil, validation
	}
	body, err := s.repository.Update(ctx, userID, payload.Slug, favoriteID, input)
	if errors.Is(err, store.ErrNotFound) {
		return nil, workspaceFavoriteNotFoundError()
	}
	if errors.Is(err, store.ErrAlreadyExists) {
		message := "Favorite already exists"
		return nil, &api.WorkspaceFavoriteInvalid{Message: &message}
	}
	if err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return decodeWorkspaceFavorite(body)
}

func (s *workspaceFavoritesService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return err
	}
	favoriteID, err := uuid.Parse(payload.FavoriteID)
	if err != nil {
		return workspaceFavoriteNotFoundError()
	}
	err = s.repository.Delete(ctx, userID, payload.Slug, favoriteID)
	if errors.Is(err, store.ErrNotFound) {
		return workspaceFavoriteNotFoundError()
	}
	if err != nil {
		return workspaceFavoriteInternalError()
	}
	return nil
}

func (s *workspaceFavoritesService) Group(ctx context.Context, payload *api.GroupPayload) ([]*api.WorkspaceFavorite, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	parentID, err := uuid.Parse(payload.FavoriteID)
	if err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	body, err := s.repository.Group(ctx, userID, payload.Slug, parentID)
	if err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return decodeWorkspaceFavorites(body)
}

func workspaceFavoriteCreateInput(payload *api.CreatePayload) (store.CreateInput, error) {
	var result store.CreateInput
	entityType, ok := payload.EntityType.Value()
	if !payload.EntityType.Present() {
		return result, &api.WorkspaceFavoriteInvalid{EntityType: []string{"This field is required."}}
	}
	if !ok {
		return result, &api.WorkspaceFavoriteInvalid{EntityType: []string{"This field may not be null."}}
	}
	if strings.TrimSpace(entityType) == "" {
		return result, &api.WorkspaceFavoriteInvalid{EntityType: []string{"This field may not be blank."}}
	}
	result.EntityType = entityType
	if invalid := setWorkspaceFavoriteUUID(payload.EntityIdentifier, &result.EntityIdentifier); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{EntityIdentifier: invalid}
	}
	if value, present := payload.Name.Value(); present {
		result.Name = &value
	}
	if value, present := payload.IsFolder.Value(); present {
		result.IsFolder = value
	} else if payload.IsFolder.IsNull() {
		return result, &api.WorkspaceFavoriteInvalid{IsFolder: []string{"This field may not be null."}}
	}
	if value, present := payload.Sequence.Value(); present {
		result.Sequence = &value
	} else if payload.Sequence.IsNull() {
		return result, &api.WorkspaceFavoriteInvalid{Sequence: []string{"This field may not be null."}}
	}
	if invalid := setWorkspaceFavoriteUUID(payload.Parent, &result.Parent); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{Parent: invalid}
	}
	if invalid := setWorkspaceFavoriteUUID(payload.ProjectID, &result.ProjectID); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{ProjectID: invalid}
	}
	return result, nil
}

func workspaceFavoriteUpdateInput(payload *api.PatchPayload) (store.UpdateInput, error) {
	result := store.UpdateInput{
		EntityTypePresent:       payload.EntityType.Present(),
		EntityIdentifierPresent: payload.EntityIdentifier.Present(),
		NamePresent:             payload.Name.Present(),
		IsFolderPresent:         payload.IsFolder.Present(),
		SequencePresent:         payload.Sequence.Present(),
		ParentPresent:           payload.Parent.Present(),
		ProjectIDPresent:        payload.ProjectID.Present(),
	}
	if value, ok := payload.EntityType.Value(); ok {
		if strings.TrimSpace(value) == "" {
			return result, &api.WorkspaceFavoriteInvalid{EntityType: []string{"This field may not be blank."}}
		}
		result.EntityType = &value
	} else if payload.EntityType.IsNull() {
		return result, &api.WorkspaceFavoriteInvalid{EntityType: []string{"This field may not be null."}}
	}
	if invalid := setWorkspaceFavoriteUUID(payload.EntityIdentifier, &result.EntityIdentifier); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{EntityIdentifier: invalid}
	}
	if value, ok := payload.Name.Value(); ok {
		result.Name = &value
	}
	if value, ok := payload.IsFolder.Value(); ok {
		result.IsFolder = &value
	} else if payload.IsFolder.IsNull() {
		return result, &api.WorkspaceFavoriteInvalid{IsFolder: []string{"This field may not be null."}}
	}
	if value, ok := payload.Sequence.Value(); ok {
		result.Sequence = &value
	} else if payload.Sequence.IsNull() {
		return result, &api.WorkspaceFavoriteInvalid{Sequence: []string{"This field may not be null."}}
	}
	if invalid := setWorkspaceFavoriteUUID(payload.Parent, &result.Parent); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{Parent: invalid}
	}
	if invalid := setWorkspaceFavoriteUUID(payload.ProjectID, &result.ProjectID); invalid != nil {
		return result, &api.WorkspaceFavoriteInvalid{ProjectID: invalid}
	}
	return result, nil
}

func setWorkspaceFavoriteUUID(value loom.Nullable[string], destination **uuid.UUID) []string {
	if text, ok := value.Value(); ok {
		parsed, err := uuid.Parse(text)
		if err != nil {
			return []string{"Must be a valid UUID."}
		}
		*destination = &parsed
	}
	return nil
}

func decodeWorkspaceFavorite(body json.RawMessage) (*api.WorkspaceFavorite, error) {
	var result api.WorkspaceFavorite
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return &result, nil
}

func decodeWorkspaceFavorites(body json.RawMessage) ([]*api.WorkspaceFavorite, error) {
	var result []*api.WorkspaceFavorite
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, workspaceFavoriteInternalError()
	}
	return result, nil
}

func workspaceFavoriteNotFoundError() error {
	return &api.WorkspaceFavoriteNotFound{Message: "The required object does not exist."}
}

func workspaceFavoriteInternalError() error {
	return &api.WorkspaceFavoriteInternal{Message: "Something went wrong please try again later"}
}

var _ workspaceFavoritesRepository = (*store.Repository)(nil)
