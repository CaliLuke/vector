package systemapi

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/modulereads"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type apiModuleRepositoryStub struct {
	err     error
	reads   int
	request store.APIV1Request
	body    any
}

func (r *apiModuleRepositoryStub) AuthorizeAPIV1(context.Context, string, string, string, bool) error {
	return r.err
}
func (r *apiModuleRepositoryStub) ReadAPIV1(_ context.Context, _ string, request store.APIV1Request) (any, error) {
	r.reads++
	r.request = request
	return r.body, nil
}

func TestAPIModulePermissionPrecedesThrottleAndTimezone(t *testing.T) {
	repository := &apiModuleRepositoryStub{err: store.ErrForbidden}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "not-a-timezone"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	value, err := NewAPIModuleReads(repository, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiModuleReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "Bearer literal", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDModules(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesPayload{Slug: "alpha", ProjectID: uuid.NewString()})
	var denied *api.APIModuleReadsErrorResponseBody
	if !errors.As(err, &denied) || denied.Name != "status_403" || throttle.calls != 0 || repository.reads != 0 || keys.key != "Bearer literal" {
		t.Fatalf("permission precedence err=%v throttle=%d reads=%d", err, throttle.calls, repository.reads)
	}
}

func TestAPIModulePreservesPathIdentityAndOpaqueJSONTimestamps(t *testing.T) {
	repository := &apiModuleRepositoryStub{body: map[string]any{"target_date": "2020-01-01", "description_text": map[string]any{"created_at": "2020-01-01T00:00:00Z"}, "description_html": map[string]any{"updated_at": "2020-01-01T00:00:00Z"}}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Asia/Kolkata"}}
	value, err := NewAPIModuleReads(repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiModuleReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"module_id": {"attacker"}, "per_page": {"١_٠"}})
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesPayload{ModuleID: "real-module", Slug: "alpha", ProjectID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"description_html":{"updated_at":"2020-01-01T00:00:00Z"}`) || repository.request.Params["module_id"] != "real-module" || repository.request.Params["per_page"] != "١_٠" || !strings.Contains(string(data), `"target_date":"2020-01-01"`) || !strings.Contains(string(data), `"description_text":{"created_at":"2020-01-01T00:00:00Z"}`) {
		t.Fatalf("request=%#v body=%s", repository.request, data)
	}
}
