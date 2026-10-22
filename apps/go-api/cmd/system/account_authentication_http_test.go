package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	loomhttp "github.com/CaliLuke/loom/http"
)

type accountAuthenticationProtocolFallbackStub struct {
	calls        int
	optionsCalls int
}

func (stub *accountAuthenticationProtocolFallbackStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 204, nil
}

func (stub *accountAuthenticationProtocolFallbackStub) AuthorizeSessionResourceProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any) {
	stub.calls++
	return 204, nil
}

func (stub *accountAuthenticationProtocolFallbackStub) SessionProtocolOptionsMetadata(context.Context, string, systemapi.SessionProtocolResource, json.RawMessage, json.RawMessage) (int, any, bool) {
	stub.optionsCalls++
	return http.StatusBadRequest, map[string]string{"error": "The required key does not exist."}, true
}

type accountAuthenticationProtocolAuthorizerStub struct{ calls int }

func (stub *accountAuthenticationProtocolAuthorizerStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	stub.calls++
	return 401, map[string]any{"detail": "Authentication credentials were not provided."}
}

type authenticatedAccountProtocolStub struct{}

func (authenticatedAccountProtocolStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (authenticatedAccountProtocolStub) AuthorizeSessionResourceProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any) {
	return 0, nil
}

type accountProtocolThrottleStub struct{ calls int }

func (accountProtocolThrottleStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (accountProtocolThrottleStub) AuthorizeSessionResourceProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any) {
	return 0, nil
}

func (stub *accountProtocolThrottleStub) AuthorizePublicProtocolRequest(context.Context) (int, map[string]any, http.Header) {
	stub.calls++
	headers := make(http.Header)
	headers.Set("Retry-After", "60")
	return http.StatusTooManyRequests, map[string]any{"error_code": 5900}, headers
}

func TestAccountAuthenticationProtocolsPreserveSessionAuthorizationBoundary(t *testing.T) {
	protocols := accountAuthenticationProtocols()
	if len(protocols) != 8 {
		t.Fatalf("account authentication protocols=%d, want 8", len(protocols))
	}
	for _, protocol := range protocols {
		if protocol.allow != "POST, OPTIONS" || !protocol.authenticateMethod || !protocol.throttleBeforeTimezone || !protocol.csrfForAuthenticatedSession {
			t.Fatalf("protocol %q has unexpected method policy: %#v", protocol.pattern, protocol)
		}
	}

	account := &accountAuthenticationProtocolAuthorizerStub{}
	fallback := &accountAuthenticationProtocolFallbackStub{}
	dispatch := accountAuthenticationSessionProtocolDispatch{account: account, fallback: fallback}

	for _, path := range accountAuthenticationProtocolPaths[:6] {
		status, _ := dispatch.AuthorizeSessionResourceProtocol(context.Background(), "", systemapi.SessionProtocolResource{Pattern: path})
		if status != 0 {
			t.Fatalf("public OPTIONS auth for %s = %d, want allow", path, status)
		}
	}
	if account.calls != 0 || fallback.calls != 0 {
		t.Fatalf("public endpoints reached an authorizer: account=%d fallback=%d", account.calls, fallback.calls)
	}

	for _, path := range accountAuthenticationProtocolPaths[6:] {
		status, body := dispatch.AuthorizeSessionResourceProtocol(context.Background(), "", systemapi.SessionProtocolResource{Pattern: path})
		if status != http.StatusUnauthorized || body["detail"] != "Authentication credentials were not provided." {
			t.Fatalf("private OPTIONS auth for %s = %d %v", path, status, body)
		}
	}
	if account.calls != 2 {
		t.Fatalf("private authorizer calls=%d, want 2", account.calls)
	}

	status, _ := dispatch.AuthorizeSessionResourceProtocol(context.Background(), "", systemapi.SessionProtocolResource{Pattern: "/api/workspaces/{slug}/webhooks/"})
	if status != 204 || fallback.calls != 1 {
		t.Fatalf("unrelated protocol dispatch status=%d fallback calls=%d", status, fallback.calls)
	}
}

func TestAccountAuthenticationOptionsUseAPIViewMetadataWithoutTimezoneLookup(t *testing.T) {
	fallback := &accountAuthenticationProtocolFallbackStub{}
	dispatch := accountAuthenticationSessionProtocolDispatch{fallback: fallback}
	for _, path := range accountAuthenticationProtocolPaths {
		status, body, handled := dispatch.SessionProtocolOptionsMetadata(
			context.Background(),
			"session-with-invalid-timezone",
			systemapi.SessionProtocolResource{Method: http.MethodOptions, Pattern: path},
			[]byte(`{"name":"source APIView metadata"}`),
			nil,
		)
		if handled || status != 0 || body != nil {
			t.Fatalf("auth OPTIONS metadata for %s = (%d, %v, %v), want unhandled APIView metadata", path, status, body, handled)
		}
	}
	if fallback.optionsCalls != 0 {
		t.Fatalf("fallback timezone metadata calls=%d, want 0 for auth APIViews", fallback.optionsCalls)
	}
	_, _, handled := dispatch.SessionProtocolOptionsMetadata(
		context.Background(), "session", systemapi.SessionProtocolResource{Method: http.MethodOptions, Pattern: "/api/workspaces/{slug}/webhooks/"}, nil, nil,
	)
	if !handled || fallback.optionsCalls != 1 {
		t.Fatalf("unrelated session metadata delegated=%v calls=%d, want one fallback call", handled, fallback.optionsCalls)
	}
}

func TestAccountAuthenticationAPIViewThrottleCoversOptionsAndUnsupportedMethods(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	account := &accountProtocolThrottleStub{}
	handler := newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		accountAuthenticationSessionProtocolDispatch{account: account},
		policy,
		sessionServiceMiddlewareStack(apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}, nil, djangoSessionCookieConfig{}),
		djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{},
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	handler = readServicesCompatibilityMiddleware(handler)

	for _, method := range []string{http.MethodOptions, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/auth/email-check/", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" {
				t.Fatalf("%s response=%d headers=%v body=%s", method, response.Code, response.Header(), response.Body.String())
			}
		})
	}
	if account.calls != 2 {
		t.Fatalf("protocol throttle calls=%d, want 2", account.calls)
	}

	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", strings.NewReader(`{"email":"a@example.com"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || account.calls != 2 {
		t.Fatalf("generated POST response=%d protocol throttle calls=%d; generated handler must not consume protocol throttle", response.Code, account.calls)
	}
}

func TestAccountAuthenticationCSRFBlocksAuthenticatedUnsafeRequests(t *testing.T) {
	secret := "abcdefghijklmnopqrstuvwxyzABCDEF"
	token, err := systemapi.NewCSRFToken().CsrfToken(context.Background(), &csrftoken.CsrfTokenPayload{CsrfCookie: &secret})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		cookie     string
		token      string
		origin     string
		referer    string
		trusted    []string
		requestURL string
		status     int
		wantCalled bool
	}{
		{name: "missing cookie", status: http.StatusForbidden},
		{name: "valid same-origin token", cookie: secret, token: token.CsrfToken, origin: "https://app.example.test", status: http.StatusNoContent, wantCalled: true},
		{name: "same-origin referer path and query", cookie: secret, token: token.CsrfToken, referer: "https://app.example.test/settings/profile?tab=security", status: http.StatusNoContent, wantCalled: true},
		{name: "trusted-origin referer path and query", cookie: secret, token: token.CsrfToken, referer: "https://trusted.example.test/settings/profile?tab=security", trusted: []string{"https://trusted.example.test"}, status: http.StatusNoContent, wantCalled: true},
		{name: "trusted HTTPS origin on HTTP request", cookie: secret, token: token.CsrfToken, origin: "https://trusted.example.test", trusted: []string{"https://trusted.example.test"}, requestURL: "http://app.example.test/auth/change-password/", status: http.StatusNoContent, wantCalled: true},
		{name: "wrong origin", cookie: secret, token: token.CsrfToken, origin: "https://attacker.example.test", status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				called = true
				response.WriteHeader(http.StatusNoContent)
			})
			handler := accountAuthenticationCSRFMiddleware(test.trusted)(next)
			requestURL := test.requestURL
			if requestURL == "" {
				requestURL = "https://app.example.test/auth/change-password/"
			}
			request := httptest.NewRequest(http.MethodPost, requestURL, strings.NewReader(`[]`))
			request = request.WithContext(systemapi.WithReadServiceSessionPrincipal(request.Context(), "user-1", "UTC"))
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: test.cookie})
			}
			if test.token != "" {
				request.Header.Set("X-CSRFToken", test.token)
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if test.referer != "" {
				request.Header.Set("Referer", test.referer)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || called != test.wantCalled {
				t.Fatalf("status=%d called=%v body=%s, want status=%d called=%v", response.Code, called, response.Body.String(), test.status, test.wantCalled)
			}
			if test.status == http.StatusForbidden && !strings.Contains(response.Body.String(), "CSRF Failed:") {
				t.Fatalf("CSRF rejection body=%s", response.Body.String())
			}
		})
	}
}

func TestAccountAuthenticationUnsupportedUnsafeMethodChecksCSRFBefore405(t *testing.T) {
	secret := "abcdefghijklmnopqrstuvwxyzABCDEF"
	token, err := systemapi.NewCSRFToken().CsrfToken(context.Background(), &csrftoken.CsrfTokenPayload{CsrfCookie: &secret})
	if err != nil {
		t.Fatal(err)
	}
	protocol := newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		accountAuthenticationSessionProtocolDispatch{account: authenticatedAccountProtocolStub{}},
		loomhttp.RuntimeCORSPolicy{},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				ctx := systemapi.WithReadServiceSessionPrincipal(request.Context(), "authenticated-user", "UTC")
				next.ServeHTTP(response, request.WithContext(ctx))
			})
		},
		djangoSessionCookieConfig{Name: "session-id", Configured: true},
		false,
		apiV1CompatibilityOptions{},
		[]string{"https://trusted.example.test"},
	)(http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodPut, "https://app.example.test/auth/change-password/", strings.NewReader(`[]`))
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: secret})
	request.Header.Set("X-CSRFToken", token.CsrfToken)
	request.Header.Set("Referer", "https://trusted.example.test/settings/security?from=profile")
	response := httptest.NewRecorder()
	protocol.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("trusted-origin Referer unsupported method status=%d body=%s, want 405 after CSRF validation", response.Code, response.Body.String())
	}
}

func TestAccountAuthenticationPublicUnsupportedMethodChecksCSRFOnlyForSessionUser(t *testing.T) {
	for _, test := range []struct {
		name          string
		authenticated bool
		wantStatus    int
	}{
		{name: "anonymous gets method not allowed", wantStatus: http.StatusMethodNotAllowed},
		{name: "authenticated gets CSRF failure first", authenticated: true, wantStatus: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			stack := func(next http.Handler) http.Handler {
				if !test.authenticated {
					return next
				}
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					ctx := systemapi.WithReadServiceSessionPrincipal(request.Context(), "authenticated-user", "UTC")
					next.ServeHTTP(response, request.WithContext(ctx))
				})
			}
			handler := newSessionServiceProtocolMiddleware(
				notificationProtocolMetadata,
				accountAuthenticationSessionProtocolDispatch{account: authenticatedAccountProtocolStub{}},
				loomhttp.RuntimeCORSPolicy{}, stack,
				djangoSessionCookieConfig{}, false, apiV1CompatibilityOptions{},
			)(http.NotFoundHandler())
			request := httptest.NewRequest(http.MethodPut, "https://app.example.test/auth/email-check/", strings.NewReader(`[]`))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
		})
	}
}
