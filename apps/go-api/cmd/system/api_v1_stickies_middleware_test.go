package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	stickiessvr "github.com/CaliLuke/go-plane/apiv1stickies/gen/http/api_v1_stickies/server"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

type apiV1StickyMiddlewareRepository struct {
	resolveErr    error
	createCalls   int
	retrieveCalls int
}

func (repository *apiV1StickyMiddlewareRepository) ResolveAPIWorkspace(context.Context, string, stickystore.UserID) (stickystore.WorkspaceID, error) {
	if repository.resolveErr != nil {
		return stickystore.WorkspaceID{}, repository.resolveErr
	}
	return stickystore.WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701")), nil
}

func (repository *apiV1StickyMiddlewareRepository) UserExists(context.Context, stickystore.UserID) (bool, error) {
	return true, nil
}

func (repository *apiV1StickyMiddlewareRepository) Create(_ context.Context, id stickystore.StickyID, scope stickystore.Scope, input stickystore.CreateStickyInput) (stickystore.Sticky, error) {
	repository.createCalls++
	name := "note"
	if input.Name != nil {
		name = *input.Name
	}
	return stickystore.Sticky{
		ID: id, OwnerID: scope.UserID, WorkspaceID: scope.WorkspaceID,
		CreatedAt: time.Date(2026, time.June, 30, 23, 0, 0, 123456000, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 30, 23, 0, 0, 123456000, time.UTC),
		Name:      &name, Description: []byte(`{}`), DescriptionHTML: "<p></p>", LogoProps: []byte(`{}`),
	}, nil
}

func (*apiV1StickyMiddlewareRepository) List(context.Context, stickystore.Scope, stickystore.ListOptions) (stickystore.StickyPage, error) {
	return stickystore.StickyPage{}, nil
}

func (repository *apiV1StickyMiddlewareRepository) Retrieve(_ context.Context, id stickystore.StickyID, scope stickystore.Scope) (stickystore.Sticky, error) {
	repository.retrieveCalls++
	name := "note"
	return stickystore.Sticky{
		ID: id, OwnerID: scope.UserID, WorkspaceID: scope.WorkspaceID,
		CreatedAt: time.Date(2026, time.June, 30, 23, 0, 0, 123456000, time.UTC),
		UpdatedAt: time.Date(2026, time.June, 30, 23, 0, 0, 123456000, time.UTC),
		Name:      &name, Description: []byte(`{}`), DescriptionHTML: "<p></p>", LogoProps: []byte(`{}`),
	}, nil
}

func (*apiV1StickyMiddlewareRepository) PatchOwned(context.Context, stickystore.StickyID, stickystore.Scope, stickystore.PatchStickyInput) (stickystore.Sticky, error) {
	return stickystore.Sticky{}, errors.New("unexpected Sticky patch")
}

func (*apiV1StickyMiddlewareRepository) DeleteOwned(context.Context, stickystore.StickyID, stickystore.Scope) error {
	return errors.New("unexpected Sticky delete")
}

type apiV1StickyMiddlewareKeys struct{ calls int }

func (keys *apiV1StickyMiddlewareKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	keys.calls++
	if key == "" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return apikey.Principal{UserID: uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"), Timezone: "UTC"}, nil
}

type apiV1StickyMiddlewareThrottle struct {
	calls   int
	allowed bool
}

func (throttle *apiV1StickyMiddlewareThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	throttle.calls++
	return apikey.Rate{Allowed: throttle.allowed, Remaining: "59", Reset: 42, Wait: time.Second}, nil
}

type apiV1StickyMiddlewareEffects struct{}

func (apiV1StickyMiddlewareEffects) SoftDelete(context.Context, string, string, string) error {
	return nil
}

func newAPIV1StickyMiddlewareHandler(t *testing.T, service api.Service, dispatch *activityHTTPService, logger *apiV1ActivityLoggerStub, bodyLimit int64) http.Handler {
	t.Helper()
	mux := loomhttp.NewMuxer()
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	options := apiV1CompatibilityOptions{activityLogger: logger, activitySecret: "sticky-test-secret", bodySizeLimit: bodyLimit}
	stack := activityReadsMiddlewareStack(options, nil, djangoSessionCookieConfig{})
	server := stickiessvr.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, policy)
	server.Use(apiV1StickyActivityMiddlewareStack(options, nil, djangoSessionCookieConfig{}))
	stickiessvr.Mount(mux, server)
	handler := newActivityReadsProtocolMiddleware(membershipActivityProtocolDispatch{ActivityReadsProtocolAuthorizer: dispatch, stickies: service.(systemapi.APIV1StickyProtocolAuthorizer)}, policy, stack, options, djangoSessionCookieConfig{}, bodyLimit, true)(mux)
	handler = readServicesCompatibilityMiddleware(handler)
	handler = readServicesDjangoPathConverterMiddleware(policy, &options)(handler)
	handler = loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(handler)
	return handler
}

func TestAPIV1StickyFormatSuffixDispatchesWithoutRedirect(t *testing.T) {
	repository := &apiV1StickyMiddlewareRepository{}
	keys := &apiV1StickyMiddlewareKeys{}
	throttle := &apiV1StickyMiddlewareThrottle{allowed: true}
	service, err := systemapi.NewAPIV1Stickies(repository, keys, throttle, apiV1StickyMiddlewareEffects{})
	if err != nil {
		t.Fatal(err)
	}
	dispatch := &activityHTTPService{}
	logger := &apiV1ActivityLoggerStub{}
	handler := newAPIV1StickyMiddlewareHandler(t, service, dispatch, logger, defaultFileSizeLimit)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies.json", strings.NewReader(`{"name":"note"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "sticky-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || response.Header().Get("Location") != "" || repository.createCalls != 1 {
		t.Fatalf("slashless collection format alias response=%d location=%q creates=%d body=%q", response.Code, response.Header().Get("Location"), repository.createCalls, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/stickies/00000000-0000-4000-8000-000000000811.json", nil)
	request.Header.Set("X-Api-Key", "sticky-key")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Location") != "" || repository.retrieveCalls != 1 {
		t.Fatalf("slashless detail format alias response=%d location=%q retrieves=%d body=%q", response.Code, response.Header().Get("Location"), repository.retrieveCalls, response.Body.String())
	}

	priorAuthCalls := keys.calls
	request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/stickies/00000000-0000-4000-8000-000000000811.xml", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != `{"detail":"Not found."}` || keys.calls != priorAuthCalls {
		t.Fatalf("unsupported format response=%d body=%q auth_calls=%d, want source 404 before auth", response.Code, response.Body.String(), keys.calls)
	}
}

func TestAPIV1StickyProtocolCompatibilityHeadersAndMethodDispatch(t *testing.T) {
	repository := &apiV1StickyMiddlewareRepository{}
	keys := &apiV1StickyMiddlewareKeys{}
	throttle := &apiV1StickyMiddlewareThrottle{allowed: true}
	service, err := systemapi.NewAPIV1Stickies(repository, keys, throttle, apiV1StickyMiddlewareEffects{})
	if err != nil {
		t.Fatal(err)
	}
	dispatch := &activityHTTPService{}
	logger := &apiV1ActivityLoggerStub{}
	handler := newAPIV1StickyMiddlewareHandler(t, service, dispatch, logger, 4)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", strings.NewReader(`{"name":"note"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "sticky-key")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || keys.calls != 0 || throttle.calls != 0 || len(logger.records) != 0 {
		t.Fatalf("request-size stage response=%d authorization=%d throttling=%d audit=%d, want request-size rejection before authorization, throttling and audit", response.Code, keys.calls, throttle.calls, len(logger.records))
	}

	handler = newAPIV1StickyMiddlewareHandler(t, service, dispatch, logger, defaultFileSizeLimit)
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", strings.NewReader(`{"name":"note"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "sticky-key")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || keys.calls != 1 || throttle.calls != 1 || repository.createCalls != 1 {
		t.Fatalf("authorization/throttling stages response=%d body=%q authorization=%d throttling=%d creates=%d", response.Code, response.Body.String(), keys.calls, throttle.calls, repository.createCalls)
	}
	for header, want := range map[string]string{
		"Allow":                      "GET, POST, HEAD, OPTIONS",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Referrer-Policy":            "same-origin",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
	} {
		if got := response.Header().Get(header); got != want {
			t.Errorf("create %s = %q, want %q", header, got, want)
		}
	}
	if response.Header().Get("X-Ratelimit-Remaining") != "" || response.Header().Get("X-Ratelimit-Reset") != "" {
		t.Fatalf("successful Sticky response has unsupported API rate headers: %v", response.Header())
	}
	if len(logger.records) != 1 || logger.records[0].ResponseCode != http.StatusCreated {
		t.Fatalf("create audit records = %#v, want one successful request", logger.records)
	}

	repository.resolveErr = stickystore.ErrWorkspaceAccessDenied
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/other/stickies/", strings.NewReader(`{"name":"note"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "sticky-key")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || keys.calls != 2 || throttle.calls != 1 || repository.createCalls != 1 {
		t.Fatalf("authorization stage response=%d authorization=%d throttling=%d creates=%d, want workspace authorization before throttling/write", response.Code, keys.calls, throttle.calls, repository.createCalls)
	}

	repository.resolveErr = nil
	throttle.allowed = false
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/", strings.NewReader(`{"name":"note"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "sticky-key")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests || keys.calls != 3 || throttle.calls != 2 || repository.createCalls != 1 || response.Header().Get("Retry-After") != "1" {
		t.Fatalf("throttling stage response=%d retry=%q authorization=%d throttling=%d creates=%d", response.Code, response.Header().Get("Retry-After"), keys.calls, throttle.calls, repository.createCalls)
	}
	if len(logger.records) != 3 || logger.records[1].ResponseCode != http.StatusForbidden || logger.records[2].ResponseCode != http.StatusTooManyRequests {
		t.Fatalf("write audit records = %#v, want 201/403/429", logger.records)
	}

	throttle.allowed = true
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/stickies/00000000-0000-4000-8000-000000000001/", nil)
	request.Header.Set("X-Api-Key", "sticky-key")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || dispatch.protocolCalls != 0 || keys.calls != 4 || throttle.calls != 3 {
		t.Fatalf("unsupported method response=%d body=%q dispatch_calls=%d method=%q", response.Code, response.Body.String(), dispatch.protocolCalls, dispatch.protocolMethod)
	}
	if response.Header().Get("Allow") != "GET, PUT, PATCH, DELETE, HEAD, OPTIONS" {
		t.Fatalf("unsupported detail Allow = %q", response.Header().Get("Allow"))
	}
	if len(logger.records) != 4 || logger.records[3].ResponseCode != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method audit records = %#v, want one additional 405", logger.records)
	}
}
