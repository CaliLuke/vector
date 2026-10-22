package systemapi

import (
	"context"
	"errors"
	"testing"

	api "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type apiLabelRepositoryStub struct {
	err     error
	reads   int
	request store.APIV1Request
	body    any
}

func (r *apiLabelRepositoryStub) AuthorizeAPIV1(context.Context, string, string, string, bool) error {
	return r.err
}
func (r *apiLabelRepositoryStub) ReadAPIV1(_ context.Context, _ string, request store.APIV1Request) (any, error) {
	r.reads++
	r.request = request
	return r.body, nil
}

func TestAPILabelPermissionPrecedesThrottleAndTimezone(t *testing.T) {
	repository := &apiLabelRepositoryStub{err: store.ErrForbidden}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "not-a-timezone"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	value, err := NewAPILabelReads(repository, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*apiLabelReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "Bearer literal", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels(ctx, &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsPayload{Slug: "alpha", ProjectID: uuid.NewString()})
	var denied *api.APILabelReadsErrorResponseBody
	if !errors.As(err, &denied) || denied.Name != "status_403" || throttle.calls != 0 || repository.reads != 0 || keys.key != "Bearer literal" {
		t.Fatalf("permission precedence err=%v throttle=%d reads=%d", err, throttle.calls, repository.reads)
	}
}
