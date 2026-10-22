package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_properties"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loom "github.com/CaliLuke/loom/pkg"
)

type userPropertiesRepositoryStub struct {
	authorized        bool
	projectAuthorized bool
	err               error
	body              json.RawMessage
	patched           map[string]json.RawMessage
}

func (r *userPropertiesRepositoryStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}
func (r *userPropertiesRepositoryStub) AuthorizedProject(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return r.projectAuthorized, r.err
}
func (r *userPropertiesRepositoryStub) GetWorkspace(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) PatchWorkspace(context.Context, uuid.UUID, string, map[string]json.RawMessage) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) GetProject(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) PatchProject(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, values map[string]json.RawMessage) (json.RawMessage, error) {
	r.patched = values
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) GetCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) PatchCycle(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) GetModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *userPropertiesRepositoryStub) PatchModule(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error) {
	return r.body, r.err
}

type userPropertiesSessionStub struct{ err error }

func (s userPropertiesSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if s.err != nil {
		return djangosession.Principal{}, s.err
	}
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001"}, nil
}

func userPropertiesJSON() json.RawMessage {
	return json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000010","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z","workspace":"00000000-0000-4000-8000-000000000011","user":"00000000-0000-4000-8000-000000000001","filters":{},"display_filters":{},"display_properties":{},"rich_filters":{},"sort_order":7.5}`)
}

func TestUserPropertiesAPIKeyAuthMapsInvalidSession(t *testing.T) {
	repo := &userPropertiesRepositoryStub{}
	service, err := NewUserProperties(repo, userPropertiesSessionStub{err: errors.New("expired")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(api.Authorizer).APIKeyAuth(context.Background(), "expired", nil)
	var unauth *api.UserPropertiesUnauthenticated
	if !errors.As(err, &unauth) || unauth.Detail != unauthenticatedDetail {
		t.Fatalf("auth error=%T %v", err, err)
	}
}

func TestUserPropertiesProjectPatchPassesAllWritableFields(t *testing.T) {
	repo := &userPropertiesRepositoryStub{projectAuthorized: true, body: userPropertiesJSON()}
	service, err := NewUserProperties(repo, userPropertiesSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ProjectPatch(ctx, &api.ProjectPatchPayload{
		Slug:        "demo",
		ProjectID:   "00000000-0000-4000-8000-000000000002",
		Preferences: loom.NullableValue(loom.JSONValue(`{"pages":{"block_display":false}}`)),
		SortOrder:   loom.NullableValue(7.5),
	})
	if err != nil || result == nil {
		t.Fatalf("patch result=%v err=%v", result, err)
	}
	if string(repo.patched["sort_order"]) != "7.5" || string(repo.patched["preferences"]) != `{"pages":{"block_display":false}}` {
		t.Fatalf("patched fields=%v", repo.patched)
	}
}

func TestUserPropertiesRejectsNonProjectMember(t *testing.T) {
	repo := &userPropertiesRepositoryStub{projectAuthorized: false}
	service, err := NewUserProperties(repo, userPropertiesSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ProjectGet(ctx, &api.ProjectGetPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002"})
	var forbidden *api.UserPropertiesForbidden
	if !errors.As(err, &forbidden) || forbidden.Message != forbiddenDetail {
		t.Fatalf("forbidden error=%T %v", err, err)
	}
}

func (r *userPropertiesRepositoryStub) CheckScopedPatch(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, string) error {
	return r.err
}
