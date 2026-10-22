package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type sessionProtocolAuthorizerStub struct{ allowed bool }

func (s sessionProtocolAuthorizerStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	if s.allowed {
		return 0, nil
	}
	return 401, map[string]any{"detail": "Authentication credentials were not provided."}
}
func sessionProtocolTestHandler(t *testing.T, authenticated bool) http.Handler {
	t.Helper()
	return sessionProtocolTestHandlerWithOptions(t, authenticated, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})
}
func sessionProtocolTestHandlerWithOptions(t *testing.T, authenticated bool, options apiV1CompatibilityOptions) http.Handler {
	t.Helper()
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	stack := sessionServiceMiddlewareStack(options, nil, djangoSessionCookieConfig{})
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	handler := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{authenticated}, policy, stack, djangoSessionCookieConfig{}, true, options)(next)
	return readServicesCompatibilityMiddleware(handler)
}
func TestSessionServiceProtocolAuthenticatesMetadataAndUnsupportedMethods(t *testing.T) {
	for _, test := range []struct {
		name, method, path string
		authenticated      bool
		status             int
		detail             string
	}{
		{"anonymous metadata", http.MethodOptions, "/api/workspaces/demo/users/notifications/", false, 401, "Authentication credentials were not provided."},
		{"anonymous unsupported", http.MethodPut, "/api/workspaces/demo/users/notifications/", false, 401, "Authentication credentials were not provided."},
		{"authenticated metadata", http.MethodOptions, "/api/workspaces/demo/users/notifications/", true, 200, ""},
		{"authenticated unsupported", http.MethodPut, "/api/workspaces/demo/users/notifications/", true, 405, `Method "PUT" not allowed.`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			sessionProtocolTestHandler(t, test.authenticated).ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if test.detail != "" {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["detail"] != test.detail {
					t.Fatalf("body=%v", body)
				}
			}
		})
	}
}

func TestAccountAuthenticationSlashlessRedirectMatchesCommonMiddlewareOrdering(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}
	stack := sessionServiceMiddlewareStack(options, nil, djangoSessionCookieConfig{})
	handler := newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		accountAuthenticationSessionProtocolDispatch{account: authenticatedAccountProtocolStub{}},
		policy, stack, djangoSessionCookieConfig{}, false, options,
	)(http.NotFoundHandler())
	handler = readServicesCompatibilityMiddleware(handler)

	for _, method := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
	} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/auth/email-check?email=new%40example.com", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/auth/email-check/?email=new%40example.com" {
				t.Fatalf("slashless %s = %d location=%q body=%s", method, response.Code, response.Header().Get("Location"), response.Body.String())
			}
			if response.Header().Get("X-Frame-Options") != "" {
				t.Fatalf("slash redirect ran frame middleware before CommonMiddleware: %v", response.Header())
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
				t.Fatalf("slash redirect is missing source outer middleware headers: %v", response.Header())
			}
		})
	}

	debugHandler := newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		accountAuthenticationSessionProtocolDispatch{account: authenticatedAccountProtocolStub{}},
		policy, stack, djangoSessionCookieConfig{}, true, options,
	)(http.NotFoundHandler())
	debugResponse := httptest.NewRecorder()
	debugHandler.ServeHTTP(debugResponse, httptest.NewRequest(http.MethodPost, "/auth/email-check", strings.NewReader(`{"email":"a@example.com"}`)))
	if debugResponse.Code != http.StatusInternalServerError {
		t.Fatalf("DEBUG slashless POST = %d, want 500", debugResponse.Code)
	}
}

func TestNotificationActionProtocolUsesSourceMethods(t *testing.T) {
	for _, test := range []struct {
		path, method, wantPattern, wantAllow string
	}{
		{"/api/workspaces/demo/users/notifications/mark-all-read/", http.MethodPost, "/api/workspaces/{slug}/users/notifications/mark-all-read/", "POST, OPTIONS"},
		{"/api/workspaces/demo/users/notifications/mark-all-read/", http.MethodGet, "/api/workspaces/{slug}/users/notifications/mark-all-read/", "POST, OPTIONS"},
		{"/api/workspaces/demo/users/notifications/11111111-1111-4111-8111-111111111111/", http.MethodPatch, "/api/workspaces/{slug}/users/notifications/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS"},
	} {
		protocol, ok := resolveReadServiceProtocol(test.path)
		if !ok {
			t.Fatalf("no session protocol for %s", test.path)
		}
		if protocol.pattern != test.wantPattern || protocol.allow != test.wantAllow {
			t.Fatalf("%s %s resolved to pattern=%q allow=%q, want %q %q", test.method, test.path, protocol.pattern, protocol.allow, test.wantPattern, test.wantAllow)
		}
	}
}

func TestSessionServiceProtocolPreflightBypassesSessionAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/demo/users/notifications/", nil)
	request.Header.Set("Origin", "https://app.example.test")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response := httptest.NewRecorder()
	sessionProtocolTestHandler(t, false).ServeHTTP(response, request)
	if response.Code != 200 || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
		t.Fatalf("preflight=%d %v %s", response.Code, response.Header(), response.Body.String())
	}
}

func TestSessionServiceProtocolFinalizesSessionsAfterEarlyResponses(t *testing.T) {
	for _, test := range []struct {
		name, method, path, accept string
		status                     int
	}{
		{"redirect", http.MethodGet, "/api/workspaces/demo/users/notifications", "", 301},
		{"negotiation", http.MethodOptions, "/api/workspaces/demo/users/notifications/", "text/plain", 406},
		{"malformed UUID", http.MethodGet, "/api/workspaces/demo/users/notifications/not-a-uuid/", "", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{earlyRefresh: &djangosession.SessionRefresh{SessionKey: "refreshed", BrowserClose: true}}
			cookie := djangoSessionCookieConfig{Name: "session-id", Configured: true}
			policy, err := djangoRuntimeCORSPolicy(nil)
			if err != nil {
				t.Fatal(err)
			}
			options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}
			handler := readServicesDjangoPathConverterMiddleware(policy, &options)(sessionProtocolTestHandler(t, true))
			handler = activityReadsSessionFinalizationMiddleware(sessions, cookie)(handler)
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Accept", test.accept)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "current"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || !strings.Contains(response.Header().Get("Set-Cookie"), "session-id=refreshed") {
				t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
			}
		})
	}
}

func TestDedicatedAuthenticationRoutesDoNotRunReadServiceSessionFinalizer(t *testing.T) {
	for _, path := range []string{
		"/auth/forgot-password/",
		"/auth/change-password/",
		"/auth/sign-in/",
		"/auth/google/callback/",
		"/auth/reset-password/dXNlcg/token/",
		"/auth/spaces/reset-password/dXNlcg/token/",
	} {
		t.Run(path, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{}
			handler := activityReadsSessionFinalizationMiddleware(sessions, djangoSessionCookieConfig{Name: "session-id", Configured: true})(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusOK)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
			if response.Code != http.StatusOK || sessions.calls != 0 || sessions.earlyCalls != 0 {
				t.Fatalf("status=%d session calls=%d finalizations=%d", response.Code, sessions.calls, sessions.earlyCalls)
			}
		})
	}
}

func TestSessionServiceProtocolSlashlessSizeLimitPrecedesRedirect(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			handler := sessionProtocolTestHandlerWithOptions(t, false, apiV1CompatibilityOptions{bodySizeLimit: 1})
			request := httptest.NewRequest(method, "/api/workspaces/demo/users/notifications", strings.NewReader("oversized"))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) || response.Header().Get("Location") != "" {
				t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
			}
		})
	}
}
func TestSessionServiceProtocolSlashlessAuditRecordsInnerNotFound(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			logger := &apiV1ActivityLoggerStub{}
			handler := sessionProtocolTestHandlerWithOptions(t, false, apiV1CompatibilityOptions{activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit})
			const path = "/api/workspaces/demo/users/notifications"
			request := httptest.NewRequest(method, path, strings.NewReader("body"))
			request.Header.Set("X-Api-Key", "audit-only")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			wantStatus := http.StatusMovedPermanently
			if method == http.MethodPost {
				wantStatus = http.StatusInternalServerError
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if len(logger.records) != 1 {
				t.Fatalf("audit records=%v", logger.records)
			}
			record := logger.records[0]
			if record.Path != path || record.TokenIdentifier == "" || record.ResponseCode != http.StatusNotFound || record.ResponseBody != string(djangoNotFoundBody) {
				t.Fatalf("slashless audit=%#v", record)
			}
		})
	}
}
