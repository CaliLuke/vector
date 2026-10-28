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
	"time"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	workspaceapi "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
)

type workspaceLifecycleMiddlewareTrace struct {
	events          []string
	role            int
	invalidTimezone bool
}

type workspaceLifecycleTraceBody struct {
	io.ReadCloser
	trace *workspaceLifecycleMiddlewareTrace
	read  bool
}

func (body *workspaceLifecycleTraceBody) Read(value []byte) (int, error) {
	if !body.read {
		body.read = true
		body.trace.add("request-size")
	}
	return body.ReadCloser.Read(value)
}

func (trace *workspaceLifecycleMiddlewareTrace) add(event string) {
	trace.events = append(trace.events, event)
}

type workspaceLifecycleMiddlewareSessions struct {
	trace *workspaceLifecycleMiddlewareTrace
}

// Catch signature drift before the middleware's optional type assertion fails.
var _ sessionPrincipalInspector = workspaceLifecycleMiddlewareSessions{}

func (sessions workspaceLifecycleMiddlewareSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	sessions.trace.add("session-authentication")
	return djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: "UTC"}, nil
}

func (sessions workspaceLifecycleMiddlewareSessions) InspectReadServiceSession(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	sessions.trace.add("session-authentication")
	if len(request.CookiesNamed("session-id")) == 0 {
		return false, false, nil, djangosession.Principal{}, nil
	}
	sessions.trace.add("timezone")
	if sessions.trace.invalidTimezone {
		return false, false, nil, djangosession.Principal{}, djangosession.ErrTimezoneNotFound
	}
	return false, false, nil, djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: "UTC"}, nil
}

func (sessions workspaceLifecycleMiddlewareSessions) InspectReadServiceSessionWithoutTimezone(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	sessions.trace.add("session-authentication")
	if len(request.CookiesNamed("session-id")) == 0 {
		return false, false, nil, djangosession.Principal{}, nil
	}
	return false, false, nil, djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: "UTC"}, nil
}

func (workspaceLifecycleMiddlewareSessions) InspectSession(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	return false, false, nil, cycleMiddlewareActor, nil
}

func (workspaceLifecycleMiddlewareSessions) InspectSessionWithTimezone(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	return false, false, nil, cycleMiddlewareActor, nil
}

func (workspaceLifecycleMiddlewareSessions) RefreshSession(_ context.Context, _ *http.Request, refresh djangosession.SessionRefresh) (djangosession.SessionRefresh, error) {
	return refresh, nil
}

func (workspaceLifecycleMiddlewareSessions) AllowRequest(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return true, 0, false, false, nil, cycleMiddlewareActor, nil
}

func (workspaceLifecycleMiddlewareSessions) AllowRequestWithoutTimezone(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return true, 0, false, false, nil, cycleMiddlewareActor, nil
}

func (workspaceLifecycleMiddlewareSessions) FinalizeUnaccessedSession(context.Context, *http.Request, int) (bool, *djangosession.SessionRefresh, error) {
	return false, nil, nil
}

type workspaceLifecycleMiddlewareRepository struct {
	trace *workspaceLifecycleMiddlewareTrace
}

func (repository workspaceLifecycleMiddlewareRepository) WorkspaceRole(context.Context, uuid.UUID, string) (int, error) {
	repository.trace.add("authorization")
	return repository.trace.role, nil
}

func (workspaceLifecycleMiddlewareRepository) WorkspaceCreationDisabled(context.Context) (bool, error) {
	return false, nil
}

func (workspaceLifecycleMiddlewareRepository) CreateWorkspace(context.Context, uuid.UUID, map[string]json.RawMessage, string) (json.RawMessage, error) {
	return nil, nil
}

func (repository workspaceLifecycleMiddlewareRepository) UpdateWorkspace(context.Context, uuid.UUID, string, map[string]json.RawMessage) (json.RawMessage, error) {
	repository.trace.add("dispatch")
	return json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000001","name":"Updated","slug":"demo"}`), nil
}

func (workspaceLifecycleMiddlewareRepository) PrepareWorkspaceDeletion(context.Context, uuid.UUID, string) (instancemanagement.WorkspaceDeletion, error) {
	return instancemanagement.WorkspaceDeletion{}, nil
}

func (workspaceLifecycleMiddlewareRepository) SoftDeleteWorkspace(context.Context, uuid.UUID, instancemanagement.WorkspaceDeletion, time.Time) error {
	return nil
}

type workspaceLifecycleMiddlewarePublisher struct{}

func (workspaceLifecycleMiddlewarePublisher) Task(context.Context, string, []any, map[string]any) error {
	return nil
}

func (workspaceLifecycleMiddlewarePublisher) SoftDelete(context.Context, string, string, string) error {
	return nil
}

func newWorkspaceLifecycleMiddlewareHandler(t *testing.T, trace *workspaceLifecycleMiddlewareTrace, bodyLimit int64) http.Handler {
	t.Helper()
	sessions := workspaceLifecycleMiddlewareSessions{trace: trace}
	service, err := systemapi.NewWorkspaceLifecycle(
		workspaceLifecycleMiddlewareRepository{trace: trace},
		sessions,
		workspaceLifecycleMiddlewarePublisher{},
	)
	if err != nil {
		t.Fatal(err)
	}
	activity := &apiV1ActivityLoggerStub{beforeLog: func() { trace.add("audit") }}
	return extensionHTTPHandler(pendingReplacementEndpoints{
		workspaceLifecycle:     workspaceapi.NewEndpoints(service),
		readServicesSession:    sessions,
		readServicesCookie:     djangoSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsBodyLimit: bodyLimit,
		activityReadsActivity:  activity,
	})
}

func TestWorkspaceLifecycleMiddlewareOrderRequestSizeAuthenticationTimezoneAuthorizationDispatchAudit(t *testing.T) {
	const bodyLimit = 32
	for _, test := range []struct {
		name            string
		method          string
		cookie          bool
		role            int
		audit           bool
		invalidTimezone bool
		body            string
		wantStatus      int
		wantEvents      []string
	}{
		{name: "authorized update", method: http.MethodPatch, cookie: true, role: 20, audit: true, body: `{"name":"Updated"}`, wantStatus: http.StatusOK, wantEvents: []string{"request-size", "session-authentication", "timezone", "authorization", "dispatch", "audit"}},
		{name: "member denied before dispatch", method: http.MethodPatch, cookie: true, role: 15, audit: true, body: `{"name":"Denied"}`, wantStatus: http.StatusForbidden, wantEvents: []string{"request-size", "session-authentication", "timezone", "authorization", "audit"}},
		{name: "anonymous rejected before authorization", method: http.MethodPut, body: `{}`, wantStatus: http.StatusUnauthorized, wantEvents: []string{"request-size", "session-authentication"}},
		{name: "invalid timezone before authorization", method: http.MethodOptions, cookie: true, invalidTimezone: true, wantStatus: http.StatusBadRequest, wantEvents: []string{"session-authentication", "timezone"}},
		{name: "request size before authentication and audit", method: http.MethodPatch, cookie: true, role: 20, audit: true, body: `{"name":"` + strings.Repeat("x", bodyLimit+1) + `"}`, wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			trace := &workspaceLifecycleMiddlewareTrace{role: test.role, invalidTimezone: test.invalidTimezone}
			handler := newWorkspaceLifecycleMiddlewareHandler(t, trace, bodyLimit)
			request := httptest.NewRequest(test.method, "/api/workspaces/demo/", strings.NewReader(test.body))
			if test.body != "" {
				request.Body = &workspaceLifecycleTraceBody{ReadCloser: request.Body, trace: trace}
			}
			request.Header.Set("Content-Type", "application/json")
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			if test.audit {
				request.Header.Set("X-Api-Key", "middleware-audit-test-key")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s want=%d", response.Code, response.Body.String(), test.wantStatus)
			}
			if !reflect.DeepEqual(trace.events, test.wantEvents) {
				t.Fatalf("middleware/effect events=%v want=%v", trace.events, test.wantEvents)
			}
			// HTTP 413 proves the request-size stage stopped the request before
			// session authentication, authorization, dispatch, or audit.
			if test.wantStatus == http.StatusRequestEntityTooLarge && len(trace.events) != 0 {
				t.Fatalf("request-size boundary ran after downstream effects: %v", trace.events)
			}
		})
	}
}

func TestWorkspaceLifecycleRejectsArrayBodyWithClientError(t *testing.T) {
	trace := &workspaceLifecycleMiddlewareTrace{role: 20}
	handler := newWorkspaceLifecycleMiddlewareHandler(t, trace, 32)
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces/", strings.NewReader("[]"))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	request.Body = &workspaceLifecycleTraceBody{ReadCloser: request.Body, trace: trace}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("array request status=%d body=%s, want 400", response.Code, response.Body.String())
	}
	if !reflect.DeepEqual(trace.events, []string{"request-size", "session-authentication", "timezone"}) {
		t.Fatalf("array request unexpectedly reached authorization or dispatch: %v", trace.events)
	}
}
