package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	storeuuid "uuid"
)

type projectLifecycleMiddlewareRepository struct {
	*projects.LifecycleRepository
	recorder *cycleMiddlewareRecorder
}

func (r projectLifecycleMiddlewareRepository) AuthorizeProjectWrite(context.Context, string, string, string, string, bool) error {
	r.recorder.add("authorization")
	if r.recorder.denied {
		return projects.ErrForbidden
	}
	return nil
}
func (r projectLifecycleMiddlewareRepository) AuthorizeAPIProjectRead(context.Context, storeuuid.UUID, string, bool) error {
	return r.AuthorizeProjectWrite(context.Background(), "", "", "", "", false)
}
func (projectLifecycleMiddlewareRepository) CheckProjectMutation(context.Context, string, string, string, bool) error {
	return nil
}
func (r projectLifecycleMiddlewareRepository) CreateProject(context.Context, string, string, djangoinput.Input, bool) (json.RawMessage, error) {
	r.recorder.add("dispatch")
	return json.RawMessage(`{}`), nil
}
func (r projectLifecycleMiddlewareRepository) APIProjectSummary(context.Context, storeuuid.UUID, string, storeuuid.UUID, projects.APIProjectSummaryOptions) (json.RawMessage, error) {
	r.recorder.add("dispatch")
	return json.RawMessage(`{"counts": {"cycles": 0, "intakes": 0, "issues": 0, "labels": 0, "members": 4, "modules": 0, "pages": 0, "states": 1}, "id": "00000000-0000-4000-8000-000000000001", "identifier": "BASE", "name": "Source Project"}`), nil
}

func newProjectLifecycleMiddlewareTestHandler(t *testing.T, recorder *cycleMiddlewareRecorder, sessions *cycleMiddlewareSessions, timezone string, activity *cycleMiddlewareActivity, bodyLimit int64) http.Handler {
	t.Helper()
	repository := projectLifecycleMiddlewareRepository{recorder: recorder}
	service, err := systemapi.NewProjectLifecycle(repository, sessions, cycleMiddlewareAPIKeys{recorder: recorder, timezone: timezone}, cycleMiddlewareThrottle{recorder: recorder})
	if err != nil {
		t.Fatal(err)
	}
	protocolActivity := &activityHTTPService{}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			projectLifecycle: api.NewEndpoints(service), projectLifecycleMetadata: service,
			activityReads: activityapi.NewEndpoints(protocolActivity), activityReadsMetadata: protocolActivity,
			readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsActivity: activity, activityReadsSecret: "project-lifecycle-test-secret", activityReadsBodyLimit: bodyLimit,
		},
	)
}

func TestProjectLifecycleProductionMiddlewareOrderRequestSizeSessionAPIThrottleDispatchAudit(t *testing.T) {
	const limit = 32
	for _, test := range []struct {
		name                                   string
		apiKey, denied, invalidZone, oversized bool
		wantStatus                             int
		wantEvents                             []string
		wantAuditCount                         int
	}{
		{name: "API permission then throttle dispatch and audit", apiKey: true, wantStatus: 201, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "dispatch", "audit"}, wantAuditCount: 1},
		{name: "API denial before throttle", apiKey: true, denied: true, wantStatus: 403, wantEvents: []string{"api-key-authentication", "authorization", "audit"}, wantAuditCount: 1},
		{name: "API timezone after permission and throttle", apiKey: true, invalidZone: true, wantStatus: 400, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "audit"}, wantAuditCount: 1},
		{name: "API request-size before authentication", apiKey: true, oversized: true, wantStatus: 413},
		{name: "session timezone then permission and dispatch", wantStatus: 201, wantEvents: []string{"session-inspection", "timezone", "authorization", "dispatch"}},
		{name: "session permission denies dispatch", denied: true, wantStatus: 403, wantEvents: []string{"session-inspection", "timezone", "authorization"}},
		{name: "session timezone before permission", invalidZone: true, wantStatus: 400, wantEvents: []string{"session-inspection", "timezone"}},
		{name: "session request-size before inspection", oversized: true, wantStatus: 413},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: test.denied}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC", invalidZone: test.invalidZone && !test.apiKey}
			zone := "UTC"
			if test.invalidZone && test.apiKey {
				zone = "Not/AZone"
			}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			handler := newProjectLifecycleMiddlewareTestHandler(t, recorder, sessions, zone, activity, limit)
			path := "/api/workspaces/demo/projects/"
			if test.apiKey {
				path = "/api/v1/workspaces/demo/projects/"
			}
			body := `{}`
			if test.oversized {
				body = strings.Repeat("x", limit+1)
			}
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			if test.apiKey {
				request.Header.Set("X-Api-Key", "valid-api-key")
			} else if !test.oversized {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if !reflect.DeepEqual(recorder.events, test.wantEvents) {
				t.Fatalf("events=%v want %v", recorder.events, test.wantEvents)
			}
			if len(activity.records) != test.wantAuditCount {
				t.Fatalf("audit count=%d want %d", len(activity.records), test.wantAuditCount)
			}
		})
	}
}

func TestProjectLifecycleProductionMalformedUUIDPrecedesAuthentication(t *testing.T) {
	for _, test := range []struct{ method, path string }{
		{http.MethodPatch, "/api/v1/workspaces/demo/projects/not-a-uuid/"},
		{http.MethodPost, "/api/workspaces/demo/projects/not-a-uuid/archive/"},
	} {
		recorder := &cycleMiddlewareRecorder{}
		sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
		activity := &cycleMiddlewareActivity{recorder: recorder}
		handler := newProjectLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, strings.NewReader(`{}`)))
		if response.Code != 404 || len(recorder.events) != 0 || len(activity.records) != 0 {
			t.Errorf("%s %s: status=%d events=%v audit=%v", test.method, test.path, response.Code, recorder.events, activity.records)
		}
		if response.Header().Get("Allow") != "" {
			t.Errorf("malformed path leaked endpoint Allow=%s", response.Header().Get("Allow"))
		}
		if strings.HasPrefix(test.path, "/api/v1/") {
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "Page not found." {
				t.Errorf("API resolver body=%s", response.Body.String())
			}
		} else if response.Body.String() != string(djangoNotFoundBody) {
			t.Errorf("session resolver body=%s", response.Body.String())
		}
	}
}

func TestProjectLifecycleSessionOptionsInspectsTimezoneBeforeAuthenticationAndMetadata(t *testing.T) {
	for _, route := range []struct{ path, allow string }{
		{"/api/workspaces/demo/projects/", "POST, OPTIONS"},
		{"/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/", "PUT, PATCH, DELETE, OPTIONS"},
		{"/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/archive/", "POST, DELETE, OPTIONS"},
	} {
		for _, invalidZone := range []bool{false, true} {
			t.Run(route.path+map[bool]string{false: "valid timezone", true: "invalid timezone"}[invalidZone], func(t *testing.T) {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC", invalidZone: invalidZone}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler := newProjectLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
				request := httptest.NewRequest(http.MethodOptions, route.path, nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				wantStatus := http.StatusOK
				wantEvents := []string{"session-inspection", "timezone", "session-authentication"}
				if invalidZone {
					wantStatus = http.StatusBadRequest
					wantEvents = []string{"session-inspection", "timezone"}
					var body map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "The required key does not exist." {
						t.Fatalf("timezone error=%s,%v", response.Body.String(), err)
					}
				}
				if response.Code != wantStatus || response.Header().Get("Allow") != route.allow || !reflect.DeepEqual(recorder.events, wantEvents) || len(activity.records) != 0 {
					t.Fatalf("status=%d allow=%q events=%v audit=%v body=%s", response.Code, response.Header().Get("Allow"), recorder.events, activity.records, response.Body.String())
				}
			})
		}
	}
}

func TestProjectLifecycleSlashlessSessionReadsRedirectWithoutAuthentication(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/demo/projects/",
		"/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/",
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+path, func(t *testing.T) {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler := newProjectLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, strings.TrimSuffix(path, "/")+"?x=1", nil))
				if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != path+"?x=1" || response.Body.Len() != 0 || len(recorder.events) != 0 || len(activity.records) != 0 {
					t.Fatalf("status=%d location=%q events=%v audit=%v body=%s", response.Code, response.Header().Get("Location"), recorder.events, activity.records, response.Body.String())
				}
			})
		}
	}
}

func TestProjectLifecycleProductionSummaryUsesRenderedLengthForGzipThreshold(t *testing.T) {
	recorder := &cycleMiddlewareRecorder{}
	sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
	activity := &cycleMiddlewareActivity{recorder: recorder}
	handler := newProjectLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/summary/", nil)
	request.Header.Set("X-Api-Key", "valid-api-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Body.Len() >= 200 || strings.Contains(response.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("SQL whitespace changed compression policy: len=%d Vary=%s", response.Body.Len(), response.Header().Get("Vary"))
	}
}
