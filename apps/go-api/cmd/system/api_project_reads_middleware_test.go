package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	storeuuid "uuid"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/gen/system"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
)

type apiProjectReadsMiddlewareRepository struct {
	projectLifecycleMiddlewareRepository
	listOptions projects.APIProjectListOptions
}

func (repository *apiProjectReadsMiddlewareRepository) APIProjectList(_ context.Context, _ storeuuid.UUID, _ string, options projects.APIProjectListOptions) (json.RawMessage, error) {
	repository.recorder.add("dispatch")
	repository.listOptions = options
	return json.RawMessage(`{"results":[]}`), nil
}

func (repository *apiProjectReadsMiddlewareRepository) APIProject(context.Context, storeuuid.UUID, string, storeuuid.UUID, projects.APIProjectDetailOptions) (json.RawMessage, error) {
	repository.recorder.add("dispatch")
	return json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000001"}`), nil
}

func newAPIProjectReadsMiddlewareTestHandler(t *testing.T, repository *apiProjectReadsMiddlewareRepository, recorder *cycleMiddlewareRecorder, sessions *cycleMiddlewareSessions, activity *cycleMiddlewareActivity, bodyLimit int64) http.Handler {
	t.Helper()
	service, err := systemapi.NewProjectLifecycle(repository, sessions, cycleMiddlewareAPIKeys{recorder: recorder, timezone: "UTC"}, cycleMiddlewareThrottle{recorder: recorder})
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
			activityReadsActivity: activity, activityReadsSecret: "api-project-reads-test-secret", activityReadsBodyLimit: bodyLimit,
		},
	)
}

func TestAPIProjectReadsMiddlewareOrderRequestSizeAuthenticationAuthorizationThrottlingDispatchAudit(t *testing.T) {
	// The API-key read chain is request-size, authentication, permission authorization,
	// throttling, handler dispatch, and API-key audit. Anonymous calls stop at auth.
	for _, test := range []struct {
		name       string
		apiKey     bool
		oversized  bool
		wantStatus int
		wantEvents []string
		wantAudit  int
	}{
		{name: "authenticated list dispatch and audit", apiKey: true, wantStatus: http.StatusOK, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "dispatch", "audit"}, wantAudit: 1},
		{name: "anonymous list denied before dispatch", wantStatus: http.StatusUnauthorized, wantEvents: []string{"api-key-authentication"}},
		{name: "request-size before authentication", oversized: true, wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			repository := &apiProjectReadsMiddlewareRepository{projectLifecycleMiddlewareRepository: projectLifecycleMiddlewareRepository{recorder: recorder}}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			handler := newAPIProjectReadsMiddlewareTestHandler(t, repository, recorder, sessions, activity, 32)
			body := ""
			if test.oversized {
				body = strings.Repeat("x", 33)
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/?cursor=2%3A0%3A0&per_page=2&fields=id%2Cname%2Cworkspace&expand=workspace&order_by=name", strings.NewReader(body))
			if test.apiKey {
				request.Header.Set("X-Api-Key", "valid-api-key")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if strings.Join(recorder.events, ",") != strings.Join(test.wantEvents, ",") {
				t.Fatalf("middleware events=%v want %v", recorder.events, test.wantEvents)
			}
			if len(activity.records) != test.wantAudit {
				t.Fatalf("audit records=%d want %d", len(activity.records), test.wantAudit)
			}
			if test.apiKey {
				if repository.listOptions.Fields != "id,name,workspace" || repository.listOptions.Expand != "workspace" || repository.listOptions.Cursor == nil || *repository.listOptions.Cursor != "2:0:0" || repository.listOptions.PerPage == nil || *repository.listOptions.PerPage != "2" || repository.listOptions.OrderBy == nil || *repository.listOptions.OrderBy != "name" {
					t.Fatalf("list query options were not preserved: %#v", repository.listOptions)
				}
				if activity.records[0].Path != "/api/v1/workspaces/demo/projects/" || activity.records[0].Method != http.MethodGet || activity.records[0].ResponseCode != http.StatusOK {
					t.Fatalf("audit record does not match request: %#v", activity.records[0])
				}
			}
		})
	}
}
