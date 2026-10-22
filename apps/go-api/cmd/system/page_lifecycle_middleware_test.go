package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type pageLifecycleAuthRecorder struct {
	pageUnauthorizedService
	calls int
}

type pageLifecycleProtocolAuthorizerStub struct {
	status int
	calls  []systemapi.SessionProtocolResource
}

func (authorizer *pageLifecycleProtocolAuthorizerStub) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return 0, nil
}

func (authorizer *pageLifecycleProtocolAuthorizerStub) AuthorizeSessionResourceProtocol(_ context.Context, _ string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	authorizer.calls = append(authorizer.calls, resource)
	if authorizer.status != 0 {
		return authorizer.status, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	return 0, nil
}

func (service *pageLifecycleAuthRecorder) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	service.calls++
	return service.pageUnauthorizedService.APIKeyAuth(ctx, key, scheme)
}

func TestPageLifecycleMountedRequestSizeAuthenticationPermissionTimezoneDispatchAndNoAudit(t *testing.T) {
	service := &pageLifecycleAuthRecorder{}
	mux := loomhttp.NewMuxer()
	server := pagereadssvr.New(pagereads.NewEndpoints(service), mux,
		loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	pagereadssvr.Mount(mux, server)
	mountPageReadsProtocolRoutes(mux, server, loomhttp.RuntimeCORSPolicy{})
	activity := &cycleMiddlewareActivity{recorder: &cycleMiddlewareRecorder{}}
	options := apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "page-test-secret", bodySizeLimit: 4}
	handler := readServicesCompatibilityMiddleware(activityReadsRequestSizeMiddleware(4)(activityReadsAuditFallbackMiddleware(options)(mux)))
	const path = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/"
	request := func(body string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		return request
	}

	// The shared request-size boundary rejects oversized bytes before generated
	// authentication and authorization or dispatch, and it removes path method headers.
	oversized := httptest.NewRecorder()
	handler.ServeHTTP(oversized, request(`{"large":true}`))
	if oversized.Code != http.StatusRequestEntityTooLarge || service.calls != 0 {
		t.Fatalf("request-size status=%d auth calls=%d body=%s", oversized.Code, service.calls, oversized.Body.String())
	}

	// A bounded page request reaches generated authentication. The page-specific
	// permission-classes-before-timezone assertion is in
	// apps/go-api/page_reads_test.go::TestPageLifecyclePermissionTimezoneAndHandlerOrder;
	// this mounted boundary verifies request-size/authentication and the absence
	// of API audit for django-session traffic.
	authenticatedBoundary := httptest.NewRecorder()
	handler.ServeHTTP(authenticatedBoundary, request(`{}`))
	if authenticatedBoundary.Code != http.StatusUnauthorized || service.calls != 1 {
		t.Fatalf("authentication status=%d auth calls=%d body=%s", authenticatedBoundary.Code, service.calls, authenticatedBoundary.Body.String())
	}
	if len(activity.records) != 0 {
		t.Fatalf("django-session page request emitted API audit records: %+v", activity.records)
	}
}

type pageLifecycleWriteService struct {
	pagereads.Service
	dispatches   int
	authCalls    int
	optionsCalls int
	denied       bool
	denyAuth     bool
	responseBody string
}

func (service *pageLifecycleWriteService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	service.authCalls++
	if service.denyAuth {
		return ctx, &pagereads.PageReadsUnauthenticated{Detail: "Authentication credentials were not provided."}
	}
	return ctx, nil
}

func (service *pageLifecycleWriteService) Create(context.Context, *pagereads.CreatePayload, io.ReadCloser) (*pagereads.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	service.dispatches++
	return service.response()
}

func (service *pageLifecycleWriteService) Update(context.Context, *pagereads.UpdatePayload, io.ReadCloser) (*pagereads.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	service.dispatches++
	return service.response()
}

func (service *pageLifecycleWriteService) CreateOptions(context.Context, *pagereads.CreateOptionsPayload) (*pagereads.RawJSONResponseMetadata, io.ReadCloser, error) {
	service.optionsCalls++
	return nil, nil, &pagereads.PageReadsUnauthenticated{Detail: "Authentication credentials were not provided."}
}

func (service *pageLifecycleWriteService) response() (*pagereads.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	if service.denied {
		body, err := loom.JSONValueFrom(map[string]any{"detail": "denied"})
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, &pagereads.PageLifecycleError{Kind: "status_403", Body: body}
	}
	body := service.responseBody
	if body == "" {
		body = `{"id":"created"}`
	}
	return &pagereads.PageLifecycleResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(body)), nil
}

func newPageLifecycleProductionWriteHandler(t *testing.T, service *pageLifecycleWriteService, sessions *cycleMiddlewareSessions, authorizer *pageLifecycleProtocolAuthorizerStub, activity *cycleMiddlewareActivity, bodyLimit int64) http.Handler {
	t.Helper()
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			pageReads: pagereads.NewEndpoints(service), pageReadsMetadata: authorizer,
			readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true},
			readServicesCORS:      []string{"https://app.example.test"},
			activityReadsActivity: activity, activityReadsSecret: "page-test-secret", activityReadsBodyLimit: bodyLimit,
		},
	)
}

func TestPageLifecycleProductionWriteTransportBoundary(t *testing.T) {
	const collection = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/"
	const detail = collection + "00000000-0000-4000-8000-000000000002/"
	for _, test := range []struct {
		name             string
		method           string
		path             string
		body             string
		accept           string
		acceptEncoding   string
		query            string
		responseBody     string
		wantGzip         bool
		apiKey           bool
		session          bool
		invalidTimezone  bool
		preflight        bool
		denied           bool
		denyAuth         bool
		wantStatus       int
		wantAudit        int
		wantCalls        int
		wantAuthCalls    int
		wantOptionsCalls int
		wantInspections  int
	}{
		{name: "oversized body is rejected before negotiation and session inspection", method: http.MethodPost, path: collection, body: `{"too":"large"}`, accept: "text/plain", apiKey: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "Accept denial with token is audited", method: http.MethodPatch, path: detail, body: `{}`, accept: "text/plain", apiKey: true, wantStatus: http.StatusNotAcceptable, wantAudit: 1},
		{name: "format denial with token is audited", method: http.MethodPatch, path: detail + "?format=unsupported", body: `{}`, apiKey: true, wantStatus: http.StatusNotFound, wantAudit: 1},
		{name: "Accept denial without token has no audit", method: http.MethodPatch, path: detail, body: `{}`, accept: "text/plain", wantStatus: http.StatusNotAcceptable},
		{name: "session inspection error with token is audited", method: http.MethodPatch, path: detail, body: `{}`, apiKey: true, session: true, invalidTimezone: true, wantStatus: http.StatusBadRequest, wantAudit: 1, wantInspections: 1},
		{name: "authentication denied PATCH with token is audited", method: http.MethodPatch, path: detail, body: `{}`, apiKey: true, denyAuth: true, wantStatus: http.StatusUnauthorized, wantAudit: 1, wantAuthCalls: 1, wantInspections: 1},
		{name: "supported PATCH with token is audited", method: http.MethodPatch, path: detail, body: `{}`, apiKey: true, session: true, wantStatus: http.StatusOK, wantAudit: 1, wantCalls: 1, wantAuthCalls: 1, wantInspections: 1},
		{name: "gzip response audit captures the uncompressed body", method: http.MethodPatch, path: detail, body: `{}`, apiKey: true, session: true, acceptEncoding: "gzip", responseBody: `{"value":"` + strings.Repeat("a", 240) + `"}`, wantGzip: true, wantStatus: http.StatusOK, wantAudit: 1, wantCalls: 1, wantAuthCalls: 1, wantInspections: 1},
		{name: "supported POST without token has no audit", method: http.MethodPost, path: collection, body: `{}`, session: true, wantStatus: http.StatusCreated, wantCalls: 1, wantAuthCalls: 1, wantInspections: 1},
		{name: "denied supported PATCH with token is audited", method: http.MethodPatch, path: detail, body: `{}`, apiKey: true, session: true, denied: true, wantStatus: http.StatusForbidden, wantAudit: 1, wantCalls: 1, wantAuthCalls: 1, wantInspections: 1},
		{name: "ordinary OPTIONS authentication denial is audited", method: http.MethodOptions, path: collection, body: `{}`, apiKey: true, wantStatus: http.StatusUnauthorized, wantAudit: 1, wantAuthCalls: 1, wantOptionsCalls: 1, wantInspections: 1},
		{name: "oversized ordinary OPTIONS is rejected before authentication", method: http.MethodOptions, path: collection, body: `{"too":"large"}`, apiKey: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "CORS preflight bypasses write audit", method: http.MethodOptions, path: collection, preflight: true, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			authorizer := &pageLifecycleProtocolAuthorizerStub{}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			service := &pageLifecycleWriteService{}
			service.denied = test.denied
			service.denyAuth = test.denyAuth
			service.responseBody = test.responseBody
			sessions.invalidZone = test.invalidTimezone
			handler := newPageLifecycleProductionWriteHandler(t, service, sessions, authorizer, activity, 4)
			request := httptest.NewRequest(test.method, test.path+test.query, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}
			if test.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", test.acceptEncoding)
			}
			if test.apiKey {
				request.Header.Set("X-Api-Key", "raw-page-token")
			}
			if test.session {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			if test.preflight {
				request.Header.Set("Origin", "https://app.example.test")
				request.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || len(activity.records) != test.wantAudit || service.dispatches != test.wantCalls || service.authCalls != test.wantAuthCalls || service.optionsCalls != test.wantOptionsCalls {
				t.Fatalf("status=%d audit=%d auth=%d options=%d dispatch=%d body=%s; want status=%d audit=%d auth=%d options=%d dispatch=%d", response.Code, len(activity.records), service.authCalls, service.optionsCalls, service.dispatches, response.Body.String(), test.wantStatus, test.wantAudit, test.wantAuthCalls, test.wantOptionsCalls, test.wantCalls)
			}
			inspections := 0
			for _, event := range recorder.events {
				if event == "session-inspection" {
					inspections++
				}
			}
			if inspections != test.wantInspections {
				t.Fatalf("session inspections=%d, want %d; events=%v", inspections, test.wantInspections, recorder.events)
			}
			if test.preflight && response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
				t.Fatalf("preflight origin header=%q", response.Header().Get("Access-Control-Allow-Origin"))
			}
			if got := response.Header().Get("Content-Encoding") == "gzip"; got != test.wantGzip {
				t.Fatalf("gzip response=%t, want %t; headers=%v", got, test.wantGzip, response.Header())
			}
			if test.wantGzip {
				reader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
				if err != nil {
					t.Fatalf("open gzip response: %v", err)
				}
				body, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("read gzip response: %v", err)
				}
				if err := reader.Close(); err != nil {
					t.Fatalf("close gzip response: %v", err)
				}
				if string(body) != test.responseBody {
					t.Fatalf("decompressed response body differs; got %q want %q", string(body), test.responseBody)
				}
			}
			if test.wantAudit == 1 {
				record := activity.records[0]
				if record.ResponseCode != test.wantStatus || record.TokenIdentifier != apiActivityTokenIdentifier("page-test-secret", "raw-page-token") {
					t.Fatalf("audit record does not contain the response and token identifier safely: %+v", record)
				}
				wantPath := strings.SplitN(test.path, "?", 2)[0]
				if record.Method != test.method || record.Path != wantPath {
					t.Fatalf("audit record path/method = %s %s", record.Method, record.Path)
				}
				headers, ok := record.Headers.(string)
				if !ok || strings.Contains(headers, "raw-page-token") || !strings.Contains(headers, "[REDACTED]") {
					t.Fatalf("audit headers are not safely redacted: %#v", record.Headers)
				}
				if test.wantGzip {
					auditedBody, ok := record.ResponseBody.(string)
					if !ok || auditedBody != test.responseBody {
						t.Fatalf("audit response body=%#v, want uncompressed %s", record.ResponseBody, test.responseBody)
					}
				}
			}
			if test.name == "oversized body is rejected before negotiation and session inspection" && (len(authorizer.calls) != 0 || len(recorder.events) != 0 || len(activity.records) != 0) {
				t.Fatalf("oversized request reached page authorization: %+v", authorizer.calls)
			}
		})
	}
}

func TestPageLifecycleUnsupportedCanonicalMethodsAuthorizeBefore405(t *testing.T) {
	const path = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/00000000-0000-4000-8000-000000000002/"
	for _, test := range []struct {
		name       string
		denied     bool
		wantStatus int
	}{
		{name: "guest is forbidden before unsupported-method dispatch", denied: true, wantStatus: http.StatusForbidden},
		{name: "authorized role reaches method-not-allowed dispatch", wantStatus: http.StatusMethodNotAllowed},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{}
			sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
			activity := &cycleMiddlewareActivity{recorder: recorder}
			authorizer := &pageLifecycleProtocolAuthorizerStub{}
			if test.denied {
				authorizer.status = http.StatusForbidden
			}
			cookie := djangoSessionCookieConfig{Name: "session-id", Configured: true}
			options := apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "page-test-secret", bodySizeLimit: 1 << 20}
			stack := sessionServiceMiddlewareStack(options, sessions, cookie)
			protocol := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, labelSessionProtocolDispatch{pages: authorizer}, loomhttp.RuntimeCORSPolicy{}, stack, cookie, false, options)
			dispatch := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
				response.WriteHeader(http.StatusMethodNotAllowed)
			})
			handler := protocol(dispatch)
			request := httptest.NewRequest(http.MethodTrace, path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if len(authorizer.calls) != 1 || authorizer.calls[0].Method != http.MethodTrace || authorizer.calls[0].Pattern != "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/" || authorizer.calls[0].PathParameters["page_id"] != "00000000-0000-4000-8000-000000000002" {
				t.Fatalf("page permission calls=%+v", authorizer.calls)
			}
			if response.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" {
				t.Fatalf("Allow=%q", response.Header().Get("Allow"))
			}
			if !test.denied && response.Code != http.StatusMethodNotAllowed {
				t.Fatal("authorized unsupported method did not reach dispatch")
			}
			if len(activity.records) != 0 {
				t.Fatalf("session request emitted API audit: %+v", activity.records)
			}
			if !json.Valid(response.Body.Bytes()) && response.Code == http.StatusForbidden {
				t.Fatalf("permission response is not JSON: %q", response.Body.String())
			}
		})
	}
}

func TestPageLifecycleProtocolDispatchOnlyClaimsDeclaredPageRoutes(t *testing.T) {
	for _, pattern := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/pages/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/",
	} {
		if !isPageLifecycleSessionProtocol(pattern) {
			t.Errorf("page lifecycle route %q was not selected", pattern)
		}
	}
	for _, pattern := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/",
	} {
		if isPageLifecycleSessionProtocol(pattern) {
			t.Errorf("non-page route %q was selected by page lifecycle dispatch", pattern)
		}
	}
}
