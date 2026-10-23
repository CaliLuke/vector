package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacefavorites"
	loom "github.com/CaliLuke/loom/pkg"
)

const favoriteTestUser = "00000000-0000-4000-8000-000000000010"
const favoriteTestID = "00000000-0000-4000-8000-000000000011"
const favoriteTestBody = `{"id":"` + favoriteTestID + `","entity_type":"folder","entity_identifier":null,"entity_data":null,"name":"Projects","is_folder":true,"sequence":65535,"parent":null,"workspace_id":"00000000-0000-4000-8000-000000000012","project_id":null}`

type favoriteRepoStub struct {
	authorized bool
	err        error
	created    store.CreateInput
	updated    store.UpdateInput
	deleted    bool
}

func (r *favoriteRepoStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}
func (r *favoriteRepoStub) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return json.RawMessage("[" + favoriteTestBody + "]"), r.err
}
func (r *favoriteRepoStub) Group(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[" + favoriteTestBody + "]"), r.err
}
func (r *favoriteRepoStub) Create(_ context.Context, _ uuid.UUID, _ string, input store.CreateInput) (json.RawMessage, error) {
	r.created = input
	return json.RawMessage(favoriteTestBody), r.err
}
func (r *favoriteRepoStub) Update(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, input store.UpdateInput) (json.RawMessage, error) {
	r.updated = input
	return json.RawMessage(favoriteTestBody), r.err
}
func (r *favoriteRepoStub) Delete(context.Context, uuid.UUID, string, uuid.UUID) error {
	r.deleted = true
	return r.err
}

type favoriteSessionStub struct{ err error }

func (s favoriteSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: favoriteTestUser}, s.err
}

func favoriteService(t *testing.T, repository *favoriteRepoStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewWorkspaceFavorites(repository, favoriteSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestWorkspaceFavoritesCRUDAndGroup(t *testing.T) {
	repository := &favoriteRepoStub{authorized: true}
	service, ctx := favoriteService(t, repository)
	name := "Projects"
	created, err := service.Create(ctx, &api.CreatePayload{
		Slug:       "demo",
		EntityType: loom.NullableValue("folder"),
		Name:       loom.NullableValue(name),
		IsFolder:   loom.NullableValue(true),
	})
	if err != nil || created.EntityType != "folder" || repository.created.Name == nil || *repository.created.Name != name {
		t.Fatalf("create=%+v input=%+v err=%v", created, repository.created, err)
	}
	newName := "Renamed"
	if _, err = service.Patch(ctx, &api.PatchPayload{Slug: "demo", FavoriteID: favoriteTestID, Name: loom.NullableValue(newName)}); err != nil {
		t.Fatal(err)
	}
	if !repository.updated.NamePresent || repository.updated.Name == nil || *repository.updated.Name != newName {
		t.Fatalf("update input=%+v", repository.updated)
	}
	if values, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err != nil || len(values) != 1 {
		t.Fatalf("list=%d err=%v", len(values), err)
	}
	if values, err := service.Group(ctx, &api.GroupPayload{Slug: "demo", FavoriteID: favoriteTestID}); err != nil || len(values) != 1 {
		t.Fatalf("group=%d err=%v", len(values), err)
	}
	if err = service.Delete(ctx, &api.DeletePayload{Slug: "demo", FavoriteID: favoriteTestID}); err != nil || !repository.deleted {
		t.Fatalf("delete=%v called=%t", err, repository.deleted)
	}
}

func TestWorkspaceFavoritesPreservesValidationAndPhantomMethods(t *testing.T) {
	repository := &favoriteRepoStub{authorized: true}
	service, ctx := favoriteService(t, repository)
	if _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo"}); err == nil {
		t.Fatal("expected required entity_type validation")
	} else if invalid, ok := err.(*api.WorkspaceFavoriteInvalid); !ok || len(invalid.EntityType) != 1 {
		t.Fatalf("validation=%T %+v", err, err)
	}
	if _, err := service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", FavoriteID: favoriteTestID}); err == nil {
		t.Fatal("expected phantom retrieve error")
	}
	if _, err := service.PatchCollection(ctx, &api.PatchCollectionPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected phantom collection patch error")
	}
	if err := service.DeleteCollection(ctx, &api.DeleteCollectionPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected phantom collection delete error")
	}
	if _, err := service.CreateDetail(ctx, &api.CreateDetailPayload{Slug: "demo", FavoriteID: favoriteTestID}); err == nil {
		t.Fatal("expected phantom detail create error")
	}
}

func TestWorkspaceFavoritesAuthorizationAndNotFound(t *testing.T) {
	repository := &favoriteRepoStub{}
	service, ctx := favoriteService(t, repository)
	if _, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden")
	} else if forbidden, ok := err.(*api.WorkspaceFavoriteForbidden); !ok || forbidden.Message != "You don't have the required permissions." {
		t.Fatalf("forbidden=%T %+v", err, err)
	}
	repository.authorized = true
	repository.err = store.ErrNotFound
	if err := service.Delete(ctx, &api.DeletePayload{Slug: "demo", FavoriteID: favoriteTestID}); err == nil {
		t.Fatal("expected not found")
	}
	invalidSession, _ := NewWorkspaceFavorites(repository, favoriteSessionStub{err: errors.New("invalid")})
	if _, err := invalidSession.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); err == nil {
		t.Fatal("expected authentication error")
	}
}
