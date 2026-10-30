package main

import (
	"strings"
	"testing"
	"time"

	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/stretchr/testify/require"
)

func resetTokenFixture(t *testing.T) (*djangoPasswordResetTokenIssuer, store.User, string, string, time.Time) {
	t.Helper()
	issuedAt := time.Date(2026, 10, 21, 12, 34, 56, 0, time.UTC)
	issuer, err := newDjangoPasswordResetTokenIssuer("token-validation-test-secret")
	require.NoError(t, err)
	issuer.now = func() time.Time { return issuedAt }
	lastLogin := issuedAt.Add(-time.Hour)
	user := store.User{ID: "abc", Email: "a@example.test", Password: "stored-password-hash", LastLogin: &lastLogin}
	uid, token, err := issuer.Issue(t.Context(), user)
	require.NoError(t, err)
	return issuer, user, uid, token, issuedAt
}

func TestPasswordResetRejectsMalformedTokensWithoutOperationalFailure(t *testing.T) {
	issuer, user, uid, token, _ := resetTokenFixture(t)
	timestamp, signature, ok := strings.Cut(token, "-")
	require.True(t, ok)
	for _, scenario := range []struct{ name, uid, token string }{
		{"malformed user encoding", "%%%", token},
		{"different user", "ZGlmZmVyZW50", token},
		{"missing signature", uid, timestamp},
		{"invalid timestamp", uid, "!" + "-" + signature},
		{"overflowing timestamp", uid, strings.Repeat("z", 30) + "-" + signature},
		{"negative timestamp", uid, "-1-" + signature},
		{"short signature", uid, timestamp + "-" + signature[:31]},
		{"modified signature", uid, timestamp + "-" + strings.Repeat("!", 32)},
		{"extra segment", uid, token + "-extra"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			valid, err := issuer.Verify(t.Context(), user, scenario.uid, scenario.token)
			require.NoError(t, err)
			require.False(t, valid)
		})
	}
}

func TestPasswordResetEnforcesTokenLifetime(t *testing.T) {
	issuer, user, uid, token, issuedAt := resetTokenFixture(t)
	for _, scenario := range []struct {
		name  string
		age   time.Duration
		valid bool
	}{
		{"issued now", 0, true},
		{"at expiry boundary", 72 * time.Hour, true},
		{"expired", 72*time.Hour + time.Second, false},
		{"future", -time.Second, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			issuer.now = func() time.Time { return issuedAt.Add(scenario.age) }
			valid, err := issuer.Verify(t.Context(), user, uid, token)
			require.NoError(t, err)
			require.Equal(t, scenario.valid, valid)
		})
	}
}

func TestPasswordResetTokenIsInvalidatedByAccountChanges(t *testing.T) {
	issuer, original, uid, token, issuedAt := resetTokenFixture(t)
	for _, scenario := range []struct {
		name   string
		update func(*store.User)
	}{
		{"password", func(user *store.User) { user.Password = "replacement-password-hash" }},
		{"email", func(user *store.User) { user.Email = "replacement@example.test" }},
		{"last login", func(user *store.User) { login := issuedAt; user.LastLogin = &login }},
		{"cleared last login", func(user *store.User) { user.LastLogin = nil }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			changed := original
			scenario.update(&changed)
			valid, err := issuer.Verify(t.Context(), changed, uid, token)
			require.NoError(t, err)
			require.False(t, valid)
		})
	}
}
