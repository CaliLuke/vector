package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
)

type exportMountRepository struct {
	denied         bool
	creates, reads int
}

func (r *exportMountRepository) Authorized(context.Context, string, string) (bool, error) {
	return !r.denied, nil
}
func (*exportMountRepository) UserEmail(context.Context, string) (string, error) {
	return "test@example.com", nil
}
func (r *exportMountRepository) CreateIssueExport(context.Context, string, string, string, []any, bool) (json.RawMessage, string, string, []any, error) {
	r.creates++
	return json.RawMessage(`{"message":"export queued"}`), "workspace-id", "export-token", []any{"project-id"}, nil
}
func (r *exportMountRepository) History(context.Context, string, string, djangoapi.Query) (json.RawMessage, error) {
	r.reads++
	return json.RawMessage(`{"results":[]}`), nil
}

type exportMountPublisher struct {
	names  []string
	kwargs []map[string]any
}

func (p *exportMountPublisher) Task(_ context.Context, name string, _ []any, kwargs map[string]any) error {
	p.names = append(p.names, name)
	p.kwargs = append(p.kwargs, kwargs)
	return nil
}

func TestWorkspaceExportsProductionAuthorizationAndEffects(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, body      string
		anonymous, denied             bool
		status, creates, reads, tasks int
	}{
		{name: "create", path: "export-issues/", method: "POST", body: `{"provider":"csv"}`, status: 200, creates: 1, tasks: 1},
		{name: "denied before body", path: "export-issues/", method: "POST", body: `{`, denied: true, status: 403},
		{name: "anonymous", path: "export-issues/", method: "POST", body: `{"provider":"csv"}`, anonymous: true, status: 401},
		{name: "invalid provider", path: "export-issues/", method: "POST", body: `{"provider":"exe"}`, status: 400},
		{name: "history", path: "export-issues/?per_page=20&cursor=0:0:0", method: "GET", status: 200, reads: 1},
		{name: "history head", path: "export-issues/?per_page=20&cursor=0:0:0", method: "HEAD", status: 200, reads: 1},
		{name: "history parameters", path: "export-issues/", method: "GET", status: 400},
		{name: "metadata ignores handler role decorator", path: "export-issues/", method: "OPTIONS", denied: true, status: 200},
		{name: "analytics validation", path: "export-analytics/", method: "POST", body: `{}`, status: 400},
		{name: "analytics method", path: "export-analytics/", method: "GET", status: 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &exportMountRepository{denied: tc.denied}
			publisher := &exportMountPublisher{}
			sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
			service, err := systemapi.NewWorkspaceExports(repo, sessions, publisher)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceExports: api.NewEndpoints(service), workspaceExportsMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
			request := httptest.NewRequest(tc.method, "/api/workspaces/demo/"+tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if !tc.anonymous {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s, want%d", response.Code, response.Body.String(), tc.status)
			}
			if repo.creates != tc.creates || repo.reads != tc.reads || len(publisher.names) != tc.tasks {
				t.Fatalf("creates=%d reads=%d tasks=%v", repo.creates, repo.reads, publisher.names)
			}
			if tc.tasks > 0 {
				if publisher.names[0] != "plane.bgtasks.export_task.issue_export_task" || publisher.kwargs[0]["provider"] != "csv" || publisher.kwargs[0]["token_id"] != "export-token" {
					t.Fatalf("unexpected task: %v %v", publisher.names, publisher.kwargs)
				}
			}
			if tc.method == "HEAD" && response.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
		})
	}
}
