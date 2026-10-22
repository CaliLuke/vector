package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacethemes"
	loom "github.com/CaliLuke/loom/pkg"
)

const themeUser = "00000000-0000-4000-8000-000000000010"
const themeID = "00000000-0000-4000-8000-000000000011"
const themeBody = `{"id":"` + themeID + `","workspace":"00000000-0000-4000-8000-000000000012","actor":"` + themeUser + `","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","deleted_at":null,"name":"Dark","colors":{"primary":"#111111"}}`

type themeRepoStub struct {
	authorized bool
	err        error
	name       *string
	colors     json.RawMessage
	deleted    bool
}

func (r *themeRepoStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}
func (r *themeRepoStub) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return json.RawMessage("[" + themeBody + "]"), r.err
}
func (r *themeRepoStub) Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(themeBody), r.err
}
func (r *themeRepoStub) Create(context.Context, uuid.UUID, string, string, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(themeBody), r.err
}
func (r *themeRepoStub) Update(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, name *string, colors json.RawMessage) (json.RawMessage, error) {
	r.name, r.colors = name, colors
	return json.RawMessage(themeBody), r.err
}
func (r *themeRepoStub) Delete(context.Context, uuid.UUID, string, uuid.UUID) error {
	r.deleted = true
	return r.err
}

type themeSessionStub struct{ err error }

func (s themeSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: themeUser}, s.err
}

func themeService(t *testing.T, repo *themeRepoStub) (api.Service, context.Context) {
	t.Helper()
	svc, err := NewWorkspaceThemes(repo, themeSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := svc.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, ctx
}

func TestWorkspaceThemesAuthCRUDAndPatch(t *testing.T) {
	repo := &themeRepoStub{authorized: true}
	svc, ctx := themeService(t, repo)
	if got, err := svc.List(ctx, &api.ListPayload{Slug: "demo"}); err != nil || len(got) != 1 {
		t.Fatalf("list=%v err=%v", got, err)
	}
	colors := loom.JSONValue(`{"primary":"#222222"}`)
	name := "Light"
	if _, err := svc.Create(ctx, &api.CreatePayload{Slug: "demo", Name: &name, Colors: colors}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: themeID, Colors: loom.NullableValue(colors)}); err != nil || repo.colors == nil {
		t.Fatalf("patch err=%v", err)
	}
	if err := svc.Delete(ctx, &api.DeletePayload{Slug: "demo", Pk: themeID}); err != nil || !repo.deleted {
		t.Fatalf("delete err=%v", err)
	}
}

func TestWorkspaceThemesForbiddenAndValidation(t *testing.T) {
	repo := &themeRepoStub{}
	svc, ctx := themeService(t, repo)
	if _, err := svc.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden")
	}
	repo.authorized = true
	if _, err := svc.Patch(ctx, &api.PatchPayload{Slug: "demo", Pk: themeID, Name: loom.NullValue[string]()}); err == nil {
		t.Fatal("expected null validation")
	}
	repo.err = store.ErrNotFound
	if _, err := svc.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Pk: themeID}); err == nil {
		t.Fatal("expected not found")
	}
	bad := themeSessionStub{err: errors.New("invalid")}
	badSvc, _ := NewWorkspaceThemes(repo, bad)
	if _, err := badSvc.(api.Authorizer).APIKeyAuth(context.Background(), "x", nil); err == nil {
		t.Fatal("expected auth error")
	}
}
