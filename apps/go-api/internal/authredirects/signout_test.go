package authredirects

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/stretchr/testify/require"
)

type logoutAccounts struct {
	*accountStub
	failure    error
	lastLogout time.Time
}

func (accounts *logoutAccounts) UpdateLogoutMetadata(_ context.Context, _, _ string, at time.Time) error {
	if accounts.failure != nil {
		return accounts.failure
	}
	accounts.lastLogout = at
	return nil
}

type logoutSessionStore struct {
	sessionStore
	failure error
	rows    map[string]bool
}

func (store *logoutSessionStore) DeleteSession(_ context.Context, key string) error {
	if store.failure != nil {
		return store.failure
	}
	delete(store.rows, key)
	return nil
}

func TestSignOutReportsRevocationFailuresAndRevokesDespiteMetadataFailure(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, scenario := range []struct {
		name                       string
		run                        func(*Service, context.Context, Request, string) Result
		space                      bool
		deleteError, metadataError error
		status                     int
		revoked                    bool
		location                   string
	}{
		{name: "success", run: (*Service).SignOut, status: http.StatusFound, revoked: true, location: "https://app.example.test"},
		{name: "session deletion fails", run: (*Service).SignOut, deleteError: failure, status: http.StatusInternalServerError},
		{name: "metadata update fails", run: (*Service).SignOut, metadataError: failure, status: http.StatusInternalServerError, revoked: true},
		{name: "administration shares revocation failure handling", run: (*Service).AdminSignOut, deleteError: failure, status: http.StatusInternalServerError},
		{name: "space retains its redirect", run: (*Service).SignOut, space: true, status: http.StatusFound, revoked: true, location: "https://app.example.test/?next_path=%2Fshared%2F"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			accounts := &logoutAccounts{accountStub: &accountStub{}, failure: scenario.metadataError}
			store := &logoutSessionStore{failure: scenario.deleteError, rows: map[string]bool{"current": true, "other": true}}
			sessions, err := browsersession.New(store, browsersession.Config{SecretKey: "test-secret-key"})
			require.NoError(t, err)
			service, err := NewService(accounts, passwordStub{}, strengthStub(3), sessions, nil)
			require.NoError(t, err)
			now := time.Date(2026, time.November, 2, 12, 0, 0, 0, time.UTC)
			request := Request{BaseURL: "https://app.example.test", NextPath: "/shared/", AllowedHosts: []string{"app.example.test"}, IsSpace: scenario.space, Now: now,
				CurrentSession: &browsersession.SessionRefresh{SessionKey: "current"}}

			result := scenario.run(service, t.Context(), request, "user")

			require.Equal(t, scenario.status, result.Status)
			require.Equal(t, !scenario.revoked, store.rows["current"], "server session state must match the reported revocation")
			require.True(t, store.rows["other"], "another session must remain valid")
			require.Equal(t, scenario.revoked, result.ExpireSession)
			if scenario.status == http.StatusFound {
				require.Equal(t, scenario.location, result.Location)
				require.Equal(t, now, accounts.lastLogout)
			} else {
				require.Empty(t, result.Location, "failed logout must not redirect as success")
				require.True(t, accounts.lastLogout.IsZero(), "failed revocation must not record a completed logout")
			}
		})
	}
}

func TestAnonymousSignOutPreservesRedirectWithoutStorage(t *testing.T) {
	accounts := &logoutAccounts{accountStub: &accountStub{}, failure: errors.New("no user to update")}
	store := &logoutSessionStore{failure: errors.New("no session to revoke")}
	sessions, err := browsersession.New(store, browsersession.Config{SecretKey: "test-secret-key"})
	require.NoError(t, err)
	service, err := NewService(accounts, passwordStub{}, strengthStub(3), sessions, nil)
	require.NoError(t, err)
	request := Request{BaseURL: "https://app.example.test", NextPath: "/shared/", AllowedHosts: []string{"app.example.test"}}

	result := service.SignOut(t.Context(), request, "")
	require.Equal(t, Result{Status: http.StatusFound, Location: "https://app.example.test"}, result)
	request.IsSpace = true
	result = service.SignOut(t.Context(), request, "")
	require.Equal(t, Result{Status: http.StatusFound, Location: "https://app.example.test/?next_path=%2Fshared%2F"}, result)
	require.True(t, accounts.lastLogout.IsZero())
}
