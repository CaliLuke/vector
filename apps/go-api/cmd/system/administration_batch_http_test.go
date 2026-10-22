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
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	store "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	api "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
)

type webhookMiddlewareRepository struct {
	*store.Repository
	recorder *cycleMiddlewareRecorder
}

func (r webhookMiddlewareRepository) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	r.recorder.add("authorization")
	return !r.recorder.denied, nil
}
func (r webhookMiddlewareRepository) List(context.Context, string) (json.RawMessage, error) {
	r.recorder.add("dispatch")
	return json.RawMessage(`[]`), nil
}

type webhookMiddlewarePublisher struct{ t *testing.T }

func (p webhookMiddlewarePublisher) SoftDelete(context.Context, string, string, string) error {
	p.t.Fatal("unexpected durable effect")
	return nil
}

func TestWebhookProductionAuthenticationPrecedesParsingAndProtocolDispatch(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, path string
		authenticated, denied    bool
		invalidTimezone          bool
		status                   int
		authorization, dispatch  bool
	}{
		{name: "anonymous malformed write", method: "POST", body: `{`, status: 401},
		{name: "request-size before authentication", method: "POST", body: strings.Repeat("x", 65), authenticated: true, status: 413},
		{name: "member malformed write", method: "POST", body: `{`, authenticated: true, denied: true, status: 403, authorization: true},
		{name: "admin malformed write", method: "POST", body: `{`, authenticated: true, status: 400, authorization: true},
		{name: "member metadata", method: "OPTIONS", authenticated: true, denied: true, status: 200},
		{name: "anonymous metadata", method: "OPTIONS", status: 401},
		{name: "member unsupported method", method: "PUT", authenticated: true, denied: true, status: 405},
		{name: "anonymous unsupported method", method: "PUT", status: 401},
		{name: "member collection delete", method: "DELETE", authenticated: true, denied: true, status: 403, authorization: true},
		{name: "admin collection delete", method: "DELETE", authenticated: true, status: 405, authorization: true},
		{name: "member head", method: "HEAD", authenticated: true, denied: true, status: 403, authorization: true},
		{name: "admin head", method: "HEAD", authenticated: true, status: 200, authorization: true, dispatch: true},
		{name: "admin list", method: "GET", authenticated: true, status: 200, authorization: true, dispatch: true},
		{name: "invalid timezone options", method: "OPTIONS", authenticated: true, invalidTimezone: true, status: 400},
		{name: "anonymous detail post", method: "POST", path: "/api/workspaces/demo/webhooks/11111111-1111-4111-8111-111111111111/", status: 401},
		{name: "member detail post", method: "POST", path: "/api/workspaces/demo/webhooks/11111111-1111-4111-8111-111111111111/", authenticated: true, denied: true, status: 403, authorization: true},
		{name: "admin detail post", method: "POST", path: "/api/workspaces/demo/webhooks/11111111-1111-4111-8111-111111111111/", authenticated: true, status: 405, authorization: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: tc.denied}
			sessions := &cycleMiddlewareSessions{recorder: recorder, invalidZone: tc.invalidTimezone, userTimezone: "UTC"}
			audit := &cycleMiddlewareActivity{recorder: recorder}
			service, err := systemapi.NewWorkspaceWebhooks(webhookMiddlewareRepository{recorder: recorder}, sessions, webhookMiddlewarePublisher{t})
			if err != nil {
				t.Fatal(err)
			}
			handler := newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{workspaceWebhooks: api.NewEndpoints(service), workspaceWebhooksMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsActivity: audit, activityReadsBodyLimit: 64})
			path := tc.path
			if path == "" {
				path = "/api/workspaces/demo/webhooks/"
			}
			request := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.authenticated {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
			if tc.status == http.StatusForbidden {
				if tc.method != http.MethodHead {
					var body map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body) != 1 || body["error"] != "You don't have the required permissions." {
						t.Fatalf("forbidden body=%s decode error=%v", response.Body.String(), err)
					}
				}
			}
			var expectedEvents []string
			if tc.status != http.StatusRequestEntityTooLarge {
				expectedEvents = append(expectedEvents, "session-inspection")
				if tc.authenticated {
					expectedEvents = append(expectedEvents, "timezone")
				}
				if tc.authorization {
					expectedEvents = append(expectedEvents, "authorization")
				}
				if tc.dispatch {
					expectedEvents = append(expectedEvents, "dispatch")
				}
			}
			if !slices.Equal(recorder.events, expectedEvents) {
				t.Fatalf("events=%v want=%v", recorder.events, expectedEvents)
			}
			if len(audit.records) != 0 {
				t.Fatalf("session request emitted API-key audit records: %v", audit.records)
			}
			if tc.status == http.StatusRequestEntityTooLarge && len(recorder.events) != 0 {
				t.Fatalf("request-size rejection reached session or domain code: %v", recorder.events)
			}
			if tc.method == "HEAD" && response.Body.Len() != 0 {
				t.Fatalf("HEAD body=%s", response.Body.String())
			}
			if tc.method == "OPTIONS" && tc.status == 200 {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["name"] != "Webhook Endpoint" {
					t.Fatalf("metadata=%s error=%v", response.Body.String(), err)
				}
				if got := response.Header().Get("Allow"); got != "GET, POST, PATCH, DELETE, HEAD, OPTIONS" {
					t.Fatalf("Allow=%q", got)
				}
			}
		})
	}
}

func TestWebhookMalformedLogUUIDUsesDjangoConverter404BeforeAuth(t *testing.T) {
	for _, authenticated := range []bool{false, true} {
		name := "anonymous"
		if authenticated {
			name = "authenticated-invalid-timezone"
		}
		t.Run(name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			sessions := &cycleMiddlewareSessions{recorder: recorder, invalidZone: authenticated, userTimezone: "Not/AZone"}
			audit := &cycleMiddlewareActivity{recorder: recorder}
			service, err := systemapi.NewWorkspaceWebhooks(webhookMiddlewareRepository{recorder: recorder}, sessions, webhookMiddlewarePublisher{t})
			if err != nil {
				t.Fatal(err)
			}
			handler := newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{workspaceWebhooks: api.NewEndpoints(service), workspaceWebhooksMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsActivity: audit, activityReadsBodyLimit: 64})
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/demo/webhook-logs/not-a-uuid/", nil)
			if authenticated {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound || response.Body.String() != `{"error":"Page not found."}` {
				t.Fatalf("converter response = %d %q, want 404 page-not-found JSON", response.Code, response.Body.String())
			}
			if len(recorder.events) != 0 || len(audit.records) != 0 {
				t.Fatalf("converter miss reached authentication/domain middleware: events=%v audit=%v", recorder.events, audit.records)
			}
			if got := response.Header().Get("Vary"); got != "Cookie, origin" {
				t.Fatalf("Vary=%q, want Cookie, origin", got)
			}
		})
	}
}

func TestAdministrationBatchPathConvertersAndUnmatchedRoutes(t *testing.T) {
	const project = "11111111-1111-4111-8111-111111111111"
	const estimate = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct {
		path      string
		malformed bool
	}{
		{"/api/workspaces/demo/projects/" + project + "/estimates/" + estimate + "/estimate-points/not-a-uuid/", false},
		{"/api/workspaces/demo/projects/not-a-uuid/estimates/" + estimate + "/estimate-points/anything/", true},
		{"/api/workspaces/demo/projects/" + project + "/estimates/not-a-uuid/estimate-points/anything/", true},
		{"/api/workspaces/demo/webhooks/not-a-uuid/", true},
		{"/api/workspaces/demo/webhook-logs/not-a-uuid/", true},
	} {
		protocol, ok := resolveReadServiceProtocolForMethod(tc.path, "PATCH")
		if !ok {
			t.Errorf("source path not registered: %s", tc.path)
			continue
		}
		if got := readServiceProtocolHasMalformedUUID(tc.path, protocol); got != tc.malformed {
			t.Errorf("%s malformed=%v want=%v", tc.path, got, tc.malformed)
		}
	}
	for _, path := range []string{
		"/api/workspaces/demo/webhooks/logs/extra/",
		"/api/workspaces/demo/webhook-logs/",
		"/api/workspaces/demo/projects/" + project + "/bulk-create-labels/extra/",
		"/api/workspaces/demo/projects/" + project + "/estimate-points/",
	} {
		if protocol, ok := resolveReadServiceProtocol(path); ok {
			t.Errorf("unmatched source path %s resolved as %s", path, protocol.pattern)
		}
	}
}
