package mcpactions_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"

	"github.com/stretchr/testify/require"

	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
)

type projectWriteStub struct {
	command          projects.ArchiveCommand
	result           projects.ArchiveResult
	err              error
	visibility       projects.ProjectWriteVisibility
	visibilityErr    error
	writes           int
	beforeRevalidate func()
}

func newProjectWriteStub() *projectWriteStub {
	return &projectWriteStub{result: projects.ArchiveResult{At: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC), Revision: "project:30000000-0000-4000-8000-000000000001:2"}, visibility: projects.ProjectWriteVisibility{Archive: true, Unarchive: true}}
}
func (s *projectWriteStub) ArchiveProject(ctx context.Context, command projects.ArchiveCommand) (projects.ArchiveResult, error) {
	s.command = command
	s.writes++
	if s.beforeRevalidate != nil {
		s.beforeRevalidate()
	}
	if command.Revalidate != nil {
		if err := command.Revalidate(ctx); err != nil {
			return projects.ArchiveResult{}, err
		}
	}
	return s.result, s.err
}
func (s *projectWriteStub) VisibleProjectWrites(context.Context, uuid.UUID, uuid.UUID, string) (projects.ProjectWriteVisibility, error) {
	return s.visibility, s.visibilityErr
}

func TestProjectArchivePreservesEmptyResultWithoutExtensions(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, action := range []string{"archive", "unarchive"} {
			t.Run(version+"/"+action, func(t *testing.T) {
				f := newFixture(t)
				result := rpcVersion(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"`+action+`","project_id":"30000000-0000-4000-8000-000000000001"}`), version)
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.Content)
				require.Empty(t, result.StructuredContent)
				require.Equal(t, 1, f.projectWrites.writes)
				require.Equal(t, action == "archive", f.projectWrites.command.Archive)
			})
		}
	}
}

func TestProjectArchiveReturnsTypedExtensionMetadata(t *testing.T) {
	f := newFixture(t)
	expires := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	f.projectWrites.result.Receipt.ReplayExpiresAt = &expires
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001","request_id":"40000000-0000-4000-8000-000000000001"}`))
	require.False(t, result.IsError, result.Content)
	require.JSONEq(t, `{"project_id":"30000000-0000-4000-8000-000000000001","revision":"project:30000000-0000-4000-8000-000000000001:2","request_id":"40000000-0000-4000-8000-000000000001","replay_expires_at":"2026-10-29T12:00:00Z","replayed":false}`, string(result.StructuredContent))
}

func TestProjectArchiveRejectsInvalidIdentifiersBeforeWrites(t *testing.T) {
	f := newFixture(t)
	request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	for _, input := range []*api.ProjectArchive{
		{Action: "archive", ProjectID: "bad"},
		{Action: "archive", ProjectID: uuid.Nil.String()},
		{Action: "archive", ProjectID: "30000000-0000-4000-8000-000000000001", RequestID: new("bad")},
		{Action: "archive", ProjectID: "30000000-0000-4000-8000-000000000001", RequestID: new(uuid.Nil.String())},
	} {
		value := api.NewProjectRequestProjectArchive(input)
		_, err = f.service.Project(ctx, &value)
		require.Error(t, err)
	}
	require.Zero(t, f.projectWrites.writes)
}

func TestArchiveRevalidatesCredentialAndActorAfterWaiting(t *testing.T) {
	for _, changedOwner := range []bool{false, true} {
		t.Run(fmt.Sprint(changedOwner), func(t *testing.T) {
			f := newFixture(t)
			f.projectWrites.beforeRevalidate = func() {
				if changedOwner {
					f.keys.owners["test-one"] = f.keys.owners["test-two"]
				} else {
					f.keys.err = apikey.ErrInvalid
				}
			}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001"}`))
			require.True(t, result.IsError)
			require.Empty(t, result.StructuredContent)
			code := "unauthorized"
			if changedOwner {
				code = "forbidden"
			}
			require.Contains(t, result.Content[0].Text, "["+code+"]")
		})
	}
}

func TestProjectExtensionsFailClosedWhenDomainMetadataIsMissing(t *testing.T) {
	f := newFixture(t)
	h := f.handler(t, nil)
	for _, input := range []string{
		`{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001","include_revision":true}`,
		`{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001","request_id":"40000000-0000-4000-8000-000000000001"}`,
	} {
		result := rpc(t, h, "test-one", "tools/call", projectCall(input))
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].Text, "[internal_error]")
		require.Empty(t, result.StructuredContent)
	}
}

func TestProjectDiscoveryPropagatesAuthorizationAndPrimaryFailures(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.DiscoverActions(t.Context())
	require.Error(t, err)
	request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	for _, memberFailure := range []bool{false, true} {
		f.projectWrites.visibilityErr = nil
		if memberFailure {
			f.directory.capabilitiesErr = errors.New("primary unavailable")
		} else {
			f.projectWrites.visibilityErr = errors.New("primary unavailable")
		}
		visible, err := f.service.DiscoverActions(ctx)
		require.Error(t, err)
		require.Nil(t, visible)
	}
}

func TestProjectRevisionGuardDoesNotRequireReceipt(t *testing.T) {
	f := newFixture(t)
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001","expected_revision":"project:30000000-0000-4000-8000-000000000001:1"}`))
	require.False(t, result.IsError, result.Content)
	require.JSONEq(t, `{"project_id":"30000000-0000-4000-8000-000000000001","revision":"project:30000000-0000-4000-8000-000000000001:2"}`, string(result.StructuredContent))
	require.Nil(t, f.projectWrites.command.RequestID)
	require.Equal(t, "project:30000000-0000-4000-8000-000000000001:1", *f.projectWrites.command.ExpectedRevision)
}
