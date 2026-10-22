package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	server "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/server"
	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type workItemCoreRepositoryStub struct {
	authorizeErr       error
	executeErr         error
	result             workitemstore.Result
	authorizeCalls     int
	putActionCalls     int
	putActionAvailable bool
	putActionErr       error
	executeCalls       int
	request            workitemstore.Request
}

func TestWorkItemSearchRetainsQueryReferencesAndProtectsPath(t *testing.T) {
	ctx := WithReadServiceQueryPresence(context.Background(), url.Values{"issue_id": {"anchor"}, "project_id": {"untrusted"}})
	q := workItemRequest(ctx, "get_api_workspaces_by_slug_projects_by_project_id_search_issues", &api.GetAPIWorkspacesBySlugProjectsByProjectIDSearchIssuesPayload{Slug: "workspace", ProjectID: "path-project"})
	if q.Params["issue_id"] != "anchor" || q.ProjectID != "path-project" {
		t.Fatalf("request=%#v", q)
	}
}

func (stub *workItemCoreRepositoryStub) AuthorizeInitial(_ context.Context, request workitemstore.Request) error {
	stub.authorizeCalls++
	stub.request = request
	return stub.authorizeErr
}

func (stub *workItemCoreRepositoryStub) Execute(_ context.Context, request workitemstore.Request, _ io.Reader, _ string) (workitemstore.Result, error) {
	stub.executeCalls++
	stub.request = request
	return stub.result, stub.executeErr
}

func workItemCoreHTTPHandler(t *testing.T, service api.Service) http.Handler {
	t.Helper()
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{
		Pattern: "http://localhost:3000", Methods: []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		Headers: []string{"Content-Type", "X-Api-Key"}, Credentials: true,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux
}

func newWorkItemCoreTestService(t *testing.T, repository *workItemCoreRepositoryStub, keys *activityAPIKeyStub, throttle *activityThrottleStub) api.Service {
	t.Helper()
	if keys == nil {
		keys = &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
	}
	if throttle == nil {
		throttle = &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	}
	service, err := NewWorkItemCore(repository, &activitySessionStub{}, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func workItemCoreAPICtx(t *testing.T, service api.Service, key string) context.Context {
	t.Helper()
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestNewWorkItemCoreRequiresEachBoundaryDependency(t *testing.T) {
	repository := &workItemCoreRepositoryStub{}
	sessions := &activitySessionStub{}
	keys := &activityAPIKeyStub{}
	throttle := &activityThrottleStub{}
	tests := []struct {
		name       string
		repository workItemRepository
		sessions   activityReadsSessions
		keys       activityReadsAPIKeys
		throttle   activityReadsThrottle
	}{
		{name: "repository", sessions: sessions, keys: keys, throttle: throttle},
		{name: "sessions", repository: repository, keys: keys, throttle: throttle},
		{name: "API keys", repository: repository, sessions: sessions, throttle: throttle},
		{name: "throttle", repository: repository, sessions: sessions, keys: keys},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewWorkItemCore(test.repository, test.sessions, test.keys, test.throttle); err == nil {
				t.Fatal("NewWorkItemCore accepted a missing dependency")
			}
		})
	}
}

func TestWorkItemCoreAuthenticationFailureDoesNotEnterDomain(t *testing.T) {
	repository := &workItemCoreRepositoryStub{}
	keys := &activityAPIKeyStub{err: apikey.ErrInvalid}
	service := newWorkItemCoreTestService(t, repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	handler := workItemCoreHTTPHandler(t, service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", strings.NewReader(`{}`))
	request.Header.Set("X-Api-Key", "invalid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["detail"] != "Given API token is not valid" {
		t.Fatalf("error body=%#v", body)
	}
	if keys.calls != 1 || repository.authorizeCalls != 0 || repository.executeCalls != 0 {
		t.Fatalf("key validations=%d authorize=%d execute=%d", keys.calls, repository.authorizeCalls, repository.executeCalls)
	}
}

func TestWorkItemCorePermissionFailurePrecedesThrottleAndExecution(t *testing.T) {
	repository := &workItemCoreRepositoryStub{authorizeErr: &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"error": "You don't have the required permissions."}}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := newWorkItemCoreTestService(t, repository, nil, throttle)
	ctx := workItemCoreAPICtx(t, service, "valid")
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssues(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesPayload{Slug: "demo", ProjectID: uuid.NewString()})
	if body != nil {
		_ = body.Close()
	}
	var apiError *api.WorkItemError
	if !errors.As(err, &apiError) || apiError.LoomErrorName() != "status_403" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if repository.authorizeCalls != 1 || throttle.calls != 0 || repository.executeCalls != 0 {
		t.Fatalf("authorize=%d throttle=%d execute=%d", repository.authorizeCalls, throttle.calls, repository.executeCalls)
	}
}

func TestWorkItemCoreThrottleRunsBeforeTimezoneValidation(t *testing.T) {
	repository := &workItemCoreRepositoryStub{}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: false}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Not/A_Timezone"}}
	service := newWorkItemCoreTestService(t, repository, keys, throttle)
	ctx := workItemCoreAPICtx(t, service, "valid")
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssues(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesPayload{Slug: "demo", ProjectID: uuid.NewString()})
	if body != nil {
		_ = body.Close()
	}
	var apiError *api.WorkItemError
	if !errors.As(err, &apiError) || apiError.LoomErrorName() != "status_429" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if repository.authorizeCalls != 1 || throttle.calls != 1 || repository.executeCalls != 0 {
		t.Fatalf("authorize=%d throttle=%d execute=%d", repository.authorizeCalls, throttle.calls, repository.executeCalls)
	}
}

func TestWorkItemCoreTimezoneValidationFollowsAuthorizationAndThrottle(t *testing.T) {
	repository := &workItemCoreRepositoryStub{}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Not/A_Timezone"}}
	service := newWorkItemCoreTestService(t, repository, keys, throttle)
	ctx := workItemCoreAPICtx(t, service, "valid")
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssues(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesPayload{Slug: "demo", ProjectID: uuid.NewString()})
	if body != nil {
		_ = body.Close()
	}
	var apiError *api.WorkItemError
	if !errors.As(err, &apiError) || apiError.LoomErrorName() != "status_400" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if repository.authorizeCalls != 1 || throttle.calls != 1 || repository.executeCalls != 0 {
		t.Fatalf("authorize=%d throttle=%d execute=%d", repository.authorizeCalls, throttle.calls, repository.executeCalls)
	}
}

func TestWorkItemCorePathValuesOverrideConflictingQueryValues(t *testing.T) {
	repository := &workItemCoreRepositoryStub{result: workitemstore.Result{Status: http.StatusOK, Body: map[string]any{}}}
	service := newWorkItemCoreTestService(t, repository, nil, nil)
	ctx := workItemCoreAPICtx(t, service, "valid")
	ctx = WithReadServiceQueryPresence(ctx, url.Values{
		"slug": {"query-workspace"}, "identifier": {"OTHER-999"}, "search": {"term"},
	})
	_, body, err := service.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier(ctx, &api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload{
		Slug: "path-workspace", Identifier: "PROJ-17",
	})
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		_ = body.Close()
	}
	request := repository.request
	if request.Slug != "path-workspace" || request.Params["slug"] != "path-workspace" || request.Params["identifier"] != "PROJ-17" || request.Params["search"] != "term" {
		t.Fatalf("path/query merge = %#v", request)
	}
}

func TestWorkItemCoreGeneratedErrorPreservesCompleteJSONBodyAndLargeIntegers(t *testing.T) {
	const expected = `{"detail":{"name":["Already used"]},"errors":[{"code":"invalid","value":9007199254740993}],"metadata":{"keys":["field","other"]}}`
	repository := &workItemCoreRepositoryStub{executeErr: &djangoapi.HTTPError{
		Status: http.StatusBadRequest,
		Body: map[string]any{
			"detail":   map[string]any{"name": []any{"Already used"}},
			"errors":   []any{map[string]any{"code": "invalid", "value": json.Number("9007199254740993")}},
			"metadata": map[string]any{"keys": []any{"field", "other"}},
		},
	}}
	service := newWorkItemCoreTestService(t, repository, nil, nil)
	handler := workItemCoreHTTPHandler(t, service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", nil)
	request.Header.Set("X-Api-Key", "valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	decode := func(raw []byte) any {
		t.Helper()
		var value any
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if got, want := decode(response.Body.Bytes()), decode([]byte(expected)); !reflect.DeepEqual(got, want) {
		t.Fatalf("error body=%#v, want %#v", got, want)
	}
	if !strings.Contains(response.Body.String(), "9007199254740993") {
		t.Fatalf("large integer changed in raw response: %s", response.Body.String())
	}
}

func TestWorkItemProtocolUsesResourceAuthorizationBeforeThrottle(t *testing.T) {
	repository := &workItemCoreRepositoryStub{authorizeErr: &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "denied"}}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := newWorkItemCoreTestService(t, repository, nil, throttle).(WorkItemProtocolAuthorizer)
	_, err := service.AuthorizeWorkItemProtocol(context.Background(), WorkItemProtocolResource{Method: "OPTIONS", Pattern: "/api/v1/workspaces/{slug}/issues/{identifier}/", APIKey: "valid", Params: map[string]string{"slug": "demo", "identifier": "PROJ-1"}})
	var apiError *api.WorkItemError
	if !errors.As(err, &apiError) || apiError.Kind != "status_403" {
		t.Fatalf("error=%#v", err)
	}
	if repository.authorizeCalls != 1 || repository.executeCalls != 0 || throttle.calls != 0 || repository.request.Params["identifier"] != "PROJ-1" || repository.request.Method != "OPTIONS" {
		t.Fatalf("authorization=%#v throttle=%d", repository, throttle.calls)
	}
}

func TestWorkItemIntakeResponseConvertsSnoozeAndPreservesJSONData(t *testing.T) {
	stamp := "2026-01-10T08:00:00Z"
	repository := &workItemCoreRepositoryStub{result: workitemstore.Result{Status: 200, Body: map[string]any{
		"results": []map[string]any{{"snoozed_till": stamp, "source_data": map[string]any{"snoozed_till": stamp}}},
	}}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Asia/Kolkata"}}
	service := newWorkItemCoreTestService(t, repository, keys, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/intake-issues/", nil)
	request.Header.Set("X-Api-Key", "valid")
	response := httptest.NewRecorder()
	workItemCoreHTTPHandler(t, service).ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var decoded struct {
		Results []struct {
			Snooze string         `json:"snoozed_till"`
			Data   map[string]any `json:"source_data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Results) != 1 || decoded.Results[0].Snooze != "2026-01-10T13:30:00+05:30" || decoded.Results[0].Data["snoozed_till"] != stamp {
		t.Fatalf("response=%s", response.Body.String())
	}
}

func (stub *workItemCoreRepositoryStub) ProtocolPutAction(_ context.Context, request workitemstore.Request) (bool, error) {
	stub.putActionCalls++
	stub.request = request
	return stub.putActionAvailable, stub.putActionErr
}

func TestWorkItemOptionsChecksPutObjectAfterAuthorizationOnlyForDetailMetadata(t *testing.T) {
	for _, test := range []struct {
		name, method, pattern   string
		available               bool
		authorizeErr, lookupErr error
		wantCalls               int
		wantOmit, wantError     bool
	}{
		{name: "missing detail", method: "OPTIONS", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/", wantCalls: 1, wantOmit: true},
		{name: "existing detail", method: "OPTIONS", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/", available: true, wantCalls: 1},
		{name: "denied before lookup", method: "OPTIONS", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/", authorizeErr: &djangoapi.HTTPError{Status: 403}, wantError: true},
		{name: "unexpected lookup failure", method: "OPTIONS", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/", lookupErr: errors.New("database unavailable"), wantCalls: 1, wantError: true},
		{name: "collection metadata", method: "OPTIONS", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/"},
		{name: "unsupported method", method: "TRACE", pattern: "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &workItemCoreRepositoryStub{authorizeErr: test.authorizeErr, putActionAvailable: test.available, putActionErr: test.lookupErr}
			service := newWorkItemCoreTestService(t, repository, nil, nil).(WorkItemProtocolAuthorizer)
			service.(*workItemCoreService).sessions = &activitySessionStub{principal: djangosession.Principal{UserID: uuid.NewString(), UserTimezone: "UTC"}}
			metadata, err := service.AuthorizeWorkItemProtocol(context.Background(), WorkItemProtocolResource{Method: test.method, Pattern: test.pattern, SessionID: "valid", Params: map[string]string{"slug": "workspace", "project_id": "project", "pk": "issue", "filters": "query"}})
			if (err != nil) != test.wantError || metadata.OmitPutAction != test.wantOmit || repository.putActionCalls != test.wantCalls || repository.executeCalls != 0 {
				t.Fatalf("metadata=%#v err=%v lookup calls=%d writes=%d", metadata, err, repository.putActionCalls, repository.executeCalls)
			}
			if test.wantCalls > 0 && (repository.request.IssueID != "issue" || repository.request.Params["filters"] != "query") {
				t.Fatalf("lookup lost request scope: %#v", repository.request)
			}
		})
	}
}
