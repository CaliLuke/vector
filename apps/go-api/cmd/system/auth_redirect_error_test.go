package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	redirectapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type authRedirectUnavailableAccounts struct {
	authredirects.AccountRepository
}

func (authRedirectUnavailableAccounts) ConfigValue(_ context.Context, _, fallback string) (string, error) {
	return fallback, nil
}

func (authRedirectUnavailableAccounts) InstanceConfigured(context.Context) (bool, error) {
	return false, context.DeadlineExceeded
}

type unusedAuthRedirectPasswords struct {
	authredirects.PasswordHasher
	authredirects.PasswordStrength
}

func TestAuthRedirectFailureRendersHTMLAndRecordsStage(t *testing.T) {
	credentials := unusedAuthRedirectPasswords{}
	core, err := authredirects.NewService(authRedirectUnavailableAccounts{}, credentials, credentials, &browsersession.Validator{}, nil)
	require.NoError(t, err)
	service, err := systemapi.NewAuthRedirects(core)
	require.NoError(t, err)
	pending := pendingReplacementEndpoints{authRedirects: redirectapi.NewEndpoints(service)}
	mux := loomhttp.NewMuxer()
	mountAuthRedirectsRoutes(t.Context(), pending, mux, loomhttp.RuntimeCORSPolicy{})

	var output bytes.Buffer
	ctx := newLogContext(t.Context(), &output, false)
	handler := log.HTTP(ctx, log.WithDisableRequestID(), log.WithDisableRequestLogging())(
		loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(nativePlainViewCSRFMiddleware(nil, defaultFileSizeLimit)(mux)),
	)
	csrf := strings.Repeat("a", 32)
	request := httptest.NewRequest(http.MethodPost, "/auth/sign-in/", strings.NewReader(
		"email=alice%40example.test&password=secret-fixture&csrfmiddlewaretoken="+csrf,
	))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: csrf})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Contains(t, response.Header().Get("Content-Type"), "text/html")
	require.Contains(t, response.Body.String(), "<h1>Authentication unavailable</h1>")
	require.Empty(t, response.Header().Values("Set-Cookie"))
	require.Empty(t, response.Header().Get("Location"))
	require.NotContains(t, response.Body.String(), "secret-fixture")
	require.Contains(t, output.String(), `"auth.stage":"read_instance_status"`)
	require.Contains(t, output.String(), `"auth.error_class":"deadline_exceeded"`)
	require.NotContains(t, output.String(), "secret-fixture")
	require.NotContains(t, output.String(), "encode HTTP response")
}
