package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	api "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type activityRepositoryStub struct {
	request        activitystore.Request
	principal      activitystore.Principal
	body           any
	authorizeErr   error
	readErr        error
	authorizeCalls int
	readCalls      int
}

func (stub *activityRepositoryStub) Authorize(_ context.Context, principal activitystore.Principal, request activitystore.Request) error {
	stub.authorizeCalls++
	stub.principal = principal
	stub.request = request
	return stub.authorizeErr
}

func (stub *activityRepositoryStub) Read(_ context.Context, principal activitystore.Principal, request activitystore.Request) (activitystore.Result, error) {
	stub.readCalls++
	stub.principal = principal
	stub.request = request
	return activitystore.Result{Body: stub.body}, stub.readErr
}

type activityAPIKeyStub struct {
	principal apikey.Principal
	err       error
	calls     int
	key       string
}

func (stub *activityAPIKeyStub) Validate(_ context.Context, key string) (apikey.Principal, error) {
	stub.calls++
	stub.key = key
	return stub.principal, stub.err
}

type activitySessionStub struct {
	principal djangosession.Principal
	err       error
	calls     int
	key       string
}

func (stub *activitySessionStub) Validate(_ context.Context, key string) (djangosession.Principal, error) {
	stub.calls++
	stub.key = key
	return stub.principal, stub.err
}

type activityThrottleStub struct {
	rate  apikey.Rate
	err   error
	calls int
}

func (stub *activityThrottleStub) Allow(context.Context, string) (apikey.Rate, error) {
	stub.calls++
	return stub.rate, stub.err
}

func emptyActivityPage() map[string]any {
	return map[string]any{
		"grouped_by": nil, "sub_grouped_by": nil, "total_count": 0,
		"next_cursor": "1000:1:0", "prev_cursor": "1000:-1:1",
		"next_page_results": false, "prev_page_results": false,
		"count": 0, "total_pages": 0, "total_results": 0,
		"extra_stats": nil, "results": []any{},
	}
}

func TestActivityReadsAPIKeyAuthorizationAndQueryForwarding(t *testing.T) {
	userID := uuid.New()
	repository := &activityRepositoryStub{body: emptyActivityPage()}
	serviceValue, err := NewActivityReads(repository, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: userID, Timezone: "Asia/Kolkata"}}, &activitySessionStub{}, &activityThrottleStub{rate: apikey.Rate{Allowed: true, Remaining: "59", Reset: time.Now().Unix() + 60}})
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*activityReadsService)
	effects := &apikey.Effects{}
	ctx := apikey.WithEffects(context.Background(), effects)
	ctx, err = service.APIKeyAuth(ctx, "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{
		"per_page": {"2"}, "order_by": {"-updated_at"}, "fields": {"id,field"},
		"slug": {"other"}, "project_id": {uuid.New().String()}, "issue_id": {uuid.New().String()}, "pk": {uuid.New().String()},
	})
	projectID := uuid.New().String()
	issueID := uuid.New().String()
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivities(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesPayload{
		Slug: "demo", ProjectID: projectID, IssueID: issueID,
	})
	if body != nil {
		testsupport.Close(t, body)
	}
	if err != nil {
		t.Fatal(err)
	}
	if repository.principal.UserID != userID || repository.principal.Timezone != "Asia/Kolkata" {
		t.Fatalf("principal=%#v", repository.principal)
	}
	if repository.request.Operation != "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities" ||
		repository.request.Params["per_page"] != "2" || repository.request.Params["order_by"] != "-updated_at" ||
		repository.request.Params["slug"] != "demo" || repository.request.Params["project_id"] != projectID || repository.request.Params["issue_id"] != issueID {
		t.Fatalf("request=%#v", repository.request)
	}
	if !effects.Authenticated || effects.Rate == nil || effects.Rate.Remaining != "59" {
		t.Fatalf("effects=%#v", effects)
	}
}
func TestActivityReadsRoutesUserAndAPIOperationsToTheirConfiguredRepositories(t *testing.T) {
	userID := uuid.New()
	userRepository := &activityRepositoryStub{body: emptyActivityPage()}
	apiRepository := &activityRepositoryStub{body: emptyActivityPage()}
	serviceValue, err := NewActivityReads(
		userRepository,
		apiRepository,
		&activityAPIKeyStub{principal: apikey.Principal{UserID: userID, Timezone: "UTC"}},
		&activitySessionStub{principal: djangosession.Principal{UserID: userID.String(), UserTimezone: "UTC"}},
		&activityThrottleStub{rate: apikey.Rate{Allowed: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*activityReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, apiBody, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesPayload{
		Slug: "demo", ProjectID: uuid.New().String(), IssueID: uuid.New().String(),
	})
	if apiBody != nil {
		testsupport.Close(t, apiBody)
	}
	if err != nil {
		t.Fatal(err)
	}
	userCtx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	_, userBody, err := service.GetAPIUsersMeActivities(userCtx, &api.GetAPIUsersMeActivitiesPayload{})
	if userBody != nil {
		testsupport.Close(t, userBody)
	}
	if err != nil {
		t.Fatal(err)
	}
	if userRepository.authorizeCalls != 1 || userRepository.readCalls != 1 {
		t.Fatalf("user repository calls = authorize %d, read %d", userRepository.authorizeCalls, userRepository.readCalls)
	}
	if apiRepository.authorizeCalls != 1 || apiRepository.readCalls != 1 {
		t.Fatalf("API repository calls = authorize %d, read %d", apiRepository.authorizeCalls, apiRepository.readCalls)
	}
}

func TestActivityReadsSessionAuthenticationAndProjection(t *testing.T) {
	userID := uuid.New()
	repository := &activityRepositoryStub{body: map[string]any{
		"grouped_by": nil, "sub_grouped_by": nil, "total_count": 1,
		"next_cursor": "1000:1:0", "prev_cursor": "1000:-1:1",
		"next_page_results": false, "prev_page_results": false,
		"count": 1, "total_pages": 1, "total_results": 1,
		"extra_stats": nil,
		"results":     []any{map[string]any{"id": uuid.New().String(), "created_at": "2026-01-10T08:00:00Z", "field": "name", "icon_prop": map[string]any{"created_at": "2026-01-10T08:00:00Z"}, "description_json": map[string]any{"updated_at": "2026-01-10T08:00:00Z"}}},
	}}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{}, &activitySessionStub{principal: djangosession.Principal{UserID: userID.String(), UserTimezone: "Asia/Kolkata"}}, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.GetAPIUsersMeActivities(ctx, &api.GetAPIUsersMeActivitiesPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if repository.principal.UserID != userID || repository.request.Operation != "get_api_users_me_activities" {
		t.Fatalf("principal=%#v request=%#v", repository.principal, repository.request)
	}
	decoded := map[string]any{}
	if decodeErr := json.NewDecoder(body).Decode(&decoded); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	testsupport.Close(t, body)
	results := decoded["results"].([]any)
	activity := results[0].(map[string]any)
	if len(results) != 1 || activity["created_at"] != "2026-01-10T13:30:00+05:30" || activity["icon_prop"].(map[string]any)["created_at"] != "2026-01-10T08:00:00Z" || activity["description_json"].(map[string]any)["updated_at"] != "2026-01-10T08:00:00Z" {
		t.Fatalf("response=%#v", decoded)
	}
}

func TestActivityReadsRejectInvalidSessionTimezones(t *testing.T) {
	for timezone, statusName := range map[string]string{
		"Etc/DoesNotExist": "status_400",
		"../UTC":           "status_500",
	} {
		t.Run(timezone, func(t *testing.T) {
			repository := &activityRepositoryStub{body: emptyActivityPage()}
			serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{}, &activitySessionStub{principal: djangosession.Principal{UserID: uuid.New().String(), UserTimezone: timezone}}, &activityThrottleStub{})
			service := serviceValue.(*activityReadsService)
			ctx, authErr := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
			if authErr != nil {
				t.Fatal(authErr)
			}
			_, body, err := service.GetAPIUsersMeActivities(ctx, &api.GetAPIUsersMeActivitiesPayload{})
			if body != nil {
				testsupport.Close(t, body)
			}
			var apiError *api.ActivityErrorResponseBody
			if !errors.As(err, &apiError) || apiError.Name != statusName || repository.authorizeCalls != 1 || repository.readCalls != 0 {
				t.Fatalf("error=%T %#v authorize=%d read=%d", err, err, repository.authorizeCalls, repository.readCalls)
			}
		})
	}
}

func TestActivityReadsRejectAuthenticationFailures(t *testing.T) {
	repository := &activityRepositoryStub{}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{err: apikey.ErrInvalid}, &activitySessionStub{err: djangosession.ErrSessionNotFound}, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)

	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	var apiError *api.ActivityErrorResponseBody
	if !errors.As(err, &apiError) || apiError.Name != "status_403" {
		t.Fatalf("API-key error=%T %#v", err, err)
	}
	_, body, err := service.GetAPIUsersMeActivities(context.Background(), &api.GetAPIUsersMeActivitiesPayload{})
	if body != nil {
		testsupport.Close(t, body)
	}
	if !errors.As(err, &apiError) || apiError.Name != "status_401" {
		t.Fatalf("session error=%T %#v", err, err)
	}
}
func TestActivityReadsProtocolAuthorizationRequiresMemberRoleBeforeThrottle(t *testing.T) {
	repository := &activityRepositoryStub{authorizeErr: activitystore.ErrForbidden}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, throttle)
	service := serviceValue.(*activityReadsService)

	status, body := service.AuthorizeAPIActivityProtocol(context.Background(), "token", "demo", uuid.New().String(), "POST")
	if status != http.StatusForbidden || body["detail"] != "You do not have permission to perform this action." {
		t.Fatalf("unsafe protocol authorization status=%d body=%#v", status, body)
	}
	if !repository.request.RequireProjectMemberRole {
		t.Fatal("unsafe protocol authorization did not require an admin or member role")
	}
	if throttle.calls != 0 {
		t.Fatalf("unsafe protocol authorization throttled before permission denial: calls=%d", throttle.calls)
	}

	repository = &activityRepositoryStub{}
	throttle = &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	serviceValue, _ = NewActivityReads(repository, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, throttle)
	status, body = serviceValue.(*activityReadsService).AuthorizeAPIActivityProtocol(context.Background(), "token", "demo", uuid.New().String(), "HEAD")
	if status != 0 || body != nil || repository.request.RequireProjectMemberRole {
		t.Fatalf("safe protocol authorization status=%d body=%#v require_member_role=%t", status, body, repository.request.RequireProjectMemberRole)
	}
}

func TestActivityReadsAuthorizeBeforeThrottle(t *testing.T) {
	repository := &activityRepositoryStub{authorizeErr: activitystore.ErrForbidden}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activitySessionStub{}, throttle)
	service := serviceValue.(*activityReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesPayload{
		Slug: "demo", ProjectID: uuid.New().String(), IssueID: uuid.New().String(),
	})
	if body != nil {
		testsupport.Close(t, body)
	}
	var apiError *api.ActivityErrorResponseBody
	if !errors.As(err, &apiError) || apiError.Name != "status_403" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if repository.authorizeCalls != 1 || repository.readCalls != 0 || throttle.calls != 0 {
		t.Fatalf("authorize=%d read=%d throttle=%d", repository.authorizeCalls, repository.readCalls, throttle.calls)
	}
}

func TestActivityReadsSessionAuthUsesRetainedCookieMetadata(t *testing.T) {
	sessions := &activitySessionStub{principal: djangosession.Principal{UserID: uuid.New().String(), UserTimezone: "UTC"}}
	serviceValue, _ := NewActivityReads(&activityRepositoryStub{}, &activityRepositoryStub{}, &activityAPIKeyStub{}, sessions, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var authErr error
	handler := loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_, authErr = service.APIKeyAuth(request.Context(), "", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/users/me/activities/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-value"})
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if authErr != nil || sessions.calls != 1 || sessions.key != "session-value" {
		t.Fatalf("auth error=%v session calls=%d key=%q", authErr, sessions.calls, sessions.key)
	}
}

func TestActivityReadsInspectedSessionCannotUseGeneratedFallbackCookie(t *testing.T) {
	sessions := &activitySessionStub{principal: djangosession.Principal{UserID: uuid.New().String(), UserTimezone: "UTC"}}
	serviceValue, _ := NewActivityReads(&activityRepositoryStub{}, &activityRepositoryStub{}, &activityAPIKeyStub{}, sessions, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)
	ctx := WithReadServiceSessionPrincipal(context.Background(), "", "")
	_, err := service.APIKeyAuth(ctx, "generated-fallback", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	var apiError *api.ActivityErrorResponseBody
	if !errors.As(err, &apiError) || apiError.Name != "status_401" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if sessions.calls != 0 {
		t.Fatalf("session validator calls=%d", sessions.calls)
	}
}

func TestActivityReadsRawJSONPreservesProjectionAndLargeIntegers(t *testing.T) {
	repository := &activityRepositoryStub{body: map[string]any{
		"id": uuid.New().String(), "new_value": json.Number("9007199254740993"), "field": "estimate_point",
	}}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{}, &activitySessionStub{}, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)
	ctx := context.WithValue(context.Background(), activityReadsPrincipalKey{}, activitystore.Principal{UserID: uuid.New(), Authenticated: true, Timezone: "UTC"})
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"fields": {"new_value, field"}})
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPk(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPkPayload{
		Slug: "demo", ProjectID: uuid.New().String(), IssueID: uuid.New().String(), Pk: uuid.New().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(body)
	testsupport.Close(t, body)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"new_value":9007199254740993}` {
		t.Fatalf("raw=%s", raw)
	}
}

func TestActivityReadCompatibilityBoundaries(t *testing.T) {
	full := map[string]any{"id": "activity", "field": "name"}
	projected := activityFields(full, ",").(map[string]any)
	if len(projected) != 2 || projected["field"] != "name" {
		t.Fatalf("comma-only fields projected %#v", projected)
	}

	timestamps := map[string]any{"issue": map[string]any{"completed_at": "2026-01-10T08:00:00Z"}}
	converted := activityTimestamps(timestamps, "Asia/Kolkata").(map[string]any)
	if converted["issue"].(map[string]any)["completed_at"] != "2026-01-10T13:30:00+05:30" {
		t.Fatalf("timestamps=%#v", converted)
	}

	status, body := activityErrorStatusAndBody(mapActivityReadError(errors.New("database unavailable")))
	if status != 500 || body["error"] != "Something went wrong please try again later" || body["detail"] != nil {
		t.Fatalf("status=%d body=%#v", status, body)
	}
}

func TestUserActivityReadsIgnoreFieldsProjection(t *testing.T) {
	repository := &activityRepositoryStub{body: map[string]any{
		"results": []any{map[string]any{"id": "activity", "field": "name"}},
	}}
	serviceValue, _ := NewActivityReads(repository, repository, &activityAPIKeyStub{}, &activitySessionStub{}, &activityThrottleStub{})
	service := serviceValue.(*activityReadsService)
	ctx := WithReadServiceSessionPrincipal(context.Background(), uuid.New().String(), "UTC")
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"fields": {"id"}})
	_, body, err := service.GetAPIUsersMeActivities(ctx, &api.GetAPIUsersMeActivitiesPayload{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(body)
	testsupport.Close(t, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"field":"name"`) {
		t.Fatalf("raw=%s", raw)
	}
}
