package systemapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "github.com/CaliLuke/go-plane/gen/http/instance_admin_session/server"
	api "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestInstanceAdminSessionStoreFailuresRemainFailures(t *testing.T) {
	for _, cause := range []error{
		djangosession.ErrSessionStore, djangosession.ErrUserStore, djangosession.ErrInvalidSessionShape, djangosession.ErrInvalidUserID,
		fmt.Errorf("%w: %w: database unavailable", djangosession.ErrStore, djangosession.ErrSessionStore),
		fmt.Errorf("%w: %w: %w", djangosession.ErrStore, djangosession.ErrUserStore, context.Canceled),
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{}, instanceAdminSessionStub{err: cause})
			require.NoError(t, err)
			result, err := service.Session(t.Context(), &api.SessionPayload{SessionID: strptr("opaque-cookie")})
			require.Nil(t, result, "a failed session lookup must not look like a successful logout")
			var internal *api.InstanceAdminSessionInternalError
			require.ErrorAs(t, err, &internal)
			require.False(t, strings.Contains(err.Error(), "database"))
			require.False(t, strings.Contains(err.Error(), "opaque-cookie"))
			_, authErr := service.(api.Authorizer).APIKeyAuth(t.Context(), "opaque-cookie", nil)
			require.ErrorAs(t, authErr, &internal, "store failure must not be reported as invalid credentials")
		})
	}
}

func TestInstanceAdminSessionInvalidCredentialsRemainAnonymous(t *testing.T) {
	for _, cause := range []error{djangosession.ErrSessionNotFound, djangosession.ErrBadSignature, djangosession.ErrInactiveUser, errors.New("invalid session")} {
		t.Run(cause.Error(), func(t *testing.T) {
			service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{}, instanceAdminSessionStub{err: cause})
			require.NoError(t, err)
			result, err := service.Session(t.Context(), &api.SessionPayload{SessionID: strptr("invalid-cookie")})
			require.NoError(t, err)
			require.False(t, result.IsAuthenticated)
			require.Nil(t, result.User)
			_, err = service.(api.Authorizer).APIKeyAuth(t.Context(), "invalid-cookie", nil)
			var unauthenticated *api.InstanceAdminSessionUnauthenticated
			require.ErrorAs(t, err, &unauthenticated)
		})
	}
}

func TestInstanceAdminSessionStoreFailureHTTP(t *testing.T) {
	service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{}, instanceAdminSessionStub{err: fmt.Errorf("%w: private storage diagnostic", djangosession.ErrStore)})
	require.NoError(t, err)
	mux := loomhttp.NewMuxer()
	handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux, handler)
	for _, path := range []string{"/api/instances/admins/session/", "/api/instances/admins/me/"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "opaque-cookie"})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusInternalServerError, response.Code)
			require.JSONEq(t, `{"error":"Something went wrong please try again later"}`, response.Body.String())
		})
	}
}

func TestInstanceAdminSessionMalformedActorRemainsAnonymous(t *testing.T) {
	service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{}, instanceAdminSessionStub{principal: djangosession.Principal{UserID: "not-a-uuid"}})
	require.NoError(t, err)
	result, err := service.Session(t.Context(), &api.SessionPayload{SessionID: strptr("invalid-actor")})
	require.NoError(t, err)
	require.False(t, result.IsAuthenticated)
	require.Nil(t, result.User)
	_, err = service.(api.Authorizer).APIKeyAuth(t.Context(), "invalid-actor", nil)
	var denied *api.InstanceAdminSessionUnauthenticated
	require.ErrorAs(t, err, &denied)
}
