package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	customapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	notificationstore "github.com/CaliLuke/go-plane/internal/notifications"
	publicationstore "github.com/CaliLuke/go-plane/internal/projectpublication"
	notificationapi "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	intakeapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	publicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	historyapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	analyticsapi "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	exportsapi "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	personalapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	insightsapi "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
)

type remainingReadMiddlewareSessions struct {
	searchMountSessions
	inspections int
}

func (s *remainingReadMiddlewareSessions) InspectReadServiceSessionWithoutTimezone(ctx context.Context, r *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	s.inspections++
	return s.searchMountSessions.InspectReadServiceSessionWithoutTimezone(ctx, r)
}

// Exercise the production middleware and generated services; only database and
// session I/O are substituted. Domain calls must not happen before rejection.
func TestRemainingReadFamiliesMiddlewareOrder(t *testing.T) {
	families := []struct {
		name, method, path string
		success            int
		prepare            func(*testing.T, *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int)
	}{

		{"member preferences", "PATCH", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/preferences/member/22222222-2222-4222-8222-222222222222/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := extensionPreferencesRepository{recorder: &cycleMiddlewareRecorder{}}
			service, err := systemapi.NewProjectMemberPreferences(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{projectMemberPreferences: preferencesapi.NewEndpoints(service), projectMemberPreferencesMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return len(r.recorder.events) }
		}},
		{"external Unsplash", "GET", "/api/unsplash/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &externalMountRepository{}
			service, err := systemapi.NewExternalIntegrations(r, s, nil)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{externalIntegrations: externalapi.NewEndpoints(service), externalIntegrationsMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.configurations + r.roles }
		}},
		{"publication", "POST", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/project-deploy-boards/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &publicationMiddlewareRepository{}
			service, err := systemapi.NewProjectPublication(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{projectPublication: publicationapi.NewEndpoints(service), projectPublicationMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.writes }
		}},
		{"notification mark all", "POST", "/api/workspaces/demo/users/notifications/mark-all-read/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &remainingNotificationMiddlewareRepository{}
			service, err := systemapi.NewNotificationActions(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{notificationActions: notificationapi.NewEndpoints(service), notificationsMetadata: service.(systemapi.SessionProtocolAuthorizer)}, func() int { return r.authorizations + r.dispatches }
		}},
		{"analytics", "GET", "/api/workspaces/demo/advance-analytics/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &analyticsMountRepository{}
			service, err := systemapi.NewWorkspaceAnalytics(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{workspaceAnalytics: analyticsapi.NewEndpoints(service), workspaceAnalyticsMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.roles + r.queries }
		}},
		{"personal dashboard", "GET", "/api/users/me/workspaces/demo/dashboard/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &personalDashboardMountRepository{}
			service, err := systemapi.NewPersonalDashboard(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{personalDashboard: personalapi.NewEndpoints(service), personalDashboardMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.calls }
		}},
		{"custom views", "POST", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/project-views/", 204, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &customViewCreationHTTPRepository{}
			service, err := systemapi.NewCustomViewCreation(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{customViewCreation: customapi.NewEndpoints(service), customViewCreationMetadata: service.(systemapi.SessionResourceProtocolAuthorizer)}, func() int { return r.projectTargets + r.projectUpdates }
		}},
		{"workspace exports", "POST", "/api/workspaces/demo/export-issues/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &exportMountRepository{}
			service, err := systemapi.NewWorkspaceExports(r, s, &exportMountPublisher{})
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{workspaceExports: exportsapi.NewEndpoints(service), workspaceExportsMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.creates + r.reads }
		}},
		{"project intake writes", "POST", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/inboxes/", 201, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &extensionIntakeRepository{}
			service, err := systemapi.NewProjectIntakeWrites(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{projectIntakeWrites: intakeapi.NewEndpoints(service)}, func() int { return r.calls }
		}},
		{"workspace user insights", "GET", "/api/workspaces/demo/user-profile/22222222-2222-4222-8222-222222222222/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &insightsMountRepository{}
			service, err := systemapi.NewWorkspaceUserInsights(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{workspaceUserInsights: insightsapi.NewEndpoints(service), workspaceUserInsightsMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.calls }
		}},
		{"workspace search", "GET", "/api/workspaces/demo/search/?query=existing", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &searchMountRepository{}
			service, err := systemapi.NewWorkspaceSearch(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{workspaceSearch: searchapi.NewEndpoints(service), workspaceSearchMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return r.queries }
		}},
		{"history metadata", "GET", "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/issues/22222222-2222-4222-8222-222222222222/meta/", 200, func(t *testing.T, s *remainingReadMiddlewareSessions) (pendingReplacementEndpoints, func() int) {
			r := &workItemHistoryHTTPRepository{allowed: true}
			service, err := systemapi.NewWorkItemHistory(r, s)
			if err != nil {
				t.Fatal(err)
			}
			return pendingReplacementEndpoints{workItemHistory: historyapi.NewEndpoints(service), workItemHistoryMetadata: service.(sessionResourceProtocolAuthorizer)}, func() int { return len(r.reads) }
		}},
	}
	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			for _, tc := range []struct {
				name                                           string
				authenticated, invalidZone, oversized, options bool
				status                                         int
			}{
				{name: "request-size before authentication", oversized: true, status: 413},
				{name: "authentication before timezone", invalidZone: true, status: 401},
				{name: "timezone before authorization and dispatch", authenticated: true, invalidZone: true, status: 400},
				{name: "OPTIONS validates timezone", authenticated: true, invalidZone: true, options: true, status: 400},
				{name: "authorization permits dispatch", authenticated: true, status: family.success},
			} {
				t.Run(tc.name, func(t *testing.T) {
					events := &cycleMiddlewareRecorder{}
					sessions := &remainingReadMiddlewareSessions{searchMountSessions: searchMountSessions{&cycleMiddlewareSessions{recorder: events, userTimezone: "UTC", invalidZone: tc.invalidZone}}}
					pending, calls := family.prepare(t, sessions)
					audit := &cycleMiddlewareActivity{recorder: events}
					pending.readServicesSession = sessions
					pending.readServicesCookie = djangoSessionCookieConfig{Name: "session-id", Configured: true}
					pending.activityReadsBodyLimit = 64
					pending.activityReadsActivity = audit
					handler := extensionHTTPHandler(pending)
					body := "{}"
					if strings.Contains(family.path, "/export-issues/") {
						body = `{"provider":"csv"}`
					} else if strings.Contains(family.path, "/inboxes/") {
						body = `{"name":"New intake"}`
					}
					if tc.oversized {
						body = strings.Repeat("x", 65)
					}
					method := family.method
					if tc.options {
						method = "OPTIONS"
					}
					request := httptest.NewRequest(method, family.path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					if tc.authenticated {
						request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
					}
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					wantStatus := tc.status
					if response.Code != wantStatus {
						t.Fatalf("status=%d want=%d body=%s", response.Code, wantStatus, response.Body.String())
					}
					if tc.oversized && sessions.inspections != 0 {
						t.Fatalf("oversized request performed %d session inspections", sessions.inspections)
					}
					dispatchExpected := tc.authenticated && !tc.invalidZone && !tc.oversized && !tc.options
					if !dispatchExpected && calls() != 0 {
						t.Fatalf("rejected request reached domain %d times", calls())
					}
					if dispatchExpected && calls() == 0 {
						t.Fatal("authorized request did not reach domain")
					}
					if len(audit.records) != 0 {
						t.Fatalf("session request emitted %d API-key audit records", len(audit.records))
					}
				})
			}
		})
	}
}

// History has source permission-classes before timezone; other family methods
// use the ordinary authorization stage covered above.
func TestHistoryPermissionClassesBeforeTimezone(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST"} {
		for _, allowed := range []bool{false, true} {
			t.Run(method+fmt.Sprint(allowed), func(t *testing.T) {
				repo := &workItemHistoryHTTPRepository{allowed: allowed}
				sessions := &remainingReadMiddlewareSessions{searchMountSessions: searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "Invalid/Zone", invalidZone: true}}}
				service, err := systemapi.NewWorkItemHistory(repo, sessions)
				if err != nil {
					t.Fatal(err)
				}
				handler := extensionHTTPHandler(pendingReplacementEndpoints{workItemHistory: historyapi.NewEndpoints(service), workItemHistoryMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
				req := httptest.NewRequest(method, "/api/workspaces/demo/projects/"+historyHTTPProjectID+"/issues/"+historyHTTPIssueID+"/history/", nil)
				req.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				want := 403
				if allowed {
					want = 400
				}
				if response.Code != want || len(repo.reads) != 0 {
					t.Fatalf("status=%d want=%d reads=%d body=%s", response.Code, want, len(repo.reads), response.Body.String())
				}
			})
		}
	}
}

type publicationMiddlewareRepository struct{ writes int }

func (*publicationMiddlewareRepository) Authorize(context.Context, string, string, string, string) error {
	return nil
}
func (*publicationMiddlewareRepository) Exists(context.Context, string, string, string) error {
	return nil
}
func (r *publicationMiddlewareRepository) Create(context.Context, string, string, string, publicationstore.CreateInput) (json.RawMessage, error) {
	r.writes++
	return json.RawMessage(`{}`), nil
}
func (r *publicationMiddlewareRepository) Update(context.Context, string, string, string, string, publicationstore.Patch) (json.RawMessage, error) {
	r.writes++
	return json.RawMessage(`{}`), nil
}
func (r *publicationMiddlewareRepository) Delete(context.Context, string, string, string, string) error {
	r.writes++
	return nil
}

type remainingNotificationMiddlewareRepository struct {
	notificationMiddlewareRepository
}

func (r *remainingNotificationMiddlewareRepository) Set(context.Context, notificationstore.Principal, string, string, string, any) (map[string]any, error) {
	r.dispatches++
	return map[string]any{}, nil
}
func (r *remainingNotificationMiddlewareRepository) MarkAllRead(context.Context, notificationstore.Principal, string, bool, bool, string) error {
	r.dispatches++
	return nil
}
