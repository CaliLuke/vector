package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	api "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	statesapi "github.com/CaliLuke/go-plane/gen/project_state_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
)

const stateMiddlewareProject = "22222222-2222-4222-8222-222222222222"

type stateWritesMiddlewareRepository struct {
	recorder     *cycleMiddlewareRecorder
	authorizeErr error
}

func (r stateWritesMiddlewareRepository) AuthorizeProject(context.Context, uuid.UUID, string, uuid.UUID) error {
	return nil
}
func (r stateWritesMiddlewareRepository) List(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}
func (r stateWritesMiddlewareRepository) Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}
func (r stateWritesMiddlewareRepository) Intake(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}
func (r stateWritesMiddlewareRepository) InvalidateAppCache(context.Context, string) {}
func (r stateWritesMiddlewareRepository) AuthorizeApp(context.Context, string, string, string, string) (bool, error) {
	r.recorder.add("authorization")
	return !r.recorder.denied, r.authorizeErr
}
func (r stateWritesMiddlewareRepository) AuthorizeAPIV1(context.Context, string, string, string, bool) error {
	r.recorder.add("authorization")
	if r.recorder.denied {
		return store.ErrForbidden
	}
	return nil
}
func (r stateWritesMiddlewareRepository) ReadAPIV1(context.Context, string, store.APIV1Request) (any, error) {
	return nil, nil
}
func (r stateWritesMiddlewareRepository) ResolveForUpdate(context.Context, store.StateWriteInput) error {
	return nil
}
func (r stateWritesMiddlewareRepository) Create(context.Context, store.StateWriteInput) (json.RawMessage, error) {
	r.recorder.add("dispatch")
	return json.RawMessage(`{"name":"State"}`), nil
}
func (r stateWritesMiddlewareRepository) Update(ctx context.Context, input store.StateWriteInput) (json.RawMessage, error) {
	return r.Create(ctx, input)
}
func (r stateWritesMiddlewareRepository) Delete(context.Context, store.StateWriteInput) error {
	r.recorder.add("dispatch")
	return nil
}
func (r stateWritesMiddlewareRepository) MarkDefault(ctx context.Context, input store.StateWriteInput) error {
	return r.Delete(ctx, input)
}
func (r stateWritesMiddlewareRepository) SerializeAPIV1(_ context.Context, raw json.RawMessage, _, _ string) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal(raw, &result)
	return result, err
}

func newStateWritesMiddlewareTestHandler(t *testing.T, repo stateWritesMiddlewareRepository, sessions *cycleMiddlewareSessions, timezone string, activity *cycleMiddlewareActivity, bodyLimit int64) http.Handler {
	t.Helper()
	appService, err := systemapi.NewProjectStateReads(repo, sessions)
	if err != nil {
		t.Fatal(err)
	}
	apiService, err := systemapi.NewAPIStateReads(repo, cycleMiddlewareAPIKeys{recorder: repo.recorder, timezone: timezone}, cycleMiddlewareThrottle{recorder: repo.recorder})
	if err != nil {
		t.Fatal(err)
	}
	protocolActivity := &activityHTTPService{}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			projectStateReads: statesapi.NewEndpoints(appService), projectStateMetadata: appService.(systemapi.ProjectStateReadsMetadataAuthorizer),
			apiStateReads: api.NewEndpoints(apiService),
			activityReads: activityapi.NewEndpoints(protocolActivity), activityReadsMetadata: protocolActivity,
			readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsActivity: activity, activityReadsSecret: "state-writes-test-secret", activityReadsBodyLimit: bodyLimit,
		})
}

func TestStateWriteAuthorizationFailurePrecedesBodyParsingAndDispatch(t *testing.T) {
	recorder := &cycleMiddlewareRecorder{}
	repository := stateWritesMiddlewareRepository{recorder: recorder, authorizeErr: errors.New("database unavailable")}
	sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
	activity := &cycleMiddlewareActivity{recorder: recorder}
	handler := newStateWritesMiddlewareTestHandler(t, repository, sessions, "UTC", activity, 1024)
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces/demo/projects/"+stateMiddlewareProject+"/states/", strings.NewReader(`{`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("authorization failure status=%d body=%s", response.Code, response.Body.String())
	}
	if !reflect.DeepEqual(recorder.events, []string{"session-inspection", "timezone", "authorization"}) {
		t.Fatalf("authorization failure events=%v", recorder.events)
	}
	if len(activity.records) != 0 {
		t.Fatalf("session write unexpectedly produced API-key audit records: %v", activity.records)
	}
}

func TestProjectStateWritesProductionMiddlewareOrderRequestSizeSessionAPIThrottleDispatchAudit(t *testing.T) {
	const bodyLimit = 32
	apiPath := "/api/v1/workspaces/demo/projects/" + stateMiddlewareProject + "/states/"
	sessionPath := "/api/workspaces/demo/projects/" + stateMiddlewareProject + "/states/"
	for _, test := range []struct {
		name           string
		apiKey         bool
		anonymous      bool
		denied         bool
		invalidZone    bool
		oversized      bool
		wantStatus     int
		wantEvents     []string
		wantAuditCount int
	}{
		{name: "anonymous API write is unauthenticated", apiKey: true, anonymous: true, wantStatus: http.StatusUnauthorized, wantEvents: []string{"api-key-authentication"}},
		{name: "anonymous app write is unauthenticated", anonymous: true, wantStatus: http.StatusUnauthorized, wantEvents: []string{"session-inspection"}},
		{name: "API key permission then throttle dispatch and audit", apiKey: true, wantStatus: http.StatusOK, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "dispatch", "audit"}, wantAuditCount: 1},
		{name: "API permission denial stops before throttle and dispatch then audits", apiKey: true, denied: true, wantStatus: http.StatusForbidden, wantEvents: []string{"api-key-authentication", "authorization", "audit"}, wantAuditCount: 1},
		{name: "API timezone validation follows authorization and throttle", apiKey: true, invalidZone: true, wantStatus: http.StatusBadRequest, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "audit"}, wantAuditCount: 1},
		{name: "request-size API rejection precedes authentication and audit", apiKey: true, oversized: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "session timezone then authorization and dispatch", wantStatus: http.StatusOK, wantEvents: []string{"session-inspection", "timezone", "authorization", "dispatch"}},
		{name: "session permission denial stops dispatch", denied: true, wantStatus: http.StatusForbidden, wantEvents: []string{"session-inspection", "timezone", "authorization"}},
		{name: "session timezone rejection precedes authorization", invalidZone: true, wantStatus: http.StatusBadRequest, wantEvents: []string{"session-inspection", "timezone"}},
		{name: "request-size session rejection precedes session inspection", oversized: true, wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: test.denied}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC", invalidZone: test.invalidZone && !test.apiKey}
			keyTimezone := "UTC"
			if test.invalidZone && test.apiKey {
				keyTimezone = "Not/AZone"
			}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			handler := newStateWritesMiddlewareTestHandler(t, stateWritesMiddlewareRepository{recorder: recorder}, sessions, keyTimezone, activity, bodyLimit)
			body := `{}`
			if test.oversized {
				body = strings.Repeat("x", bodyLimit+1)
			}
			path := sessionPath
			if test.apiKey {
				path = apiPath
			}
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			if test.apiKey && !test.anonymous {
				request.Header.Set("X-Api-Key", "valid-api-key")
			} else if !test.apiKey && !test.oversized && !test.anonymous {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if response.Code >= 400 && !test.oversized {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{}
				switch {
				case test.anonymous:
					want["detail"] = "Authentication credentials were not provided."
				case test.invalidZone:
					want["error"] = "The required key does not exist."
				case test.apiKey:
					want["detail"] = "You do not have permission to perform this action."
				default:
					want["error"] = "You don't have the required permissions."
				}
				if !reflect.DeepEqual(body, want) {
					t.Fatalf("error body=%#v, want %#v", body, want)
				}
			}
			if !reflect.DeepEqual(recorder.events, test.wantEvents) {
				t.Fatalf("events=%v, want %v", recorder.events, test.wantEvents)
			}
			if len(activity.records) != test.wantAuditCount {
				t.Fatalf("API activity audit records=%d, want %d", len(activity.records), test.wantAuditCount)
			}
		})
	}
}

func TestProjectStateWriteHeadersMatchRetainedDjangoMethods(t *testing.T) {
	for _, test := range []struct {
		method, suffix, allow, vary string
		status                      int
	}{
		{http.MethodPost, "/states/", "POST, OPTIONS", "Accept-Encoding, Cookie, origin", http.StatusOK},
		{http.MethodPatch, "/states/33333333-3333-4333-8333-333333333333/", "PATCH, DELETE, OPTIONS", "Accept-Encoding, Cookie, origin", http.StatusOK},
		{http.MethodDelete, "/states/33333333-3333-4333-8333-333333333333/", "PATCH, DELETE, OPTIONS", "Cookie, origin", http.StatusNoContent},
		{http.MethodPost, "/states/33333333-3333-4333-8333-333333333333/mark-default/", "POST, OPTIONS", "Cookie, origin", http.StatusNoContent},
	} {
		t.Run(test.method+test.suffix, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			handler := newStateWritesMiddlewareTestHandler(t, stateWritesMiddlewareRepository{recorder: recorder}, sessions, "UTC", &cycleMiddlewareActivity{recorder: recorder}, 1024)
			req := httptest.NewRequest(test.method, "/api/workspaces/demo/projects/"+stateMiddlewareProject+test.suffix, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.status || response.Header().Get("Allow") != test.allow || response.Header().Get("Vary") != test.vary {
				t.Fatalf("status=%d Allow=%q Vary=%q body=%q", response.Code, response.Header().Get("Allow"), response.Header().Get("Vary"), response.Body.String())
			}
		})
	}
}

func TestProjectStateProtocolDoesNotClaimUnrelatedPaths(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/demo/projects/" + stateMiddlewareProject + "/states/33333333-3333-4333-8333-333333333333/unknown/",
		"/api/v1/workspaces/demo/projects/" + stateMiddlewareProject + "/states/33333333-3333-4333-8333-333333333333/mark-default/",
		"/api/workspaces/demo/projects/" + stateMiddlewareProject + "/states/33333333-3333-4333-8333-333333333333/mark-default/extra/",
	} {
		if protocol, ok := resolveReadServiceProtocol(path); ok {
			t.Errorf("unrelated path %s claimed by %s", path, protocol.pattern)
		}
	}
}

func TestProjectStateProtocolAuthenticatesOptionsAndUnsupportedMethods(t *testing.T) {
	for _, suffix := range []string{"/states/", "/states/33333333-3333-4333-8333-333333333333/", "/states/33333333-3333-4333-8333-333333333333/mark-default/"} {
		for _, method := range []string{http.MethodOptions, http.MethodPut, http.MethodTrace} {
			for _, authenticated := range []bool{false, true} {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
				handler := newStateWritesMiddlewareTestHandler(t, stateWritesMiddlewareRepository{recorder: recorder}, sessions, "UTC", &cycleMiddlewareActivity{recorder: recorder}, 1024)
				req := httptest.NewRequest(method, "/api/workspaces/demo/projects/"+stateMiddlewareProject+suffix, nil)
				if authenticated {
					req.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				want := http.StatusUnauthorized
				if authenticated {
					want = http.StatusMethodNotAllowed
					if method == http.MethodOptions {
						want = http.StatusOK
					}
				}
				if response.Code != want {
					t.Errorf("%s %s auth=%v status=%d body=%q want=%d", method, suffix, authenticated, response.Code, response.Body.String(), want)
				}
				for _, event := range recorder.events {
					if event == "dispatch" {
						t.Errorf("protocol method performed domain mutation: %v", recorder.events)
					}
				}
			}
		}
	}
}

func TestProjectStateOptionsMatchesCapturedSource(t *testing.T) {
	data, err := os.ReadFile("testdata/project_state_writes_source_options.json")
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]struct {
		Body    any               `json:"body"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	for name, suffix := range map[string]string{
		"collection":   "/states/",
		"detail":       "/states/33333333-3333-4333-8333-333333333333/",
		"mark-default": "/states/33333333-3333-4333-8333-333333333333/mark-default/",
	} {
		t.Run(name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			handler := newStateWritesMiddlewareTestHandler(t, stateWritesMiddlewareRepository{recorder: recorder}, sessions, "UTC", &cycleMiddlewareActivity{recorder: recorder}, 1024)
			req := httptest.NewRequest(http.MethodOptions, "/api/workspaces/demo/projects/"+stateMiddlewareProject+suffix, nil)
			req.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			var got any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			want := source[name]
			if response.Code != http.StatusOK || !reflect.DeepEqual(got, want.Body) {
				t.Fatalf("OPTIONS status=%d body=%s, want captured source %#v", response.Code, response.Body.String(), want.Body)
			}
			if allow := response.Header().Get("Allow"); allow != want.Headers["Allow"] {
				t.Fatalf("Allow=%q, want %q", allow, want.Headers["Allow"])
			}
		})
	}
}
