package systemapi

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	api "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
)

type workspaceLifecycleTestRepository struct {
	role    int
	steps   *[]string
	deleted instancemanagement.WorkspaceDeletion
}

func (repository workspaceLifecycleTestRepository) WorkspaceRole(context.Context, uuid.UUID, string) (int, error) {
	return repository.role, nil
}

func (workspaceLifecycleTestRepository) WorkspaceCreationDisabled(context.Context) (bool, error) {
	return false, nil
}

func (workspaceLifecycleTestRepository) CreateWorkspace(context.Context, uuid.UUID, map[string]json.RawMessage, string) (json.RawMessage, error) {
	panic("unexpected workspace create")
}

func (workspaceLifecycleTestRepository) UpdateWorkspace(context.Context, uuid.UUID, string, map[string]json.RawMessage) (json.RawMessage, error) {
	panic("unexpected workspace update")
}

func (repository workspaceLifecycleTestRepository) PrepareWorkspaceDeletion(context.Context, uuid.UUID, string) (instancemanagement.WorkspaceDeletion, error) {
	*repository.steps = append(*repository.steps, "prepare")
	return repository.deleted, nil
}

func (repository workspaceLifecycleTestRepository) SoftDeleteWorkspace(_ context.Context, _ uuid.UUID, workspace instancemanagement.WorkspaceDeletion, deletedAt time.Time) error {
	if workspace.ID != repository.deleted.ID || deletedAt.IsZero() {
		return context.Canceled
	}
	*repository.steps = append(*repository.steps, "soft-delete")
	return nil
}

type workspaceLifecycleTestSessions struct{}

func (workspaceLifecycleTestSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	panic("unexpected session validation")
}

type workspaceLifecycleTestPublisher struct{ steps *[]string }

func (publisher workspaceLifecycleTestPublisher) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	if name != "plane.bgtasks.event_tracking_task.track_event" || len(args) != 0 || kwargs["event_name"] != "workspace_deleted" {
		return context.Canceled
	}
	*publisher.steps = append(*publisher.steps, "track-event")
	return nil
}

func (publisher workspaceLifecycleTestPublisher) SoftDelete(_ context.Context, app, model, id string) error {
	if app != "db" || model != "workspace" || id != "00000000-0000-4000-8000-000000000001" {
		return context.Canceled
	}
	*publisher.steps = append(*publisher.steps, "soft-delete-task")
	return nil
}

func TestWorkspaceLifecycleDeletePreservesSourceEffectOrderAndArguments(t *testing.T) {
	steps := []string{}
	service, err := NewWorkspaceLifecycle(
		workspaceLifecycleTestRepository{
			role:    20,
			steps:   &steps,
			deleted: instancemanagement.WorkspaceDeletion{ID: "00000000-0000-4000-8000-000000000001", Name: "Example", Slug: "example"},
		},
		workspaceLifecycleTestSessions{},
		workspaceLifecycleTestPublisher{steps: &steps},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), workspaceLifecyclePrincipalKey{}, workspaceLifecyclePrincipal{userID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), timezone: "UTC"})
	if err := service.DeleteWorkspace(ctx, &api.DeleteWorkspacePayload{Slug: "example"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"prepare", "track-event", "soft-delete", "soft-delete-task"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("delete effect order = %v, want %v", steps, want)
	}
}

func TestWorkspaceLifecycleDeleteRejectsMemberBeforeDeleting(t *testing.T) {
	steps := []string{}
	service, err := NewWorkspaceLifecycle(
		workspaceLifecycleTestRepository{role: 15, steps: &steps},
		workspaceLifecycleTestSessions{},
		workspaceLifecycleTestPublisher{steps: &steps},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), workspaceLifecyclePrincipalKey{}, workspaceLifecyclePrincipal{userID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), timezone: "UTC"})
	err = service.DeleteWorkspace(ctx, &api.DeleteWorkspacePayload{Slug: "example"})
	if err == nil {
		t.Fatal("member deletion unexpectedly succeeded")
	}
	status, body := workspaceLifecycleProtocolError(err)
	if status != 403 || !reflect.DeepEqual(body, map[string]any{"detail": "You do not have permission to perform this action."}) {
		t.Fatalf("member deletion error = %d %v", status, body)
	}
	if len(steps) != 0 {
		t.Fatalf("member deletion performed side effects: %v", steps)
	}
}

func TestWorkspaceLifecyclePermissionBodyMatchesMembershipState(t *testing.T) {
	ctx := context.WithValue(context.Background(), workspaceLifecyclePrincipalKey{}, workspaceLifecyclePrincipal{userID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), timezone: "UTC"})
	for _, test := range []struct {
		name string
		role int
		want map[string]any
	}{
		{name: "missing workspace", role: 0, want: map[string]any{"detail": "You do not have permission to perform this action."}},
		{name: "member cannot update settings", role: 15, want: map[string]any{"error": "You don't have the required permissions."}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &workspaceLifecycleService{repository: workspaceLifecycleTestRepository{role: test.role}}
			_, err := service.workspaceAdmin(ctx, "workspace", false)
			status, body := workspaceLifecycleProtocolError(err)
			if status != 403 || !reflect.DeepEqual(body, test.want) {
				t.Fatalf("workspace update denial = %d %v, want 403 %v", status, body, test.want)
			}
		})
	}
}

func TestWorkspaceLifecycleUnsupportedMethodUsesSourcePermission(t *testing.T) {
	service := &workspaceLifecycleService{
		repository: workspaceLifecycleTestRepository{role: 20},
	}
	ctx := WithReadServiceSessionPrincipal(context.Background(), "00000000-0000-4000-8000-000000000002", "UTC")
	status, body := service.AuthorizeSessionResourceProtocol(ctx, "", SessionProtocolResource{
		Method: http.MethodTrace, Pattern: "/api/workspaces/{slug}/", Slug: "workspace",
	})
	if status != 403 || !reflect.DeepEqual(body, map[string]any{"detail": "You do not have permission to perform this action."}) {
		t.Fatalf("TRACE authorization = %d %v, want source permission denial", status, body)
	}
}
