package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	api "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

func TestAPIV1StickyPUTAllowsOmittedSerializerFields(t *testing.T) {
	stripped := "preserved text"
	current := stickystore.Sticky{DescriptionHTML: "<p>preserved text</p>", DescriptionStripped: &stripped}
	patch, failures := apiV1StickyPatchFields(djangoinput.Input{Fields: map[string]any{}}, current, true, time.UTC)
	if len(failures) != 0 {
		t.Fatalf("empty PUT field failures = %#v", failures)
	}
	if patch.Name != nil || patch.Color != nil || patch.BackgroundColor != nil || patch.Description != nil || patch.LogoProps != nil {
		t.Fatalf("empty PUT changed omitted fields: %#v", patch)
	}
	if patch.DescriptionStripped == nil || *patch.DescriptionStripped == nil || **patch.DescriptionStripped != stripped {
		t.Fatalf("empty PUT should retain model-derived stripped HTML, got %#v", patch.DescriptionStripped)
	}
}

func TestAPIV1StickyOptionsBodiesMatchRecordedDRFMetadata(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not return the test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{
		UserID:   stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702")),
		Timezone: "UTC",
		Key:      "valid-api-key",
	})
	for _, test := range []struct {
		name     string
		fixture  string
		call     func(*apiV1StickiesService) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error)
		wantName string
	}{
		{
			name:    "collection",
			fixture: "list-options",
			call: func(service *apiV1StickiesService) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
				return service.ListOptions(ctx, &api.ListOptionsPayload{Slug: "sticky-v1-parity"})
			},
			wantName: "Sticky List",
		},
		{
			name:    "detail",
			fixture: "retrieve-options",
			call: func(service *apiV1StickiesService) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
				return service.DetailOptions(ctx, &api.DetailOptionsPayload{Slug: "sticky-v1-parity", Pk: "00000000-0000-4000-8000-000000000701"})
			},
			wantName: "Sticky Instance",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{}
			service := &apiV1StickiesService{repository: repository, throttle: &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}}
			metadata, body, err := test.call(service)
			if err != nil {
				t.Fatalf("OPTIONS service call failed: %v", err)
			}
			defer body.Close()
			if metadata == nil || metadata.ContentType != "application/json" {
				t.Fatalf("OPTIONS metadata = %#v, want application/json", metadata)
			}
			actualBytes, err := io.ReadAll(body)
			if err != nil {
				t.Fatal(err)
			}
			fixturePath := filepath.Join(repoRoot, "apps", "go-api", "internal", "parity", "testdata", "fixtures", "api_v1_stickies", test.fixture+".json")
			fixtureBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("read source fixture %s: %v", fixturePath, err)
			}
			var fixture struct {
				Response struct {
					Body any `json:"body"`
				} `json:"response"`
			}
			if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
				t.Fatalf("decode source fixture: %v", err)
			}
			var expected, actual any
			if err := json.Unmarshal(mustJSON(t, fixture.Response.Body), &expected); err != nil {
				t.Fatalf("decode fixture response body: %v", err)
			}
			if err := json.Unmarshal(actualBytes, &actual); err != nil {
				t.Fatalf("decode OPTIONS response: %v", err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("OPTIONS response differs from %s fixture\nactual: %#v\nexpected: %#v", test.fixture, actual, expected)
			}
			bodyMap, ok := actual.(map[string]any)
			if !ok || bodyMap["name"] != test.wantName {
				t.Fatalf("OPTIONS response name = %#v, want %q", bodyMap["name"], test.wantName)
			}
		})
	}
}

type apiV1StickyScopeRepositoryStub struct {
	apiV1StickiesRepository
	resolveErr    error
	workspaceID   stickystore.WorkspaceID
	retrieveErr   error
	userExists    bool
	userExistsErr error
	userIDs       []stickystore.UserID
	listCalls     int
	createCalls   int
	createInput   stickystore.CreateStickyInput
	retrieveCalls int
	retrieveScope stickystore.Scope
	sticky        stickystore.Sticky
	patchScope    stickystore.Scope
	patchCalls    int
}

type apiV1StickyThrottleStub struct {
	rate  apikey.Rate
	calls int
}

func (throttle *apiV1StickyThrottleStub) Allow(context.Context, string) (apikey.Rate, error) {
	throttle.calls++
	return throttle.rate, nil
}

func (repository *apiV1StickyScopeRepositoryStub) ResolveAPIWorkspace(context.Context, string, stickystore.UserID) (stickystore.WorkspaceID, error) {
	if repository.resolveErr != nil {
		return repository.workspaceID, repository.resolveErr
	}
	if repository.workspaceID != (stickystore.WorkspaceID{}) {
		return repository.workspaceID, nil
	}
	return stickystore.WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701")), nil
}

func (repository *apiV1StickyScopeRepositoryStub) UserExists(_ context.Context, userID stickystore.UserID) (bool, error) {
	repository.userIDs = append(repository.userIDs, userID)
	return repository.userExists, repository.userExistsErr
}

func (repository *apiV1StickyScopeRepositoryStub) List(_ context.Context, _ stickystore.Scope, _ stickystore.ListOptions) (stickystore.StickyPage, error) {
	repository.listCalls++
	return stickystore.StickyPage{}, nil
}

func (repository *apiV1StickyScopeRepositoryStub) Create(_ context.Context, _ stickystore.StickyID, scope stickystore.Scope, input stickystore.CreateStickyInput) (stickystore.Sticky, error) {
	repository.createCalls++
	repository.createInput = input
	return repository.sticky, nil
}

func (repository *apiV1StickyScopeRepositoryStub) Retrieve(_ context.Context, _ stickystore.StickyID, scope stickystore.Scope) (stickystore.Sticky, error) {
	repository.retrieveCalls++
	repository.retrieveScope = scope
	return repository.sticky, repository.retrieveErr
}

func (repository *apiV1StickyScopeRepositoryStub) PatchOwned(_ context.Context, _ stickystore.StickyID, scope stickystore.Scope, patch stickystore.PatchStickyInput) (stickystore.Sticky, error) {
	repository.patchCalls++
	repository.patchScope = scope
	if patch.Name != nil && *patch.Name != nil {
		repository.sticky.Name = *patch.Name
	}
	return repository.sticky, nil
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAPIV1StickyListRejectsNonpositivePerPageBeforeCursorParsing(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	for _, test := range []struct {
		name    string
		perPage string
		cursor  string
	}{
		{name: "zero before malformed cursor", perPage: "0", cursor: "bad"},
		{name: "zero before oversized cursor", perPage: "0", cursor: "1:" + strings.Repeat("9", 100) + ":0"},
		{name: "huge negative before narrowing", perPage: "-" + strings.Repeat("9", 4300)},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{}
			throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
			service := &apiV1StickiesService{repository: repository, throttle: throttle}
			ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
			perPage, cursor := test.perPage, test.cursor
			_, _, err := service.List(ctx, &api.ListPayload{Slug: "demo", PerPage: &perPage, Cursor: &cursor})

			var response *api.APIV1StickyErrorResponse
			if !errors.As(err, &response) || response.Name != "status_400" {
				t.Fatalf("List error = %#v, want status 400", err)
			}
			if got, want := string(response.Body), `{"detail":"Error in parsing"}`; got != want {
				t.Fatalf("List error body = %s, want %s", got, want)
			}
			if repository.listCalls != 0 {
				t.Fatalf("repository List called %d times for invalid pagination, want 0", repository.listCalls)
			}
			if throttle.calls != 1 {
				t.Fatalf("throttle calls = %d, want 1", throttle.calls)
			}
		})
	}
}

func TestAPIV1StickyRejectsNonexistentAuditUsersBeforeWrites(t *testing.T) {
	missingUser := uuid.MustParse("00000000-0000-4000-8000-000000000799")
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	stickyID := stickystore.StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703"))
	for _, test := range []struct {
		name                string
		body                string
		call                func(*apiV1StickiesService, context.Context, io.ReadCloser) error
		want                string
		wantUserExistsCalls int
	}{
		{
			name: "create validates both IDs before persistence",
			body: `{"created_by":"00000000-0000-4000-8000-000000000799","updated_by":"00000000-0000-4000-8000-000000000799"}`,
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo"}, body)
				return err
			},
			want:                `{"created_by":["Invalid pk \"00000000-0000-4000-8000-000000000799\" - object does not exist."],"updated_by":["Invalid pk \"00000000-0000-4000-8000-000000000799\" - object does not exist."]}`,
			wantUserExistsCalls: 2,
		},
		{
			name: "patch validates both IDs before model save",
			body: `{"created_by":"00000000-0000-4000-8000-000000000799","updated_by":"00000000-0000-4000-8000-000000000799"}`,
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: uuid.UUID(stickyID).String()}, body)
				return err
			},
			want:                `{"created_by":["Invalid pk \"00000000-0000-4000-8000-000000000799\" - object does not exist."],"updated_by":["Invalid pk \"00000000-0000-4000-8000-000000000799\" - object does not exist."]}`,
			wantUserExistsCalls: 2,
		},
		{
			name: "create rejects malformed user UUID before lookup",
			body: `{"created_by":"not-a-uuid"}`,
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo"}, body)
				return err
			},
			want:                `{"created_by":["Must be a valid UUID."]}`,
			wantUserExistsCalls: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{sticky: stickystore.Sticky{ID: stickyID, OwnerID: userID, Description: []byte(`{}`), DescriptionHTML: "<p>body</p>", LogoProps: []byte(`{}`)}}
			service := &apiV1StickiesService{repository: repository, throttle: &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}}
			ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
			metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var requestErr error
			handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				requestErr = test.call(service, request.Context(), io.NopCloser(strings.NewReader(test.body)))
			}))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", nil)
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
			var response *api.APIV1StickyErrorResponse
			if !errors.As(requestErr, &response) || response.Name != "status_400" {
				t.Fatalf("service error = %#v, want status 400", requestErr)
			}
			if got := string(response.Body); got != test.want {
				t.Fatalf("validation body = %s, want %s", got, test.want)
			}
			if len(repository.userIDs) != test.wantUserExistsCalls {
				t.Fatalf("UserExists calls = %#v, want %d", repository.userIDs, test.wantUserExistsCalls)
			}
			for _, userID := range repository.userIDs {
				if userID != stickystore.UserID(missingUser) {
					t.Fatalf("UserExists checked ID %s, want missing ID %s", uuid.UUID(userID), missingUser)
				}
			}
			if repository.createCalls != 0 || repository.patchCalls != 0 {
				t.Fatalf("invalid audit IDs reached persistence: creates=%d patches=%d", repository.createCalls, repository.patchCalls)
			}
		})
	}
}

func TestAPIV1StickyAuditUserLookupFailuresAreInternalErrorsBeforeWrites(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	stickyID := stickystore.StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703"))
	for _, test := range []struct {
		name string
		call func(*apiV1StickiesService, context.Context, io.ReadCloser) error
	}{
		{
			name: "create",
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo"}, body)
				return err
			},
		},
		{
			name: "patch",
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: uuid.UUID(stickyID).String()}, body)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{
				sticky:        stickystore.Sticky{ID: stickyID, OwnerID: userID, Description: []byte(`{}`), DescriptionHTML: "<p>body</p>", LogoProps: []byte(`{}`)},
				userExistsErr: errors.New("database unavailable"),
			}
			service := &apiV1StickiesService{repository: repository, throttle: &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}}
			ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
			metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var requestErr error
			handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				requestErr = test.call(service, request.Context(), io.NopCloser(strings.NewReader(`{"created_by":"00000000-0000-4000-8000-000000000799"}`)))
			}))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", nil)
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))

			var response *api.APIV1StickyErrorResponse
			if !errors.As(requestErr, &response) || response.Name != "status_500" {
				t.Fatalf("service error = %#v, want status 500", requestErr)
			}
			if got, want := string(response.Body), `{"error":"Something went wrong please try again later"}`; got != want {
				t.Fatalf("internal error body = %s, want %s", got, want)
			}
			if len(repository.userIDs) != 1 {
				t.Fatalf("UserExists calls = %#v, want one failed lookup", repository.userIDs)
			}
			if repository.createCalls != 0 || repository.patchCalls != 0 {
				t.Fatalf("lookup failure reached persistence: creates=%d patches=%d", repository.createCalls, repository.patchCalls)
			}
		})
	}
}

func TestAPIV1StickyRejectsNonDictionaryWriteBodiesBeforePersistence(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	stickyID := stickystore.StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703"))
	for _, test := range []struct {
		name string
		call func(*apiV1StickiesService, context.Context, io.ReadCloser) error
	}{
		{
			name: "create list body",
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo"}, body)
				return err
			},
		},
		{
			name: "patch list body",
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: uuid.UUID(stickyID).String()}, body)
				return err
			},
		},
		{
			name: "put list body",
			call: func(service *apiV1StickiesService, ctx context.Context, body io.ReadCloser) error {
				_, _, err := service.Update(ctx, &api.UpdatePayload{Slug: "demo", Pk: uuid.UUID(stickyID).String()}, body)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{
				sticky: stickystore.Sticky{ID: stickyID, OwnerID: userID, Description: []byte(`{}`), DescriptionHTML: "<p>body</p>", LogoProps: []byte(`{}`)},
			}
			throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
			service := &apiV1StickiesService{repository: repository, throttle: throttle}
			ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
			metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var requestErr error
			handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				requestErr = test.call(service, request.Context(), io.NopCloser(strings.NewReader(`[]`)))
			}))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", nil)
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
			var response *api.APIV1StickyErrorResponse
			if !errors.As(requestErr, &response) || response.Name != "status_400" {
				t.Fatalf("service error = %#v, want status 400", requestErr)
			}
			if got, want := string(response.Body), `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}`; got != want {
				t.Fatalf("non-dictionary body = %s, want %s", got, want)
			}
			if repository.createCalls != 0 || repository.patchCalls != 0 {
				t.Fatalf("non-dictionary body reached persistence: creates=%d patches=%d", repository.createCalls, repository.patchCalls)
			}
			if test.name != "create list body" && repository.retrieveCalls != 1 {
				t.Fatalf("detail write retrieved %d times, want lookup before body validation", repository.retrieveCalls)
			}
			if throttle.calls != 1 {
				t.Fatalf("throttle calls = %d, want 1", throttle.calls)
			}
		})
	}
}

func TestAPIV1StickyNullWriteBodyUsesDRFEmptyDataValidation(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	repository := &apiV1StickyScopeRepositoryStub{}
	service := &apiV1StickiesService{repository: repository, throttle: &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}}
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var requestErr error
	handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_, _, requestErr = service.Create(request.Context(), &api.CreatePayload{Slug: "demo"}, io.NopCloser(strings.NewReader(`null`)))
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", nil)
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
	var response *api.APIV1StickyErrorResponse
	if !errors.As(requestErr, &response) || response.Name != "status_400" {
		t.Fatalf("service error = %#v, want status 400", requestErr)
	}
	if got, want := string(response.Body), `{"non_field_errors":["No data provided"]}`; got != want {
		t.Fatalf("null body = %s, want %s", got, want)
	}
	if repository.createCalls != 0 {
		t.Fatalf("null body reached create persistence %d times", repository.createCalls)
	}
}

func TestAPIV1StickyDetailOptionsOmitsUpdateActionForMissingObject(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	stickyID := stickystore.StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703"))
	repository := &apiV1StickyScopeRepositoryStub{retrieveErr: stickystore.ErrStickyNotFound}
	throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := &apiV1StickiesService{repository: repository, throttle: throttle}
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
	metadata, body, err := service.DetailOptions(ctx, &api.DetailOptionsPayload{Slug: "demo", Pk: uuid.UUID(stickyID).String()})
	if err != nil {
		t.Fatalf("missing object OPTIONS error = %v, want metadata without PUT action", err)
	}
	defer body.Close()
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("missing object OPTIONS metadata = %#v", metadata)
	}
	var result map[string]any
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if _, exists := result["actions"]; exists {
		t.Fatalf("missing object OPTIONS contains actions: %#v", result)
	}
	if repository.retrieveCalls != 1 || throttle.calls != 1 {
		t.Fatalf("missing object OPTIONS lookup/throttle calls = %d/%d, want 1/1", repository.retrieveCalls, throttle.calls)
	}
}

func TestAPIV1StickyMalformedDetailOptionsOmitsUpdateAction(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	repository := &apiV1StickyScopeRepositoryStub{}
	throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := &apiV1StickiesService{repository: repository, throttle: throttle}
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
	metadata, body, err := service.DetailOptions(ctx, &api.DetailOptionsPayload{Slug: "demo", Pk: "not-a-uuid"})
	if err != nil {
		t.Fatalf("malformed detail OPTIONS error = %v, want metadata without PUT action", err)
	}
	defer body.Close()
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("malformed detail OPTIONS metadata = %#v", metadata)
	}
	var result map[string]any
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if _, exists := result["actions"]; exists {
		t.Fatalf("malformed detail OPTIONS contains actions: %#v", result)
	}
	if repository.retrieveCalls != 0 || throttle.calls != 1 {
		t.Fatalf("malformed detail OPTIONS lookup/throttle calls = %d/%d, want 0/1", repository.retrieveCalls, throttle.calls)
	}
}

func TestAPIV1StickySoftDeletedWorkspaceScopeMatchesDjangoQueryset(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	workspaceID := stickystore.WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701"))
	for _, operation := range []stickystore.Operation{
		stickystore.OperationList,
		stickystore.OperationRetrieve,
		stickystore.OperationUpdate,
		stickystore.OperationDelete,
	} {
		t.Run(string(operation), func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{workspaceID: workspaceID, resolveErr: stickystore.ErrStickyWorkspaceDeleted}
			service := &apiV1StickiesService{repository: repository}
			ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
			scope, err := service.scope(ctx, "demo", operation)
			if err != nil {
				t.Fatalf("scope on soft-deleted workspace = %v, want retained Sticky queryset scope", err)
			}
			if scope.WorkspaceID != workspaceID || scope.Operation != operation || scope.UserID != userID {
				t.Fatalf("scope = %#v, want workspace %s operation %q and owner %s", scope, uuid.UUID(workspaceID), operation, uuid.UUID(userID))
			}
		})
	}

	repository := &apiV1StickyScopeRepositoryStub{workspaceID: workspaceID, resolveErr: stickystore.ErrStickyWorkspaceDeleted}
	service := &apiV1StickiesService{repository: repository}
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
	if _, err := service.scope(ctx, "demo", stickystore.OperationCreate); err == nil {
		t.Fatal("create on soft-deleted workspace succeeded, want source 404")
	} else {
		var response *api.APIV1StickyErrorResponse
		if !errors.As(err, &response) || response.Name != "status_404" {
			t.Fatalf("create scope error = %#v, want source 404", err)
		}
	}
}

func TestAPIV1StickyUpdateUsesDistinctRetrieveAndUpdateScopes(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	workspaceID := stickystore.WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701"))
	stickyID := stickystore.StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703"))
	name := "before"
	repository := &apiV1StickyScopeRepositoryStub{sticky: stickystore.Sticky{
		ID: stickyID, WorkspaceID: workspaceID, OwnerID: userID, Name: &name,
		Description: []byte(`{}`), DescriptionHTML: "<p>body</p>", LogoProps: []byte(`{}`),
	}}
	service := &apiV1StickiesService{repository: repository, throttle: &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}}
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC", Key: "valid"})
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var updateErr error
	handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		body := io.NopCloser(strings.NewReader(`{"name":"after"}`))
		_, _, updateErr = service.update(request.Context(), "demo", uuid.UUID(stickyID).String(), body, false)
	}))
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/demo/stickies/"+uuid.UUID(stickyID).String()+"/", nil)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	err = updateErr
	if err != nil {
		t.Fatalf("update() error = %v", err)
	}
	if repository.retrieveCalls != 1 || repository.retrieveScope.Operation != stickystore.OperationRetrieve {
		t.Fatalf("retrieve calls/scope = %d/%q, want one retrieve operation", repository.retrieveCalls, repository.retrieveScope.Operation)
	}
	if repository.patchCalls != 1 || repository.patchScope.Operation != stickystore.OperationUpdate {
		t.Fatalf("patch calls/scope = %d/%q, want one update operation", repository.patchCalls, repository.patchScope.Operation)
	}
	if repository.sticky.Name == nil || *repository.sticky.Name != "after" {
		t.Fatalf("stored Sticky name = %v, want after", repository.sticky.Name)
	}
}

func TestAPIV1StickyPermissionAndDeletedWorkspacePrecedeMalformedPK(t *testing.T) {
	userID := stickystore.UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	ctx := context.WithValue(context.Background(), apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: userID, Timezone: "UTC"})
	for _, test := range []struct {
		name         string
		resolveErr   error
		wantName     string
		wantBody     string
		wantThrottle int
	}{
		{name: "nonmember permission wins", resolveErr: stickystore.ErrWorkspaceAccessDenied, wantName: "status_403", wantBody: `{"detail":"You do not have permission to perform this action."}`},
		{name: "deleted workspace reaches detail lookup", resolveErr: stickystore.ErrStickyWorkspaceDeleted, wantName: "status_404", wantBody: `{"detail":"Not found."}`, wantThrottle: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiV1StickyScopeRepositoryStub{resolveErr: test.resolveErr}
			throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
			service := &apiV1StickiesService{repository: repository, throttle: throttle}
			_, _, err := service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: "malformed"})
			var response *api.APIV1StickyErrorResponse
			if !errors.As(err, &response) {
				t.Fatalf("Retrieve() error = %T %v, want typed API error", err, err)
			}
			if response.Name != test.wantName || string(response.Body) != test.wantBody || repository.retrieveCalls != 0 || throttle.calls != test.wantThrottle {
				t.Fatalf("response=(%s, %s) retrieve calls=%d throttle calls=%d, want (%s, %s) and throttle calls=%d", response.Name, response.Body, repository.retrieveCalls, throttle.calls, test.wantName, test.wantBody, test.wantThrottle)
			}
		})
	}

	repository := &apiV1StickyScopeRepositoryStub{}
	throttle := &apiV1StickyThrottleStub{rate: apikey.Rate{Allowed: true}}
	service := &apiV1StickiesService{repository: repository, throttle: throttle}
	_, _, err := service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: "malformed"})
	var response *api.APIV1StickyErrorResponse
	if !errors.As(err, &response) || response.Name != "status_404" || string(response.Body) != `{"detail":"Not found."}` || repository.retrieveCalls != 0 || throttle.calls != 1 {
		t.Fatalf("member malformed PK response = %#v, retrieve calls=%d throttle calls=%d, want Sticky 404 after permission/throttle before repository lookup", response, repository.retrieveCalls, throttle.calls)
	}
}

func TestAPIV1StickySerializerNullAndWritableAuditFields(t *testing.T) {
	owner := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702")
	input := djangoinput.Input{Fields: map[string]any{
		"created_by":       owner.String(),
		"deleted_at":       nil,
		"description_html": "<p>updated</p>",
	}}
	patch, failures := apiV1StickyPatchFields(input, stickystore.Sticky{}, false, time.UTC)
	if len(failures) != 0 {
		t.Fatalf("writable audit fields failures = %#v", failures)
	}
	if patch.CreatedByID == nil || *patch.CreatedByID == nil || uuid.UUID(**patch.CreatedByID) != owner {
		t.Fatalf("created_by was not accepted: %#v", patch.CreatedByID)
	}
	if patch.DeletedAt == nil || *patch.DeletedAt != nil {
		t.Fatalf("nullable deleted_at was not accepted: %#v", patch.DeletedAt)
	}
	if patch.DescriptionHTML == nil || *patch.DescriptionHTML != "<p>updated</p>" {
		t.Fatalf("description_html = %#v", patch.DescriptionHTML)
	}
	input = djangoinput.Input{Fields: map[string]any{"description": nil}}
	_, failures = apiV1StickyPatchFields(input, stickystore.Sticky{}, false, time.UTC)
	if failures["description"] == nil {
		t.Fatalf("null description should fail validation, got %#v", failures)
	}
}

func TestAPIV1StickyDeletedAtUsesDjangoDateTimeParsing(t *testing.T) {
	parsed, err := apiV1StickyDatetime("2026-1-2T3:4:5.123456789012 +0530", time.UTC)
	if err != nil {
		t.Fatalf("Django-compatible timestamp rejected: %v", err)
	}
	want := time.Date(2026, time.January, 1, 21, 34, 5, 123456000, time.UTC)
	if !parsed.Equal(want) {
		t.Fatalf("parsed timestamp = %s, want %s", parsed, want)
	}
}

func TestAPIV1StickyInvalidDeletedAtUsesExactDRFMessage(t *testing.T) {
	_, failures := apiV1StickyPatchFields(
		djangoinput.Input{Fields: map[string]any{"deleted_at": "not-a-datetime"}},
		stickystore.Sticky{},
		false,
		time.UTC,
	)
	want := []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
	if !reflect.DeepEqual(failures["deleted_at"], want) {
		t.Fatalf("deleted_at validation = %#v, want %#v", failures["deleted_at"], want)
	}
}

func TestAPIV1StickyNaiveDeletedAtUsesActorTimezone(t *testing.T) {
	kathmandu, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := apiV1StickyDatetime("2026-02-03T04:05", kathmandu)
	if err != nil {
		t.Fatalf("naive actor-local timestamp rejected: %v", err)
	}
	want := time.Date(2026, time.February, 2, 22, 20, 0, 0, time.UTC)
	if !parsed.Equal(want) {
		t.Fatalf("naive timestamp = %s, want %s", parsed, want)
	}
}

func TestAPIV1StickyTimesUseActorTimezone(t *testing.T) {
	instant := time.Date(2026, time.June, 30, 23, 0, 0, 123456000, time.UTC)
	if got, want := apiV1StickyTimeInZone(instant, "Asia/Kathmandu"), "2026-07-01T04:45:00.123456+05:45"; got != want {
		t.Fatalf("serialized API v1 Sticky time = %q, want %q", got, want)
	}
}
