package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activitysvr "github.com/CaliLuke/go-plane/activityreads/gen/http/plane_internal_django_migration_api/server"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type activityHTTPService struct {
	authScheme        string
	authKey           string
	query             string
	alias             string
	authCalls         int
	decodedPerPage    int
	protocolCalls     int
	userProtocolCalls int
	protocolMethod    string
	protocolStatus    int
	protocolResponse  map[string]any
	userCursor        *string
	userOrder         *string
	apiCursor         *string
}

func (service *activityHTTPService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	service.authCalls++
	service.authScheme = scheme.Name
	service.authKey = key
	if scheme.Name == "ApiKeyAuth" && key == "" {
		return ctx, &activityapi.ActivityErrorResponseBody{Name: "status_401"}
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return ctx, nil
}

func activityHTTPRaw(body string) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	return &activityapi.ActivityRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(body)), nil
}

func (service *activityHTTPService) GetAPIUsersMeActivities(ctx context.Context, payload *activityapi.GetAPIUsersMeActivitiesPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.query = systemapi.ReadServiceQueryValues(ctx).Get("per_page")
	service.userCursor = payload.Cursor
	service.userOrder = payload.OrderBy
	return activityHTTPRaw(`{"grouped_by":null,"sub_grouped_by":null,"next_cursor":"2:1:0","prev_cursor":"2:-1:1","results":[],"extra_stats":null}`)
}

func (service *activityHTTPService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities(ctx context.Context, payload *activityapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.alias = "issues"
	service.query = systemapi.ReadServiceQueryValues(ctx).Get("per_page")
	service.apiCursor = payload.Cursor
	if payload.PerPage != nil {
		service.decodedPerPage = *payload.PerPage
	}
	return activityHTTPRaw(`{"grouped_by":null,"sub_grouped_by":null,"next_cursor":"2:1:0","prev_cursor":"2:-1:1","results":[],"extra_stats":null}`)
}

func (service *activityHTTPService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPk(context.Context, *activityapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPkPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.alias = "issue-detail"
	return activityHTTPRaw(`{"project":"00000000-0000-4000-8000-000000000001","workspace":"00000000-0000-4000-8000-000000000002"}`)
}

func (service *activityHTTPService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivities(context.Context, *activityapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.alias = "work-items"
	return activityHTTPRaw(`{"grouped_by":null,"sub_grouped_by":null,"next_cursor":"1000:1:0","prev_cursor":"1000:-1:1","results":[],"extra_stats":null}`)
}

func (service *activityHTTPService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesByPk(context.Context, *activityapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesByPkPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.alias = "work-item-detail"
	return activityHTTPRaw(`{"project":"00000000-0000-4000-8000-000000000001","workspace":"00000000-0000-4000-8000-000000000002"}`)
}

func (*activityHTTPService) GetAPIUsersMeActivitiesOptions(context.Context) (*activityapi.ActivityOptions, error) {
	return &activityapi.ActivityOptions{
		Name: "User Activity Endpoint", Description: "", Renders: []string{"application/json"},
		Parses: []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, nil
}
func (service *activityHTTPService) AuthorizeAPIActivityProtocol(ctx context.Context, key, _, _, method string) (int, map[string]any) {
	service.protocolCalls++
	service.authKey = key
	service.protocolMethod = method
	if service.protocolStatus != 0 {
		return service.protocolStatus, service.protocolResponse
	}
	if key == "" {
		return http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return 0, nil
}

func (service *activityHTTPService) AuthorizeUserActivityProtocol(_ context.Context, session string) (int, map[string]any) {
	service.userProtocolCalls++
	service.authKey = session
	if session == "" {
		return http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	return 0, nil
}

func newActivityHTTPHandler(service *activityHTTPService) http.Handler {
	return newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})
}

func newActivityHTTPHandlerWithOptions(service *activityHTTPService, options apiV1CompatibilityOptions) http.Handler {
	return newActivityHTTPHandlerWithSession(service, options, nil, djangoSessionCookieConfig{})
}

func newActivityHTTPHandlerWithSession(service *activityHTTPService, options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) http.Handler {
	mux := loomhttp.NewMuxer()
	corsPolicy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.com"})
	if err != nil {
		panic(err)
	}
	server := activitysvr.New(activityapi.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, corsPolicy)
	stack := activityReadsMiddlewareStack(options, sessions, cookie)
	server.Use(stack)
	activitysvr.Mount(mux, server)
	mountActivityReadsProtocolRoutes(mux, server, corsPolicy)
	handler := newActivityReadsProtocolMiddleware(service, corsPolicy, stack, options, cookie, options.bodySizeLimit, true)(mux)
	handler = readServicesCompatibilityMiddleware(handler)
	handler = readServicesDjangoPathConverterMiddleware(corsPolicy, &options)(handler)
	handler = activityReadsSessionFinalizationMiddleware(sessions, cookie)(handler)
	return loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(handler)
}

func TestActivityGeneratedHTTPRoutesExerciseMixedAuthentication(t *testing.T) {
	service := &activityHTTPService{}
	handler := newActivityHTTPHandler(service)
	projectID := "00000000-0000-4000-8000-000000000001"
	issueID := "00000000-0000-4000-8000-000000000002"

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/"+projectID+"/issues/"+issueID+"/activities/?per_page=2", nil)
	request.Header.Set("X-Api-Key", "token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.authScheme != "ApiKeyAuth" || service.authKey != "token" || service.query != "2" || service.decodedPerPage != 2 || service.alias != "issues" {
		t.Fatalf("response=%d body=%q auth=%q key=%q query=%q decoded=%d alias=%q", response.Code, response.Body.String(), service.authScheme, service.authKey, service.query, service.decodedPerPage, service.alias)
	}
	if !strings.Contains(response.Body.String(), `"results":[]`) || response.Header().Get("Content-Length") == "" {
		t.Fatalf("body=%q headers=%#v", response.Body.String(), response.Header())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/users/me/activities/?per_page=3", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.authScheme != "PlaneSessionCookie" || service.authKey != "" || service.query != "3" {
		t.Fatalf("response=%d body=%q auth=%q key=%q query=%q", response.Code, response.Body.String(), service.authScheme, service.authKey, service.query)
	}
}
func TestActivityGeneratedUserQueryAndOptionsContracts(t *testing.T) {
	service := &activityHTTPService{}
	handler := newActivityHTTPHandler(service)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities/?cursor=2:1:0&order_by=-updated_at&per_page=2", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.userCursor == nil || *service.userCursor != "2:1:0" || service.userOrder == nil || *service.userOrder != "-updated_at" || service.query != "2" {
		t.Fatalf("response=%d cursor=%v order=%v per_page=%q", response.Code, service.userCursor, service.userOrder, service.query)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodOptions, "/api/users/me/activities/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"User Activity Endpoint"`) || !strings.Contains(response.Body.String(), `"multipart/form-data"`) {
		t.Fatalf("response=%d body=%q headers=%#v", response.Code, response.Body.String(), response.Header())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodOptions, "/api/users/me/activities/", strings.NewReader("ignored"))
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Accept", "text/plain")
	request.Header.Set("X-Api-Key", "token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" || response.Header().Get("Content-Length") != "0" || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Vary") != "origin" || response.Header().Get("Allow") != "" || response.Header().Get("X-Frame-Options") != "" || response.Body.Len() != 0 {
		t.Fatalf("user preflight status=%d headers=%#v body=%q", response.Code, response.Header(), response.Body.String())
	}
}
func TestActivityMethodAuthorizerUsesPrimaryForUnsafeMethods(t *testing.T) {
	readOnly := &activityHTTPService{}
	primary := &activityHTTPService{}
	authorizer := activityReadsMethodAuthorizer{readOnly: readOnly, primary: primary}
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		authorizer.AuthorizeAPIActivityProtocol(context.Background(), "key", "slug", "project", method)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace} {
		authorizer.AuthorizeAPIActivityProtocol(context.Background(), "key", "slug", "project", method)
	}
	authorizer.AuthorizeUserActivityProtocol(context.Background(), "session")
	if readOnly.protocolCalls != 3 || readOnly.userProtocolCalls != 0 || primary.protocolCalls != 5 || primary.userProtocolCalls != 1 {
		t.Fatalf("read-only API=%d user=%d primary API=%d user=%d", readOnly.protocolCalls, readOnly.userProtocolCalls, primary.protocolCalls, primary.userProtocolCalls)
	}
}
func TestActivitySlashlessRoutesPreserveDjangoStatusAndAPIAudit(t *testing.T) {
	logger := &apiV1ActivityLoggerStub{}
	service := &activityHTTPService{}
	handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
	})
	projectID := "00000000-0000-4000-8000-000000000001"
	issueID := "00000000-0000-4000-8000-000000000002"
	routes := []string{
		"/api/users/me/activities",
		"/api/v1/workspaces/demo/projects/" + projectID + "/issues/" + issueID + "/activities",
	}
	for _, route := range routes {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace} {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(method, route+"?cursor=value", nil)
			request.Header.Set("X-Api-Key", "invalid")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "invalid"})
			handler.ServeHTTP(response, request)
			unsafe := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
			expectedStatus, expectedLocation, expectedBody := http.StatusMovedPermanently, route+"/?cursor=value", ""
			expectedVary := "Cookie, origin"
			if unsafe {
				expectedStatus, expectedLocation = http.StatusInternalServerError, ""
				expectedBody = string(djangoDebugAppendSlashBody(request))
			}
			if response.Code != expectedStatus || response.Header().Get("Location") != expectedLocation || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Vary") != expectedVary || response.Body.String() != expectedBody {
				t.Fatalf("%s %s status=%d headers=%#v body=%q", method, route, response.Code, response.Header(), response.Body.String())
			}
		}
	}
	for _, route := range routes {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, route, nil)
		request.Header.Set("Origin", "https://app.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Location") != "" || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Vary") != "origin" || response.Body.Len() != 0 {
			t.Fatalf("preflight %s status=%d headers=%#v body=%q", route, response.Code, response.Header(), response.Body.String())
		}
	}
	malformedRoutes := []string{
		"/api/v1/workspaces/demo/projects/not-a-uuid/issues/" + issueID + "/activities",
		"/api/v1/workspaces/demo/projects/" + projectID + "/issues/" + issueID + "/activities/not-a-uuid",
		"/api/v1/workspaces/demo/projects/not-a-uuid/work-items/" + issueID + "/activities",
		"/api/v1/workspaces/demo/projects/" + projectID + "/work-items/" + issueID + "/activities/not-a-uuid",
	}
	for _, route := range malformedRoutes {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, route, nil)
		request.Header.Set("X-Api-Key", "token")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || response.Header().Get("Location") != "" {
			t.Fatalf("malformed slashless %s status=%d headers=%#v body=%q", route, response.Code, response.Header(), response.Body.String())
		}
	}
	if service.authKey != "" {
		t.Fatalf("slash compatibility authenticated key %q", service.authKey)
	}
	if got, want := len(logger.records), 16; got != want {
		t.Fatalf("slashless API audit records = %d, want %d", got, want)
	}
	for _, record := range logger.records {
		if strings.HasSuffix(record.Path, "/") || record.TokenIdentifier == "" || record.ResponseCode != http.StatusNotFound || record.ResponseBody != string(djangoNotFoundBody) {
			t.Fatalf("slashless API audit record = %#v", record)
		}
	}
}
func TestActivitySlashlessPreservesExpiredSessionCookie(t *testing.T) {
	sessions := &publicProjectMetadataThrottleStub{earlyExpire: true}
	handler := newActivityHTTPHandlerWithSession(
		&activityHTTPService{},
		apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit},
		sessions,
		djangoSessionCookieConfig{Name: "session-id", Configured: true},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "invalid"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	wantCookie := `session-id=""; expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; Path=/; SameSite=Lax`
	if response.Code != http.StatusMovedPermanently || response.Header().Get("Set-Cookie") != wantCookie {
		t.Fatalf("slashless expired session status=%d headers=%#v", response.Code, response.Header())
	}
}
func TestActivitySlashlessUsesOuterSessionResponseSemantics(t *testing.T) {
	cookie := djangoSessionCookieConfig{Name: "session-id", Configured: true}

	t.Run("timezone validation stays view-local", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{err: djangosession.ErrTimezoneNotFound}
		handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}, sessions, cookie)
		request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities", nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMovedPermanently || sessions.calls != 0 || sessions.earlyCalls != 1 || sessions.earlyStatus != http.StatusMovedPermanently {
			t.Fatalf("status=%d inspected=%d finalized=%d final_status=%d headers=%#v", response.Code, sessions.calls, sessions.earlyCalls, sessions.earlyStatus, response.Header())
		}
	})

	t.Run("unsafe server error does not refresh", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{earlyRefresh: &djangosession.SessionRefresh{SessionKey: "refreshed", BrowserClose: true}}
		handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}, sessions, cookie)
		request := httptest.NewRequest(http.MethodPost, "/api/users/me/activities", nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		expectedBody := string(djangoDebugAppendSlashBody(request))
		if response.Code != http.StatusInternalServerError || response.Body.String() != expectedBody || sessions.calls != 0 || sessions.earlyCalls != 1 || sessions.earlyStatus != http.StatusInternalServerError || response.Header().Get("Set-Cookie") != "" {
			t.Fatalf("status=%d body=%q inspected=%d finalized=%d final_status=%d headers=%#v", response.Code, response.Body.String(), sessions.calls, sessions.earlyCalls, sessions.earlyStatus, response.Header())
		}
	})
	t.Run("unsafe server error honors technical response negotiation", func(t *testing.T) {
		for _, accept := range []string{"", "application/json", "text/html;q=0", "text/html;q=0.0001", "text/html;level=1"} {
			t.Run(accept, func(t *testing.T) {
				sessions := &publicProjectMetadataThrottleStub{}
				handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}, sessions, cookie)
				request := httptest.NewRequest(http.MethodPost, "/api/users/me/activities", nil)
				request.Header["Accept"] = []string{accept}
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusInternalServerError || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" || !strings.Contains(response.Body.String(), "APPEND_SLASH=False") {
					t.Fatalf("Accept=%q status=%d content_type=%q body=%q", accept, response.Code, response.Header().Get("Content-Type"), response.Body.String())
				}
			})
		}
		for _, accept := range []string{"*/*", "text/*", "text/html;q=0.1", "text/html;level=1, text/html"} {
			t.Run(accept, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, "/api/users/me/activities", nil)
				request.Header.Set("Accept", accept)
				body, contentType := djangoDebugAppendSlashResponse(request)
				if contentType != "text/html; charset=utf-8" || !strings.HasPrefix(string(body), "<!DOCTYPE html>") {
					t.Fatalf("Accept=%q content_type=%q body=%q", accept, contentType, body)
				}
			})
		}
	})
}
func TestActivityAPIV1AuthenticatesProtocolCompanionsAndMatchesPreflight(t *testing.T) {
	logger := &apiV1ActivityLoggerStub{}
	service := &activityHTTPService{}
	handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: 1,
	})
	projectID := "00000000-0000-4000-8000-000000000001"
	issueID := "00000000-0000-4000-8000-000000000002"
	activityID := "00000000-0000-4000-8000-000000000003"
	paths := []string{
		"/api/v1/workspaces/demo/projects/" + projectID + "/issues/" + issueID + "/activities/",
		"/api/v1/workspaces/demo/projects/" + projectID + "/issues/" + issueID + "/activities/" + activityID + "/",
		"/api/v1/workspaces/demo/projects/" + projectID + "/work-items/" + issueID + "/activities/",
		"/api/v1/workspaces/demo/projects/" + projectID + "/work-items/" + issueID + "/activities/" + activityID + "/",
	}

	userResponse := httptest.NewRecorder()
	userRequest := httptest.NewRequest(http.MethodHead, "/api/users/me/activities/", nil)
	userRequest.Header.Set("Origin", "https://app.example.com")
	userRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(userResponse, userRequest)
	if userResponse.Code != http.StatusOK || userResponse.Body.Len() != 0 || userResponse.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("user HEAD status=%d headers=%#v body=%q", userResponse.Code, userResponse.Header(), userResponse.Body.String())
	}

	for _, method := range []string{http.MethodHead, http.MethodOptions} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, paths[0], nil)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("Allow") != "GET" || strings.Contains(response.Header().Get("Vary"), "Cookie") {
			t.Fatalf("unauthenticated method=%s status=%d allow=%q vary=%q body=%q", method, response.Code, response.Header().Get("Allow"), response.Header().Get("Vary"), response.Body.String())
		}
	}

	for _, path := range paths {
		for _, method := range []string{http.MethodHead, http.MethodOptions} {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(method, path, nil)
			request.Header.Set("Origin", "https://app.example.com")
			request.Header.Set("X-Api-Key", "token")
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" || !strings.Contains(response.Header().Get("Vary"), "Cookie") || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
				t.Fatalf("authenticated path=%s method=%s status=%d allow=%q vary=%q headers=%#v body=%q", path, method, response.Code, response.Header().Get("Allow"), response.Header().Get("Vary"), response.Header(), response.Body.String())
			}
		}
	}

	protocolCalls := service.protocolCalls
	activityRecords := len(logger.records)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, paths[0], strings.NewReader("oversized"))
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Accept", "text/plain")
	request.Header.Set("X-Api-Key", "Bearer token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" || response.Header().Get("Content-Length") != "0" || response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Vary") != "origin" || response.Header().Get("Allow") != "" || response.Header().Get("X-Frame-Options") != "" || service.protocolCalls != protocolCalls || len(logger.records) != activityRecords {
		t.Fatalf("preflight status=%d headers=%#v body=%q protocol_calls=%d activity_records=%d", response.Code, response.Header(), response.Body.String(), service.protocolCalls, len(logger.records))
	}
}

func TestActivityUnsupportedMethodsAuthenticateBeforeMethodNotAllowed(t *testing.T) {
	logger := &apiV1ActivityLoggerStub{}
	service := &activityHTTPService{}
	handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
	})
	apiPath := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, apiPath, nil))
	if response.Code != http.StatusUnauthorized || service.protocolCalls != 1 {
		t.Fatalf("unauthenticated API POST status=%d protocol_calls=%d body=%q", response.Code, service.protocolCalls, response.Body.String())
	}

	activityRecords := len(logger.records)
	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, apiPath, nil)
	request.Header.Set("X-Api-Key", "token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet || service.protocolCalls != 2 || service.protocolMethod != http.MethodPost || len(logger.records) <= activityRecords {
		t.Fatalf("authenticated API POST status=%d allow=%q protocol_calls=%d protocol_method=%q activity_records=%d body=%q", response.Code, response.Header().Get("Allow"), service.protocolCalls, service.protocolMethod, len(logger.records), response.Body.String())
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/users/me/activities/", nil))
	if response.Code != http.StatusUnauthorized || service.userProtocolCalls != 1 {
		t.Fatalf("unauthenticated user POST status=%d protocol_calls=%d body=%q", response.Code, service.userProtocolCalls, response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/users/me/activities/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD, OPTIONS" || service.userProtocolCalls != 2 {
		t.Fatalf("authenticated user POST status=%d allow=%q protocol_calls=%d body=%q", response.Code, response.Header().Get("Allow"), service.userProtocolCalls, response.Body.String())
	}
}
func TestActivityMiddlewareOrderRequestSizeAuthenticationTimezoneAuthorizationThrottlingDispatchAudit(t *testing.T) {
	apiPath := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"

	t.Run("request-size precedes session authentication and audit", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{allowed: true, userID: "00000000-0000-4000-8000-000000000003", userTimezone: "UTC"}
		logger := &apiV1ActivityLoggerStub{}
		service := &activityHTTPService{}
		handler := newActivityHTTPHandlerWithSession(service, apiV1CompatibilityOptions{
			activityLogger: logger, activitySecret: "secret", bodySizeLimit: 1,
		}, sessions, djangoSessionCookieConfig{})
		request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", strings.NewReader("oversized"))
		request.Header.Set("X-Api-Key", "audit-only")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusRequestEntityTooLarge || sessions.calls != 0 || service.alias != "" || len(logger.records) != 0 {
			t.Fatalf("status=%d session_calls=%d dispatch=%q audit=%d", response.Code, sessions.calls, service.alias, len(logger.records))
		}
	})
	t.Run("API audit precedes session refresh failure", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{
			userID:       "00000000-0000-4000-8000-000000000003",
			userTimezone: "UTC",
			rotated:      &djangosession.SessionRefresh{SessionKey: "session"},
			refreshErr:   djangosession.ErrSessionNotFound,
		}
		logger := &apiV1ActivityLoggerStub{}
		logger.beforeLog = func() {
			if sessions.refreshCalls != 0 {
				t.Errorf("session refreshed before API audit")
			}
		}
		handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{
			activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
		}, sessions, djangoSessionCookieConfig{Name: "session-id", Configured: true})
		request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", nil)
		request.Header.Set("X-Api-Key", "audit-only")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || response.Body.String() != string(djangoBadRequestBody) || sessions.refreshCalls != 1 || len(logger.records) != 1 || logger.records[0].ResponseCode != http.StatusOK {
			t.Fatalf("status=%d body=%q refresh=%d audit=%#v", response.Code, response.Body.String(), sessions.refreshCalls, logger.records)
		}
	})
	t.Run("API audit records session inspection failures", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{err: errors.New("inspect session")}
		logger := &apiV1ActivityLoggerStub{}
		handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{
			activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
		}, sessions, djangoSessionCookieConfig{Name: "session-id", Configured: true})
		request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", nil)
		request.Header.Set("X-Api-Key", "audit-only")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || len(logger.records) != 1 || logger.records[0].ResponseCode != http.StatusInternalServerError {
			t.Fatalf("status=%d audit=%#v", response.Code, logger.records)
		}
	})
	t.Run("request-size precedes slashless compatibility", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			sessions := &publicProjectMetadataThrottleStub{allowed: true, userID: "00000000-0000-4000-8000-000000000003", userTimezone: "UTC"}
			service := &activityHTTPService{}
			handler := newActivityHTTPHandlerWithSession(service, apiV1CompatibilityOptions{bodySizeLimit: 1}, sessions, djangoSessionCookieConfig{})
			request := httptest.NewRequest(method, "/api/users/me/activities", strings.NewReader("oversized"))
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) || sessions.calls != 0 || service.alias != "" {
				t.Fatalf("method=%s status=%d body=%q session_calls=%d dispatch=%q", method, response.Code, response.Body.String(), sessions.calls, service.alias)
			}
		}
	})

	t.Run("production unsafe slashless requests redirect", func(t *testing.T) {
		handler := activityReadsSlashRedirectMiddleware(http.NotFoundHandler(), false)
		request := httptest.NewRequest(http.MethodPost, "/api/users/me/activities?value=1", strings.NewReader("body"))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/api/users/me/activities/?value=1" || response.Body.Len() != 0 {
			t.Fatalf("status=%d headers=%#v body=%q", response.Code, response.Header(), response.Body.String())
		}
	})

	t.Run("timezone precedes OPTIONS dispatch", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{err: djangosession.ErrTimezoneNotFound}
		service := &activityHTTPService{}
		handler := newActivityHTTPHandlerWithSession(service, apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}, sessions, djangoSessionCookieConfig{})
		request := httptest.NewRequest(http.MethodOptions, "/api/users/me/activities/", nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || response.Body.String() != "{\"error\":\"The required key does not exist.\"}" || sessions.calls != 1 || service.userProtocolCalls != 0 {
			t.Fatalf("status=%d body=%q session_calls=%d protocol_calls=%d", response.Code, response.Body.String(), sessions.calls, service.userProtocolCalls)
		}
	})

	t.Run("exact API key reaches authentication before dispatch and audit follows", func(t *testing.T) {
		logger := &apiV1ActivityLoggerStub{}
		service := &activityHTTPService{}
		handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
			activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
		})
		request := httptest.NewRequest(http.MethodGet, apiPath, nil)
		request.Header.Set("X-Api-Key", "Bearer invalid")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || service.authCalls != 1 || service.authKey != "Bearer invalid" || service.alias != "issues" || len(logger.records) != 1 || logger.records[0].ResponseCode != http.StatusOK {
			t.Fatalf("status=%d auth=%d key=%q dispatch=%q audit=%#v", response.Code, service.authCalls, service.authKey, service.alias, logger.records)
		}
	})

	t.Run("throttling context reaches dispatch before success audit", func(t *testing.T) {
		logger := &apiV1ActivityLoggerStub{}
		service := &activityHTTPService{}
		handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
			activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
		})
		request := httptest.NewRequest(http.MethodGet, apiPath, nil)
		request.Header.Set("X-Api-Key", "token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || service.authCalls != 1 || service.alias != "issues" || len(logger.records) != 1 || logger.records[0].ResponseCode != http.StatusOK {
			t.Fatalf("status=%d auth=%d dispatch=%q audit=%#v", response.Code, service.authCalls, service.alias, logger.records)
		}
	})
}

func TestActivityAPIV1PreservesRawIntegerSyntaxAndMalformedUUIDContract(t *testing.T) {
	service := &activityHTTPService{}
	handler := newActivityHTTPHandler(service)
	path := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path+"?per_page=%20%31_%30%20", nil)
	request.Header.Set("X-Api-Key", "token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.query != " 1_0 " || service.decodedPerPage != 10 {
		t.Fatalf("status=%d raw=%q decoded=%d body=%q", response.Code, service.query, service.decodedPerPage, response.Body.String())
	}

	for _, rawQuery := range []string{"cursor=%ZZ", "per_page=%ZZ", "per_page=2&per_page=3&per_page=%ZZ"} {
		service.apiCursor = nil
		service.query = ""
		service.decodedPerPage = 0
		response = httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodGet, path, nil)
		request.URL.RawQuery = rawQuery
		request.Header.Set("X-Api-Key", "token")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("query=%q status=%d body=%q", rawQuery, response.Code, response.Body.String())
		}
		if strings.HasPrefix(rawQuery, "cursor=") {
			if service.apiCursor == nil || *service.apiCursor != "%ZZ" {
				t.Fatalf("query=%q cursor=%v", rawQuery, service.apiCursor)
			}
		} else if service.query != "%ZZ" || service.decodedPerPage != 1 {
			t.Fatalf("query=%q raw=%q decoded=%d", rawQuery, service.query, service.decodedPerPage)
		}
	}

	authCalls := service.authCalls
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, strings.Replace(path, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-00000000000Z", 1), nil)
	request.Header.Set("X-Api-Key", "token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != `{"error":"Page not found."}` || response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Vary") != "Cookie, origin" || service.authCalls != authCalls || service.protocolCalls != 0 {
		t.Fatalf("malformed status=%d headers=%#v body=%q auth_calls=%d protocol_calls=%d", response.Code, response.Header(), response.Body.String(), service.authCalls, service.protocolCalls)
	}
}

func TestActivityAPIKeyCompatibilityPreservesExactCredentialAndAuditsUserRoute(t *testing.T) {
	projectPath := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	logger := &apiV1ActivityLoggerStub{}
	service := &activityHTTPService{}
	handler := newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, projectPath+"?per_page=1&per_page=2&fields=a%3Bb", nil)
	request.Header.Set("X-Api-Key", "Bearer token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.authCalls != 1 || service.authKey != "Bearer token" || len(logger.records) != 1 || logger.records[0].QueryParams != "per_page=1&per_page=2&fields=a%3Bb" {
		t.Fatalf("status=%d key=%q auth=%d logs=%d", response.Code, service.authKey, service.authCalls, len(logger.records))
	}

	logger.records = nil
	response = httptest.NewRecorder()
	requestBody := "audit request"
	request = httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", strings.NewReader(requestBody))
	request.Header.Set("X-Api-Key", "audit-only")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(logger.records) != 1 || logger.records[0].Body != requestBody {
		t.Fatalf("status=%d body=%q logs=%#v", response.Code, response.Body.String(), logger.records)
	}

	logger.records = nil
	handler = newActivityHTTPHandlerWithOptions(service, apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: 4,
	})
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", strings.NewReader("oversized"))
	request.Header.Set("X-Api-Key", "audit-only")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) || len(logger.records) != 0 {
		t.Fatalf("oversized status=%d body=%q logs=%#v", response.Code, response.Body.String(), logger.records)
	}
}

func TestActivityCompatibilityLogsUncompressedBodyWithOriginalQuery(t *testing.T) {
	logger := &apiV1ActivityLoggerStub{}
	body := `{"payload":"` + strings.Repeat("a", 300) + `"}`
	inner := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		apikey.EffectsFromContext(request.Context()).Authenticated = true
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(body))
	})
	handler := readServicesCompatibilityMiddleware(activityReadsMiddlewareStack(apiV1CompatibilityOptions{
		activityLogger: logger, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit,
	}, nil, djangoSessionCookieConfig{})(inner))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/?per_page=1&per_page=2&fields=a%3Bb", nil)
	request.Header.Set("X-Api-Key", "token")
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "gzip" || len(logger.records) != 1 {
		t.Fatalf("status=%d headers=%#v records=%d", response.Code, response.Header(), len(logger.records))
	}
	if logger.records[0].ResponseBody != body || logger.records[0].QueryParams != "per_page=1&per_page=2&fields=a%3Bb" {
		t.Fatalf("activity=%#v", logger.records[0])
	}
}

func TestActivityFinalizesSessionsOutsideEarlyResponses(t *testing.T) {
	cookie := djangoSessionCookieConfig{Name: "session-id", Domain: ".example.test", Secure: true, Configured: true}
	validPath := "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/"
	tests := []struct {
		name, method, path, body, accept, apiKey string
		status                                   int
	}{
		{"api success", http.MethodGet, validPath, "", "", "token", http.StatusOK},
		{"malformed uuid", http.MethodGet, strings.Replace(validPath, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-00000000000Z", 1), "", "", "token", http.StatusNotFound},
		{"Django-shaped nil UUID authenticates", http.MethodGet, strings.Replace(validPath, "00000000-0000-4000-8000-000000000001", "00000000-0000-0000-0000-000000000000", 1), "", "", "", http.StatusUnauthorized},
		{"negotiation failure", http.MethodGet, validPath, "", "text/plain", "token", http.StatusNotAcceptable},
		{"exact API key", http.MethodGet, validPath, "", "", "Bearer token", http.StatusOK},
		{"user oversized", http.MethodGet, "/api/users/me/activities/", "oversized", "", "audit-only", http.StatusRequestEntityTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{earlyRefresh: &djangosession.SessionRefresh{SessionKey: "refreshed", BrowserClose: true}}
			handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{bodySizeLimit: 4}, sessions, cookie)
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}
			request.Header.Set("X-Api-Key", test.apiKey)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "current"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || sessions.earlyCalls != 1 || sessions.calls != 0 || !strings.Contains(response.Header().Get("Set-Cookie"), "session-id=refreshed") {
				t.Fatalf("status=%d early=%d inspected=%d headers=%#v body=%q", response.Code, sessions.earlyCalls, sessions.calls, response.Header(), response.Body.String())
			}
		})
	}

	sessions := &publicProjectMetadataThrottleStub{}
	handler := newActivityHTTPHandlerWithSession(&activityHTTPService{}, apiV1CompatibilityOptions{bodySizeLimit: 1}, sessions, cookie)
	request := httptest.NewRequest(http.MethodOptions, validPath, strings.NewReader("oversized"))
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "current"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || sessions.earlyCalls != 0 || sessions.calls != 0 {
		t.Fatalf("preflight status=%d early=%d inspected=%d headers=%#v", response.Code, sessions.earlyCalls, sessions.calls, response.Header())
	}
}
