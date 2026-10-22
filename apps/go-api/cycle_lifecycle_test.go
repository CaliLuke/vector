package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	api "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type cycleLifecycleSessionValidatorStub struct {
	principal djangosession.Principal
	err       error
	key       string
}

func (stub *cycleLifecycleSessionValidatorStub) Validate(_ context.Context, key string) (djangosession.Principal, error) {
	stub.key = key
	return stub.principal, stub.err
}

type cycleLifecycleAPIKeyValidatorStub struct {
	principal apikey.Principal
	err       error
	key       string
}

func (stub *cycleLifecycleAPIKeyValidatorStub) Validate(_ context.Context, key string) (apikey.Principal, error) {
	stub.key = key
	return stub.principal, stub.err
}

func TestCycleLifecycleAuthUsesSchemeSpecificCredentialValidator(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	sessions := &cycleLifecycleSessionValidatorStub{principal: djangosession.Principal{UserID: userID.String(), UserTimezone: "UTC"}}
	keys := &cycleLifecycleAPIKeyValidatorStub{principal: apikey.Principal{UserID: userID, Timezone: "America/Los_Angeles"}}
	service := &cycleLifecycleService{sessions: sessions, keys: keys}

	apiContext, err := service.APIKeyAuth(context.Background(), "api-token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatalf("API key auth returned error: %v", err)
	}
	apiPrincipal, err := cycleLifecycleActor(apiContext)
	if err != nil {
		t.Fatalf("read API principal: %v", err)
	}
	if !apiPrincipal.apiKey || apiPrincipal.userID != userID || apiPrincipal.timezone != "America/Los_Angeles" || apiPrincipal.key != "api-token" {
		t.Fatalf("unexpected API principal: %#v", apiPrincipal)
	}
	if keys.key != "api-token" || sessions.key != "" {
		t.Fatalf("wrong validator used: API key %q, session key %q", keys.key, sessions.key)
	}

	sessionContext, err := service.APIKeyAuth(context.Background(), "session-id", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatalf("session auth returned error: %v", err)
	}
	sessionPrincipal, err := cycleLifecycleActor(sessionContext)
	if err != nil {
		t.Fatalf("read session principal: %v", err)
	}
	if sessionPrincipal.apiKey || sessionPrincipal.userID != userID || sessionPrincipal.timezone != "UTC" {
		t.Fatalf("unexpected session principal: %#v", sessionPrincipal)
	}
	if sessions.key != "session-id" {
		t.Fatalf("session validator received %q", sessions.key)
	}
}

func TestCycleLifecycleAuthPreservesDjangoAPIKeyFailures(t *testing.T) {
	service := &cycleLifecycleService{
		sessions: &cycleLifecycleSessionValidatorStub{},
		keys:     &cycleLifecycleAPIKeyValidatorStub{err: apikey.ErrInvalid},
	}
	_, err := service.APIKeyAuth(context.Background(), "bad-token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	var response *api.CycleLifecycleError
	if !errors.As(err, &response) {
		t.Fatalf("expected generated API error, got %T %v", err, err)
	}
	if response.Kind != "status_403" {
		t.Fatalf("invalid API key returned %s", response.Kind)
	}
}

func TestCyclePatchSeparatesWritablePersistenceFromRawActivityData(t *testing.T) {
	repository := &cycleLifecycleRepositoryStub{result: map[string]any{}}
	service := &cycleLifecycleService{repo: repository}
	ctx := context.WithValue(context.Background(), cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
		userID: uuid.MustParse("00000000-0000-4000-8000-000000000303"),
	})
	body := io.NopCloser(strings.NewReader(`{"created_by":"not-a-uuid","updated_by":17,"deleted_at":"not-a-date","name":"changed"}`))
	if _, err := service.patch(ctx, "workspace", "project", "cycle", body, false); err != nil {
		t.Fatalf("read-only input should be ignored before validation: %v", err)
	}
	if len(repository.requested) != 1 || repository.requested["name"] != "changed" {
		t.Fatalf("persistence input should contain only writable validated fields: %#v", repository.requested)
	}
	for key, want := range map[string]any{"created_by": "not-a-uuid", "updated_by": json.Number("17"), "deleted_at": "not-a-date", "name": "changed"} {
		if got := repository.activity[key]; got != want {
			t.Errorf("activity request %s = %#v, want %#v", key, got, want)
		}
	}
}

func TestCyclePatchFiltersCompletedCycleFieldsBeforeValidation(t *testing.T) {
	end := time.Now().Add(-time.Hour)
	repository := &cycleLifecycleRepositoryStub{
		cycle:  store.CycleLifecycleResource{EndDate: &end},
		result: map[string]any{},
	}
	service := &cycleLifecycleService{repo: repository}
	ctx := context.WithValue(context.Background(), cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
		userID: uuid.MustParse("00000000-0000-4000-8000-000000000303"),
	})
	body := io.NopCloser(strings.NewReader(`{"sort_order":12,"name":""}`))
	if _, err := service.patch(ctx, "workspace", "project", "cycle", body, false); err != nil {
		t.Fatalf("completed-cycle ignored name should be filtered before validation: %v", err)
	}
	if len(repository.requested) != 1 || repository.requested["sort_order"] != float64(12) {
		t.Fatalf("completed cycle persistence input = %#v, want sort_order only", repository.requested)
	}
}

func TestCycleLifecycleLegacyArchiveAndDeleteUseSourceMissingObjectBody(t *testing.T) {
	for _, operation := range []string{"archive", "delete-authorize", "delete"} {
		t.Run(operation, func(t *testing.T) {
			repository := &cycleLifecycleRepositoryStub{}
			switch operation {
			case "archive":
				repository.archiveErr = store.ErrNotFound
			case "delete-authorize":
				repository.deleteAuthorizeErr = store.ErrNotFound
			case "delete":
				repository.deleteErr = store.ErrNotFound
			}
			service := &cycleLifecycleService{repo: repository}
			ctx := context.WithValue(context.Background(), cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
				userID: uuid.MustParse("00000000-0000-4000-8000-000000000303"),
			})
			var err error
			if operation == "archive" {
				_, err = service.archive(ctx, "workspace", "project", "cycle", false, true)
			} else {
				err = service.delete(ctx, "workspace", "project", "cycle", false)
			}
			var response *api.CycleLifecycleError
			if !errors.As(err, &response) || response.Kind != "status_404" {
				t.Fatalf("error = %T %v, want status_404", err, err)
			}
			body, marshalErr := json.Marshal(response.Body)
			if marshalErr != nil || string(body) != `{"error":"The required object does not exist."}` {
				t.Fatalf("error body = %s (%v)", body, marshalErr)
			}
		})
	}
}

type cycleLifecycleRepositoryStub struct {
	input              store.CycleCreate
	requested          map[string]any
	activity           map[string]any
	api                bool
	userID             string
	slug               string
	projectID          string
	permission         error
	cycleError         error
	archiveErr         error
	deleteAuthorizeErr error
	deleteErr          error
	cycleCalls         int
	appProjectCalls    int
	appDeleteCalls     int
	createCalls        int
	result             map[string]any
	cycle              store.CycleLifecycleResource
}

func (stub *cycleLifecycleRepositoryStub) AuthorizeProject(_ context.Context, userID, slug, projectID string) error {
	stub.userID, stub.slug, stub.projectID = userID, slug, projectID
	return stub.permission
}

func (stub *cycleLifecycleRepositoryStub) AuthorizeAppProject(_ context.Context, userID, slug, projectID string) error {
	stub.appProjectCalls++
	stub.userID, stub.slug, stub.projectID = userID, slug, projectID
	return stub.permission
}

func (stub *cycleLifecycleRepositoryStub) AuthorizeAppDelete(_ context.Context, slug, projectID, cycleID, actor string) error {
	stub.appDeleteCalls++
	stub.slug, stub.projectID, stub.userID = slug, projectID, actor
	return stub.deleteAuthorizeErr
}

func (stub *cycleLifecycleRepositoryStub) AuthorizeReadProject(_ context.Context, userID, slug, projectID string) error {
	stub.userID, stub.slug, stub.projectID = userID, slug, projectID
	return stub.permission
}

func (stub *cycleLifecycleRepositoryStub) DateCheck(context.Context, string, string, *string, *string, *string) (bool, error) {
	return true, nil
}

func (stub *cycleLifecycleRepositoryStub) SetArchived(context.Context, string, string, string, string, bool) (time.Time, error) {
	return time.Time{}, stub.archiveErr
}

func (stub *cycleLifecycleRepositoryStub) AuthorizeDelete(context.Context, string, string, string, string, bool) error {
	return stub.deleteAuthorizeErr
}

func (stub *cycleLifecycleRepositoryStub) Delete(context.Context, string, string, string, string, bool) error {
	return stub.deleteErr
}

func (stub *cycleLifecycleRepositoryStub) Cycle(context.Context, string, string, string) (store.CycleLifecycleResource, error) {
	stub.cycleCalls++
	return stub.cycle, stub.cycleError
}

func (stub *cycleLifecycleRepositoryStub) Patch(_ context.Context, _, _, _, _ string, api bool, requested, activity map[string]any) (map[string]any, error) {
	stub.api, stub.requested, stub.activity = api, requested, activity
	return stub.result, nil
}

func (stub *cycleLifecycleRepositoryStub) PutNoop(context.Context, string, string, string, string) (map[string]any, error) {
	return stub.result, nil
}

func (stub *cycleLifecycleRepositoryStub) Create(_ context.Context, actor, slug, projectID string, input store.CycleCreate, api bool, requested map[string]any) (map[string]any, error) {
	stub.createCalls++
	stub.input, stub.api, stub.requested = input, api, requested
	return stub.result, nil
}

func TestCycleSessionDeletePermissionUsesCreatorDecoratorOnlyForDetail(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	for _, test := range []struct {
		name        string
		pattern     string
		wantCreator bool
	}{
		{name: "cycle detail creator grant", pattern: "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", wantCreator: true},
		{name: "archive collection write role", pattern: "/api/workspaces/{slug}/projects/{project_id}/archived-cycles/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &cycleLifecycleRepositoryStub{}
			service := &cycleLifecycleService{
				repo: repository,
				sessions: &cycleLifecycleSessionValidatorStub{principal: djangosession.Principal{
					UserID: userID.String(), UserTimezone: "UTC",
				}},
			}
			resource := SessionProtocolResource{
				Method: http.MethodDelete, Pattern: test.pattern, Slug: "workspace", ProjectID: "project",
				ID: "cycle", PathParameters: map[string]string{"pk": "cycle"},
			}
			status, body := service.AuthorizeSessionResourceProtocol(context.Background(), "session", resource)
			if status != 0 {
				t.Fatalf("authorization status=%d body=%#v", status, body)
			}
			if test.wantCreator && (repository.appDeleteCalls != 1 || repository.appProjectCalls != 0) {
				t.Fatalf("detail delete used appProject=%d appDelete=%d, want creator decorator", repository.appProjectCalls, repository.appDeleteCalls)
			}
			if !test.wantCreator && (repository.appDeleteCalls != 0 || repository.appProjectCalls != 1) {
				t.Fatalf("archive delete used appProject=%d appDelete=%d, want normal write role", repository.appProjectCalls, repository.appDeleteCalls)
			}
		})
	}
}

func TestCycleCreateChecksRawPairedDatesBeforeSerializerValidation(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	for _, apiV1 := range []bool{false, true} {
		name := "app"
		if apiV1 {
			name = "v1"
		}
		t.Run(name, func(t *testing.T) {
			repository := &cycleLifecycleRepositoryStub{}
			service := &cycleLifecycleService{
				repo:     repository,
				throttle: cycleLifecycleThrottleStub{rate: apikey.Rate{Allowed: true}},
			}
			ctx := context.WithValue(context.Background(), cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
				userID: userID, timezone: "UTC", key: "api-token", apiKey: apiV1,
			})
			_, _, err := service.create(ctx, "workspace", "project", io.NopCloser(strings.NewReader(`{"start_date":"2026-10-20"}`)), apiV1)
			var lifecycleError *api.CycleLifecycleError
			if !errors.As(err, &lifecycleError) || lifecycleError.Kind != "status_400" {
				t.Fatalf("create error=%v, want source 400 pair error", err)
			}
			encoded, marshalErr := json.Marshal(lifecycleError.Body)
			if marshalErr != nil || string(encoded) != `{"error":"Both start date and end date are either required or are to be null"}` {
				t.Fatalf("pair error body=%s marshal_err=%v", encoded, marshalErr)
			}
			if repository.createCalls != 0 {
				t.Fatalf("repository create called %d times after raw pair validation failed", repository.createCalls)
			}
		})
	}
}

func TestCycleOptionsResourceLookupUsesReadScopeBeforeCycleLookup(t *testing.T) {
	resource := SessionProtocolResource{
		Method: http.MethodOptions, Pattern: "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/",
		Slug: "acme", ProjectID: "project-1", ID: "cycle-1",
	}
	for _, test := range []struct {
		name        string
		permission  error
		cycleError  error
		wantExists  bool
		wantLookups int
	}{
		{name: "existing member cycle", wantExists: true, wantLookups: 1},
		{name: "missing cycle", cycleError: store.ErrNotFound, wantExists: false, wantLookups: 1},
		{name: "outside active member queryset", permission: store.ErrNotFound, wantExists: false, wantLookups: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &cycleLifecycleRepositoryStub{permission: test.permission, cycleError: test.cycleError}
			service := &cycleLifecycleService{repo: repository}
			ctx := WithReadServiceSessionPrincipal(context.Background(), "user-1", "UTC")
			exists, err := service.SessionProtocolResourceExists(ctx, resource)
			if err != nil || exists != test.wantExists {
				t.Fatalf("resource exists = (%t, %v), want (%t, nil)", exists, err, test.wantExists)
			}
			if repository.cycleCalls != test.wantLookups {
				t.Fatalf("cycle lookups = %d, want %d", repository.cycleCalls, test.wantLookups)
			}
			if repository.userID != "user-1" || repository.slug != "acme" || repository.projectID != "project-1" {
				t.Fatalf("read scope = user:%q slug:%q project:%q", repository.userID, repository.slug, repository.projectID)
			}
		})
	}
}

type cycleLifecycleThrottleStub struct {
	rate apikey.Rate
}

func (stub cycleLifecycleThrottleStub) Allow(context.Context, string) (apikey.Rate, error) {
	return stub.rate, nil
}

func TestCycleLifecycleLegacyCreateUsesOnlyWritableFields(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	repository := &cycleLifecycleRepositoryStub{result: map[string]any{
		"id": "00000000-0000-4000-8000-000000000202", "name": "Draft cycle", "created_by": userID.String(),
		"description": "", "end_date": nil, "start_date": nil, "owned_by_id": userID.String(),
		"external_id": nil, "external_source": nil, "logo_props": map[string]any{}, "progress_snapshot": map[string]any{},
		"project_id": "00000000-0000-4000-8000-000000000303", "workspace_id": "00000000-0000-4000-8000-000000000404",
		"sort_order": float64(65535), "version": 1, "status": "DRAFT", "total_issues": 0, "completed_issues": 0,
		"is_favorite": false, "assignee_ids": []any{},
	}}
	service := &cycleLifecycleService{
		repo: repository, sessions: &cycleLifecycleSessionValidatorStub{principal: djangosession.Principal{UserID: userID.String(), UserTimezone: "UTC"}},
	}
	ctx, err := service.APIKeyAuth(context.Background(), "session-token", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.PostAPIWorkspacesBySlugProjectsByProjectIDCycles(ctx, &api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesPayload{
		Slug: "cycle-lifecycle", ProjectID: "00000000-0000-4000-8000-000000000303",
	}, io.NopCloser(strings.NewReader(`{"name":"Draft cycle","deleted_at":"2099-01-01T00:00:00Z","created_by":"00000000-0000-4000-8000-000000000999"}`)))
	if err != nil {
		t.Fatalf("legacy create: %v", err)
	}
	if repository.userID != userID.String() || repository.slug != "cycle-lifecycle" || repository.projectID != "00000000-0000-4000-8000-000000000303" {
		t.Fatalf("wrong project authorization scope: %#v", repository)
	}
	if repository.input.Name != "Draft cycle" || repository.input.OwnedBy != nil || repository.api {
		t.Fatalf("wrong legacy input: %#v", repository)
	}
	if len(repository.requested) != 3 || repository.requested["deleted_at"] == nil || repository.requested["created_by"] == nil {
		t.Fatalf("requested activity payload was not the original body: %#v", repository.requested)
	}
	if response.Name == nil || *response.Name != "Draft cycle" || response.Version == nil || *response.Version != 1 {
		t.Fatalf("unexpected app response: %#v", response)
	}
}

func TestCycleLifecycleV1CreateRunsPermissionBeforeThrottleAndPassesAPIFields(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	repository := &cycleLifecycleRepositoryStub{result: map[string]any{
		"id": "00000000-0000-4000-8000-000000000202", "name": "Cycle", "created_by": userID.String(),
		"description": "", "end_date": nil, "start_date": nil, "owned_by": userID.String(),
		"external_id": nil, "external_source": nil, "logo_props": map[string]any{}, "progress_snapshot": map[string]any{},
		"project": "00000000-0000-4000-8000-000000000303", "workspace": "00000000-0000-4000-8000-000000000404",
		"sort_order": float64(65535), "version": 1, "timezone": "UTC", "view_props": map[string]any{},
	}}
	service := &cycleLifecycleService{
		repo:     repository,
		keys:     &cycleLifecycleAPIKeyValidatorStub{principal: apikey.Principal{UserID: userID, Timezone: "UTC"}},
		throttle: cycleLifecycleThrottleStub{rate: apikey.Rate{Allowed: true, Remaining: "59", Reset: 1}},
	}
	ctx, err := service.APIKeyAuth(context.Background(), "api-token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.PostAPIV1WorkspacesBySlugProjectsByProjectIDCycles(ctx, &api.PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesPayload{
		Slug: "cycle-lifecycle", ProjectID: "00000000-0000-4000-8000-000000000303",
	}, io.NopCloser(strings.NewReader(`{"name":"Cycle","start_date":"2099-02-01","end_date":"2099-02-14"}`)))
	if err != nil {
		t.Fatalf("v1 create: %v", err)
	}
	if repository.input.Name != "Cycle" || !repository.api || repository.input.StartDate == nil || *repository.input.StartDate != "2099-02-01" {
		t.Fatalf("wrong v1 create input: %#v", repository.input)
	}
	if response.Name == nil || *response.Name != "Cycle" || response.Timezone == nil || *response.Timezone != api.TimezoneEnum("UTC") {
		t.Fatalf("unexpected v1 response: %#v", response)
	}
	if response.CompletedIssues != nil || response.TotalIssues != nil {
		t.Fatalf("v1 cycle create unexpectedly included read aggregates: %#v", response)
	}
}
