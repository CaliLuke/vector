package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	adminapi "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	api "github.com/CaliLuke/go-plane/gen/instance_management"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	store "github.com/CaliLuke/go-plane/internal/instancemanagement"
)

type instanceHTTPRepository struct {
	*store.Repository
	denied   bool
	recorder *cycleMiddlewareRecorder
}

func (r instanceHTTPRepository) IsAdmin(context.Context, uuid.UUID) (bool, error) {
	if r.recorder != nil {
		r.recorder.add("permission-classes")
	}
	return !r.denied, nil
}
func (instanceHTTPRepository) Root(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{"is_activated":false,"is_setup_done":false}`), nil
}
func (r instanceHTTPRepository) ListAdmins(context.Context) (json.RawMessage, error) {
	if r.recorder != nil {
		r.recorder.add("dispatch")
	}
	return json.RawMessage(`[]`), nil
}

type instanceHTTPSessions struct{ *cycleMiddlewareSessions }

func (s instanceHTTPSessions) InspectReadServiceSession(ctx context.Context, r *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	cookie, err := r.Cookie("admin-session-id")
	if err != nil {
		return false, false, nil, djangosession.Principal{}, nil
	}
	principal, err := s.Validate(ctx, cookie.Value)
	if err != nil {
		return true, false, nil, djangosession.Principal{}, nil
	}
	return false, false, nil, principal, nil
}
func (s instanceHTTPSessions) InspectReadServiceSessionWithoutTimezone(ctx context.Context, r *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	return s.InspectReadServiceSession(ctx, r)
}

type instanceHTTPDependencies struct{ t *testing.T }

type instanceHTTPThrottle struct {
	calls int
	limit int
	wait  time.Duration
	err   error
}

func (throttle *instanceHTTPThrottle) AllowAnonymousRequest(_ context.Context, _ *http.Request) (bool, time.Duration, error) {
	throttle.calls++
	if throttle.err != nil {
		return false, 0, throttle.err
	}
	return throttle.calls <= throttle.limit, throttle.wait, nil
}

func (d instanceHTTPDependencies) Encrypt(string) (string, error) {
	d.t.Fatal("unexpected encryption")
	return "", nil
}
func (d instanceHTTPDependencies) Decrypt(string) (string, error) {
	d.t.Fatal("unexpected decryption")
	return "", nil
}
func (d instanceHTTPDependencies) Send(context.Context, instancemail.SMTPConfig, instancemail.Message) error {
	d.t.Fatal("unexpected email")
	return nil
}
func (instanceHTTPDependencies) Invalidate(context.Context, string, string, bool) error { return nil }

func newInstanceManagementTestHandler(t *testing.T, denied bool) http.Handler {
	t.Helper()
	return newInstanceManagementMiddlewareTestHandler(t, denied, "UTC", 1<<20, &cycleMiddlewareRecorder{}, nil)
}

func newInstanceManagementMiddlewareTestHandler(t *testing.T, denied bool, timezone string, limit int64, recorder *cycleMiddlewareRecorder, audit *cycleMiddlewareActivity) http.Handler {
	return newInstanceManagementMiddlewareTestHandlerWithThrottle(t, denied, timezone, limit, recorder, audit, nil)
}

func newInstanceManagementMiddlewareTestHandlerWithThrottle(t *testing.T, denied bool, timezone string, limit int64, recorder *cycleMiddlewareRecorder, audit *cycleMiddlewareActivity, throttle instanceManagementAnonymousThrottle) http.Handler {
	t.Helper()
	if audit == nil {
		audit = &cycleMiddlewareActivity{recorder: recorder}
	}
	sessions := instanceHTTPSessions{&cycleMiddlewareSessions{recorder: recorder, userTimezone: timezone}}
	dependency := instanceHTTPDependencies{t}
	service, err := systemapi.NewInstanceManagement(instanceHTTPRepository{denied: denied, recorder: recorder}, sessions, dependency, dependency, dependency, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	legacy := adminapi.NewEndpoints(&instanceAdminSessionHTTPStub{authenticated: true})
	return newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, legacy, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{instanceManagement: api.NewEndpoints(service), instanceManagementMetadata: service.(sessionResourceProtocolAuthorizer), instanceManagementThrottle: throttle, readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, publicProjectMetadataAdminCookie: djangoSessionCookieConfig{Name: "admin-session-id", Configured: true}, activityReadsActivity: audit, activityReadsBodyLimit: limit})
}

func TestInstanceManagementAnonymousThrottleSharedRootAndSignupBudget(t *testing.T) {
	throttle := &instanceHTTPThrottle{limit: 30, wait: 17 * time.Second}
	handler := newInstanceManagementMiddlewareTestHandlerWithThrottle(t, false, "UTC", 1<<20, &cycleMiddlewareRecorder{}, nil, throttle)
	for index := 0; index < 30; index++ {
		request := httptest.NewRequest(http.MethodGet, "/api/instances/", nil)
		request.RemoteAddr = "192.0.2.55:4000"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("root request %d status=%d body=%s", index+1, response.Code, response.Body.String())
		}
	}
	blocked := httptest.NewRequest(http.MethodPost, "/api/instances/admins/sign-up-screen-visited/", strings.NewReader(`{}`))
	blocked.RemoteAddr = "192.0.2.55:4000"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, blocked)
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "17" || response.Body.String() != `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}` {
		t.Fatalf("shared anonymous budget response=%d retry=%q body=%s", response.Code, response.Header().Get("Retry-After"), response.Body.String())
	}
	if throttle.calls != 31 {
		t.Fatalf("anonymous throttle calls=%d want=31", throttle.calls)
	}
}

func TestInstanceManagementAnonymousThrottleAuthenticatedBypassAndPrivatePrecedence(t *testing.T) {
	throttle := &instanceHTTPThrottle{limit: 0}
	handler := newInstanceManagementMiddlewareTestHandlerWithThrottle(t, false, "UTC", 1<<20, &cycleMiddlewareRecorder{}, nil, throttle)
	request := httptest.NewRequest(http.MethodGet, "/api/instances/", nil)
	request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || throttle.calls != 0 {
		t.Fatalf("authenticated root response=%d calls=%d body=%s", response.Code, throttle.calls, response.Body.String())
	}
	private := httptest.NewRequest(http.MethodGet, "/api/instances/admins/", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, private)
	if response.Code != http.StatusUnauthorized || throttle.calls != 0 {
		t.Fatalf("private endpoint response=%d throttle calls=%d body=%s", response.Code, throttle.calls, response.Body.String())
	}
	patch := httptest.NewRequest(http.MethodPatch, "/api/instances/", strings.NewReader(`{}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, patch)
	if response.Code != http.StatusUnauthorized || throttle.calls != 0 {
		t.Fatalf("root PATCH response=%d throttle calls=%d body=%s", response.Code, throttle.calls, response.Body.String())
	}
}

func TestInstanceManagementAnonymousThrottleRequestSelection(t *testing.T) {
	for _, testCase := range []struct {
		method, path string
		want         bool
	}{
		{http.MethodGet, "/api/instances/", true},
		{http.MethodHead, "/api/instances/", true},
		{http.MethodOptions, "/api/instances/", true},
		{http.MethodPost, "/api/instances/", true},
		{http.MethodPatch, "/api/instances/", false},
		{http.MethodGet, "/api/instances/admins/sign-up-screen-visited/", true},
		{http.MethodOptions, "/api/instances/admins/sign-up-screen-visited/", true},
		{http.MethodGet, "/api/instances/admins/", false},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		if got := instanceManagementAnonymousThrottleRequest(request); got != testCase.want {
			t.Errorf("%s %s selected=%v want=%v", testCase.method, testCase.path, got, testCase.want)
		}
	}
}

func TestInstanceManagementPlainRedirectProtocolMatchesDjangoHeaders(t *testing.T) {
	handler := readServicesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusFound)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/instances/admins/sign-in/", strings.NewReader("email=x"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("supported plain-view status=%d", response.Code)
	}
	if response.Header().Get("Allow") != "" {
		t.Fatalf("successful plain-view response unexpectedly advertises Allow=%q", response.Header().Get("Allow"))
	}
	if response.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" || response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("security headers=%v", response.Header())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/instances/admins/sign-in/", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatalf("unsupported plain-view response=%d Allow=%q body=%s", response.Code, response.Header().Get("Allow"), response.Body.String())
	}
}

func TestInstanceManagementAdminRedirectUsesPlainDjangoGzipContract(t *testing.T) {
	handler := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(response, strings.Repeat("x", 300))
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/instances/admins/sign-in/", strings.NewReader("email=x"))
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "" {
		t.Fatalf("plain view response=%d encoding=%q", response.Code, response.Header().Get("Content-Encoding"))
	}
	for _, value := range response.Header().Values("Vary") {
		if strings.Contains(strings.ToLower(value), "accept-encoding") {
			t.Fatalf("plain view unexpectedly varies on Accept-Encoding: %q", value)
		}
	}
}

func TestInstanceManagementMiddlewareAuthorizationTimezoneDispatchAudit(t *testing.T) {
	for _, tc := range []struct {
		name, body, timezone  string
		authenticated, denied bool
		status                int
		events                []string
	}{
		{"request-size before authentication", strings.Repeat("x", 65), "UTC", true, false, 413, nil},
		{"authentication before authorization", "", "UTC", false, false, 401, nil},
		{"permission-classes before invalid timezone", "", "Not/AZone", true, true, 403, []string{"session-authentication", "permission-classes"}},
		{"timezone before dispatch", "", "Not/AZone", true, false, 400, []string{"session-authentication", "permission-classes"}},
		{"authorized dispatch", "", "UTC", true, false, 200, []string{"session-authentication", "permission-classes", "dispatch"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			audit := &cycleMiddlewareActivity{recorder: recorder}
			handler := newInstanceManagementMiddlewareTestHandler(t, tc.denied, tc.timezone, 64, recorder, audit)
			request := httptest.NewRequest(http.MethodGet, "/api/instances/admins/", strings.NewReader(tc.body))
			if tc.authenticated {
				request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || !slices.Equal(recorder.events, tc.events) {
				t.Fatalf("status=%d events=%v body=%s; want%d %v", response.Code, recorder.events, response.Body.String(), tc.status, tc.events)
			}
			if len(audit.records) != 0 {
				t.Fatalf("admin-session request emitted API-key audit: %v", audit.records)
			}
		})
	}
}

func TestInstanceManagementAdminCookieAndProtectedCompanions(t *testing.T) {
	const detail = "/api/instances/admins/11111111-1111-4111-8111-111111111111/"
	for _, tc := range []struct {
		name, method, path, cookie, detail string
		denied                             bool
		status                             int
	}{
		{name: "public root", method: "GET", path: "/api/instances/", status: 200},
		{name: "application session cannot administer", method: "GET", path: "/api/instances/admins/", cookie: "session-id=valid-session", status: 401},
		{name: "admin session", method: "GET", path: "/api/instances/admins/", cookie: "admin-session-id=valid-session", status: 200},
		{name: "role before malformed write", method: "POST", path: "/api/instances/admins/", cookie: "admin-session-id=valid-session", denied: true, status: 403},
		{name: "authentication before malformed write", method: "POST", path: "/api/instances/admins/", status: 401},
		{name: "metadata requires admin", method: "OPTIONS", path: "/api/instances/admins/", cookie: "admin-session-id=valid-session", denied: true, status: 403},
		{name: "disable email anonymous", method: "DELETE", path: "/api/instances/configurations/disable-email-feature/", status: 401, detail: "Authentication credentials were not provided."},
		{name: "disable email non-admin", method: "DELETE", path: "/api/instances/configurations/disable-email-feature/", cookie: "admin-session-id=valid-session", denied: true, status: 403, detail: "You do not have permission to perform this action."},
		{name: "workspaces get anonymous", method: "GET", path: "/api/instances/workspaces/", status: 401, detail: "Authentication credentials were not provided."},
		{name: "workspaces get non-admin", method: "GET", path: "/api/instances/workspaces/", cookie: "admin-session-id=valid-session", denied: true, status: 403, detail: "You do not have permission to perform this action."},
		{name: "detail post anonymous", method: "POST", path: detail, status: 401},
		{name: "detail post denied", method: "POST", path: detail, cookie: "admin-session-id=valid-session", denied: true, status: 403},
		{name: "detail post authenticated405", method: "POST", path: detail, cookie: "admin-session-id=valid-session", status: 405},
		{name: "existing me literal", method: "GET", path: "/api/instances/admins/me/", cookie: "admin-session-id=valid-session", status: 200},
		{name: "existing session literal", method: "GET", path: "/api/instances/admins/session/", cookie: "admin-session-id=valid-session", status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newInstanceManagementTestHandler(t, tc.denied)
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Cookie", tc.cookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
			if tc.detail != "" {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode error body: %v; body=%s", err, response.Body.String())
				}
				if body["detail"] != tc.detail {
					t.Fatalf("detail=%v want=%q body=%s", body["detail"], tc.detail, response.Body.String())
				}
			}
		})
	}
}

func TestInstanceManagementExpiredCookieDoesNotExpireApplicationCookie(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/instances/admins/", nil)
	request.Header.Set("Cookie", "session-id=valid-session; admin-session-id=expired-session")
	response := httptest.NewRecorder()
	newInstanceManagementTestHandler(t, false).ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "admin-session-id" || cookies[0].MaxAge != -1 {
		t.Fatalf("expired cookies=%v", cookies)
	}
}

func TestInstanceAdminLiteralPathsRemainOutsideGenericDetailProtocol(t *testing.T) {
	for _, segment := range []string{"me", "session"} {
		if protocol, ok := resolveReadServiceProtocol("/api/instances/admins/" + segment + "/"); ok {
			t.Errorf("literal %s claimed by %s", segment, protocol.pattern)
		}
	}
}

func TestInstanceAdminRedirectProtocolMethods(t *testing.T) {
	handler := newInstanceManagementTestHandler(t, false)
	for _, path := range []string{"sign-in", "sign-up", "sign-out"} {
		t.Run(path, func(t *testing.T) {
			options := httptest.NewRequest(http.MethodOptions, "/api/instances/admins/"+path+"/", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, options)
			if response.Code != http.StatusOK || response.Header().Get("Allow") != "POST, OPTIONS" {
				t.Fatalf("OPTIONS status=%d Allow=%q body=%s", response.Code, response.Header().Get("Allow"), response.Body.String())
			}
			unsupported := httptest.NewRequest(http.MethodPut, "/api/instances/admins/"+path+"/", nil)
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, unsupported)
			if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "CSRF Verification Failed") {
				t.Fatalf("PUT without CSRF status=%d body=%s", response.Code, response.Body.String())
			}
			secret := strings.Repeat("a", 32)
			unsupported = httptest.NewRequest(http.MethodPut, "/api/instances/admins/"+path+"/", nil)
			unsupported.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: secret})
			unsupported.Header.Set("X-CSRFToken", secret)
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, unsupported)
			if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "POST, OPTIONS" {
				t.Fatalf("PUT with valid CSRF status=%d Allow=%q body=%s", response.Code, response.Header().Get("Allow"), response.Body.String())
			}
		})
	}
}

func (instanceHTTPRepository) Telemetry(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{"logs_and_traces":{"status":"enabled","destination":"http://logal:4318"}}`), nil
}

func TestInstanceTelemetryRequiresAdminAndDisablesCaching(t *testing.T) {
	for _, tc := range []struct {
		name, cookie string
		denied       bool
		status       int
	}{
		{name: "anonymous", status: http.StatusUnauthorized},
		{name: "ordinary session", cookie: "session-id=valid-session", status: http.StatusUnauthorized},
		{name: "non-admin", cookie: "admin-session-id=valid-session", denied: true, status: http.StatusForbidden},
		{name: "admin", cookie: "admin-session-id=valid-session", status: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newInstanceManagementTestHandler(t, tc.denied)
			request := httptest.NewRequest(http.MethodGet, "/api/instances/telemetry/", nil)
			request.Header.Set("Cookie", tc.cookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.status == http.StatusOK {
				if response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), "http://logal:4318") {
					t.Fatalf("unexpected telemetry response: %v %s", response.Header(), response.Body.String())
				}
			} else if strings.Contains(response.Body.String(), "logal") {
				t.Fatal("destination leaked without admin authorization")
			}
		})
	}
}
