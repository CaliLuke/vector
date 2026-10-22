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
	store "github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	api "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
)

type insightsMountRepository struct {
	calls  int
	denied bool
}

func (r *insightsMountRepository) AuthorizeWorkspace(context.Context, string, string, bool) error {
	if r.denied {
		return store.ErrForbidden
	}
	return nil
}
func (r *insightsMountRepository) Profile(context.Context, string, string, string) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`{"display_name":"Existing member"}`), nil
}
func (r *insightsMountRepository) Activity(context.Context, string, string, string, []string, djangoapi.Query) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`{"results":[{"comment":"Existing activity"}]}`), nil
}
func (r *insightsMountRepository) Stats(context.Context, string, string, string, map[string]string) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`{"assigned_issues":3}`), nil
}
func (r *insightsMountRepository) ExportActivity(context.Context, string, string, string, string) ([]store.ActivityCSVRow, error) {
	r.calls++
	return nil, nil
}
func (r *insightsMountRepository) CanExportActivity(context.Context, string, string) (bool, error) {
	return !r.denied, nil
}

func TestWorkspaceUserInsightsProductionMounts(t *testing.T) {
	const user = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct {
		name, method, path, query, body, contains, optionsName string
		anonymous, denied                                      bool
		status, calls                                          int
	}{
		{name: "profile", method: "GET", path: "user-profile/" + user + "/", contains: "Existing member", status: 200, calls: 1},
		{name: "profile metadata", method: "OPTIONS", path: "user-profile/" + user + "/", optionsName: "Workspace User Profile Endpoint", status: 200},
		{name: "stats", method: "GET", path: "user-stats/" + user + "/", contains: "assigned_issues", status: 200, calls: 1},
		{name: "stats metadata", method: "OPTIONS", path: "user-stats/" + user + "/", optionsName: "Workspace User Profile Stats Endpoint", status: 200},
		{name: "activity", method: "GET", path: "user-activity/" + user + "/", contains: "Existing activity", status: 200, calls: 1},
		{name: "activity query UUID validation", method: "GET", path: "user-activity/" + user + "/", query: "?project=__invalid__", status: 400},
		{name: "activity head", method: "HEAD", path: "user-activity/" + user + "/", status: 200, calls: 1},
		{name: "activity metadata", method: "OPTIONS", path: "user-activity/" + user + "/", optionsName: "Workspace User Activity Endpoint", status: 200},
		{name: "activity anonymous", method: "GET", path: "user-activity/" + user + "/", anonymous: true, status: 401},
		{name: "activity forbidden", method: "GET", path: "user-activity/" + user + "/", denied: true, status: 403},
		{name: "activity method", method: "PATCH", path: "user-activity/" + user + "/", status: 405},
		{name: "export", method: "POST", path: "user-activity/" + user + "/export/", body: `{"date":"2026-01-01"}`, contains: "\r\n", status: 200, calls: 1},
		{name: "export metadata", method: "OPTIONS", path: "user-activity/" + user + "/export/", optionsName: "Export Workspace User Activity Endpoint", status: 200},
		{name: "export validation", method: "POST", path: "user-activity/" + user + "/export/", body: `{}`, status: 400},
		{name: "export forbidden", method: "POST", path: "user-activity/" + user + "/export/", body: `{"date":"2026-01-01"}`, denied: true, status: 403},
		{name: "negative similar path", method: "GET", path: "user-activities/" + user + "/", status: 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &insightsMountRepository{denied: tc.denied}
			sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
			service, err := systemapi.NewWorkspaceUserInsights(repository, sessions)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceUserInsights: api.NewEndpoints(service), workspaceUserInsightsMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
			request := httptest.NewRequest(tc.method, "/api/workspaces/demo/"+tc.path+tc.query, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if !tc.anonymous {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), tc.status)
			}
			if repository.calls != tc.calls {
				t.Fatalf("repository calls=%d, want %d", repository.calls, tc.calls)
			}
			if tc.contains != "" && !strings.Contains(response.Body.String(), tc.contains) {
				t.Fatalf("missing expected body %q in %q", tc.contains, response.Body.String())
			}
			if tc.optionsName != "" {
				var metadata struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil || metadata.Name != tc.optionsName {
					t.Fatalf("OPTIONS metadata=%s, name=%q, err=%v; want name %q", response.Body.String(), metadata.Name, err, tc.optionsName)
				}
			}
			if tc.method == "HEAD" && response.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
			if tc.name == "export" && !strings.Contains(response.Header().Get("Content-Disposition"), "workspace-user-activity.csv") {
				t.Fatalf("export headers=%v", response.Header())
			}
		})
	}
}
