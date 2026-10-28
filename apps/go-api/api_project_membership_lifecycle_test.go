package systemapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	client "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/client"
	server "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type projectMembershipLifecycleRepositoryStub struct {
	authorizeErr   error
	executeErr     error
	result         any
	command        projectmembershipreads.LifecycleCommand
	authorizeCalls int
	executeCalls   int
}

func (stub *projectMembershipLifecycleRepositoryStub) AuthorizeLifecycle(_ context.Context, command projectmembershipreads.LifecycleCommand) error {
	stub.authorizeCalls++
	stub.command = command
	return stub.authorizeErr
}

func (stub *projectMembershipLifecycleRepositoryStub) ExecuteLifecycle(_ context.Context, command projectmembershipreads.LifecycleCommand) (any, error) {
	stub.executeCalls++
	stub.command = command
	return stub.result, stub.executeErr
}

func lifecycleService(t *testing.T, repository *projectMembershipLifecycleRepositoryStub, keys *activityAPIKeyStub, sessions *activitySessionStub, throttle *activityThrottleStub) *projectMembershipLifecycleService {
	t.Helper()
	value, err := NewProjectMembershipLifecycle(repository, keys, throttle, sessions)
	if err != nil {
		t.Fatal(err)
	}
	return value.(*projectMembershipLifecycleService)
}

func membershipLifecycleStatus(t *testing.T, err error, status string) {
	t.Helper()
	var response *api.MembershipLifecycleErrorResponseBody
	if !errors.As(err, &response) || response.Name != "status_"+status {
		t.Fatalf("error = %T %#v, want status_%s", err, err, status)
	}
}

func TestProjectMembershipLifecycleAPIKeyAuthenticationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		key  string
		err  error
		want string
	}{
		{name: "missing", err: errors.New("missing"), want: "401"},
		{name: "invalid", key: "bad-token", err: apikey.ErrInvalid, want: "403"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := lifecycleService(t, &projectMembershipLifecycleRepositoryStub{}, &activityAPIKeyStub{err: test.err}, &activitySessionStub{}, &activityThrottleStub{})
			_, err := service.APIKeyAuth(context.Background(), test.key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
			membershipLifecycleStatus(t, err, test.want)
		})
	}
}

func TestProjectMembershipLifecycleAPIKeyAndSessionAuthenticationStayIsolated(t *testing.T) {
	userID := uuid.New()
	repository := &projectMembershipLifecycleRepositoryStub{result: map[string]any{"ok": true}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: userID, Timezone: "UTC"}}
	sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "session-user", UserTimezone: "UTC"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := lifecycleService(t, repository, keys, sessions, throttle)
	effects := &apikey.Effects{}
	apiContext, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "api-secret", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembers(apiContext, &api.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"}, io.NopCloser(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	_ = body.Close()
	if repository.command.UserID != userID.String() || !effects.Authenticated {
		t.Fatalf("API-key command user=%q effects=%+v", repository.command.UserID, effects)
	}
	if throttle.calls != 1 || keys.calls != 1 || sessions.calls != 0 {
		t.Fatalf("API-key calls: keys=%d sessions=%d throttle=%d", keys.calls, sessions.calls, throttle.calls)
	}

	sessionContext, err := service.APIKeyAuth(context.Background(), "session-secret", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err = service.PostAPIWorkspacesBySlugProjectsByProjectIDMembers(sessionContext, &api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"}, io.NopCloser(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	_ = body.Close()
	if repository.command.UserID != "session-user" || repository.command.Operation != "legacy-create-members" {
		t.Fatalf("session command=%+v", repository.command)
	}
	if throttle.calls != 1 || sessions.calls != 1 {
		t.Fatalf("session calls: sessions=%d throttle=%d", sessions.calls, throttle.calls)
	}
}

func TestProjectMembershipLifecycleForbiddenPermissionDoesNotThrottleOrExecute(t *testing.T) {
	repository := &projectMembershipLifecycleRepositoryStub{authorizeErr: &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"detail": "forbidden"}}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, throttle)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"})
	membershipLifecycleStatus(t, err, "403")
	if repository.authorizeCalls != 1 || repository.executeCalls != 0 || throttle.calls != 0 {
		t.Fatalf("authorize=%d execute=%d throttle=%d", repository.authorizeCalls, repository.executeCalls, throttle.calls)
	}
}

func TestProjectMembershipLifecycleThrottleRejectionRecordsEffectsWithoutExecution(t *testing.T) {
	repository := &projectMembershipLifecycleRepositoryStub{}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: false, Wait: 2 * time.Second}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, throttle)
	effects := &apikey.Effects{}
	ctx, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembers(ctx, &api.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"}, io.NopCloser(strings.NewReader(`{}`)))
	membershipLifecycleStatus(t, err, "429")
	if !effects.Authenticated || effects.Rate == nil || effects.Rate.Allowed || effects.Rate.Wait != 2*time.Second {
		t.Fatalf("effects=%+v", effects)
	}
	if repository.authorizeCalls != 1 || repository.executeCalls != 0 || throttle.calls != 1 {
		t.Fatalf("authorize=%d execute=%d throttle=%d", repository.authorizeCalls, repository.executeCalls, throttle.calls)
	}
}

func TestProjectMembershipLifecycleReturnsRawJSONMetadataAndResponse(t *testing.T) {
	repository := &projectMembershipLifecycleRepositoryStub{result: map[string]any{"ok": true, "count": 2}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	metadata, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ContentType != "application/json" || string(data) != `{"count":2,"ok":true}` {
		t.Fatalf("metadata=%+v response=%s", metadata, data)
	}
	if repository.command.Operation != "v1-list-members" || repository.command.Method != http.MethodGet || repository.command.Slug != "alpha" || repository.command.ProjectID != "project-id" {
		t.Fatalf("command=%+v", repository.command)
	}
}

func TestProjectMembershipLifecycleLegacySessionDoesNotUseAPIKeyThrottleEffects(t *testing.T) {
	repository := &projectMembershipLifecycleRepositoryStub{result: map[string]any{"ok": true}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{}, &activitySessionStub{principal: djangosession.Principal{UserID: "session-user", UserTimezone: "UTC"}}, throttle)
	effects := &apikey.Effects{}
	ctx, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "session-secret", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.PostAPIWorkspacesBySlugProjectsByProjectIDMembers(ctx, &api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: "project-id"}, io.NopCloser(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	_ = body.Close()
	if throttle.calls != 0 || effects.Authenticated || effects.Rate != nil {
		t.Fatalf("throttle=%d effects=%+v", throttle.calls, effects)
	}
}

type projectMembershipLifecycleCookieDoer struct {
	doer loomhttp.Doer
}

func (doer projectMembershipLifecycleCookieDoer) Do(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Cookie", "session-id=session-token")
	if request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return doer.doer.Do(request)
}

type projectMembershipLifecycleStatusDoer struct {
	doer      loomhttp.Doer
	status    int
	path      string
	query     string
	errorName string
	body      string
	addQuery  string
}

func (doer *projectMembershipLifecycleStatusDoer) Do(request *http.Request) (*http.Response, error) {
	if doer.addQuery != "" {
		request.URL.RawQuery = doer.addQuery
	}
	doer.path = request.URL.Path
	doer.query = request.URL.RawQuery
	response, err := doer.doer.Do(request)
	if response != nil {
		doer.status = response.StatusCode
		doer.errorName = response.Header.Get("loom-error")
		if response.Body != nil {
			body, readAllErr := io.ReadAll(response.Body)
			if readAllErr != nil {
				return response, readAllErr
			}
			_ = response.Body.Close()
			doer.body = string(body)
			response.Body = io.NopCloser(strings.NewReader(doer.body))
		}
	}
	return response, err
}

func projectMembershipLifecycleHTTPClient(t *testing.T, service api.Service, doer loomhttp.Doer) (*client.Client, *httptest.Server) {
	t.Helper()
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "https://app.example.test", Methods: []string{"GET", "POST", "DELETE", "PATCH"}, Headers: []string{"Content-Type", "X-Api-Key"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie", "Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	serverURL, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	if doer == nil {
		doer = httpServer.Client()
	}
	return client.NewClient(serverURL.Scheme, serverURL.Host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, true), httpServer
}

func TestProjectMembershipLifecycleGeneratedClientAndServerPreserveRawJSON(t *testing.T) {
	projectID := uuid.NewString()
	repository := &projectMembershipLifecycleRepositoryStub{result: map[string]any{"member": "user-1", "role": 15}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
	service := lifecycleService(t, repository, keys, &activitySessionStub{}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	pathDoer := &projectMembershipLifecycleStatusDoer{addQuery: "slug=query-spoof&project_id=" + uuid.NewString()}
	httpClient, httpServer := projectMembershipLifecycleHTTPClient(t, service, pathDoer)
	pathDoer.doer = httpServer.Client()
	credential := "api-token"
	responseValue, err := httpClient.API.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers()(context.Background(), &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{
		Slug: "alpha", ProjectID: projectID, APIKeyAuthCredential: &credential,
	})
	if err != nil {
		t.Fatalf("generated client call: %T %#v; keys=%d authorized=%d executed=%d", err, err, keys.calls, repository.authorizeCalls, repository.executeCalls)
	}
	response := responseValue.(*api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersResponseData)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Result.ContentType != "application/json" || string(data) != `{"member":"user-1","role":15}` {
		t.Fatalf("metadata=%+v response=%s", response.Result, data)
	}
	if repository.command.Operation != "v1-list-members" || repository.command.Method != http.MethodGet || repository.command.Slug != "alpha" || repository.command.ProjectID != projectID {
		t.Fatalf("command=%+v", repository.command)
	}
	wantPath := "/api/v1/workspaces/alpha/projects/" + projectID + "/members/"
	if pathDoer.path != wantPath || pathDoer.query == "" {
		t.Fatalf("generated request path=%q query=%q, want path %q and injected query", pathDoer.path, pathDoer.query, wantPath)
	}
}

type projectMembershipLifecycleMarshalFailure struct{}

func (projectMembershipLifecycleMarshalFailure) MarshalJSON() ([]byte, error) {
	return nil, errors.New("cannot serialize test value")
}

func TestProjectMembershipLifecycleGeneratedPathMapsRawSerializerFailure(t *testing.T) {
	projectID := uuid.NewString()
	repository := &projectMembershipLifecycleRepositoryStub{result: projectMembershipLifecycleMarshalFailure{}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	statusDoer := &projectMembershipLifecycleStatusDoer{}
	httpClient, httpServer := projectMembershipLifecycleHTTPClient(t, service, statusDoer)
	statusDoer.doer = httpServer.Client()
	credential := "api-token"
	_, err := httpClient.API.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers()(context.Background(), &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{
		Slug: "alpha", ProjectID: projectID, APIKeyAuthCredential: &credential,
	})
	var response *api.MembershipLifecycleErrorResponseBody
	if !errors.As(err, &response) || statusDoer.status != http.StatusInternalServerError {
		t.Fatalf("error=%T %#v HTTP status=%d", err, err, statusDoer.status)
	}
	if repository.authorizeCalls != 1 || repository.executeCalls != 1 {
		t.Fatalf("authorize=%d execute=%d", repository.authorizeCalls, repository.executeCalls)
	}
}

func TestProjectMembershipLifecycleGeneratedPathPreservesFieldValidationErrors(t *testing.T) {
	projectID := uuid.NewString()
	repository := &projectMembershipLifecycleRepositoryStub{executeErr: &djangoapi.HTTPError{
		Status: http.StatusBadRequest,
		Body: map[string]any{
			"member": []string{"This field is required."},
			"role":   []string{"A valid integer is required."},
		},
	}}
	service := lifecycleService(t, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	statusDoer := &projectMembershipLifecycleStatusDoer{}
	httpClient, httpServer := projectMembershipLifecycleHTTPClient(t, service, statusDoer)
	statusDoer.doer = httpServer.Client()
	credential := "api-token"
	_, err := httpClient.API.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembers()(context.Background(), &api.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembersRequestData{
		Payload: &api.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: projectID, APIKeyAuthCredential: &credential},
		Body:    io.NopCloser(strings.NewReader(`{}`)),
	})
	var response *api.MembershipLifecycleErrorResponseBody
	if !errors.As(err, &response) {
		t.Fatalf("error=%T %#v, want a typed validation response", err, err)
	}
	if statusDoer.status != http.StatusBadRequest || statusDoer.errorName != "" || response.Name != "status_400" {
		t.Fatalf("error=%T %#v HTTP status=%d loom-error=%q response-name=%q", err, err, statusDoer.status, statusDoer.errorName, response.Name)
	}
	if strings.TrimSuffix(statusDoer.body, "\n") != `{"member":["This field is required."],"role":["A valid integer is required."]}` {
		t.Fatalf("wire error body changed: %q", statusDoer.body)
	}
	if string(response.Member) != `["This field is required."]` || string(response.Role) != `["A valid integer is required."]` {
		t.Fatalf("typed error member=%s role=%s", response.Member, response.Role)
	}
	wantPath := "/api/v1/workspaces/alpha/projects/" + projectID + "/members/"
	if statusDoer.path != wantPath || statusDoer.query != "" {
		t.Fatalf("generated request path=%q query=%q, want path %q and no query", statusDoer.path, statusDoer.query, wantPath)
	}
	if repository.command.Operation != "v1-create-members" || repository.command.ProjectID != projectID || repository.command.Slug != "alpha" {
		t.Fatalf("command=%+v", repository.command)
	}
}

func TestProjectMembershipLifecycleGeneratedLegacyRouteAuthenticatesSessionCookie(t *testing.T) {
	projectID := uuid.NewString()
	repository := &projectMembershipLifecycleRepositoryStub{result: map[string]any{"created": true}}
	keys := &activityAPIKeyStub{}
	sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "session-user", UserTimezone: "UTC"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := lifecycleService(t, repository, keys, sessions, throttle)
	httpClient, _ := projectMembershipLifecycleHTTPClient(t, service, projectMembershipLifecycleCookieDoer{doer: http.DefaultClient})
	responseValue, err := httpClient.API.PostAPIWorkspacesBySlugProjectsByProjectIDMembers()(context.Background(), &api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersRequestData{
		Payload: &api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersPayload{Slug: "alpha", ProjectID: projectID},
		Body:    io.NopCloser(strings.NewReader(`{"members":[{"member_id":"member-1","role":15}]}`)),
	})
	if err != nil {
		t.Fatalf("generated session call: %T %#v; sessions=%d keys=%d authorized=%d executed=%d", err, err, sessions.calls, keys.calls, repository.authorizeCalls, repository.executeCalls)
	}
	response := responseValue.(*api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersResponseData)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Result.ContentType != "application/json" || string(data) != `{"created":true}` {
		t.Fatalf("metadata=%+v response=%s", response.Result, data)
	}
	if repository.command.UserID != "session-user" || repository.command.Operation != "legacy-create-members" || repository.command.Method != http.MethodPost {
		t.Fatalf("command=%+v", repository.command)
	}
	if sessions.calls != 1 || keys.calls != 0 || throttle.calls != 0 {
		t.Fatalf("sessions=%d keys=%d throttle=%d", sessions.calls, keys.calls, throttle.calls)
	}
}
