package mcpauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"

	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
)

type accessTokenIssuer interface {
	Introspect(context.Context, string) (oauth.Identity, error)
}
type linkedAccountReader interface {
	Resolve(context.Context, string) (oauth.LinkedUser, error)
}

type OAuth struct {
	issuer      accessTokenIssuer
	links       linkedAccountReader
	memberships membershipReader
	throttle    rateLimiter
}

func NewOAuth(issuer accessTokenIssuer, links linkedAccountReader, memberships membershipReader, throttle rateLimiter) (*OAuth, error) {
	if issuer == nil || links == nil || memberships == nil || throttle == nil {
		return nil, fmt.Errorf("MCP OAuth issuer, account links, memberships and throttle are required")
	}
	return &OAuth{issuer: issuer, links: links, memberships: memberships, throttle: throttle}, nil
}

func (a *OAuth) Authenticate(ctx context.Context, request *http.Request) (context.Context, error) {
	token, slug, err := credentialHeaders(request.Header, false)
	if err != nil {
		return nil, err
	}
	principal, identity, err := a.verify(ctx, token, slug)
	if err != nil {
		return nil, err
	}
	if !principal.read {
		return nil, mcptransport.ErrInsufficientScope
	}
	if err = throttleCredential(ctx, a.throttle, principal.UserID, token); err != nil {
		return nil, err
	}
	verify := func(ctx context.Context) (Principal, error) {
		current, assertion, err := a.verify(ctx, token, slug)
		if err != nil {
			return Principal{}, err
		}
		if current.UserID != principal.UserID || assertion.Issuer != identity.Issuer || assertion.Subject != identity.Subject || assertion.ClientID != identity.ClientID {
			return Principal{}, mcptransport.ErrInvalidCredential
		}
		return current, nil
	}
	return withWorkspaceResolver(ctx, principal, verify, a.memberships), nil
}

func (a *OAuth) verify(ctx context.Context, token, slug string) (Principal, oauth.Identity, error) {
	identity, err := a.issuer.Introspect(ctx, token)
	if err != nil {
		return Principal{}, oauth.Identity{}, oauthAuthError(err)
	}
	linked, err := a.links.Resolve(ctx, identity.Subject)
	if err != nil {
		return Principal{}, oauth.Identity{}, oauthAuthError(err)
	}
	if linked.ID == uuid.Nil {
		return Principal{}, oauth.Identity{}, mcptransport.ErrInvalidCredential
	}
	principal := Principal{UserID: linked.ID, Timezone: linked.Timezone, oauth: true,
		read: slices.Contains(identity.Scopes, oauth.ReadScope), write: slices.Contains(identity.Scopes, oauth.WriteScope)}
	if slug != "" {
		workspace, err := a.memberships.Membership(ctx, linked.ID, slug)
		if err != nil {
			return Principal{}, oauth.Identity{}, err
		}
		principal.WorkspaceSlug, principal.WorkspaceID, principal.WorkspaceRole = slug, workspace.ID, workspace.Role
	}
	return principal, identity, nil
}

func oauthAuthError(err error) error {
	if errors.Is(err, oauth.ErrInvalidCredential) {
		return mcptransport.ErrInvalidCredential
	}
	return err
}

func throttleCredential(ctx context.Context, throttle rateLimiter, user uuid.UUID, token string) error {
	digest := sha256.Sum256([]byte(token))
	for _, key := range []string{":1:mcp:actor:" + user.String(), ":1:mcp:credential:" + hex.EncodeToString(digest[:])} {
		rate, err := throttle.AllowCacheKey(ctx, key, "MCP")
		if err != nil {
			return err
		}
		if !rate.Allowed {
			return &mcptransport.RateLimitError{RetryAfter: rate.Wait}
		}
	}
	return nil
}
