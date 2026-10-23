package systemapi

import (
	"context"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type serviceSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

// serviceSessionIdentity uses the middleware-validated principal when present;
// otherwise it validates the supplied credential. It preserves a principal
// returned alongside an error so each adapter can retain its current policy.
func serviceSessionIdentity(ctx context.Context, credential string, sessions serviceSessionValidator) (string, string, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if inspected {
		return userID, timezone, nil
	}
	principal, err := sessions.Validate(ctx, credential)
	return principal.UserID, principal.UserTimezone, err
}

// serviceSessionIdentityFromRequest preserves adapters that use the request's
// session cookie when the middleware did not supply a principal and the caller
// did not pass a credential.
func serviceSessionIdentityFromRequest(ctx context.Context, credential string, sessions serviceSessionValidator) (string, string, error) {
	if _, _, inspected := readServiceSessionPrincipal(ctx); !inspected && credential == "" {
		credential = sessionCredentialFromContext(ctx)
	}
	return serviceSessionIdentity(ctx, credential, sessions)
}

// validateSessionUserUUID always validates the supplied credential, even when
// middleware identity is present. Some API-key adapters preserve this policy.
func validateSessionUserUUID(ctx context.Context, credential string, sessions serviceSessionValidator) (uuid.UUID, error) {
	principal, err := sessions.Validate(ctx, credential)
	if err != nil {
		return uuid.Nil(), err
	}
	return uuid.Parse(principal.UserID)
}
