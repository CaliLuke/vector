package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/platform"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	projectmembershiplifecycleclient "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/client"
	projectmembershiplifecycleserver "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	projectmembershiplifecycleapi "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type apiV1ActivityLoggerStub struct {
	records   []publicprojectmetadatareadsstore.APIActivityLog
	err       error
	beforeLog func()
}

func (logger *apiV1ActivityLoggerStub) Log(_ context.Context, record publicprojectmetadatareadsstore.APIActivityLog) error {
	if logger.beforeLog != nil {
		logger.beforeLog()
	}
	logger.records = append(logger.records, record)
	return logger.err
}

func TestAPIV1CompatibilityGeneratedClientDecodesDjangoShapedError(t *testing.T) {
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:3000"}}})
	if err != nil {
		t.Fatal(err)
	}
	generated := projectmembershiplifecycleserver.New(
		projectmembershiplifecycleapi.NewEndpoints(apiV1MembershipErrorService{}),
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
		cors,
	)
	generated.Mount(mux)
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})(mux)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var contract *loomhttp.ResponseContractCase
	for _, candidate := range projectmembershiplifecycleserver.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkResponseContractCases() {
		if candidate.ID == "ProjectMembershipLifecycle.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk.error.status_400.400" {
			contract = &candidate
			break
		}
	}
	if contract == nil || !contract.ErrorNameFromStatus {
		t.Fatalf("generated status 400 response contract = %+v, want explicit status-derived identity", contract)
	}
	capture := &apiV1ResponseCaptureDoer{doer: server.Client(), contract: contract}
	client := projectmembershiplifecycleclient.NewClient(serverURL.Scheme, serverURL.Host, capture, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, true)
	_, err = client.API.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk()(context.Background(), &projectmembershiplifecycleapi.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkPayload{
		Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111111", Pk: "22222222-2222-4222-8222-222222222222",
	})
	var response *projectmembershiplifecycleapi.MembershipLifecycleErrorResponseBody
	if !errors.As(err, &response) {
		t.Fatalf("generated client error = %T %#v, want typed Django-shaped response", err, err)
	}
	if response.Name != "status_400" {
		t.Fatalf("generated error name = %q, want status-derived status_400", response.Name)
	}
	if capture.headers.Get("Loom-Error") != "" {
		t.Fatalf("Django-compatible response unexpectedly includes Loom-Error: %v", capture.headers)
	}
	if capture.contractErr != nil {
		t.Fatalf("generated response contract: %v", capture.contractErr)
	}
	if string(response.Member) != `["This field is required."]` {
		t.Fatalf("member error = %s", response.Member)
	}
}

type apiV1MembershipErrorService struct {
	projectmembershiplifecycleapi.Service
}

func (apiV1MembershipErrorService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (apiV1MembershipErrorService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk(context.Context, *projectmembershiplifecycleapi.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkPayload) error {
	return &projectmembershiplifecycleapi.MembershipLifecycleErrorResponseBody{
		Name:   "status_400",
		Member: loom.MustJSONValueFrom([]string{"This field is required."}),
	}
}

type apiV1ResponseCaptureDoer struct {
	doer        loomhttp.Doer
	contract    *loomhttp.ResponseContractCase
	contractErr error
	headers     http.Header
}

func (doer *apiV1ResponseCaptureDoer) Do(request *http.Request) (*http.Response, error) {
	response, err := doer.doer.Do(request)
	if response != nil {
		doer.headers = response.Header.Clone()
		if doer.contract != nil {
			doer.contractErr = loomhttp.ValidateResponseContract(response, *doer.contract)
		}
	}
	return response, err
}

func TestAPIV1CompatibilityAddsRateHeadersAndPublishesRedactedActivity(t *testing.T) {
	activity := &apiV1ActivityLoggerStub{}
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "activity-secret", bodySizeLimit: defaultFileSizeLimit})(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		endpoint, ok := platform.PublicRequestEndpoint(request.Context())
		if !ok || endpoint != "https://api.example.com" {
			t.Fatalf("public request endpoint = %q, %t", endpoint, ok)
		}
		effects := apikey.EffectsFromContext(request.Context())
		if effects == nil {
			t.Fatal("API-key effects are missing")
		}
		if values := request.Header.Values("X-Api-Key"); len(values) != 1 || values[0] != "plane-secret-token,second-token" {
			t.Fatalf("normalized API key = %q", values)
		}
		effects.Authenticated = true
		effects.Rate = &apikey.Rate{Allowed: true, Remaining: "41", Reset: 1_788_480_060}
		body, err := io.ReadAll(request.Body)
		if err != nil || string(body) != `{"name":"request"}` {
			t.Fatalf("request body = %q, %v", body, err)
		}
		response.Header().Set("Vary", "Accept, Cookie")
		response.Header().Set("Loom-Error", "internal")
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"name":"response"}`))
	}))

	request := httptest.NewRequest(http.MethodPost, "http://internal/api/v1/workspaces/acme/modules/?expand=lead", strings.NewReader(`{"name":"request"}`))
	request.Host = "api.example.com"
	request.Header.Set("X-Forwarded-Proto", "https, http")
	request.Header.Set("X-Api-Key", "plane-secret-token")
	request.Header.Add("X-Api-Key", "second-token")
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Cookie", "session=secret")
	request.Header.Set("User-Agent", "compatibility-test")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated || response.Body.String() != `{"name":"response"}` {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("X-RateLimit-Remaining") != "41" || response.Header().Get("X-RateLimit-Reset") != "1788480060" {
		t.Fatalf("rate headers = %v", response.Header())
	}
	if response.Header().Get("Vary") != "Accept, Cookie, origin" || response.Header().Get("Loom-Error") != "" {
		t.Fatalf("compatibility headers = %v", response.Header())
	}
	if len(activity.records) != 1 {
		t.Fatalf("activity records = %d", len(activity.records))
	}
	record := activity.records[0]
	if record.TokenIdentifier != apiActivityTokenIdentifier("activity-secret", "plane-secret-token,second-token") || record.Path != "/api/v1/workspaces/acme/modules/" || record.Method != http.MethodPost || record.QueryParams != "expand=lead" {
		t.Fatalf("activity request metadata = %+v", record)
	}
	if record.Body != `{"name":"request"}` || record.ResponseBody != `{"name":"response"}` || record.ResponseCode != http.StatusCreated {
		t.Fatalf("activity bodies = %+v", record)
	}
	headers, ok := record.Headers.(string)
	if !ok || strings.Contains(headers, "plane-secret-token") || strings.Contains(headers, "Bearer secret") || strings.Contains(headers, "session=secret") || !strings.Contains(headers, "[REDACTED]") {
		t.Fatalf("redacted activity headers = %#v", record.Headers)
	}
}

func TestAPIV1CompatibilityRendersDjangoThrottleResponse(t *testing.T) {
	activity := &apiV1ActivityLoggerStub{}
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "activity-secret", bodySizeLimit: defaultFileSizeLimit})(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		apikey.EffectsFromContext(request.Context()).Rate = &apikey.Rate{Allowed: false, Wait: 1500 * time.Millisecond}
		apikey.EffectsFromContext(request.Context()).Authenticated = true
		response.Header().Set("Content-Type", "application/vnd.api+json")
		response.Header().Set("Content-Length", "999")
		response.Header().Set("Vary", "Cookie")
		response.WriteHeader(http.StatusTooManyRequests)
		_, _ = response.Write([]byte(`{"detail":"Request was throttled."}`))
	}))

	request := httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/workspaces/acme/modules/", nil)
	request.Header.Set("X-Api-Key", "plane-secret-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests || response.Body.String() != `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}` {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") != "2" || response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Content-Length") != strconv.Itoa(response.Body.Len()) || response.Header().Get("Vary") != "Cookie, origin" {
		t.Fatalf("throttle headers = %v", response.Header())
	}
	if len(activity.records) != 1 || activity.records[0].ResponseBody != `{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}` {
		t.Fatalf("throttle activity = %+v", activity.records)
	}
}

func TestAPIV1CompatibilityOmitsUnavailableRetryAfter(t *testing.T) {
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})(
		http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			apikey.EffectsFromContext(request.Context()).Rate = &apikey.Rate{Allowed: false}
			response.Header().Set("Retry-After", "stale")
			response.WriteHeader(http.StatusTooManyRequests)
		}),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/workspaces/acme/modules/", nil))
	if response.Header().Get("Retry-After") != "" {
		t.Fatalf("Retry-After = %q", response.Header().Get("Retry-After"))
	}
}

func TestAPIV1CompatibilitySuppressesHEADBodyButAuditsRenderedBody(t *testing.T) {
	activity := &apiV1ActivityLoggerStub{}
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit})(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = response.Write([]byte(`{"rendered":true}`))
		}),
	)
	request := httptest.NewRequest(http.MethodHead, "http://api.example.com/api/v1/workspaces/acme/modules/", nil)
	request.Header.Set("X-Api-Key", "token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Body.Len() != 0 {
		t.Fatalf("HEAD body = %q", response.Body.String())
	}
	if response.Header().Get("Content-Length") != strconv.Itoa(len(`{"rendered":true}`)) {
		t.Fatalf("HEAD Content-Length = %q", response.Header().Get("Content-Length"))
	}
	if len(activity.records) != 1 || activity.records[0].ResponseBody != `{"rendered":true}` {
		t.Fatalf("HEAD activity = %+v", activity.records)
	}
}

type apiV1FailingRequestBody struct{}

func (apiV1FailingRequestBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (apiV1FailingRequestBody) Close() error             { return nil }

func TestAPIV1CompatibilityHandlesUnreadableRequestBody(t *testing.T) {
	activity := &apiV1ActivityLoggerStub{}
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{activityLogger: activity, activitySecret: "secret", bodySizeLimit: defaultFileSizeLimit})(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unreadable body reached endpoint") }),
	)
	request := httptest.NewRequest(http.MethodPost, "http://api.example.com/api/v1/workspaces/acme/modules/", nil)
	request.Body = apiV1FailingRequestBody{}
	request.Header.Set("X-Api-Key", "token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Content-Length") != strconv.Itoa(len(djangoServerErrorBody)) || response.Header().Get("Vary") != "origin" || response.Body.String() != string(djangoServerErrorBody) {
		t.Fatalf("response headers/body = %v %q", response.Header(), response.Body.String())
	}
	if len(activity.records) != 0 {
		t.Fatalf("activity = %+v", activity.records)
	}
}

func TestAPIV1CompatibilityRejectsDeclaredOversizedBodyBeforeEndpoint(t *testing.T) {
	activity := &apiV1ActivityLoggerStub{}
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{
		activityLogger: activity,
		activitySecret: "secret",
		bodySizeLimit:  4,
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("oversized body reached endpoint") }))
	request := httptest.NewRequest(http.MethodPost, "http://api.example.com/api/v1/workspaces/acme/modules/", strings.NewReader("12345"))
	request.Header.Set("X-Api-Key", "token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Content-Length") != strconv.Itoa(len(djangoRequestTooLargeBody)) || response.Header().Get("Vary") != "origin" {
		t.Fatalf("headers = %v", response.Header())
	}
	if len(activity.records) != 0 {
		t.Fatalf("oversized request activity = %+v", activity.records)
	}
}

func TestParseConfigReadsCeleryBrokerURL(t *testing.T) {
	t.Setenv("AMQP_URL", "amqp://plane:plane@mq:5672/plane")
	t.Setenv("SECRET_KEY", "secret")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.amqpURL != "amqp://plane:plane@mq:5672/plane" {
		t.Fatalf("AMQP URL = %q", cfg.amqpURL)
	}
}

func TestParseConfigBuildsDjangoRabbitMQBrokerURL(t *testing.T) {
	t.Setenv("AMQP_URL", "")
	t.Setenv("RABBITMQ_USER", "plane-user")
	t.Setenv("RABBITMQ_PASSWORD", "plane-password")
	t.Setenv("RABBITMQ_HOST", "plane-mq")
	t.Setenv("RABBITMQ_PORT", "5673")
	t.Setenv("RABBITMQ_VHOST", "plane-vhost")
	t.Setenv("SECRET_KEY", "secret")
	cfg, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.amqpURL != "amqp://plane-user:plane-password@plane-mq:5673/plane-vhost" {
		t.Fatalf("AMQP URL = %q", cfg.amqpURL)
	}
}

func TestPublicRequestEndpointRejectsUntrustedForwardedScheme(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/workspaces/acme/modules/", nil)
	request.Header.Set("X-Forwarded-Proto", "javascript")
	if endpoint := publicRequestEndpoint(request); endpoint != "http://api.example.com" {
		t.Fatalf("public request endpoint = %q", endpoint)
	}
}
