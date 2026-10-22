package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
)

type personalDashboardMountRepository struct {
	calls int
	month string
}

func (r *personalDashboardMountRepository) Dashboard(_ context.Context, _, _, month string) (json.RawMessage, error) {
	r.calls++
	r.month = month
	return json.RawMessage(`{"assigned_issues":3}`), nil
}

func TestPersonalDashboardRejectsMalformedMonthsBeforeQuery(t *testing.T) {
	for _, suffix := range []string{"dashboard", "issues-completed-graph"} {
		for _, month := range []string{"", "bad", "1.5"} {
			t.Run(suffix+"/"+month, func(t *testing.T) {
				repository := &personalDashboardMountRepository{}
				sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
				service, err := systemapi.NewPersonalDashboard(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{personalDashboard: api.NewEndpoints(service), personalDashboardMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
				request := httptest.NewRequest(http.MethodGet, "/api/users/me/workspaces/demo/"+suffix+"/?month="+url.QueryEscape(month), nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusBadRequest || repository.calls != 0 {
					t.Fatalf("status=%d repository calls=%d body=%s", response.Code, repository.calls, response.Body.String())
				}
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "Invalid month." {
					t.Fatalf("unexpected validation response: %s", response.Body.String())
				}
			})
		}
	}
}
func (r *personalDashboardMountRepository) ActivityGraph(context.Context, string, string) (json.RawMessage, error) {
	r.calls++
	return json.RawMessage(`[{"activity_count":2}]`), nil
}
func (r *personalDashboardMountRepository) CompletedGraph(_ context.Context, _, _, month string) (json.RawMessage, error) {
	r.calls++
	r.month = month
	return json.RawMessage(`[{"completed_count":1}]`), nil
}

func TestPersonalDashboardProductionMounts(t *testing.T) {
	for _, suffix := range []string{"dashboard", "activity-graph", "issues-completed-graph"} {
		for _, tc := range []struct {
			method string
			auth   bool
			want   int
		}{
			{"GET", true, 200}, {"HEAD", true, 200}, {"OPTIONS", true, 200},
			{"POST", true, 405}, {"GET", false, 401}, {"OPTIONS", false, 401},
		} {
			t.Run(suffix+"/"+tc.method+"/"+map[bool]string{true: "authenticated", false: "anonymous"}[tc.auth], func(t *testing.T) {
				repository := &personalDashboardMountRepository{}
				sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
				service, err := systemapi.NewPersonalDashboard(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{personalDashboard: api.NewEndpoints(service), personalDashboardMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
				request := httptest.NewRequest(tc.method, "/api/users/me/workspaces/demo/"+suffix+"/", nil)
				if tc.auth {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != tc.want {
					t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), tc.want)
				}
				wantCalls := 0
				if tc.auth && (tc.method == "GET" || tc.method == "HEAD") {
					wantCalls = 1
				}
				if repository.calls != wantCalls {
					t.Fatalf("repository calls=%d, want %d", repository.calls, wantCalls)
				}
				if tc.method == "HEAD" && response.Body.Len() != 0 {
					t.Fatal("HEAD returned a response body")
				}
			})
		}
	}
	for _, path := range []string{"/api/users/me/workspaces/demo/dashboard/extra/", "/api/users/me/workspaces/demo/activity-graphs/"} {
		handler := extensionHTTPHandler(pendingReplacementEndpoints{})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 404 {
			t.Fatalf("unowned path %s returned %d", path, response.Code)
		}
	}
}

func TestPersonalDashboardNormalizesSourceIntegerMonths(t *testing.T) {
	for _, suffix := range []string{"dashboard", "issues-completed-graph"} {
		for _, tc := range []struct{ input, normalized string }{{"２", "2"}, {"0_2", "2"}, {"2147483648", "2147483648"}, {" +02 ", "2"}} {
			t.Run(suffix+"/"+tc.input, func(t *testing.T) {
				repository := &personalDashboardMountRepository{}
				sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}}
				service, err := systemapi.NewPersonalDashboard(repository, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{personalDashboard: api.NewEndpoints(service), personalDashboardMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
				request := httptest.NewRequest(http.MethodGet, "/api/users/me/workspaces/demo/"+suffix+"/?month="+url.QueryEscape(tc.input), nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusOK || repository.calls != 1 || repository.month != tc.normalized {
					t.Fatalf("status=%d calls=%d month=%q body=%s", response.Code, repository.calls, repository.month, response.Body.String())
				}
			})
		}
	}
}
