package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type instanceAdminSessionRepoStub struct {
	admin      bool
	membership bool
	user       json.RawMessage
	err        error
}

func (r instanceAdminSessionRepoStub) IsAdmin(context.Context, uuid.UUID) (bool, error) {
	return r.admin, r.err
}
func (r instanceAdminSessionRepoStub) HasAdminMembership(context.Context, uuid.UUID) (bool, error) {
	return r.membership, r.err
}
func (r instanceAdminSessionRepoStub) User(context.Context, uuid.UUID) (json.RawMessage, error) {
	return r.user, r.err
}

type instanceAdminSessionStub struct {
	principal djangosession.Principal
	err       error
}

func (s instanceAdminSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return s.principal, s.err
}

func TestInstanceAdminSessionMe(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	raw := json.RawMessage(`{"id":"00000000-0000-0000-0000-000000000001","avatar":"","avatar_url":null,"cover_image":null,"date_joined":"2026-01-01T00:00:00Z","display_name":"Admin","email":"a@example.com","first_name":"A","last_name":"D","is_active":true,"is_bot":false,"is_email_verified":false,"user_timezone":"UTC","username":"admin","is_password_autoset":false}`)
	service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{admin: true, membership: true, user: raw}, instanceAdminSessionStub{principal: djangosession.Principal{UserID: id.String()}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "cookie", nil)
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.Me(ctx, &api.MePayload{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := uuid.Parse(user.ID)
	if err != nil || parsed != id || !user.AvatarURL.IsNull() {
		t.Fatalf("user=%+v err=%v", user, err)
	}
	service, err = NewInstanceAdminSession(instanceAdminSessionRepoStub{admin: false, membership: false, user: raw}, instanceAdminSessionStub{principal: djangosession.Principal{UserID: id.String()}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = service.(api.Authorizer).APIKeyAuth(context.Background(), "cookie", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Me(ctx, &api.MePayload{}); err == nil || err.Error() != instanceAdminForbiddenDetail {
		t.Fatalf("want forbidden, got %v", err)
	}
	service, err = NewInstanceAdminSession(instanceAdminSessionRepoStub{admin: true, membership: true, user: raw}, instanceAdminSessionStub{err: errors.New("bad")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.(api.Authorizer).APIKeyAuth(context.Background(), "cookie", nil); err == nil || err.Error() != instanceAdminUnauthenticatedDetail {
		t.Fatalf("want unauthenticated, got %v", err)
	}
}

func TestInstanceAdminSessionAllowAny(t *testing.T) {
	service, err := NewInstanceAdminSession(instanceAdminSessionRepoStub{admin: false, membership: false}, instanceAdminSessionStub{err: errors.New("bad")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Session(context.Background(), &api.SessionPayload{SessionID: strptr("cookie")})
	if err != nil || result.IsAuthenticated {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func strptr(v string) *string { return &v }
