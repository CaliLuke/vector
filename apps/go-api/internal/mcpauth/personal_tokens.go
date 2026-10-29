// Package mcpauth verifies credentials and workspace scope for remote MCP calls.
package mcpauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
)

type tokenValidator interface {
	Validate(context.Context, string) (apikey.Principal, error)
}

type membershipReader interface {
	Membership(context.Context, uuid.UUID, string) (Workspace, error)
}

type rateLimiter interface {
	AllowCacheKey(context.Context, string, string) (apikey.Rate, error)
}

// PersonalTokens authenticates against the application's existing token owner.
// The validator and membership reader must use the primary database so revocation
// and membership changes are visible on the next request or invocation.
type PersonalTokens struct {
	keys        tokenValidator
	memberships membershipReader
	throttle    rateLimiter
}

func NewPersonalTokens(keys tokenValidator, memberships membershipReader, throttle rateLimiter) (*PersonalTokens, error) {
	if keys == nil || memberships == nil || throttle == nil {
		return nil, fmt.Errorf("MCP personal-token validator, memberships and throttle are required")
	}
	return &PersonalTokens{keys: keys, memberships: memberships, throttle: throttle}, nil
}

// Authenticate accepts one Bearer credential and an optional workspace header.
// Cookies, token workspace metadata and alternate API-key headers cannot choose
// an identity or override the explicit workspace. Domain permissions remain the
// responsibility of each application action.
func (a *PersonalTokens) Authenticate(ctx context.Context, request *http.Request) (context.Context, error) {
	token, slug, err := credentialHeaders(request.Header)
	if err != nil {
		return nil, err
	}
	verify := func(ctx context.Context) (Principal, error) {
		return a.verify(ctx, token, slug)
	}
	principal, err := verify(ctx)
	if err != nil {
		return nil, err
	}
	if err := throttleCredential(ctx, a.throttle, principal.UserID, token); err != nil {
		return nil, err
	}

	return withWorkspaceResolver(ctx, principal, verify, a.memberships), nil
}

func (a *PersonalTokens) verify(ctx context.Context, token, slug string) (Principal, error) {
	owner, err := a.keys.Validate(ctx, token)
	if errors.Is(err, apikey.ErrInvalid) {
		return Principal{}, mcptransport.ErrInvalidCredential
	}
	if err != nil {
		return Principal{}, err
	}
	if owner.UserID == uuid.Nil {
		return Principal{}, fmt.Errorf("MCP token owner is missing")
	}
	principal := Principal{UserID: owner.UserID, Timezone: owner.Timezone}
	if slug == "" {
		return principal, nil
	}
	workspace, err := a.memberships.Membership(ctx, owner.UserID, slug)
	if err != nil {
		return Principal{}, err
	}
	principal.WorkspaceID, principal.WorkspaceSlug, principal.WorkspaceRole = workspace.ID, slug, workspace.Role
	return principal, nil
}
