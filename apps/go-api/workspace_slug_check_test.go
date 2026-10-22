package systemapi

import (
	"context"
	"errors"
	"testing"

	workspaceapi "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
)

type workspaceSlugRepositoryStub struct {
	exists bool
	err    error
	slug   string
}

func (stub *workspaceSlugRepositoryStub) Exists(_ context.Context, slug string) (bool, error) {
	stub.slug = slug
	return stub.exists, stub.err
}

type workspaceSlugSessionStub struct{ err error }

func (stub workspaceSlugSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}, nil
}

func TestWorkspaceSlugCheck(t *testing.T) {
	for _, test := range []struct {
		name       string
		slug       *string
		database   bool
		wantStatus bool
		wantError  string
	}{
		{name: "missing", wantError: "Workspace Slug is required"},
		{name: "empty", slug: stringPointer(""), wantError: "Workspace Slug is required"},
		{name: "existing", slug: stringPointer("taken"), database: true},
		{name: "restricted", slug: stringPointer("admin")},
		{name: "available", slug: stringPointer("new-workspace"), wantStatus: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &workspaceSlugRepositoryStub{exists: test.database}
			service, err := NewWorkspaceSlugCheck(repository, workspaceSlugSessionStub{})
			if err != nil {
				t.Fatal(err)
			}
			response, err := service.Check(context.Background(), &workspaceapi.CheckPayload{Slug: test.slug})
			if test.wantError != "" {
				var typed *workspaceapi.WorkspaceSlugError
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

func TestWorkspaceSlugCheckAuthentication(t *testing.T) {
	repository := &workspaceSlugRepositoryStub{}
	service, err := NewWorkspaceSlugCheck(repository, workspaceSlugSessionStub{err: errors.New("invalid")})
	if err != nil {
		t.Fatal(err)
	}
	authenticator := service.(workspaceapi.Authorizer)
	_, err = authenticator.APIKeyAuth(context.Background(), "", nil)
	var unauthenticated *workspaceapi.WorkspaceSlugUnauthenticated
	if !errors.As(err, &unauthenticated) || unauthenticated.Detail != unauthenticatedDetail {
		t.Fatalf("error = %v, want Django unauthenticated response", err)
	}
}

func TestWorkspaceSlugCheckAllRestrictedSlugs(t *testing.T) {
	service, err := NewWorkspaceSlugCheck(&workspaceSlugRepositoryStub{}, workspaceSlugSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range workspacevalidation.RestrictedSlugs() {
		value := slug
		response, err := service.Check(context.Background(), &workspaceapi.CheckPayload{Slug: &value})
		if err != nil || response == nil || response.Status {
			t.Fatalf("slug %q: response = %#v, error = %v, want unavailable", slug, response, err)
		}
	}
}

func TestWorkspaceSlugCheckRepositoryFailure(t *testing.T) {
	service, err := NewWorkspaceSlugCheck(&workspaceSlugRepositoryStub{err: errors.New("database unavailable")}, workspaceSlugSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	slug := "new-workspace"
	_, err = service.Check(context.Background(), &workspaceapi.CheckPayload{Slug: &slug})
	var internal *workspaceapi.WorkspaceSlugInternalError
	if !errors.As(err, &internal) || internal.Message != "Something went wrong please try again later" {
		t.Fatalf("error = %v, want Django-compatible internal error", err)
	}
}

func stringPointer(value string) *string { return &value }
