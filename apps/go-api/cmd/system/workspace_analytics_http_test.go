package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
)

type analyticsMountRepository struct {
	denied         bool
	roles, queries int
}

func (r *analyticsMountRepository) Authorized(context.Context, uuid.UUID, string, string) (bool, error) {
	r.roles++
	return !r.denied, nil
}
func (r *analyticsMountRepository) AuthorizedProjectChart(context.Context, uuid.UUID, string, string) (bool, error) {
	r.roles++
	return !r.denied, nil
}
func (r *analyticsMountRepository) AuthorizedDefault(context.Context, uuid.UUID, string) (bool, error) {
	r.roles++
	return !r.denied, nil
}
func (r *analyticsMountRepository) Chart(context.Context, uuid.UUID, string, string, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) result() (json.RawMessage, error) {
	r.queries++
	return json.RawMessage(`{"fixture":true}`), nil
}
func (r *analyticsMountRepository) Overview(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) WorkItemStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) ProjectAssigneeStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) ProjectStats(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) ProjectChart(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) JSON(context.Context, string, ...any) (json.RawMessage, error) {
	return r.result()
}
func (r *analyticsMountRepository) SavedView(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	r.queries++
	return json.RawMessage(`{"query":{},"query_dict":{"x_axis":"state_id","y_axis":"issue_count"}}`), nil
}

func TestWorkspaceAnalyticsProductionMounts(t *testing.T) {
	replacer := strings.NewReplacer("{slug}", "demo", "{project_id}", "11111111-1111-4111-8111-111111111111", "{analytic_id}", "22222222-2222-4222-8222-222222222222")
	for _, definition := range workspaceAnalyticsProtocolDefinitions {
		for _, tc := range []struct {
			name, method          string
			authenticated, denied bool
			status                int
		}{
			{"anonymous", "GET", false, false, 401},
			{"denied", "GET", true, true, 403},
			{"read", "GET", true, false, 200},
			{"head", "HEAD", true, false, 200},
			{"metadata", "OPTIONS", true, true, 200},
			{"anonymous metadata", "OPTIONS", false, false, 401},
		} {
			t.Run(definition.path+"/"+tc.name, func(t *testing.T) {
				repository := &analyticsMountRepository{denied: tc.denied}
				sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
				service, err := systemapi.NewWorkspaceAnalytics(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{workspaceAnalytics: api.NewEndpoints(service), workspaceAnalyticsMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
				query := "?x_axis=state_id&y_axis=issue_count"
				if strings.Contains(definition.path, "/projects/{project_id}/advance-analytics-charts/") {
					query += "&type=custom-work-items"
				}
				request := httptest.NewRequest(tc.method, replacer.Replace(definition.path)+query, nil)
				if tc.authenticated {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != tc.status {
					t.Fatalf("status=%d body=%s; want%d", response.Code, response.Body.String(), tc.status)
				}
				if tc.method == "OPTIONS" || !tc.authenticated {
					if repository.roles != 0 || repository.queries != 0 {
						t.Fatalf("metadata/anonymous reached domain: roles=%d queries=%d", repository.roles, repository.queries)
					}
				} else if tc.denied {
					if repository.roles == 0 || repository.queries != 0 {
						t.Fatalf("denied request reached data: roles=%d queries=%d", repository.roles, repository.queries)
					}
				} else if repository.roles == 0 || repository.queries == 0 {
					t.Fatalf("authorized read did not dispatch: roles=%d queries=%d", repository.roles, repository.queries)
				}
				if tc.method == "HEAD" && response.Body.Len() != 0 {
					t.Fatalf("HEAD body=%s", response.Body.String())
				}
				if tc.method == "GET" && tc.status == 200 && !json.Valid(response.Body.Bytes()) {
					t.Fatalf("invalid JSON response: %s", response.Body.String())
				}
			})
		}
	}
}
