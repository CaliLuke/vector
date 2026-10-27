package mcpauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tokenFake struct {
	principal apikey.Principal
	err       error
	calls     int
}

func (f *tokenFake) Validate(context.Context, string) (apikey.Principal, error) {
	f.calls++
	return f.principal, f.err
}

type membershipFake struct {
	workspace Workspace
	err       error
	calls     int
}

func (f *membershipFake) Membership(context.Context, uuid.UUID, string) (Workspace, error) {
	f.calls++
	return f.workspace, f.err
}

type throttleFake struct {
	keys []string
	rate apikey.Rate
	err  error
}

func (f *throttleFake) AllowCacheKey(_ context.Context, key, _ string) (apikey.Rate, error) {
	f.keys = append(f.keys, key)
	return f.rate, f.err
}

func personalTokenFixture(t *testing.T) (*PersonalTokens, *tokenFake, *membershipFake, *throttleFake) {
	t.Helper()
	keys := &tokenFake{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
	memberships := &membershipFake{workspace: Workspace{ID: uuid.New(), Role: 15}}
	throttle := &throttleFake{rate: apikey.Rate{Allowed: true}}
	auth, err := NewPersonalTokens(keys, memberships, throttle)
	require.NoError(t, err)
	return auth, keys, memberships, throttle
}

func tokenRequest() *http.Request {
	r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	r.Header.Set("Authorization", "Bearer test-only-personal-token")
	r.Header.Set("X-Workspace-Slug", "workspace-one")
	return r
}

func TestPersonalTokensRejectMalformedHeadersBeforeCredentialLookup(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(http.Header)
		want   error
	}{
		{"missing workspace", func(h http.Header) { h.Del("X-Workspace-Slug") }, mcptransport.ErrInvalidRequest},
		{"empty workspace", func(h http.Header) { h.Set("X-Workspace-Slug", "") }, mcptransport.ErrInvalidRequest},
		{"duplicate workspace", func(h http.Header) { h.Add("X-Workspace-Slug", "workspace-one") }, mcptransport.ErrInvalidRequest},
		{"combined workspace", func(h http.Header) { h.Set("X-Workspace-Slug", "one,two") }, mcptransport.ErrInvalidRequest},
		{"invalid workspace", func(h http.Header) { h.Set("X-Workspace-Slug", "../one") }, mcptransport.ErrInvalidRequest},
		{"workspace whitespace", func(h http.Header) { h.Set("X-Workspace-Slug", " one ") }, mcptransport.ErrInvalidRequest},
		{"workspace too long", func(h http.Header) { h.Set("X-Workspace-Slug", strings.Repeat("a", 256)) }, mcptransport.ErrInvalidRequest},
		{"duplicate authorization", func(h http.Header) { h.Add("Authorization", "Bearer second") }, mcptransport.ErrInvalidRequest},
		{"combined bearer", func(h http.Header) { h.Set("Authorization", "Bearer first, Bearer second") }, mcptransport.ErrInvalidRequest},
		{"multiple bearer", func(h http.Header) { h.Set("Authorization", "Bearer first Bearer second") }, mcptransport.ErrInvalidRequest},
		{"alternate API key", func(h http.Header) { h.Set("X-Api-Key", "second") }, mcptransport.ErrInvalidRequest},
		{"empty alternate API key", func(h http.Header) { h.Set("X-Api-Key", "") }, mcptransport.ErrInvalidRequest},
		{"missing authorization", func(h http.Header) { h.Del("Authorization") }, mcptransport.ErrInvalidCredential},
		{"empty bearer", func(h http.Header) { h.Set("Authorization", "Bearer ") }, mcptransport.ErrInvalidCredential},
		{"session cookie only", func(h http.Header) { h.Del("Authorization"); h.Set("Cookie", "session-id=test-only") }, mcptransport.ErrInvalidCredential},
		{"different scheme", func(h http.Header) { h.Set("Authorization", "Basic test-only") }, mcptransport.ErrInvalidCredential},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth, keys, memberships, throttle := personalTokenFixture(t)
			r := tokenRequest()
			tc.change(r.Header)
			ctx, err := auth.Authenticate(r.Context(), r)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, ctx)
			assert.Zero(t, keys.calls)
			assert.Zero(t, memberships.calls)
			assert.Empty(t, throttle.keys)
		})
	}
}

func TestPersonalTokensRevalidateCredentialAndMembership(t *testing.T) {
	auth, keys, memberships, throttle := personalTokenFixture(t)
	r := tokenRequest()
	ctx, err := auth.Authenticate(r.Context(), r)
	require.NoError(t, err)
	principal, ok := PrincipalFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, keys.principal.UserID, principal.UserID)
	assert.Equal(t, "personal_token", principal.CredentialClass())
	assert.Empty(t, principal.ClientID)
	assert.Equal(t, memberships.workspace.ID, principal.WorkspaceID)
	assert.Equal(t, "workspace-one", principal.WorkspaceSlug)
	assert.Equal(t, 15, principal.WorkspaceRole)
	require.Len(t, throttle.keys, 2)
	assert.Contains(t, throttle.keys[0], keys.principal.UserID.String())
	assert.NotContains(t, strings.Join(throttle.keys, " "), "test-only-personal-token")

	memberships.workspace.Role = 5
	updated, err := Revalidate(ctx)
	require.NoError(t, err)
	principal, ok = PrincipalFromContext(updated)
	require.True(t, ok)
	assert.Equal(t, 5, principal.WorkspaceRole)
	assert.Equal(t, 2, keys.calls)
	assert.Equal(t, 2, memberships.calls)
	assert.Len(t, throttle.keys, 2, "revalidation must not charge another HTTP request")

	keys.err = apikey.ErrInvalid
	_, err = Revalidate(ctx)
	require.ErrorIs(t, err, mcptransport.ErrInvalidCredential)
	assert.Equal(t, 2, memberships.calls)
	keys.err = nil
	memberships.err = mcptransport.ErrAccessDenied
	_, err = Revalidate(ctx)
	require.ErrorIs(t, err, mcptransport.ErrAccessDenied)
}

func TestPersonalTokensFailClosedAndApplySharedThrottle(t *testing.T) {
	for _, source := range []string{"token", "membership", "throttle"} {
		t.Run(source, func(t *testing.T) {
			auth, keys, memberships, throttle := personalTokenFixture(t)
			outage := errors.New("private backend error")
			switch source {
			case "token":
				keys.err = outage
			case "membership":
				memberships.err = outage
			case "throttle":
				throttle.err = outage
			}
			ctx, err := auth.Authenticate(context.Background(), tokenRequest())
			require.ErrorIs(t, err, outage)
			assert.Nil(t, ctx)
		})
	}
	auth, _, _, throttle := personalTokenFixture(t)
	throttle.rate = apikey.Rate{Wait: 2 * time.Second}
	_, err := auth.Authenticate(context.Background(), tokenRequest())
	var limited *mcptransport.RateLimitError
	require.ErrorAs(t, err, &limited)
	require.NotContains(t, err.Error(), "test-only-personal-token")
	assert.Equal(t, 2*time.Second, limited.RetryAfter)
}

func TestPersonalTokenWorkspaceSelectorMustMatchHeader(t *testing.T) {
	auth, _, _, _ := personalTokenFixture(t)
	ctx, err := auth.Authenticate(context.Background(), tokenRequest())
	require.NoError(t, err)
	for _, slug := range []string{"", "workspace-one"} {
		principal, selectErr := SelectWorkspace(ctx, slug)
		require.NoError(t, selectErr)
		assert.Equal(t, "workspace-one", principal.WorkspaceSlug)
	}
	_, err = SelectWorkspace(ctx, "workspace-two")
	require.ErrorIs(t, err, ErrWorkspaceMismatch)
	_, err = SelectWorkspace(context.Background(), "workspace-one")
	require.ErrorIs(t, err, mcptransport.ErrInvalidCredential)
}

func TestPersonalTokenConstructionAndMissingIdentityFailClosed(t *testing.T) {
	_, keys, memberships, throttle := personalTokenFixture(t)
	_, err := NewPersonalTokens(nil, memberships, throttle)
	require.Error(t, err)
	_, err = NewPersonalTokens(keys, nil, throttle)
	require.Error(t, err)
	_, err = NewPersonalTokens(keys, memberships, nil)
	require.Error(t, err)
	_, err = NewWorkspaceReader(nil)
	require.Error(t, err)
	_, err = Revalidate(context.Background())
	require.ErrorIs(t, err, mcptransport.ErrInvalidCredential)
	auth, err := NewPersonalTokens(keys, memberships, throttle)
	require.NoError(t, err)
	keys.principal.UserID = uuid.Nil
	ctx, err := auth.Authenticate(context.Background(), tokenRequest())
	require.Error(t, err)
	assert.Nil(t, ctx)
	assert.Zero(t, memberships.calls)
}

func TestPersonalTokensShareActorLimitAcrossCredentials(t *testing.T) {
	auth, _, _, throttle := personalTokenFixture(t)
	first := tokenRequest()
	_, err := auth.Authenticate(first.Context(), first)
	require.NoError(t, err)
	second := tokenRequest()
	second.Header.Set("Authorization", "Bearer second-test-only-token")
	_, err = auth.Authenticate(second.Context(), second)
	require.NoError(t, err)
	require.Len(t, throttle.keys, 4)
	assert.Equal(t, throttle.keys[0], throttle.keys[2], "rotating credentials cannot reset the actor limit")
	assert.NotEqual(t, throttle.keys[1], throttle.keys[3])
}
