package mcpauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type oauthIssuerFake struct {
	identity oauth.Identity
	err      error
}

func (f *oauthIssuerFake) Introspect(context.Context, string) (oauth.Identity, error) {
	return f.identity, f.err
}

type oauthLinksFake struct {
	user oauth.LinkedUser
	err  error
}

func (f *oauthLinksFake) Resolve(context.Context, string) (oauth.LinkedUser, error) {
	return f.user, f.err
}

type oauthFixture struct {
	auth        *OAuth
	issuer      *oauthIssuerFake
	links       *oauthLinksFake
	memberships *membershipFake
	throttle    *throttleFake
}

func newOAuthFixture(t *testing.T) oauthFixture {
	t.Helper()
	f := oauthFixture{
		issuer:      &oauthIssuerFake{identity: oauth.Identity{Issuer: "https://issuer.example", Subject: "person", ClientID: "agent", Scopes: []string{oauth.ReadScope, oauth.WriteScope}}},
		links:       &oauthLinksFake{user: oauth.LinkedUser{ID: uuid.New(), Timezone: "UTC"}},
		memberships: &membershipFake{workspace: Workspace{ID: uuid.New(), Role: 15}},
		throttle:    &throttleFake{rate: apikey.Rate{Allowed: true}},
	}
	var err error
	f.auth, err = NewOAuth(f.issuer, f.links, f.memberships, f.throttle)
	require.NoError(t, err)
	return f
}

func oauthRequest() *http.Request {
	r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp", nil)
	r.Header.Set("Authorization", "Bearer test-oauth-token")
	return r
}

func TestOAuthKeepsContextFreeIdentityAndResolvesCurrentWorkspace(t *testing.T) {
	f := newOAuthFixture(t)
	ctx, err := f.auth.Authenticate(t.Context(), oauthRequest())
	require.NoError(t, err)
	principal, err := SelectWorkspace(ctx, "")
	require.NoError(t, err)
	require.Equal(t, f.links.user.ID, principal.UserID)
	require.Equal(t, uuid.Nil, principal.WorkspaceID)
	require.NoError(t, RequireScopes(ctx, true))
	current, err := Revalidate(ctx)
	require.NoError(t, err)
	principal, err = SelectWorkspace(current, "workspace-one")
	require.NoError(t, err)
	require.Equal(t, f.memberships.workspace.ID, principal.WorkspaceID)
	require.Equal(t, 15, principal.WorkspaceRole)
	f.memberships.err = mcptransport.ErrAccessDenied
	_, err = SelectWorkspace(current, "workspace-one")
	require.ErrorIs(t, err, mcptransport.ErrAccessDenied)
	for _, slug := range []string{"../escape", strings.Repeat("a", 256)} {
		_, err = SelectWorkspace(current, slug)
		require.ErrorIs(t, err, ErrWorkspaceMismatch)
	}
	require.Len(t, f.throttle.keys, 2)
	require.NotContains(t, strings.Join(f.throttle.keys, " "), "test-oauth-token")
}

func TestOAuthHeaderWorkspaceRemainsBindingAfterRevalidation(t *testing.T) {
	f := newOAuthFixture(t)
	r := oauthRequest()
	r.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), r)
	require.NoError(t, err)
	_, err = SelectWorkspace(ctx, "workspace-two")
	require.ErrorIs(t, err, ErrWorkspaceMismatch)
	f.memberships.workspace.Role = 5
	current, err := Revalidate(ctx)
	require.NoError(t, err)
	principal, err := SelectWorkspace(current, "workspace-one")
	require.NoError(t, err)
	require.Equal(t, 5, principal.WorkspaceRole)
	f.memberships.err = mcptransport.ErrAccessDenied
	_, err = Revalidate(ctx)
	require.ErrorIs(t, err, mcptransport.ErrAccessDenied)
}

func TestOAuthRevalidationRejectsChangedCredentialOwnership(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		change func(oauthFixture)
	}{
		{"issuer", func(f oauthFixture) { f.issuer.identity.Issuer = "https://different.example" }},
		{"subject", func(f oauthFixture) { f.issuer.identity.Subject = "other-person" }},
		{"client", func(f oauthFixture) { f.issuer.identity.ClientID = "other-agent" }},
		{"account", func(f oauthFixture) { f.links.user.ID = uuid.New() }},
		{"revoked", func(f oauthFixture) { f.issuer.err = oauth.ErrInvalidCredential }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			ctx, err := f.auth.Authenticate(t.Context(), oauthRequest())
			require.NoError(t, err)
			scenario.change(f)
			_, err = Revalidate(ctx)
			require.ErrorIs(t, err, mcptransport.ErrInvalidCredential)
		})
	}
}

func TestOAuthScopesAreRefreshedBeforeReadsAndWrites(t *testing.T) {
	f := newOAuthFixture(t)
	ctx, err := f.auth.Authenticate(t.Context(), oauthRequest())
	require.NoError(t, err)
	f.issuer.identity.Scopes = []string{oauth.ReadScope}
	current, err := Revalidate(ctx)
	require.NoError(t, err)
	require.NoError(t, RequireScopes(current, false))
	require.ErrorIs(t, RequireScopes(current, true), mcptransport.ErrAccessDenied)
	f.issuer.identity.Scopes = []string{oauth.WriteScope}
	current, err = Revalidate(ctx)
	require.NoError(t, err)
	require.ErrorIs(t, RequireScopes(current, false), mcptransport.ErrAccessDenied)
	_, err = f.auth.Authenticate(t.Context(), oauthRequest())
	require.ErrorIs(t, err, mcptransport.ErrInsufficientScope)
	require.ErrorIs(t, RequireScopes(t.Context(), false), mcptransport.ErrInvalidCredential)
	pat, _, _, _ := personalTokenFixture(t)
	patContext, err := pat.Authenticate(t.Context(), tokenRequest())
	require.NoError(t, err)
	require.NoError(t, RequireScopes(patContext, true))
}

func TestOAuthAdmissionFailsClosed(t *testing.T) {
	outage := errors.New("dependency unavailable")
	for _, scenario := range []struct {
		name   string
		change func(oauthFixture, *http.Request)
		want   error
	}{
		{"duplicate authorization", func(_ oauthFixture, r *http.Request) { r.Header.Add("Authorization", "Bearer other") }, mcptransport.ErrInvalidRequest},
		{"issuer unavailable", func(f oauthFixture, _ *http.Request) { f.issuer.err = outage }, outage},
		{"unlinked", func(f oauthFixture, _ *http.Request) { f.links.err = oauth.ErrInvalidCredential }, mcptransport.ErrInvalidCredential},
		{"link store unavailable", func(f oauthFixture, _ *http.Request) { f.links.err = outage }, outage},
		{"missing account", func(f oauthFixture, _ *http.Request) { f.links.user.ID = uuid.Nil }, mcptransport.ErrInvalidCredential},
		{"throttle unavailable", func(f oauthFixture, _ *http.Request) { f.throttle.err = outage }, outage},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			r := oauthRequest()
			scenario.change(f, r)
			ctx, err := f.auth.Authenticate(t.Context(), r)
			require.ErrorIs(t, err, scenario.want)
			require.Nil(t, ctx)
		})
	}
}

func TestOAuthConstructionRequiresAllAuthorizationDependencies(t *testing.T) {
	f := newOAuthFixture(t)
	_, err := NewOAuth(nil, f.links, f.memberships, f.throttle)
	require.Error(t, err)
	_, err = NewOAuth(f.issuer, nil, f.memberships, f.throttle)
	require.Error(t, err)
	_, err = NewOAuth(f.issuer, f.links, nil, f.throttle)
	require.Error(t, err)
	_, err = NewOAuth(f.issuer, f.links, f.memberships, nil)
	require.Error(t, err)
}
