package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	system "github.com/CaliLuke/go-plane/gen/system"
	notificationsapi "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type workItemProtocolAuthorizerStub struct {
	resource systemapi.WorkItemProtocolResource
	metadata systemapi.WorkItemProtocolMetadata
	err      error
	calls    int
}

type protocolNotificationsStub struct{}

func (*protocolNotificationsStub) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (*protocolNotificationsStub) List(context.Context, *notificationsapi.ListPayload) (*notificationsapi.NotificationArrayOrPaginatedNotifications, error) {
	return nil, nil
}

func (*protocolNotificationsStub) Counts(context.Context, *notificationsapi.CountsPayload) (*notificationsapi.UnreadCounts, error) {
	return nil, nil
}

func (*protocolNotificationsStub) Get(context.Context, *notificationsapi.GetPayload) (*notificationsapi.Notification, error) {
	return nil, nil
}

func inertWorkItemEndpoints() *workitemapi.Endpoints {
	endpoint := loom.Endpoint(func(context.Context, any) (any, error) { return nil, nil })
	endpoints := &workitemapi.Endpoints{}
	value := reflect.ValueOf(endpoints).Elem()
	for index := 0; index < value.NumField(); index++ {
		value.Field(index).Set(reflect.ValueOf(endpoint))
	}
	return endpoints
}

func (stub *workItemProtocolAuthorizerStub) AuthorizeWorkItemProtocol(_ context.Context, resource systemapi.WorkItemProtocolResource) (systemapi.WorkItemProtocolMetadata, error) {
	stub.calls++
	stub.resource = resource
	return stub.metadata, stub.err
}

func TestWorkItemProtocolMethodSelectionPreservesRetiredGet(t *testing.T) {
	const path = "/api/v1/workspaces/acme/projects/project/cycles/cycle/cycle-issues/"
	old, matched := resolveReadServiceProtocol(path)
	if !matched {
		t.Fatal("source API cycle-issue GET route did not resolve")
	}
	gotGet, matched := resolveReadServiceProtocolForMethod(path, http.MethodGet)
	if !matched || gotGet.allow != old.allow || gotGet.name != old.name || gotGet.derivedHead != old.derivedHead || gotGet.authenticateMethod != old.authenticateMethod {
		t.Fatalf("GET protocol = %#v, want existing retired contract %#v", gotGet, old)
	}
	gotPost, matched := resolveReadServiceProtocolForMethod(path, http.MethodPost)
	if !matched || gotPost.allow != "POST" || gotPost.name != "Cycle Issue List Create Api Endpoint" || !gotPost.workItem || gotPost.authenticateMethod {
		t.Fatalf("POST protocol = %#v, want work-item source contract", gotPost)
	}
	gotOptions, matched := resolveReadServiceProtocolForMethod(path, http.MethodOptions)
	// This API rejects OPTIONS: its existing resource-wide Allow must remain,
	// just as it does for other unsupported methods. App metadata is separate.
	if !matched || gotOptions.allow != old.allow || gotOptions.name != old.name || !gotOptions.workItem {
		t.Fatalf("OPTIONS protocol = %#v, want existing rejected-method contract %#v", gotOptions, old)
	}
}

func TestWorkItemProtocolMiddlewarePassesRetiredGetForEveryCollision(t *testing.T) {
	for _, coreProtocol := range workItemProtocols {
		old := mustReadServiceProtocol(coreProtocol.pattern)
		if !readServiceMethodAllowed(old.allow, http.MethodGet) || workItemMethodOwned(coreProtocol.pattern, http.MethodGet) {
			continue
		}
		pattern := coreProtocol.pattern
		segments := strings.Split(pattern, "/")
		for index, segment := range segments {
			if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
				segments[index] = "value"
			}
		}
		path := strings.Join(segments, "/")
		t.Run(pattern, func(t *testing.T) {
			authorizer := &workItemProtocolAuthorizerStub{}
			middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
			handler := middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusNoContent)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNoContent || authorizer.calls != 0 {
				t.Fatalf("retired GET status=%d auth calls=%d", response.Code, authorizer.calls)
			}
		})
	}
}

func TestWorkItemProtocolMiddlewareAuthorizesUnsupportedCoreMethods(t *testing.T) {
	for _, method := range []string{http.MethodHead, http.MethodOptions, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			authorizer := &workItemProtocolAuthorizerStub{}
			middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
			handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("unsupported method reached generated route")
			}))
			request := httptest.NewRequest(method, "/api/v1/workspaces/acme/projects/project/issues/", nil)
			request.Header.Set("X-Api-Key", "key-123")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusMethodNotAllowed || authorizer.calls != 1 {
				t.Fatalf("status=%d auth calls=%d body=%q", response.Code, authorizer.calls, response.Body.String())
			}
			if authorizer.resource.Method != method || authorizer.resource.Pattern != "/api/v1/workspaces/{slug}/projects/{project_id}/issues/" || authorizer.resource.APIKey != "key-123" {
				t.Fatalf("protocol resource = %#v", authorizer.resource)
			}
			if authorizer.resource.Params["slug"] != "acme" || authorizer.resource.Params["project_id"] != "project" {
				t.Fatalf("path params = %#v", authorizer.resource.Params)
			}
			if method == http.MethodHead {
				if response.Body.Len() != 0 {
					t.Fatalf("HEAD body=%q, want empty", response.Body.String())
				}
				return
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["detail"] == "" {
				t.Fatalf("405 body=%q decode err=%v", response.Body.String(), err)
			}
		})
	}
}

func TestWorkItemProtocolMiddlewareRejectsEncodedIdentifierSlashesBeforeAuth(t *testing.T) {
	cases := []string{
		"/api/workspaces/acme/work-items/IC%2FOTHER-1/",
		"/api/workspaces/acme/work-items/IC-1%2F2/",
		"/api/v1/workspaces/acme/projects/project/issues/IC%2FOTHER-1/",
		"/api/v1/workspaces/acme/projects/project/issues/IC-1%2F2/",
	}
	for _, path := range cases {
		for _, authenticated := range []bool{false, true} {
			t.Run(path+"/authenticated="+strconv.FormatBool(authenticated), func(t *testing.T) {
				authorizer := &workItemProtocolAuthorizerStub{}
				middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
				handler := workItemIdentifierResolver404Middleware(middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					t.Fatal("encoded slash reached generated route")
				})), nil, loomhttp.RuntimeCORSPolicy{})
				request := httptest.NewRequest(http.MethodGet, path, nil)
				if authenticated {
					request.Header.Set("X-Api-Key", "valid-looking-api-key")
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-looking-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusNotFound || authorizer.calls != 0 || response.Body.String() != `{"error":"Page not found."}` {
					t.Fatalf("status=%d auth calls=%d body=%q", response.Code, authorizer.calls, response.Body.String())
				}
				for name, want := range map[string]string{
					"Content-Type":               "application/json",
					"Cross-Origin-Opener-Policy": "same-origin",
					"Referrer-Policy":            "same-origin",
					"Vary":                       "Cookie, origin",
					"X-Content-Type-Options":     "nosniff",
					"X-Frame-Options":            "DENY",
				} {
					if got := response.Header().Get(name); got != want {
						t.Errorf("header %s = %q, want %q", name, got, want)
					}
				}
			})
		}
	}
}

func buildWorkItemProductionProtocolHandler(
	ctx context.Context,
	workItemAuthorizer *workItemProtocolAuthorizerStub,
	activity *activityHTTPService,
	activityLog *apiV1ActivityLoggerStub,
) http.Handler {
	return newHTTPHandlerWithPendingReplacements(
		ctx,
		system.NewEndpoints(systemapi.NewSystem()),
		false,
		nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			notifications:          notificationsapi.NewEndpoints(&protocolNotificationsStub{}),
			notificationsMetadata:  sessionProtocolAuthorizerStub{allowed: true},
			workItemCore:           inertWorkItemEndpoints(),
			workItemProtocol:       workItemAuthorizer,
			activityReads:          activityapi.NewEndpoints(activity),
			activityReadsMetadata:  activity,
			activityReadsActivity:  activityLog,
			activityReadsSecret:    "test-secret",
			activityReadsBodyLimit: defaultFileSizeLimit,
			readServicesCORS:       []string{"https://app.example.test"},
		},
	)
}

func TestWorkItemProductionCompositionOwnsProtocolBeforeOlderStacks(t *testing.T) {
	ctx := newLogContext(context.Background(), io.Discard, false)
	for _, testCase := range []struct {
		name, method, path string
		apiKey, session    string
		status             int
		body               string
		auditRecords       int
	}{
		{
			name:    "app draft OPTIONS remains core 500",
			method:  http.MethodOptions,
			path:    "/api/workspaces/issue-core-contract/draft-to-issue/00000000-0000-4000-8000-000000000001/",
			session: "session-token",
			status:  http.StatusInternalServerError,
			body:    `{"error":"Something went wrong please try again later"}`,
		},
		{
			name:         "API identifier OPTIONS remains core 405",
			method:       http.MethodOptions,
			path:         "/api/v1/workspaces/issue-core-contract/issues/IC-1/",
			apiKey:       "valid-api-token",
			status:       http.StatusMethodNotAllowed,
			body:         `{"detail":"Method \"OPTIONS\" not allowed."}`,
			auditRecords: 1,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			activity := &activityHTTPService{}
			activityLog := &apiV1ActivityLoggerStub{}
			workItemAuth := &workItemProtocolAuthorizerStub{
				err: &workitemapi.WorkItemError{
					Kind: "status_" + strconv.Itoa(testCase.status),
					Body: loom.MustJSONValueFrom(json.RawMessage(testCase.body)),
				},
			}
			handler := buildWorkItemProductionProtocolHandler(ctx, workItemAuth, activity, activityLog)
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			if testCase.apiKey != "" {
				request.Header.Set("X-Api-Key", testCase.apiKey)
			}
			if testCase.session != "" {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: testCase.session})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.status || response.Body.String() != testCase.body {
				t.Fatalf("status=%d body=%q; want %d %q", response.Code, response.Body.String(), testCase.status, testCase.body)
			}
			if workItemAuth.calls != 1 {
				t.Fatalf("core protocol authorizer calls=%d, want 1", workItemAuth.calls)
			}
			if activity.protocolCalls != 0 || activity.authCalls != 0 || len(activityLog.records) != testCase.auditRecords {
				t.Fatalf("older API stack intercepted core protocol: protocol=%d auth=%d audit=%d", activity.protocolCalls, activity.authCalls, len(activityLog.records))
			}
		})
	}
}

func TestWorkItemProductionCompositionRejectsEncodedSlashBeforeAPIAuthAndAudit(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/acme/work-items/IC%2FOTHER-1/",
		"/api/v1/workspaces/acme/projects/project/issues/IC%2FOTHER-1/",
		"/api/v1/workspaces/acme/projects/project/issues/IC-1%2F2/",
	} {
		for _, authenticated := range []bool{false, true} {
			t.Run(path+"/authenticated="+strconv.FormatBool(authenticated), func(t *testing.T) {
				activity := &activityHTTPService{}
				activityLog := &apiV1ActivityLoggerStub{}
				workItemAuth := &workItemProtocolAuthorizerStub{}
				handler := buildWorkItemProductionProtocolHandler(newLogContext(context.Background(), io.Discard, false), workItemAuth, activity, activityLog)
				request := httptest.NewRequest(http.MethodGet, path, nil)
				if authenticated {
					request.Header.Set("X-Api-Key", "valid-api-token")
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusNotFound || response.Body.String() != `{"error":"Page not found."}` {
					t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
				}
				wantAudits := 0
				if authenticated {
					wantAudits = 1
				}
				if workItemAuth.calls != 0 || activity.protocolCalls != 0 || activity.authCalls != 0 || len(activityLog.records) != wantAudits {
					t.Fatalf("encoded slash crossed resolver boundary: core=%d protocol=%d auth=%d audit=%d", workItemAuth.calls, activity.protocolCalls, activity.authCalls, len(activityLog.records))
				}
				if authenticated {
					wantPath := strings.Replace(path, "%2F", "/", 1)
					if audit := activityLog.records[0]; audit.ResponseCode != http.StatusNotFound || audit.ResponseBody != response.Body.String() || audit.Path != wantPath {
						t.Fatalf("API resolver 404 audit = %+v, want decoded path %q", audit, wantPath)
					}
				}
				for name, want := range map[string]string{
					"Content-Type":               "application/json",
					"Cross-Origin-Opener-Policy": "same-origin",
					"Referrer-Policy":            "same-origin",
					"Vary":                       "Cookie, origin",
					"X-Content-Type-Options":     "nosniff",
					"X-Frame-Options":            "DENY",
				} {
					if got := response.Header().Get(name); got != want {
						t.Errorf("header %s = %q, want %q", name, got, want)
					}
				}
			})
		}
	}
}

func TestWorkItemProtocolMiddlewareReturnsAuthorizationErrorBeforeMethodResponse(t *testing.T) {
	want := map[string]any{"detail": "Given API token is not valid"}
	body := loom.MustJSONValueFrom(want)
	authorizer := &workItemProtocolAuthorizerStub{err: &workitemapi.WorkItemError{Kind: "status_403", Body: body}}
	middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unsupported method reached generated route")
	}))
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/acme/projects/project/issues/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden || response.Header().Get("Content-Type") != "application/json" || !reflect.DeepEqual(json.RawMessage(response.Body.Bytes()), json.RawMessage(body)) {
		t.Fatalf("status=%d content-type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
}

func TestWorkItemProtocolMiddlewareAppOptionsUsesSourceActionsAfterAuthorization(t *testing.T) {
	authorizer := &workItemProtocolAuthorizerStub{}
	middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("OPTIONS reached generated route")
	}))
	request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/acme/projects/project/issues/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-123"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || authorizer.calls != 1 || authorizer.resource.SessionID != "session-123" {
		t.Fatalf("status=%d calls=%d resource=%#v body=%q", response.Code, authorizer.calls, authorizer.resource, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type=%q", response.Header().Get("Content-Type"))
	}
	fixturePath := filepath.Join("..", "..", "internal", "parity", "testdata", "fixtures", "work_item_core", "work-item-core-protocol-app-options-issue-list.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Response struct {
			Body any `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(fixture, &source); err != nil {
		t.Fatal(err)
	}
	var actual any
	if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, source.Response.Body) {
		t.Fatalf("OPTIONS metadata differs from source fixture\nactual: %#v\nsource: %#v", actual, source.Response.Body)
	}
}

func TestWorkItemProtocolOptionsUsesSessionGzipStackLikeSource(t *testing.T) {
	cases := []struct {
		fixture string
		varyAE  bool
	}{
		{fixture: "work-item-core-protocol-app-options-issue-detail.json", varyAE: true},
		{fixture: "work-item-core-protocol-app-options-draft-to-issue.json", varyAE: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.fixture, func(t *testing.T) {
			fixturePath := filepath.Join("..", "..", "internal", "parity", "testdata", "fixtures", "work_item_core", testCase.fixture)
			fixtureBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Request struct {
					Path    string            `json:"path"`
					Cookies map[string]string `json:"cookies"`
				} `json:"request"`
				Response struct {
					Status  int               `json:"status"`
					Body    json.RawMessage   `json:"body"`
					Headers map[string]string `json:"headers"`
				} `json:"response"`
			}
			if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
				t.Fatal(err)
			}

			authorizer := &workItemProtocolAuthorizerStub{}
			stack := sessionServiceMiddlewareStack(apiV1CompatibilityOptions{bodySizeLimit: 1 << 20}, nil, djangoSessionCookieConfig{})
			middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, stack, djangoSessionCookieConfig{})
			handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("OPTIONS reached generated route")
			}))
			request := httptest.NewRequest(http.MethodOptions, fixture.Request.Path, nil)
			for name, value := range fixture.Request.Cookies {
				request.AddCookie(&http.Cookie{Name: name, Value: value})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != fixture.Response.Status {
				t.Fatalf("status=%d, source fixture status=%d body=%q", response.Code, fixture.Response.Status, response.Body.String())
			}
			var actualBody, sourceBody any
			if err := json.Unmarshal(response.Body.Bytes(), &actualBody); err != nil {
				t.Fatalf("decode target response: %v; body=%q", err, response.Body.String())
			}
			if err := json.Unmarshal(fixture.Response.Body, &sourceBody); err != nil {
				t.Fatalf("decode source fixture response: %v", err)
			}
			if !reflect.DeepEqual(actualBody, sourceBody) {
				t.Fatalf("OPTIONS body differs from source fixture")
			}
			targetVariesByAcceptEncoding := false
			for _, value := range strings.Split(response.Header().Get("Vary"), ",") {
				if strings.EqualFold(strings.TrimSpace(value), "Accept-Encoding") {
					targetVariesByAcceptEncoding = true
				}
			}
			if targetVariesByAcceptEncoding != testCase.varyAE {
				t.Fatalf("Vary=%q, source fixture=%q", response.Header().Get("Vary"), fixture.Response.Headers["Vary"])
			}
		})
	}
}

func TestWorkItemResponseContentTypeOmittedOnlyForCoreNoContent(t *testing.T) {
	corePath := "/api/workspaces/acme/draft-issues/draft-id/"
	core := workItemResponseContentTypeMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusNoContent)
	}))
	coreResponse := httptest.NewRecorder()
	core.ServeHTTP(coreResponse, httptest.NewRequest(http.MethodPatch, corePath, nil))
	if coreResponse.Code != http.StatusNoContent || coreResponse.Header().Get("Content-Type") != "" {
		t.Fatalf("core 204 status=%d Content-Type=%q", coreResponse.Code, coreResponse.Header().Get("Content-Type"))
	}

	other204 := workItemResponseContentTypeMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusNoContent)
	}))
	otherResponse := httptest.NewRecorder()
	other204.ServeHTTP(otherResponse, httptest.NewRequest(http.MethodPatch, "/api/workspaces/acme/unrelated/", nil))
	if otherResponse.Code != http.StatusNoContent || otherResponse.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unrelated 204 status=%d Content-Type=%q", otherResponse.Code, otherResponse.Header().Get("Content-Type"))
	}
}

func TestWorkItemProtocolMissingIssueOmitsPutActionWithoutChangingSharedMetadata(t *testing.T) {
	authorizer := &workItemProtocolAuthorizerStub{metadata: systemapi.WorkItemProtocolMetadata{OmitPutAction: true}}
	middleware := workItemProtocolMiddleware(authorizer, loomhttp.RuntimeCORSPolicy{}, func(next http.Handler) http.Handler { return next }, djangoSessionCookieConfig{})
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("OPTIONS reached generated route") }))
	for _, missing := range []bool{true, false, true} {
		authorizer.metadata.OmitPutAction = missing
		request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/acme/projects/project/issues/issue/?filters=discarded&filters=filter-value&project_id=untrusted&issue_id=untrusted&pk=untrusted", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		_, hasActions := body["actions"]
		if response.Code != http.StatusOK || hasActions == missing || body["name"] != "Issue" {
			t.Fatalf("missing=%v status=%d metadata=%s", missing, response.Code, response.Body.String())
		}
		if authorizer.resource.Params["filters"] != "filter-value" || authorizer.resource.Params["project_id"] != "project" || authorizer.resource.Params["pk"] != "issue" || authorizer.resource.Params["issue_id"] != "" {
			t.Fatalf("query filter and protected path not propagated: %#v", authorizer.resource.Params)
		}
	}
}

func TestWorkItemResolver404PreservesCORSBeforeAudit(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/acme/work-items/IC%2FOTHER-1/",
		"/api/v1/workspaces/acme/projects/project/issues/IC%2FOTHER-1/",
	} {
		for _, method := range []string{http.MethodGet, http.MethodOptions} {
			for _, origin := range []string{"https://app.example.test", "https://denied.example.test", "https://["} {
				t.Run(path+method+origin, func(t *testing.T) {
					authorizer := &workItemProtocolAuthorizerStub{}
					activity := &activityHTTPService{}
					logger := &apiV1ActivityLoggerStub{}
					handler := buildWorkItemProductionProtocolHandler(newLogContext(context.Background(), io.Discard, false), authorizer, activity, logger)
					request := httptest.NewRequest(method, path, nil)
					request.Header.Set("Origin", origin)
					request.Header.Set("X-Api-Key", "valid-api-token")
					if method == http.MethodOptions {
						request.Header.Set("Access-Control-Request-Method", http.MethodDelete)
					}
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					wantStatus, wantAudits, wantBody := http.StatusNotFound, 1, `{"error":"Page not found."}`
					if method == http.MethodOptions {
						wantStatus, wantAudits, wantBody = http.StatusOK, 0, ""
					}
					if response.Code != wantStatus || response.Body.String() != wantBody || len(logger.records) != wantAudits {
						t.Fatalf("status=%d body=%q audit=%d; want %d %q %d", response.Code, response.Body.String(), len(logger.records), wantStatus, wantBody, wantAudits)
					}
					if authorizer.calls != 0 || activity.authCalls != 0 || activity.protocolCalls != 0 {
						t.Fatalf("resolver entered endpoint authentication: core=%d activity=%d/%d", authorizer.calls, activity.authCalls, activity.protocolCalls)
					}
					wantOrigin, wantCredentials := "", ""
					if origin == "https://app.example.test" {
						wantOrigin, wantCredentials = origin, "true"
					}
					if response.Header().Get("Access-Control-Allow-Origin") != wantOrigin || response.Header().Get("Access-Control-Allow-Credentials") != wantCredentials {
						t.Fatalf("CORS headers=%v", response.Header())
					}
					if method == http.MethodOptions && (response.Header().Get("Allow") != "" || response.Header().Get("Content-Type") != "text/html; charset=utf-8") {
						t.Fatalf("preflight headers=%v", response.Header())
					}
				})
			}
		}
	}
}
