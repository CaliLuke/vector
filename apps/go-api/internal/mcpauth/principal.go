package mcpauth

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
	"github.com/google/uuid"
)

// Principal contains verified identity and workspace membership, never secrets.
// A personal token carries its owner's permissions, not OAuth scopes.
type Principal struct {
	UserID        uuid.UUID
	ClientID      string
	Timezone      string
	WorkspaceID   uuid.UUID
	WorkspaceSlug string
	WorkspaceRole int
	oauth         bool
	read          bool
	write         bool
}

type principalContextKey struct{}

type verifiedIdentity struct {
	principal  Principal
	verify     func(context.Context) (Principal, error)
	workspaces membershipReader
}

var ErrWorkspaceMismatch = errors.New("workspace selector must match X-Workspace-Slug")

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	identity, ok := ctx.Value(principalContextKey{}).(verifiedIdentity)
	return identity.principal, ok
}

// Revalidate refreshes live authentication before an invocation or receipt replay.
// Callers must use the returned context and perform current domain authorization.
func Revalidate(ctx context.Context) (context.Context, error) {
	identity, ok := ctx.Value(principalContextKey{}).(verifiedIdentity)
	if !ok {
		return nil, mcptransport.ErrInvalidCredential
	}
	principal, err := identity.verify(ctx)
	if err != nil {
		return nil, err
	}
	return withWorkspaceResolver(ctx, principal, identity.verify, identity.workspaces), nil
}

// SelectWorkspace binds optional tool selectors to the authenticated workspace.
// It does not grant project or action permission.
func SelectWorkspace(ctx context.Context, slug string) (Principal, error) {
	identity, ok := ctx.Value(principalContextKey{}).(verifiedIdentity)
	if !ok {
		return Principal{}, mcptransport.ErrInvalidCredential
	}
	principal := identity.principal
	if slug == "" {
		return principal, nil
	}
	if len(slug) > 255 || !workspacevalidation.ValidSlug(slug) {
		return Principal{}, ErrWorkspaceMismatch
	}
	if principal.WorkspaceSlug != "" {
		if slug != principal.WorkspaceSlug {
			return Principal{}, ErrWorkspaceMismatch
		}
		return principal, nil
	}
	if identity.workspaces == nil {
		return Principal{}, ErrWorkspaceMismatch
	}
	workspace, err := identity.workspaces.Membership(ctx, principal.UserID, slug)
	if err != nil {
		return Principal{}, err
	}
	principal.WorkspaceID, principal.WorkspaceSlug, principal.WorkspaceRole = workspace.ID, slug, workspace.Role
	return principal, nil
}

// RequireScopes checks the current credential class without assigning synthetic
// OAuth scopes to personal tokens. Call after Revalidate, including for replay.
func RequireScopes(ctx context.Context, mutation bool) error {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return mcptransport.ErrInvalidCredential
	}
	if !principal.AllowsAction(mutation) {
		return mcptransport.ErrAccessDenied
	}
	return nil
}

func withWorkspaceResolver(ctx context.Context, principal Principal, verify func(context.Context) (Principal, error), workspaces membershipReader) context.Context {
	return context.WithValue(ctx, principalContextKey{}, verifiedIdentity{principal: principal, verify: verify, workspaces: workspaces})
}

// CredentialClass identifies the verified authentication mode without exposing a token.
func (p Principal) CredentialClass() string {
	if p.oauth {
		return "oauth"
	}
	return "personal_token"
}

// AllowsAction reports current scope availability for authenticated discovery.
// Domain permissions remain the application's responsibility.
func (p Principal) AllowsAction(mutation bool) bool {
	return !p.oauth || p.read && (!mutation || p.write)
}
