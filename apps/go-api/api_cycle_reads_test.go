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

	cycleserver "github.com/CaliLuke/go-plane/apicyclereads/gen/http/api_cycle_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"

	api "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type apiCycleRepositoryStub struct {
	err     error
	reads   int
	request store.APIV1Request
	body    any
}

func (r *apiCycleRepositoryStub) AuthorizeAPIV1(context.Context, string, string, string, bool) error {
	return r.err
}
func (r *apiCycleRepositoryStub) ReadAPIV1(_ context.Context, _ string, request store.APIV1Request) (any, error) {
	r.reads++
	r.request = request
	return r.body, nil
}

func TestAPICyclePermissionPrecedesThrottleAndTimezone(t *testing.T) {
	repository := &apiCycleRepositoryStub{err: store.ErrForbidden}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "not-a-timezone"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	value, err := NewAPICycleReads(repository, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiCycleReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "Bearer literal", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDCycles(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesPayload{Slug: "alpha", ProjectID: uuid.NewString()})
	var denied *api.APICycleReadsErrorResponseBody
	if !errors.As(err, &denied) || denied.Name != "status_403" || throttle.calls != 0 || repository.reads != 0 || keys.key != "Bearer literal" {
		t.Fatalf("permission precedence err=%v throttle=%d reads=%d", err, throttle.calls, repository.reads)
	}
}

func TestAPICycleDoesNotAcceptAnotherAdapterPrincipal(t *testing.T) {
	repository := &apiCycleRepositoryStub{}
	value, err := NewAPICycleReads(repository, &activityAPIKeyStub{}, &activityThrottleStub{})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiCycleReadsService)
	ctx := context.WithValue(context.Background(), apiLabelPrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}, "label-key"})
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDCycles(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesPayload{})
	var unauthorized *api.APICycleReadsErrorResponseBody
	if !errors.As(err, &unauthorized) || unauthorized.Name != "status_401" || repository.reads != 0 {
		t.Fatalf("error=%v reads=%d", err, repository.reads)
	}
}

func TestAPICyclePreservesPathIdentityAndOpaqueJSONTimestamps(t *testing.T) {
	repository := &apiCycleRepositoryStub{body: map[string]any{"end_date": "2020-01-01T00:00:00Z", "progress_snapshot": map[string]any{"end_date": "2020-01-01T00:00:00Z"}}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Asia/Kolkata"}}
	value, err := NewAPICycleReads(repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiCycleReadsService)
	effects := &apikey.Effects{}
	ctx, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "key", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"cycle_id": {"attacker"}, "per_page": {"١_٠"}})
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesPayload{CycleID: "real-cycle", Slug: "alpha", ProjectID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if repository.request.Params["cycle_id"] != "real-cycle" || repository.request.Params["per_page"] != "١_٠" || !strings.Contains(string(data), `"end_date":"2020-01-01T05:30:00+05:30"`) || !strings.Contains(string(data), `"progress_snapshot":{"end_date":"2020-01-01T00:00:00Z"}`) {
		t.Fatalf("request=%#v body=%s", repository.request, data)
	}
	if !effects.Authenticated || effects.Rate == nil || !effects.Rate.Allowed {
		t.Fatalf("request effects=%+v", effects)
	}
}

func TestAPICycleGeneratedHTTPAcceptsCanonicalStatus(t *testing.T) {
	for _, route := range []string{"cycles-lite", "cycles"} {
		t.Run(route, func(t *testing.T) {
			repository := &apiCycleRepositoryStub{body: map[string]any{"results": []any{}}}
			service, err := NewAPICycleReads(repository, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
			if err != nil {
				t.Fatal(err)
			}
			mux := loomhttp.NewMuxer()
			cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:8000"}}})
			if err != nil {
				t.Fatal(err)
			}
			server := cycleserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
			server.Mount(mux)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/alpha/projects/30000000-0000-4000-8000-000000000001/"+route+"/?status=current", nil)
			request.Header.Set("X-Api-Key", "test-key")
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code != 200 || repository.request.Params["status"] != "current" || !repository.request.Present["status"] {
				t.Fatalf("status=%d body=%s request=%+v", recorder.Code, recorder.Body, repository.request)
			}
		})
	}
}
