package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_settings"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type userSettingsAdapterSessionStub struct{ err error }

func (s userSettingsAdapterSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000011"}, s.err
}

type userSettingsAdapterRepositoryStub struct {
	body json.RawMessage
	err  error
}

func (r *userSettingsAdapterRepositoryStub) Get(context.Context, uuid.UUID) (json.RawMessage, error) {
	return r.body, r.err
}

func TestUserSettingsGetDecodesFallbackWorkspace(t *testing.T) {
	repository := &userSettingsAdapterRepositoryStub{body: json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000011","email":"user@example.com","workspace":{"last_workspace_id":null,"last_workspace_slug":null,"fallback_workspace_id":"00000000-0000-4000-8000-000000000012","fallback_workspace_slug":"demo","invites":2}}`)}
	service, err := NewUserSettings(repository, userSettingsAdapterSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Get(ctx, &api.GetPayload{})
	if err != nil || result.Email != "user@example.com" || result.Workspace.Invites != 2 {
		t.Fatalf("Get() = %#v, %v", result, err)
	}
}

func TestUserSettingsMapsAuthenticationAndRepositoryFailures(t *testing.T) {
	service, err := NewUserSettings(&userSettingsAdapterRepositoryStub{}, userSettingsAdapterSessionStub{err: errors.New("expired")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "expired", nil); err == nil {
		t.Fatal("APIKeyAuth() returned nil error for expired session")
	}

	service, err = NewUserSettings(&userSettingsAdapterRepositoryStub{err: errors.New("database unavailable")}, userSettingsAdapterSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(ctx, &api.GetPayload{}); err == nil {
		t.Fatal("Get() returned nil error for repository failure")
	} else {
		var internal *api.UserSettingsInternal
		if !errors.As(err, &internal) {
			t.Fatalf("Get() error = %T %v, want internal", err, err)
		}
	}
}
