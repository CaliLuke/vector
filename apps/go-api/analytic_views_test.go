package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/analytic_views"
	store "github.com/CaliLuke/go-plane/internal/analyticviews"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loom "github.com/CaliLuke/loom/pkg"
	"uuid"
)

const analyticTestID = "00000000-0000-4000-8000-000000000011"

type analyticRepoTest struct {
	authorized bool
	authErr    error
	raw        json.RawMessage
	err        error
	query      json.RawMessage
	queryDict  json.RawMessage
	updated    json.RawMessage
	deleted    uuid.UUID
}

func (r *analyticRepoTest) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.authErr
}
func (r *analyticRepoTest) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return r.raw, r.err
}
func (r *analyticRepoTest) Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return r.raw, r.err
}
func (r *analyticRepoTest) Create(_ context.Context, _ uuid.UUID, _, _, _ string, query, queryDict json.RawMessage) (json.RawMessage, error) {
	r.query, r.queryDict = query, queryDict
	return r.raw, r.err
}
func (r *analyticRepoTest) Update(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ *string, _ *string, query json.RawMessage, _ bool) (json.RawMessage, error) {
	r.updated = query
	return r.raw, r.err
}
func (r *analyticRepoTest) Delete(_ context.Context, _ uuid.UUID, _ string, id uuid.UUID) error {
	r.deleted = id
	return r.err
}

type analyticSessionTest struct{ err error }

func (s analyticSessionTest) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001"}, s.err
}

func analyticRaw(name string) json.RawMessage {
	return json.RawMessage(`{"id":"` + analyticTestID + `","created_at":"2026-08-26T00:00:00Z","updated_at":"2026-08-26T00:00:00Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"deleted_at":null,"workspace":"00000000-0000-4000-8000-000000000002","name":"` + name + `","description":"Parity","query":{},"query_dict":{"x_axis":"state_id","y_axis":"issue_count"}}`)
}

func analyticService(t *testing.T, repository *analyticRepoTest) (api.Service, context.Context) {
	t.Helper()
	service, err := NewAnalyticViews(repository, analyticSessionTest{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestAnalyticViewsCRUDAndQueryCompatibility(t *testing.T) {
	repository := &analyticRepoTest{authorized: true, raw: analyticRaw("Issues by state")}
	service, ctx := analyticService(t, repository)
	queryDict := loom.JSONValue(`{"x_axis":"state_id","y_axis":"issue_count","state":["00000000-0000-4000-8000-000000000031"]}`)
	created, err := service.Create(ctx, &api.CreatePayload{Slug: "demo", Name: "Issues by state", QueryDict: queryDict})
	if err != nil || created.Name != "Issues by state" {
		t.Fatalf("create=%#v err=%v", created, err)
	}
	if strings.Contains(string(repository.query), "x_axis") || !strings.Contains(string(repository.query), "state__in") {
		t.Fatalf("derived query = %s", repository.query)
	}
	if !strings.Contains(string(repository.queryDict), "x_axis") {
		t.Fatalf("query_dict = %s", repository.queryDict)
	}

	repository.raw = json.RawMessage(`[` + string(analyticRaw("Issues by state")) + `]`)
	listed, err := service.List(ctx, &api.ListPayload{Slug: "demo"})
	if err != nil || len(listed) != 1 {
		t.Fatalf("list=%#v err=%v", listed, err)
	}
	repository.raw = analyticRaw("Updated")
	if _, err = service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: analyticTestID}); err != nil {
		t.Fatal(err)
	}
	name := "Updated"
	if _, err = service.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: analyticTestID, Name: &name, QueryData: queryDict}); err != nil {
		t.Fatal(err)
	}
	if string(repository.updated) != `{}` {
		t.Fatalf("PATCH query = %s, want Django-compatible empty object", repository.updated)
	}
	if err = service.Delete(ctx, &api.DeletePayload{Slug: "demo", Pk: analyticTestID}); err != nil || repository.deleted.String() != analyticTestID {
		t.Fatalf("delete id=%s err=%v", repository.deleted, err)
	}
}

func TestAnalyticViewsAuthenticationAuthorizationAndNotFound(t *testing.T) {
	if _, err := NewAnalyticViews(nil, analyticSessionTest{}); err == nil {
		t.Fatal("expected dependency error")
	}
	repository := &analyticRepoTest{}
	service, ctx := analyticService(t, repository)
	if _, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden error")
	}
	if _, err := service.List(context.Background(), &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected unauthenticated error")
	}
	badSession, _ := NewAnalyticViews(repository, analyticSessionTest{err: errors.New("invalid")})
	if _, err := badSession.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); err == nil {
		t.Fatal("expected session error")
	}

	repository.authorized, repository.err = true, store.ErrNotFound
	if _, err := service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: analyticTestID}); err == nil {
		t.Fatal("expected not-found error")
	}
	if err := service.Delete(ctx, &api.DeletePayload{Slug: "demo", Pk: analyticTestID}); err == nil {
		t.Fatal("expected delete not-found error")
	}
}
