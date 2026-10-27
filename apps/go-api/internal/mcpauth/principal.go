package mcpauth

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
)

// Principal contains verified identity and workspace membership, never secrets.
// A personal token carries its owner's permissions, not OAuth scopes.
type Principal struct {
	UserID        uuid.UUID
	Timezone      string
	WorkspaceID   uuid.UUID
	WorkspaceSlug string
	WorkspaceRole int
}

type principalContextKey struct{}

type verifiedIdentity struct {
	principal Principal
	verify    func(context.Context) (Principal, error)
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
	return withPrincipal(ctx, principal, identity.verify), nil
}

// SelectWorkspace binds optional tool selectors to the authenticated workspace.
// It does not grant project or action permission.
func SelectWorkspace(ctx context.Context, slug string) (Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return Principal{}, mcptransport.ErrInvalidCredential
	}
	if slug != "" && slug != principal.WorkspaceSlug {
		return Principal{}, ErrWorkspaceMismatch
	}
	return principal, nil
}

func withPrincipal(ctx context.Context, principal Principal, verify func(context.Context) (Principal, error)) context.Context {
	return context.WithValue(ctx, principalContextKey{}, verifiedIdentity{principal: principal, verify: verify})
}
