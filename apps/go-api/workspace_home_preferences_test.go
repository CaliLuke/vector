package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacehomepreferences"
	loom "github.com/CaliLuke/loom/pkg"
)

type homeRepoTest struct {
	authorized bool
	authErr    error
	err        error
	raw        json.RawMessage
	newKey     *string
	enabled    *bool
	sortOrder  *float64
}

func (r *homeRepoTest) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.authErr
}
func (r *homeRepoTest) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return r.raw, r.err
}
func (r *homeRepoTest) Get(context.Context, uuid.UUID, string, string) (json.RawMessage, error) {
	return nil, r.err
}
func (r *homeRepoTest) Update(_ context.Context, _ uuid.UUID, _, _ string, newKey *string, enabled *bool, sortOrder *float64) (json.RawMessage, error) {
	r.newKey, r.enabled, r.sortOrder = newKey, enabled, sortOrder
	return r.raw, r.err
}

type homeSessionTest struct{ err error }

func (s homeSessionTest) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001"}, s.err
}

func homePreferenceService(t *testing.T, repository *homeRepoTest) (api.Service, context.Context) {
	t.Helper()
	service, err := NewWorkspaceHomePreferences(repository, homeSessionTest{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestWorkspaceHomePreferencesListAndPhantomOperations(t *testing.T) {
	repository := &homeRepoTest{
		authorized: true,
		raw:        json.RawMessage(`[{"key":"quick_links","is_enabled":true,"config":{},"sort_order":999}]`),
	}
	service, ctx := homePreferenceService(t, repository)
	preferences, err := service.List(ctx, &api.ListPayload{Slug: "demo"})
	if err != nil || len(preferences) != 1 || string(preferences[0].Config) != `{}` {
		t.Fatalf("list=%#v err=%v", preferences, err)
	}
	if _, err = service.PatchCollection(ctx, &api.PatchCollectionPayload{Slug: "demo"}); err == nil {
		t.Fatal("collection PATCH must preserve Django's internal error")
	} else {
		var internal *api.WorkspaceHomePreferenceInternal
		if !errors.As(err, &internal) {
			t.Fatalf("collection PATCH error = %T, want internal", err)
		}
	}
	if _, err = service.Retrieve(ctx, &api.RetrievePayload{Slug: "demo", Key: "quick_links"}); err == nil {
		t.Fatal("detail GET must preserve Django's internal error")
	}
}

func TestWorkspaceHomePreferencesPatchRenamesAndUpdates(t *testing.T) {
	repository := &homeRepoTest{
		authorized: true,
		raw:        json.RawMessage(`{"key":"renamed","is_enabled":false,"sort_order":7}`),
	}
	service, ctx := homePreferenceService(t, repository)
	result, err := service.Patch(ctx, &api.PatchPayload{
		Slug: "demo",
		Key:  "quick_links",
		Body: &api.WorkspaceHomePreferencePatch{
			Key:       loom.NullableValue("renamed"),
			IsEnabled: loom.NullableValue(false),
			SortOrder: loom.NullableValue(7.0),
		},
	})
	if err != nil || result.Key != "renamed" {
		t.Fatalf("patch=%#v err=%v", result, err)
	}
	if repository.newKey == nil || *repository.newKey != "renamed" || repository.enabled == nil || *repository.enabled || repository.sortOrder == nil || *repository.sortOrder != 7 {
		t.Fatalf("patch args key=%v enabled=%v sort=%v", repository.newKey, repository.enabled, repository.sortOrder)
	}
}

func TestWorkspaceHomePreferencesAuthForbiddenValidationAndNotFound(t *testing.T) {
	if _, err := NewWorkspaceHomePreferences(nil, homeSessionTest{}); err == nil {
		t.Fatal("expected nil repository error")
	}
	repository := &homeRepoTest{}
	service, ctx := homePreferenceService(t, repository)
	if _, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden error")
	}
	if _, err := service.List(context.Background(), &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected unauthenticated error")
	}
	badAuth, _ := NewWorkspaceHomePreferences(repository, homeSessionTest{err: errors.New("invalid session")})
	if _, err := badAuth.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); err == nil {
		t.Fatal("expected session error")
	}

	repository.authorized = true
	if _, err := service.Patch(ctx, &api.PatchPayload{
		Slug: "demo",
		Key:  "quick_links",
		Body: &api.WorkspaceHomePreferencePatch{Key: loom.NullableValue(" ")},
	}); err == nil {
		t.Fatal("expected blank-key validation error")
	}
	repository.err = store.ErrNotFound
	if _, err := service.Patch(ctx, &api.PatchPayload{Slug: "demo", Key: "missing"}); err == nil {
		t.Fatal("expected not-found error")
	} else {
		var notFound *api.WorkspaceHomePreferenceNotFound
		if !errors.As(err, &notFound) || notFound.Message != "Preference not found" {
			t.Fatalf("not-found error = %#v", err)
		}
	}
}
