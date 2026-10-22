package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	store "github.com/CaliLuke/go-plane/internal/externalintegrations"
)

type externalMountRepository struct {
	*store.Repository
	denied                bool
	configurations, roles int
}

func (r *externalMountRepository) Configuration(context.Context, []string, map[string]string, bool) (map[string]string, error) {
	r.configurations++
	return map[string]string{}, nil
}
func (r *externalMountRepository) WorkspaceRole(context.Context, string, string) (int, error) {
	r.roles++
	if r.denied {
		return 5, nil
	}
	return 20, nil
}
func (r *externalMountRepository) ProjectAllowed(context.Context, string, string, string) (bool, error) {
	r.roles++
	return !r.denied, nil
}

func TestExternalIntegrationProductionMounts(t *testing.T) {
	const workspace = "/api/workspaces/demo/ai-assistant/"
	const project = "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/ai-assistant/"
	for _, tc := range []struct {
		name, method, path, body, wantBody string
		authenticated, denied              bool
		status, configurations, roles      int
	}{
		{name: "anonymous unsplash", method: "GET", path: "/api/unsplash/", status: 401},
		{name: "unconfigured unsplash", method: "GET", path: "/api/unsplash/", authenticated: true, status: 200, wantBody: "[]", configurations: 1},
		{name: "anonymous workspace AI", method: "POST", path: workspace, body: `{`, status: 401},
		{name: "anonymous project AI", method: "POST", path: project, body: `{`, status: 401},
		{name: "workspace role before malformed JSON", method: "POST", path: workspace, body: `{`, authenticated: true, denied: true, status: 403, roles: 1},
		{name: "project role before malformed JSON", method: "POST", path: project, body: `{`, authenticated: true, denied: true, status: 403, roles: 1},
		{name: "workspace config before malformed JSON", method: "POST", path: workspace, body: `{`, authenticated: true, status: 400, wantBody: "LLM provider API key and model are required", roles: 1, configurations: 1},
		{name: "project config before malformed JSON", method: "POST", path: project, body: `{`, authenticated: true, status: 400, wantBody: "LLM provider API key and model are required", roles: 1, configurations: 1},
		{name: "workspace metadata no method role", method: "OPTIONS", path: workspace, authenticated: true, denied: true, status: 200},
		{name: "project metadata no method role", method: "OPTIONS", path: project, authenticated: true, denied: true, status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &externalMountRepository{denied: tc.denied}
			sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
			service, err := systemapi.NewExternalIntegrations(repository, sessions, nil)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{externalIntegrations: api.NewEndpoints(service), externalIntegrationsMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.authenticated {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Fatalf("status=%d body=%s; want status%d containing %q", response.Code, response.Body.String(), tc.status, tc.wantBody)
			}
			if repository.configurations != tc.configurations || repository.roles != tc.roles {
				t.Fatalf("configurations=%d roles=%d, want %d/%d", repository.configurations, repository.roles, tc.configurations, tc.roles)
			}
		})
	}
}
