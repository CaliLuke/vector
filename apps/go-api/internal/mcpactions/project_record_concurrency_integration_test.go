//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func projectRecordMutationFixture(t *testing.T) (milestoneFixture, *projects.LifecycleRepository, projectmutations.Command) {
	t.Helper()
	f := newMilestoneFixture(t)
	repo := projectLifecycle(t, f.pool)
	cmd := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Action: "create", Fields: map[string]any{"name": "Project concurrency", "identifier": "CONCUR"}}
	created, err := repo.MutateResource(t.Context(), cmd)
	require.NoError(t, err)
	cmd.ProjectID, cmd.ResourceID, cmd.Action = uuid.MustParse(created.ID), uuid.MustParse(created.ID), "update"
	cmd.ExpectedRevision = &created.Revision
	key := uuid.New()
	cmd.RequestID = &key
	cmd.Fields = map[string]any{"name": "First edit"}
	return f, repo, cmd
}

func TestProjectRecordContentionCommitsOneMutation(t *testing.T) {
	for _, mode := range []string{"identical request", "changed request input", "competing revisions"} {
		t.Run(mode, func(t *testing.T) {
			f, repo, first := projectRecordMutationFixture(t)
			second := first
			if mode != "identical request" {
				second.Fields = map[string]any{"name": "Second edit"}
			}
			if mode == "competing revisions" {
				key := uuid.New()
				second.RequestID = &key
			}
			var beforeEffects int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects`).Scan(&beforeEffects))
			calls := [2]func(context.Context) (projectmutations.Result, error){}
			for i, cmd := range []projectmutations.Command{first, second} {
				calls[i] = func(ctx context.Context) (projectmutations.Result, error) { return repo.MutateResource(ctx, cmd) }
			}
			result := competingScopeWrites(t, f, projectMutationLock, first.ProjectID, calls)
			if result[0].err != nil {
				result[0], result[1] = result[1], result[0]
			}
			require.NoError(t, result[0].err)
			switch mode {
			case "identical request":
				require.NoError(t, result[1].err)
				assertConcurrentReceiptReplay(t, result[0].value, result[1].value)
			case "changed request input":
				require.ErrorIs(t, result[1].err, mutationreceipts.ErrConflict)
			case "competing revisions":
				require.ErrorIs(t, result[1].err, projects.ErrRevisionConflict)
			}
			var committed struct{ Name string }
			require.NoError(t, json.Unmarshal(result[0].value.Record, &committed))
			var name string
			var receipts, effects int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name FROM projects WHERE id=$1`, first.ProjectID).Scan(&name))
			require.Equal(t, committed.Name, name)
			require.NotEqual(t, *first.ExpectedRevision, result[0].value.Revision)
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool='project' AND project_id=$1 AND action='update'`, first.ProjectID).Scan(&receipts))
			require.Equal(t, 1, receipts)
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects`).Scan(&effects))
			require.Equal(t, beforeEffects+1, effects)
		})
	}
}

func TestProjectRecordCancelledLockWaitLeavesRequestReusable(t *testing.T) {
	f, repo, cmd := projectRecordMutationFixture(t)
	before := projectMutationStorageSnapshot(t, f)
	cancelScopeWriteWhileLocked(t, f, projectMutationLock, cmd.ProjectID, func(ctx context.Context) error { _, err := repo.MutateResource(ctx, cmd); return err })
	require.Equal(t, before, projectMutationStorageSnapshot(t, f))
	cmd.Fields["name"] = "Accepted retry"
	accepted, err := repo.MutateResource(t.Context(), cmd)
	require.NoError(t, err)
	require.NotNil(t, accepted.Replayed)
	require.False(t, *accepted.Replayed)
	require.Contains(t, string(accepted.Record), "Accepted retry")
	after := projectMutationStorageSnapshot(t, f)
	replay, err := repo.MutateResource(t.Context(), cmd)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	require.Equal(t, accepted.Revision, replay.Revision)
	require.Equal(t, after, projectMutationStorageSnapshot(t, f))
}
