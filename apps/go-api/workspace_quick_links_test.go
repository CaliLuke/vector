package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacequicklinks"
	pkg "github.com/CaliLuke/loom/pkg"
)

const quickUser = "00000000-0000-4000-8000-000000000010"
const quickID = "00000000-0000-4000-8000-000000000011"
const quickBody = `{"id":"` + quickID + `","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","deleted_at":null,"workspace":"00000000-0000-4000-8000-000000000012","owner":"` + quickUser + `","title":"Docs","url":"http://example.com","metadata":{}}`

type quickRepoStub struct {
	authorized bool
	err        error
	url        string
	deleted    bool
}

func (r *quickRepoStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}
func (r *quickRepoStub) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return json.RawMessage("[" + quickBody + "]"), r.err
}
func (r *quickRepoStub) Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(quickBody), r.err
}
func (r *quickRepoStub) Create(_ context.Context, _ uuid.UUID, _ string, url string, _ *string, _ json.RawMessage) (json.RawMessage, error) {
	r.url = url
	return json.RawMessage(quickBody), r.err
}
func (r *quickRepoStub) Update(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ bool, _ *string, url *string, _ bool, _ json.RawMessage) (json.RawMessage, error) {
	if url != nil {
		r.url = *url
	}
	return json.RawMessage(quickBody), r.err
}
func (r *quickRepoStub) Delete(context.Context, uuid.UUID, string, uuid.UUID) error {
	r.deleted = true
	return r.err
}

type quickSessionStub struct{ err error }

func (s quickSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: quickUser}, s.err
}
func quickService(t *testing.T, r *quickRepoStub) (api.Service, context.Context) {
	t.Helper()
	svc, e := NewWorkspaceQuickLinks(r, quickSessionStub{})
	if e != nil {
		t.Fatal(e)
	}
	ctx, e := svc.(api.Authorizer).APIKeyAuth(context.Background(), "s", nil)
	if e != nil {
		t.Fatal(e)
	}
	return svc, ctx
}

func TestWorkspaceQuickLinksCRUDNormalizesAndScopes(t *testing.T) {
	r := &quickRepoStub{authorized: true}
	svc, ctx := quickService(t, r)
	title := "Docs"
	if _, e := svc.Create(ctx, &api.CreatePayload{Slug: "demo", URL: "example.com/docs", Title: pkg.NullableValue(title)}); e != nil || r.url != "http://example.com/docs" {
		t.Fatalf("create url=%q err=%v", r.url, e)
	}
	url := "example.org/new"
	if _, e := svc.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: quickID, URL: &url}); e != nil || r.url != "http://example.org/new" {
		t.Fatalf("patch url=%q err=%v", r.url, e)
	}
	if e := svc.Delete(ctx, &api.DeletePayload{Slug: "demo", Pk: quickID}); e != nil || !r.deleted {
		t.Fatalf("delete=%v", e)
	}
}
func TestWorkspaceQuickLinksForbiddenValidationAndNotFound(t *testing.T) {
	r := &quickRepoStub{}
	svc, ctx := quickService(t, r)
	if _, e := svc.List(ctx, &api.ListPayload{Slug: "demo"}); e == nil {
		t.Fatal("expected forbidden")
	}
	r.authorized = true
	if _, e := svc.Create(ctx, &api.CreatePayload{Slug: "demo", URL: "not a url"}); e == nil {
		t.Fatal("expected URL validation")
	}
	r.err = store.ErrNotFound
	if _, e := svc.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: quickID}); e == nil {
		t.Fatal("expected not found")
	}
	bad, _ := NewWorkspaceQuickLinks(r, quickSessionStub{err: errors.New("bad")})
	if _, e := bad.(api.Authorizer).APIKeyAuth(context.Background(), "x", nil); e == nil {
		t.Fatal("expected auth error")
	}
}
