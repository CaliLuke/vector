// Package mcpactions projects authenticated application operations onto MCP.
package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/useridentity"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type identityReader interface {
	CurrentIdentity(context.Context, uuid.UUID) (json.RawMessage, error)
}

type Service struct{ identity identityReader }

func New(identity identityReader) (*Service, error) {
	if identity == nil {
		return nil, errors.New("MCP identity application is required")
	}
	return &Service{identity: identity}, nil
}

func (s *Service) Member(ctx context.Context, input *api.MemberMe) (*api.MemberIdentity, error) {
	if input == nil || input.Action != "me" {
		return nil, api.MakeInvalidParams(errors.New("invalid member action"))
	}
	ctx, err := mcpauth.Revalidate(ctx)
	if err != nil {
		return nil, authenticationError(err)
	}
	slug := ""
	if input.WorkspaceSlug != nil {
		slug = *input.WorkspaceSlug
	}
	actor, err := mcpauth.SelectWorkspace(ctx, slug)
	if err != nil {
		return nil, api.MakeInvalidParams(errors.New("invalid workspace selector"))
	}
	if timezoneErr := djangosession.ValidateUserTimezone(actor.Timezone); timezoneErr != nil {
		return nil, identityFailure()
	}
	body, err := s.identity.CurrentIdentity(ctx, uuid.UUID(actor.UserID))
	if errors.Is(err, useridentity.ErrNotFound) {
		return nil, api.MakeNotFound(errors.New("identity unavailable"))
	}
	if err != nil {
		return nil, identityFailure()
	}
	// The pinned SDK fills absent UserLite properties with null. Keep that
	// normalization in this projection; HTTP retains its existing wire shape.
	result := &api.MemberIdentity{
		ID: loom.NullValue[string](), FirstName: loom.NullValue[string](),
		LastName: loom.NullValue[string](), Email: loom.NullValue[string](),
		Avatar: loom.NullValue[string](), AvatarURL: loom.NullValue[string](),
		DisplayName: loom.NullValue[string](),
	}
	if decodeErr := json.Unmarshal(body, result); decodeErr != nil {
		return nil, identityFailure()
	}
	return result, nil
}

func authenticationError(err error) error {
	switch {
	case errors.Is(err, mcptransport.ErrInvalidCredential):
		return api.MakeUnauthorized(errors.New("authentication expired"))
	case errors.Is(err, mcptransport.ErrAccessDenied):
		return api.MakeForbidden(errors.New("access revoked"))
	default:
		return identityFailure()
	}
}

func identityFailure() error {
	return api.MakeInternalError(errors.New("identity read failed"))
}

var _ api.Service = (*Service)(nil)
