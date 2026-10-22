package systemapi

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type apiStateRepositoryStub struct {
	authorizeErr error
	readErr      error
	reads        int
	request      store.APIV1Request
	body         any
}

func (r *apiStateRepositoryStub) AuthorizeAPIV1(context.Context, string, string, string, bool) error {
	return r.authorizeErr
}
func (r *apiStateRepositoryStub) ReadAPIV1(_ context.Context, _ string, request store.APIV1Request) (any, error) {
	r.reads++
	r.request = request
	return r.body, r.readErr
}

func TestAPIStatePermissionPrecedesThrottleAndTimezone(t *testing.T) {
	repository := &apiStateRepositoryStub{authorizeErr: store.ErrForbidden}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "not-a-timezone"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	value, err := NewAPIStateReads(repository, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiStateReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDStates(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesPayload{Slug: "alpha", ProjectID: uuid.NewString()})
	var denied *api.APIStateReadsErrorResponseBody
	if !errors.As(err, &denied) || denied.Name != "status_403" || throttle.calls != 0 || repository.reads != 0 {
		t.Fatalf("permission precedence err=%v throttle=%d reads=%d", err, throttle.calls, repository.reads)
	}
}

func TestAPIStateDetailPreservesPathIdentityAndRawQuery(t *testing.T) {
	repository := &apiStateRepositoryStub{body: map[string]any{"created_at": "2020-01-01T00:00:00Z", "name": "Backlog"}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "Asia/Kolkata"}}
	value, err := NewAPIStateReads(repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiStateReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"state_id": {"attacker"}, "fields": {"name"}})
	_, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDPayload{Slug: "alpha", ProjectID: uuid.NewString(), StateID: "path-state"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if repository.request.Operation != "detail" || repository.request.Params["state_id"] != "path-state" || repository.request.Params["fields"] != "name" || !strings.Contains(string(data), `"name":"Backlog"`) || !strings.Contains(string(data), `"created_at":"2020-01-01T05:30:00+05:30"`) {
		t.Fatalf("request=%#v body=%s", repository.request, data)
	}
}

func TestAPIStateMapsPaginationSourceError(t *testing.T) {
	repository := &apiStateRepositoryStub{readErr: &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
	value, err := NewAPIStateReads(repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiStateReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "token", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"cursor": {"1:9223372036854775808:0"}})
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDStates(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesPayload{Slug: "alpha", ProjectID: uuid.NewString()})
	var response *api.APIStateReadsErrorResponseBody
	if !errors.As(err, &response) || response.Name != "status_500" || repository.reads != 1 {
		t.Fatalf("err=%v reads=%d", err, repository.reads)
	}
}
