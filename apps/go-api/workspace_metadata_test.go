package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

const workspaceMetadataTestUser = "00000000-0000-4000-8000-000000000010"

type workspaceMetadataRepositoryStub struct {
	authorized          bool
	authorizeErr        error
	labels, states      json.RawMessage
	labelsErr, stateErr error
}

func (s *workspaceMetadataRepositoryStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return s.authorized, s.authorizeErr
}
func (s *workspaceMetadataRepositoryStub) ListLabels(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return s.labels, s.labelsErr
}
func (s *workspaceMetadataRepositoryStub) ListStates(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return s.states, s.stateErr
}

type workspaceMetadataSessionStub struct{ err error }

func (s workspaceMetadataSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: workspaceMetadataTestUser}, s.err
}

func workspaceMetadataServiceForTest(t *testing.T, repository *workspaceMetadataRepositoryStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewWorkspaceMetadata(repository, workspaceMetadataSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestWorkspaceMetadataReturnsRepositoryContracts(t *testing.T) {
	repository := &workspaceMetadataRepositoryStub{
		authorized: true,
		labels:     json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000001","name":"Bug","color":"#f00","parent":null,"project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","sort_order":10}]`),
		states:     json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000001","project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","name":"Started","color":"#0f0","group":"started","default":false,"description":null,"sequence":1,"order":0.5}]`),
	}
	service, ctx := workspaceMetadataServiceForTest(t, repository)
	labels, err := service.Labels(ctx, &api.LabelsPayload{Slug: "demo"})
	if err != nil || len(labels) != 1 || labels[0].Name != "Bug" {
		t.Fatalf("Labels() = %#v, %v", labels, err)
	}
	states, err := service.States(ctx, &api.StatesPayload{Slug: "demo"})
	if err != nil || len(states) != 1 || states[0].Group != "started" {
		t.Fatalf("States() = %#v, %v", states, err)
	}
}

func TestWorkspaceMetadataAuthorizationAndRepositoryFailures(t *testing.T) {
	service, ctx := workspaceMetadataServiceForTest(t, &workspaceMetadataRepositoryStub{})
	if _, err := service.Labels(ctx, &api.LabelsPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden error")
	} else if _, ok := err.(*api.WorkspaceMetadataForbidden); !ok {
		t.Fatalf("error = %T, want forbidden", err)
	}

	service, ctx = workspaceMetadataServiceForTest(t, &workspaceMetadataRepositoryStub{authorized: true, labelsErr: errors.New("database unavailable")})
	if _, err := service.Labels(ctx, &api.LabelsPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected internal error")
	} else if _, ok := err.(*api.WorkspaceMetadataInternal); !ok {
		t.Fatalf("error = %T, want internal", err)
	}

	service, ctx = workspaceMetadataServiceForTest(t, &workspaceMetadataRepositoryStub{authorized: true, labels: json.RawMessage("not-json")})
	if _, err := service.Labels(ctx, &api.LabelsPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected decoding error")
	} else if _, ok := err.(*api.WorkspaceMetadataInternal); !ok {
		t.Fatalf("error = %T, want internal", err)
	}
}
