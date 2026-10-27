package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"

	api "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestMCPLinkBoundaryRequiresSessionCSRFExceptExactProviderCallback(t *testing.T) {
	for _, test := range []struct {
		name, path, method, body, contentType, origin string
		csrf                                          bool
		want                                          int
	}{
		{"status", "/api/users/me/mcp-link/", "GET", "", "", "", false, 204},
		{"start requires csrf", "/api/users/me/mcp-link/start/", "POST", "", "", "", false, 403},
		{"confirm requires csrf", "/api/users/me/mcp-link/confirm/", "POST", "{}", "application/json", "https://vector.example", false, 403},
		{"disconnect requires csrf", "/api/users/me/mcp-link/", "DELETE", "", "", "https://vector.example", false, 403},
		{"same origin", "/api/users/me/mcp-link/start/", "POST", "", "", "https://vector.example", true, 204},
		{"foreign origin", "/api/users/me/mcp-link/start/", "POST", "", "", "https://foreign.example", true, 403},
		{"provider form", "/auth/mcp/callback/", "POST", "state=state&code=code", "application/x-www-form-urlencoded", "https://issuer.example", false, 204},
		{"duplicate state", "/auth/mcp/callback/", "POST", "state=one&state=two&code=secret", "application/x-www-form-urlencoded", "", false, 400},
		{"duplicate code", "/auth/mcp/callback/", "POST", "state=one&code=secret&code=two", "application/x-www-form-urlencoded", "", false, 400},
		{"callback query", "/auth/mcp/callback/?code=secret", "POST", "state=one", "application/x-www-form-urlencoded", "", false, 400},
		{"callback json", "/auth/mcp/callback/", "POST", "{}", "application/json", "", false, 415},
		{"callback suffix is not exempt", "/auth/mcp/callback/extra", "POST", "", "", "", false, 403},
		{"callback alternate method is not exempt", "/auth/mcp/callback/", "DELETE", "", "", "", false, 403},
		{"oversize", "/auth/mcp/callback/", "POST", "code=" + strings.Repeat("s", mcpLinkBodyLimit), "application/x-www-form-urlencoded", "", false, 413},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := mcpOAuthLinkBoundary("https://vector.example/mcp", nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
			r := httptest.NewRequest(test.method, "https://vector.example"+test.path, strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.contentType)
			if test.origin != "" {
				r.Header.Set("Origin", test.origin)
			}
			if test.csrf {
				token := strings.Repeat("a", 32)
				r.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: token})
				r.Header.Set("X-CSRFToken", token)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, test.want, w.Code)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			require.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
			require.NotContains(t, w.Body.String(), "secret")
			require.NotContains(t, w.Body.String(), "foreign.example")
		})
	}
}

func TestMCPLinkRejectsMalformedOrUnboundedCallbackStreams(t *testing.T) {
	for _, test := range []struct {
		name, target, body, contentType string
		want                            int
	}{
		{"chunked oversize", "https://vector.example/auth/mcp/callback/", "code=" + strings.Repeat("s", mcpLinkBodyLimit), "application/x-www-form-urlencoded", 413},
		{"invalid escape", "https://vector.example/auth/mcp/callback/", "code=private%ZZ", "application/x-www-form-urlencoded", 400},
		{"invalid content type", "https://vector.example/auth/mcp/callback/", "code=private", "invalid;=", 415},
		{"wrong host", "https://other.example/auth/mcp/callback/", "code=private", "application/x-www-form-urlencoded", 400},
		{"empty query", "https://vector.example/auth/mcp/callback/?", "code=private", "application/x-www-form-urlencoded", 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := mcpOAuthLinkBoundary("https://vector.example/mcp", nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("invalid request reached the application")
			}))
			r := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(test.body))
			r.ContentLength = -1
			r.Header.Set("Content-Type", test.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, test.want, w.Code)
			require.NotContains(t, w.Body.String(), "private")
		})
	}
}

func TestMCPLinkProductionErrorsKeepStatusAndDiscardPrivateDetails(t *testing.T) {
	for _, status := range []int{400, 401, 403, 409, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			endpoint := &api.Endpoints{Status: func(context.Context, any) (any, error) {
				return nil, &api.LinkError{Kind: "status_" + strconv.Itoa(status), Message: "private-provider-response"}
			}}
			w := httptest.NewRecorder()
			mcpLinkProductionHandler(endpoint).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://vector.example/api/users/me/mcp-link/", nil))
			require.Equal(t, status, w.Code)
			require.NotContains(t, w.Body.String(), "private-provider-response")
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		})
	}
	endpoint := &api.Endpoints{Status: func(context.Context, any) (any, error) {
		return nil, errors.New("private-database-response")
	}}
	w := httptest.NewRecorder()
	mcpLinkProductionHandler(endpoint).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://vector.example/api/users/me/mcp-link/", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "private-database-response")
}

type mcpLinkFailedWriter struct{ http.ResponseWriter }

func (mcpLinkFailedWriter) Write([]byte) (int, error) {
	return 0, errors.New("private-response-write-error")
}

func TestMCPLinkResponseWriteFailureIsSanitizedBeforeLogging(t *testing.T) {
	endpoints := &api.Endpoints{Status: func(context.Context, any) (any, error) {
		return &api.LinkStatus{Issuer: "https://issuer.example"}, nil
	}}
	mux := loomhttp.NewMuxer()
	var reported error
	cors, err := djangoRuntimeCORSPolicy(nil)
	require.NoError(t, err)
	mountMCPOAuthLinks(mux, pendingReplacementEndpoints{mcpLinks: endpoints, mcpResourceURL: "https://vector.example/mcp"}, cors,
		func(_ context.Context, _ http.ResponseWriter, failure error) { reported = failure })
	mux.ServeHTTP(mcpLinkFailedWriter{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, "https://vector.example/api/users/me/mcp-link/", nil))
	require.Error(t, reported)
	require.NotContains(t, reported.Error(), "private-response-write-error")
}

func TestMCPLinkGeneratedValidationDoesNotEchoCallbackValues(t *testing.T) {
	endpoints := &api.Endpoints{Callback: func(context.Context, any) (any, error) {
		t.Fatal("invalid callback reached application")
		return nil, nil
	}}
	handler := mcpLinkProductionHandler(endpoints)
	r := httptest.NewRequest("POST", "https://vector.example/auth/mcp/callback/", strings.NewReader("state=private-short-state&code=private-code"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	require.Equal(t, 400, w.Code)
	require.NotContains(t, w.Body.String(), "private")
	require.Contains(t, w.Body.String(), "Invalid account linking request.")
}

func mcpLinkProductionHandler(endpoints *api.Endpoints) http.Handler {
	return newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{mcpLinks: endpoints, mcpResourceURL: "https://vector.example/mcp", activityReadsBodyLimit: 1 << 20})
}

func TestMCPLinkProductionMountAcceptsProviderFormWithoutSessionCookie(t *testing.T) {
	var received *api.CallbackPayload
	endpoint := &api.Endpoints{Callback: func(_ context.Context, payload any) (any, error) {
		received = payload.(*api.CallbackPayload)
		return &api.CallbackResult{Location: "https://vector.example/settings/profile/api-tokens/"}, nil
	}}
	handler := mcpLinkProductionHandler(endpoint)
	request := httptest.NewRequest("POST", "https://vector.example/auth/mcp/callback/", strings.NewReader("state="+strings.Repeat("a", 43)+"&code=one-time-proof"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://issuer.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, 303, response.Code, response.Body.String())
	require.NotNil(t, received)
	require.Equal(t, "one-time-proof", *received.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", response.Header().Get("Referrer-Policy"))
}

func TestDisabledMCPLinkRoutesDoNotExposeApplicationEndpoints(t *testing.T) {
	handler := mcpLinkProductionHandler(nil)
	for _, path := range []string{"/api/users/me/mcp-link/", "/api/users/me/mcp-link/start/", "/api/users/me/mcp-link/confirm/", "/auth/mcp/callback/"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", "https://vector.example"+path, nil))
		require.Equal(t, 404, response.Code, path)
	}
}
