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
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	moduleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
)

type moduleLifecycleMiddlewareRepository struct {
	moduleLifecycleTransportRepository
	recorder *cycleMiddlewareRecorder
}

func (repo moduleLifecycleMiddlewareRepository) AuthorizeAppProject(context.Context, string, string, string) error {
	repo.recorder.add("authorization")
	if repo.recorder.denied {
		return modulereadsstore.ErrForbidden
	}
	return nil
}

func (repo moduleLifecycleMiddlewareRepository) AuthorizeEntityProject(context.Context, string, string, string) error {
	repo.recorder.add("authorization")
	if repo.recorder.denied {
		return modulereadsstore.ErrForbidden
	}
	return nil
}

func (repo moduleLifecycleMiddlewareRepository) CreateModule(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error) {
	repo.recorder.add("dispatch")
	return json.RawMessage(`{}`), nil
}

func (repo moduleLifecycleMiddlewareRepository) CreateModuleV1(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error) {
	repo.recorder.add("dispatch")
	return json.RawMessage(`{"id":"33333333-3333-4333-8333-333333333333"}`), nil
}

func newModuleLifecycleMiddlewareTestHandler(t *testing.T, recorder *cycleMiddlewareRecorder, sessions *cycleMiddlewareSessions, timezone string, activity *cycleMiddlewareActivity, bodyLimit int64) (http.Handler, *activityHTTPService) {
	t.Helper()
	repo := moduleLifecycleMiddlewareRepository{recorder: recorder}
	keys := cycleMiddlewareAPIKeys{recorder: recorder, timezone: timezone}
	throttle := cycleMiddlewareThrottle{recorder: recorder}
	protocolActivity := &activityHTTPService{}
	service, err := systemapi.NewModuleLifecycle(&repo, sessions, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			moduleLifecycle:         moduleapi.NewEndpoints(service),
			moduleLifecycleMetadata: service,
			activityReads:           activityapi.NewEndpoints(protocolActivity),
			activityReadsMetadata:   protocolActivity,
			readServicesSession:     sessions,
			readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsActivity:   activity,
			activityReadsSecret:     "module-lifecycle-test-secret",
			activityReadsBodyLimit:  bodyLimit,
		},
	)
	return handler, protocolActivity
}

func TestModuleLifecycleProductionMiddlewareOrderRequestSizeSessionAPIThrottleDispatchAudit(t *testing.T) {
	const bodyLimit = 32
	apiPath := "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/"
	sessionPath := "/api/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/"
	for _, test := range []struct {
		name           string
		apiKey         bool
		denied         bool
		invalidZone    bool
		oversized      bool
		wantStatus     int
		wantEvents     []string
		wantAuditCount int
	}{
		{name: "API key permission then throttle dispatch and audit", apiKey: true, wantStatus: http.StatusCreated, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "dispatch", "audit"}, wantAuditCount: 1},
		{name: "API permission denial stops before throttle and dispatch then audits", apiKey: true, denied: true, wantStatus: http.StatusForbidden, wantEvents: []string{"api-key-authentication", "authorization", "audit"}, wantAuditCount: 1},
		{name: "API timezone validation follows authorization and throttle", apiKey: true, invalidZone: true, wantStatus: http.StatusBadRequest, wantEvents: []string{"api-key-authentication", "authorization", "throttling", "audit"}, wantAuditCount: 1},
		{name: "request-size API rejection precedes authentication and audit", apiKey: true, oversized: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "session timezone then authorization and dispatch", wantStatus: http.StatusCreated, wantEvents: []string{"session-inspection", "timezone", "authorization", "dispatch"}},
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
			handler, _ := newModuleLifecycleMiddlewareTestHandler(t, recorder, sessions, keyTimezone, activity, bodyLimit)
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
			if test.apiKey {
				request.Header.Set("X-Api-Key", "valid-api-key")
			} else if !test.oversized {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
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

func TestModuleLifecycleAPIV1OptionsAuthenticatesBeforeMethodDispatch(t *testing.T) {
	paths := []struct {
		name  string
		path  string
		allow string
	}{
		{
			name:  "collection",
			path:  "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/",
			allow: http.MethodPost,
		},
		{
			name:  "detail",
			path:  "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/",
			allow: "PATCH, DELETE",
		},
		{
			name:  "archive action",
			path:  "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/" + moduleLifecycleTransportModule + "/archive/",
			allow: http.MethodPost,
		},
		{
			name:  "unarchive action alias",
			path:  "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/archived-modules/" + moduleLifecycleTransportModule + "/unarchive/",
			allow: http.MethodDelete,
		},
	}
	for _, path := range paths {
		for _, test := range []struct {
			name       string
			key        string
			wantStatus int
			wantAudits int
		}{
			{name: "anonymous request is rejected", wantStatus: http.StatusUnauthorized},
			{name: "valid key reaches method dispatch", key: "valid-api-key", wantStatus: http.StatusMethodNotAllowed, wantAudits: 1},
		} {
			t.Run(path.name+"/"+test.name, func(t *testing.T) {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler, protocolActivity := newModuleLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
				request := httptest.NewRequest(http.MethodOptions, path.path, nil)
				if test.key != "" {
					request.Header.Set("X-Api-Key", test.key)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)

				if response.Code != test.wantStatus {
					t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
				}
				if protocolActivity.protocolCalls != 1 || protocolActivity.authKey != test.key || protocolActivity.protocolMethod != http.MethodOptions {
					t.Fatalf("protocol authorization calls=%d key=%q method=%q", protocolActivity.protocolCalls, protocolActivity.authKey, protocolActivity.protocolMethod)
				}
				if response.Header().Get("Allow") != path.allow {
					t.Errorf("Allow=%q, want %q", response.Header().Get("Allow"), path.allow)
				}
				for name, want := range map[string]string{
					"Cross-Origin-Opener-Policy": "same-origin",
					"Referrer-Policy":            "same-origin",
					"X-Content-Type-Options":     "nosniff",
					"X-Frame-Options":            "DENY",
				} {
					if got := response.Header().Get(name); got != want {
						t.Errorf("%s=%q, want %q", name, got, want)
					}
				}
				if !strings.Contains(response.Header().Get("Vary"), "origin") {
					t.Errorf("Vary=%q, want origin", response.Header().Get("Vary"))
				}
				if len(activity.records) != test.wantAudits {
					t.Errorf("API activity audit records=%d, want %d", len(activity.records), test.wantAudits)
				}
			})
		}
	}
}

func TestModuleLifecycleAPIV1ActionAliasesUseAuthenticatedMethodBoundary(t *testing.T) {
	project := moduleLifecycleTransportProject
	module := moduleLifecycleTransportModule
	paths := []struct {
		name        string
		path        string
		allow       string
		unsupported []string
	}{
		{
			name:        "archive",
			path:        "/api/v1/workspaces/demo/projects/" + project + "/modules/" + module + "/archive/",
			allow:       http.MethodPost,
			unsupported: []string{http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodPut, http.MethodPatch, http.MethodOptions, http.MethodTrace},
		},
		{
			name:        "unarchive alias",
			path:        "/api/v1/workspaces/demo/projects/" + project + "/archived-modules/" + module + "/unarchive/",
			allow:       http.MethodDelete,
			unsupported: []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodOptions, http.MethodTrace},
		},
	}
	for _, path := range paths {
		for _, method := range path.unsupported {
			t.Run(path.name+"/"+method, func(t *testing.T) {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler, protocolActivity := newModuleLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
				request := httptest.NewRequest(method, path.path, nil)
				request.Header.Set("X-Api-Key", "valid-api-key")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)

				if response.Code != http.StatusMethodNotAllowed {
					t.Fatalf("status=%d body=%q; want 405", response.Code, response.Body.String())
				}
				if response.Header().Get("Allow") != path.allow {
					t.Errorf("Allow=%q, want %q", response.Header().Get("Allow"), path.allow)
				}
				if response.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type=%q, want application/json", response.Header().Get("Content-Type"))
				}
				if !strings.Contains(response.Header().Get("Vary"), "origin") {
					t.Errorf("Vary=%q, want origin", response.Header().Get("Vary"))
				}
				if method != http.MethodHead {
					var body map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["detail"] != `Method "`+method+`" not allowed.` {
						t.Errorf("body=%q, want DRF method error; decode error=%v", response.Body.String(), err)
					}
				}
				if protocolActivity.protocolCalls != 1 || protocolActivity.authKey != "valid-api-key" || protocolActivity.protocolMethod != method {
					t.Errorf("API authorization calls=%d key=%q method=%q", protocolActivity.protocolCalls, protocolActivity.authKey, protocolActivity.protocolMethod)
				}
				if len(activity.records) != 1 {
					t.Errorf("API activity audit records=%d, want 1", len(activity.records))
				}
			})
		}
	}
}

func TestModuleLifecycleAPIV1SlashlessRoutesRedirectForEveryMethod(t *testing.T) {
	module := moduleLifecycleTransportModule
	paths := []struct {
		name string
		path string
	}{
		{
			name: "archive",
			path: "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/modules/" + module + "/archive/",
		},
		{
			name: "unarchive alias",
			path: "/api/v1/workspaces/demo/projects/" + moduleLifecycleTransportProject + "/archived-modules/" + module + "/unarchive/",
		},
	}
	for _, path := range paths {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace} {
			t.Run(path.name+"/"+method, func(t *testing.T) {
				recorder := &cycleMiddlewareRecorder{}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler, _ := newModuleLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
				request := httptest.NewRequest(method, strings.TrimSuffix(path.path, "/"), nil)
				request.Header.Set("X-Api-Key", "valid-api-key")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)

				if response.Code != http.StatusMovedPermanently {
					t.Fatalf("status=%d body=%q; want slashless 301", response.Code, response.Body.String())
				}
				if response.Header().Get("Location") != path.path {
					t.Errorf("Location=%q, want %q", response.Header().Get("Location"), path.path)
				}
				if !strings.Contains(response.Header().Get("Vary"), "Cookie") || !strings.Contains(response.Header().Get("Vary"), "origin") {
					t.Errorf("Vary=%q, want Cookie and origin", response.Header().Get("Vary"))
				}
				if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
					t.Errorf("Content-Type=%q, want Django redirect HTML", response.Header().Get("Content-Type"))
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Errorf("HEAD body=%q; want empty body", response.Body.String())
				}
			})
		}
	}
}

func TestModuleLifecycleUnsupportedSessionMethodsUseSourcePermissionBoundary(t *testing.T) {
	project := moduleLifecycleTransportProject
	module := moduleLifecycleTransportModule
	paths := []struct {
		name          string
		path          string
		method        string
		wantStatus    int
		wantAuthorize bool
	}{
		{
			name:       "ordinary module collection is authentication only",
			path:       "/api/workspaces/demo/projects/" + project + "/modules/",
			method:     http.MethodPut,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "ordinary module detail is authentication only",
			path:       "/api/workspaces/demo/projects/" + project + "/modules/" + module + "/",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "ordinary module TRACE remains authentication only",
			path:       "/api/workspaces/demo/projects/" + project + "/modules/" + module + "/",
			method:     http.MethodTrace,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:          "archived collection checks entity permission",
			path:          "/api/workspaces/demo/projects/" + project + "/archived-modules/",
			method:        http.MethodTrace,
			wantStatus:    http.StatusForbidden,
			wantAuthorize: true,
		},
		{
			name:          "archived detail checks entity permission",
			path:          "/api/workspaces/demo/projects/" + project + "/archived-modules/" + module + "/",
			method:        http.MethodTrace,
			wantStatus:    http.StatusForbidden,
			wantAuthorize: true,
		},
	}
	for _, test := range paths {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: true}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			handler, _ := newModuleLifecycleMiddlewareTestHandler(t, recorder, sessions, "UTC", activity, 1<<20)
			request := httptest.NewRequest(test.method, test.path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			authorizeCalls := 0
			for _, event := range recorder.events {
				if event == "authorization" {
					authorizeCalls++
				}
			}
			if got := authorizeCalls > 0; got != test.wantAuthorize {
				t.Errorf("authorization events=%v; authorized=%v, want %v", recorder.events, got, test.wantAuthorize)
			}
		})
	}
}
