package systemapi

import (
	"context"
	"errors"
	"testing"
	"uuid"

	userinstanceadminapi "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type userInstanceAdminRepositoryStub struct {
	isAdmin bool
	err     error
	userID  uuid.UUID
}

func (stub *userInstanceAdminRepositoryStub) IsInstanceAdmin(_ context.Context, userID uuid.UUID) (bool, error) {
	stub.userID = userID
	return stub.isAdmin, stub.err
}

type userInstanceAdminSessionStub struct {
	principal djangosession.Principal
	err       error
}

func (stub userInstanceAdminSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return stub.principal, stub.err
}

func TestUserInstanceAdminRetrieve(t *testing.T) {
	for _, test := range []struct {
		name    string
		isAdmin bool
	}{{"member", false}, {"admin", true}} {
		t.Run(test.name, func(t *testing.T) {
			repository := &userInstanceAdminRepositoryStub{isAdmin: test.isAdmin}
			service, err := NewUserInstanceAdmin(repository, userInstanceAdminSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := service.(userinstanceadminapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := service.Retrieve(ctx, &userinstanceadminapi.RetrievePayload{})
			if err != nil || response == nil || response.IsInstanceAdmin != test.isAdmin {
				t.Fatalf("response = %#v, error = %v", response, err)
			}
		})
	}
}

func TestUserInstanceAdminAuthenticationAndFailures(t *testing.T) {
	service, err := NewUserInstanceAdmin(&userInstanceAdminRepositoryStub{}, userInstanceAdminSessionStub{err: errors.New("invalid")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(userinstanceadminapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if !containsLoomError(err, "unauthenticated") {
		t.Fatalf("error = %v, want unauthenticated", err)
	}

	service, err = NewUserInstanceAdmin(&userInstanceAdminRepositoryStub{}, userInstanceAdminSessionStub{principal: djangosession.Principal{UserID: "not-a-uuid"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(userinstanceadminapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if !containsLoomError(err, "unauthenticated") {
		t.Fatalf("error = %v, want unauthenticated", err)
	}

	service, err = NewUserInstanceAdmin(&userInstanceAdminRepositoryStub{err: errors.New("boom")}, userInstanceAdminSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(userinstanceadminapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Retrieve(ctx, &userinstanceadminapi.RetrievePayload{})
	if !containsLoomError(err, "internal_error") {
		t.Fatalf("error = %v, want internal_error", err)
	}
}
