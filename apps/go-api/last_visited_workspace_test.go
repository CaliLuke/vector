package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

const lastVisitedWorkspaceTestUser = "00000000-0000-4000-8000-000000000010"

type lastVisitedWorkspaceRepositoryStub struct {
	body json.RawMessage
	err  error
}

func (stub lastVisitedWorkspaceRepositoryStub) Get(context.Context, uuid.UUID) (json.RawMessage, error) {
	return stub.body, stub.err
}

type lastVisitedWorkspaceSessionStub struct{ err error }

func (stub lastVisitedWorkspaceSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: lastVisitedWorkspaceTestUser}, stub.err
}

func TestLastVisitedWorkspaceReturnsRepositoryContract(t *testing.T) {
	repository := lastVisitedWorkspaceRepositoryStub{body: json.RawMessage(`{"project_details":[],"workspace_details":{}}`)}
	service, err := NewLastVisitedWorkspace(repository, lastVisitedWorkspaceSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Get(ctx, &api.GetPayload{})
	if err != nil || string(result.ProjectDetails) != `[]` || string(result.WorkspaceDetails) != `{}` {
		t.Fatalf("Get() = %#v, %v", result, err)
	}
}

func TestLastVisitedWorkspaceAuthenticationAndRepositoryFailures(t *testing.T) {
	service, err := NewLastVisitedWorkspace(lastVisitedWorkspaceRepositoryStub{}, lastVisitedWorkspaceSessionStub{err: errors.New("invalid")})
	if err != nil {
		t.Fatal(err)
	}
	if _, apiKeyAuthErr := service.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); apiKeyAuthErr == nil {
		t.Fatal("expected authentication error")
	}

	service, err = NewLastVisitedWorkspace(lastVisitedWorkspaceRepositoryStub{err: errors.New("database unavailable")}, lastVisitedWorkspaceSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(ctx, &api.GetPayload{}); err == nil {
		t.Fatal("expected internal error")
	} else if _, ok := err.(*api.LastVisitedWorkspaceInternal); !ok {
		t.Fatalf("error = %T", err)
	}
}
