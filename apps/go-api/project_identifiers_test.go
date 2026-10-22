package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_identifiers"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectidentifiers"
)

const projectIdentifierAdapterUser = "00000000-0000-4000-8000-000000000010"

type projectIdentifierAdapterSessionStub struct{ err error }

func (s projectIdentifierAdapterSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: projectIdentifierAdapterUser}, s.err
}

type projectIdentifierAdapterRepositoryStub struct {
	authorized   bool
	authorizeErr error
	body         json.RawMessage
	err          error
	deleted      string
}

func (r *projectIdentifierAdapterRepositoryStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.authorizeErr
}
func (r *projectIdentifierAdapterRepositoryStub) Get(context.Context, uuid.UUID, string, string) (json.RawMessage, error) {
	return r.body, r.err
}
func (r *projectIdentifierAdapterRepositoryStub) Delete(_ context.Context, _ uuid.UUID, _ string, name string) error {
	r.deleted = name
	return r.err
}

func projectIdentifierAdapterContext(t *testing.T, repository *projectIdentifierAdapterRepositoryStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewProjectIdentifiers(repository, projectIdentifierAdapterSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestProjectIdentifiersGetDecodesRepositoryResult(t *testing.T) {
	repository := &projectIdentifierAdapterRepositoryStub{
		authorized: true,
		body:       json.RawMessage(`{"exists":1,"identifiers":[{"id":7,"name":"ABC","project":"00000000-0000-4000-8000-000000000001"}]}`),
	}
	service, ctx := projectIdentifierAdapterContext(t, repository)
	result, err := service.Get(ctx, &api.GetPayload{Slug: "demo", Name: "abc"})
	if err != nil || result.Exists != 1 || len(result.Identifiers) != 1 || result.Identifiers[0].ID != 7 {
		t.Fatalf("Get() = %#v, %v", result, err)
	}
}

func TestProjectIdentifiersMapsDjangoValidationAndConflictErrors(t *testing.T) {
	service, ctx := projectIdentifierAdapterContext(t, &projectIdentifierAdapterRepositoryStub{authorized: true, err: store.ErrNameRequired})
	if _, err := service.Get(ctx, &api.GetPayload{Slug: "demo"}); err == nil {
		t.Fatal("Get() returned nil error for empty name")
	} else {
		var invalid *api.ProjectIdentifierInvalid
		if !errors.As(err, &invalid) || invalid.Message != "Name is required" {
			t.Fatalf("Get() error = %T %v", err, err)
		}
	}

	repository := &projectIdentifierAdapterRepositoryStub{authorized: true, err: store.ErrProjectExists}
	service, ctx = projectIdentifierAdapterContext(t, repository)
	if err := service.Delete(ctx, &api.DeletePayload{Slug: "demo", Body: &api.ProjectIdentifierDelete{Name: "ABC"}}); err == nil {
		t.Fatal("Delete() returned nil error for existing project")
	} else {
		var conflict *api.ProjectIdentifierConflict
		if !errors.As(err, &conflict) || conflict.Message != "Cannot delete an identifier of an existing project" {
			t.Fatalf("Delete() error = %T %v", err, err)
		}
	}
}

func TestProjectIdentifiersRejectsUnauthorizedWorkspace(t *testing.T) {
	service, ctx := projectIdentifierAdapterContext(t, &projectIdentifierAdapterRepositoryStub{})
	if _, err := service.Get(ctx, &api.GetPayload{Slug: "demo", Name: "ABC"}); err == nil {
		t.Fatal("Get() returned nil error for unauthorized workspace")
	} else {
		var forbidden *api.ProjectIdentifierForbidden
		if !errors.As(err, &forbidden) {
			t.Fatalf("Get() error = %T %v, want forbidden", err, err)
		}
	}
}
