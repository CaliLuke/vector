package systemapi

import (
	"context"
	"errors"
	"testing"
	"uuid"

	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type instanceWorkspaceSlugRepositoryStub struct {
	exists, admin            bool
	err, adminErr, existsErr error
}

func (stub *instanceWorkspaceSlugRepositoryStub) IsInstanceAdmin(context.Context, uuid.UUID) (bool, error) {
	return stub.admin, firstError(stub.adminErr, stub.err)
}

func (stub *instanceWorkspaceSlugRepositoryStub) Exists(context.Context, string) (bool, error) {
	return stub.exists, firstError(stub.existsErr, stub.err)
}

func firstError(primary, fallback error) error {
	if primary != nil {
		return primary
	}
	return fallback
}

type instanceWorkspaceSlugSessionStub struct {
	principal djangosession.Principal
	err       error
}

func (stub instanceWorkspaceSlugSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return stub.principal, stub.err
}

func TestInstanceWorkspaceSlugCheck(t *testing.T) {
	for _, test := range []struct {
		name       string
		slug       *string
		exists     bool
		wantStatus bool
		wantError  string
	}{
		{name: "missing", wantError: "Workspace Slug is required"},
		{name: "empty", slug: stringPointer(""), wantError: "Workspace Slug is required"},
		{name: "existing", slug: stringPointer("taken"), exists: true},
		{name: "restricted", slug: stringPointer("admin")},
		{name: "restricted case variant", slug: stringPointer("ADMIN"), wantStatus: true},
		{name: "available", slug: stringPointer("new-workspace"), wantStatus: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{exists: test.exists}, instanceWorkspaceSlugSessionStub{})
			if err != nil {
				t.Fatal(err)
			}
			response, err := service.Check(context.Background(), &instanceworkspaceapi.CheckPayload{Slug: test.slug})
			if test.wantError != "" {
				var typed *instanceworkspaceapi.InstanceWorkspaceSlugError
				if !errors.As(err, &typed) || typed.Message != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || response == nil || response.Status != test.wantStatus {
				t.Fatalf("response = %#v, error = %v", response, err)
			}
		})
	}
}

func TestInstanceWorkspaceSlugAuthentication(t *testing.T) {
	tests := []struct {
		name      string
		session   instanceWorkspaceSlugSessionStub
		admin     bool
		wantError string
	}{
		{name: "anonymous", session: instanceWorkspaceSlugSessionStub{err: errors.New("invalid")}, wantError: "unauthenticated"},
		{name: "ordinary user", session: instanceWorkspaceSlugSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}}, wantError: "forbidden"},
		{name: "admin", session: instanceWorkspaceSlugSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}}, admin: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{admin: test.admin}, test.session)
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.(instanceworkspaceapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err == nil || !containsLoomError(err, test.wantError) {
				t.Fatalf("error = %v, want %s", err, test.wantError)
			}
		})
	}
}

func TestInstanceWorkspaceSlugRepositoryFailure(t *testing.T) {
	service, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{adminErr: errors.New("database unavailable")}, instanceWorkspaceSlugSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(instanceworkspaceapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	var internal *instanceworkspaceapi.InstanceWorkspaceSlugInternalError
	if !errors.As(err, &internal) || internal.Message != "Something went wrong please try again later" {
		t.Fatalf("error = %v, want Django-compatible internal error", err)
	}
}

func TestInstanceWorkspaceSlugCheckRepositoryFailure(t *testing.T) {
	service, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{existsErr: errors.New("database unavailable"), admin: true}, instanceWorkspaceSlugSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	value := "available"
	_, err = service.Check(context.Background(), &instanceworkspaceapi.CheckPayload{Slug: &value})
	var internal *instanceworkspaceapi.InstanceWorkspaceSlugInternalError
	if !errors.As(err, &internal) || internal.Message != "Something went wrong please try again later" {
		t.Fatalf("error = %v, want Django-compatible internal error", err)
	}
}

func TestInstanceWorkspaceSlugCheckInvalidPrincipal(t *testing.T) {
	service, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{admin: true}, instanceWorkspaceSlugSessionStub{principal: djangosession.Principal{UserID: "not-a-uuid"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(instanceworkspaceapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if !containsLoomError(err, "unauthenticated") {
		t.Fatalf("error = %v, want unauthenticated", err)
	}
}

func TestInstanceWorkspaceSlugCheckConstructorValidation(t *testing.T) {
	if _, err := NewInstanceWorkspaceSlugCheck(nil, instanceWorkspaceSlugSessionStub{}); err == nil {
		t.Fatal("nil repository accepted")
	}
	if _, err := NewInstanceWorkspaceSlugCheck(&instanceWorkspaceSlugRepositoryStub{}, nil); err == nil {
		t.Fatal("nil session validator accepted")
	}
}

func containsLoomError(err error, name string) bool {
	var named interface{ LoomErrorName() string }
	return errors.As(err, &named) && named.LoomErrorName() == name
}
