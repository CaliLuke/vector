// Package mcpauth verifies credentials and workspace scope for remote MCP calls.
package mcpauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
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

// Authenticate accepts only one Bearer credential and one workspace header.
// Cookies, token workspace metadata and alternate API-key headers cannot choose
// an identity or override the explicit workspace. Domain permissions remain the
// responsibility of each application action.
func (a *PersonalTokens) Authenticate(ctx context.Context, request *http.Request) (context.Context, error) {
	token, slug, err := personalTokenHeaders(request.Header)
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
	digest := sha256.Sum256([]byte(token))
	for _, key := range []string{
		":1:mcp:actor:" + principal.UserID.String(),
		":1:mcp:credential:" + hex.EncodeToString(digest[:]),
	} {
		rate, err := a.throttle.AllowCacheKey(ctx, key, "MCP")
		if err != nil {
			return nil, err
		}
		if !rate.Allowed {
			return nil, &mcptransport.RateLimitError{RetryAfter: rate.Wait}
		}
	}
	return withPrincipal(ctx, principal, verify), nil
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
	workspace, err := a.memberships.Membership(ctx, owner.UserID, slug)
	if err != nil {
		return Principal{}, err
	}
	return Principal{
		UserID: owner.UserID, Timezone: owner.Timezone,
		WorkspaceID: workspace.ID, WorkspaceSlug: slug, WorkspaceRole: workspace.Role,
	}, nil
}

func personalTokenHeaders(header http.Header) (token, slug string, err error) {
	authorization := header.Values("Authorization")
	workspaces := header.Values("X-Workspace-Slug")
	if len(header.Values("X-Api-Key")) > 0 || len(authorization) > 1 || len(workspaces) != 1 {
		return "", "", mcptransport.ErrInvalidRequest
	}
	slug = workspaces[0]
	if len(slug) > 255 || !workspacevalidation.ValidSlug(slug) {
		return "", "", mcptransport.ErrInvalidRequest
	}
	if len(authorization) == 0 {
		return "", "", mcptransport.ErrInvalidCredential
	}
	if strings.Contains(authorization[0], ",") {
		return "", "", mcptransport.ErrInvalidRequest
	}
	scheme, token, found := strings.Cut(authorization[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", "", mcptransport.ErrInvalidCredential
	}
	if strings.ContainsFunc(token, unicode.IsSpace) {
		return "", "", mcptransport.ErrInvalidRequest
	}
	return token, slug, nil
}
