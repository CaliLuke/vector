package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/userapitokens"
	"github.com/CaliLuke/loom/security"
)

type apiTokensRepository interface {
	List(context.Context, uuid.UUID) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, uuid.UUID, string, string, string, *string) (json.RawMessage, error)
	Patch(context.Context, uuid.UUID, uuid.UUID, *string, *string) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}
type apiTokensSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type apiTokensService struct {
	repository apiTokensRepository
	sessions   apiTokensSessionValidator
}

func NewUserAPITokens(r apiTokensRepository, s apiTokensSessionValidator) (api.Service, error) {
	if r == nil {
		return nil, fmt.Errorf("api tokens repository is required")
	}
	if s == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &apiTokensService{r, s}, nil
}
func (s *apiTokensService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.APITokenUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.APITokenUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return context.WithValue(ctx, apiTokenPrincipalKey{}, id), nil
}
func (s *apiTokensService) id(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(apiTokenPrincipalKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.APITokenUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return id, nil
}
func (s *apiTokensService) List(ctx context.Context, _ *api.ListPayload) ([]*api.APITokenRead, error) {
	id, e := s.id(ctx)
	if e != nil {
		return nil, e
	}
	b, e := s.repository.List(ctx, id)
	if e != nil {
		return nil, apiTokenInternal()
	}
	var out []*api.APITokenRead
	if json.Unmarshal(b, &out) != nil {
		return nil, apiTokenInternal()
	}
	return out, nil
}
func (s *apiTokensService) Create(ctx context.Context, p *api.CreatePayload) (*api.APIToken, error) {
	id, e := s.id(ctx)
	if e != nil {
		return nil, e
	}
	label := strings.ReplaceAll(uuid.New().String(), "-", "")
	if p.Label != nil {
		label = *p.Label
	}
	description := ""
	if p.Description != nil {
		description = *p.Description
	}
	var expiredAt *string
	if value, ok := p.ExpiredAt.Value(); ok {
		expiredAt = &value
	}
	b, e := s.repository.Create(ctx, id, label, description, "plane_api_"+strings.ReplaceAll(uuid.New().String(), "-", ""), expiredAt)
	if e != nil {
		return nil, apiTokenInternal()
	}
	var out api.APIToken
	if json.Unmarshal(b, &out) != nil {
		return nil, apiTokenInternal()
	}
	return &out, nil
}
func (s *apiTokensService) Retrieve(ctx context.Context, p *api.RetrievePayload) (*api.APITokenRead, error) {
	id, e := s.id(ctx)
	if e != nil {
		return nil, e
	}
	token, e := uuid.Parse(p.Pk)
	if e != nil {
		return nil, apiTokenNotFound()
	}
	b, e := s.repository.Get(ctx, id, token)
	if errors.Is(e, store.ErrNotFound) {
		return nil, apiTokenNotFound()
	}
	if e != nil {
		return nil, apiTokenInternal()
	}
	var out api.APITokenRead
	if json.Unmarshal(b, &out) != nil {
		return nil, apiTokenInternal()
	}
	return &out, nil
}
func (s *apiTokensService) Patch(ctx context.Context, p *api.PatchPayload) (*api.APIToken, error) {
	id, e := s.id(ctx)
	if e != nil {
		return nil, e
	}
	token, e := uuid.Parse(p.Pk)
	if e != nil {
		return nil, apiTokenNotFound()
	}
	b, e := s.repository.Patch(ctx, id, token, p.Label, p.Description)
	if errors.Is(e, store.ErrNotFound) {
		return nil, apiTokenNotFound()
	}
	if e != nil {
		return nil, apiTokenInternal()
	}
	var out api.APIToken
	if json.Unmarshal(b, &out) != nil {
		return nil, apiTokenInternal()
	}
	return &out, nil
}
func (s *apiTokensService) Delete(ctx context.Context, p *api.DeletePayload) error {
	id, e := s.id(ctx)
	if e != nil {
		return e
	}
	token, e := uuid.Parse(p.Pk)
	if e != nil {
		return apiTokenNotFound()
	}
	e = s.repository.Delete(ctx, id, token)
	if errors.Is(e, store.ErrNotFound) {
		return apiTokenNotFound()
	}
	if e != nil {
		return apiTokenInternal()
	}
	return nil
}
func (s *apiTokensService) PatchCollection(context.Context, *api.PatchCollectionPayload) error {
	return apiTokenInternal()
}
func (s *apiTokensService) DeleteCollection(context.Context, *api.DeleteCollectionPayload) error {
	return apiTokenInternal()
}
func (s *apiTokensService) PostDetail(context.Context, *api.PostDetailPayload) error {
	return apiTokenInternal()
}
func apiTokenNotFound() error {
	return &api.APITokenNotFound{Message: "The required object does not exist."}
}
func apiTokenInternal() error {
	return &api.APITokenInternalError{Message: "Something went wrong please try again later"}
}

type apiTokenPrincipalKey struct{}
