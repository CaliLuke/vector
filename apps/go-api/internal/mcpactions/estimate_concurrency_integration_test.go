//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func estimatePointWriteFixture(t *testing.T) (milestoneFixture, *projectconfiguration.Repository, projectmutations.Command) {
	t.Helper()
	f := newMilestoneFixture(t)
	repository, err := projectconfiguration.NewRepository(f.pool)
	require.NoError(t, err)
	command := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one", Action: "create", Fields: map[string]any{"name": "Concurrent sizing"}}
	created, err := repository.MutateEstimate(t.Context(), command)
	require.NoError(t, err)
	var estimate struct{ ID uuid.UUID }
	require.NoError(t, json.Unmarshal(created.Record, &estimate))
	key := uuid.New()
	command.Action, command.ResourceID, command.RequestID, command.ExpectedRevision = "create_points", estimate.ID, &key, &created.Revision
	command.Fields = map[string]any{"points": `[{"value":"1"},{"value":"2"}]`}
	return f, repository, command
}

func estimatePointEditFixture(t *testing.T) (milestoneFixture, *projectconfiguration.Repository, projectmutations.Command) {
	t.Helper()
	f, repository, command := estimatePointWriteFixture(t)
	created, err := repository.MutateEstimate(t.Context(), command)
	require.NoError(t, err)
	var points []struct{ ID uuid.UUID }
	require.NoError(t, json.Unmarshal(created.Record, &points))
	require.Len(t, points, 2)
	key := uuid.New()
	command.Action, command.ParentID, command.RequestID, command.ExpectedRevision = "update_point", points[0].ID, &key, &created.Revision
	command.Fields = map[string]any{"value": "mcp"}
	return f, repository, command
}

func TestEstimateConcurrentPointBatchesCommitOnce(t *testing.T) {
	for _, mode := range []string{"identical receipt", "changed input", "competing revision"} {
		t.Run(mode, func(t *testing.T) {
			f, repository, first := estimatePointWriteFixture(t)
			second := first
			if mode == "changed input" {
				second.Fields = map[string]any{"points": `[{"value":"3"},{"value":"4"}]`}
			}
			if mode == "competing revision" {
				key := uuid.New()
				second.RequestID = &key
			}
			var calls [2]func(context.Context) (projectmutations.Result, error)
			for i, command := range []projectmutations.Command{first, second} {
				calls[i] = func(ctx context.Context) (projectmutations.Result, error) {
					return repository.MutateEstimate(ctx, command)
				}
			}
			outcomes := competingScopeWrites(t, f, projectMutationLock, first.ProjectID, calls)
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			switch mode {
			case "identical receipt":
				require.NoError(t, outcomes[1].err)
				assertConcurrentReceiptReplay(t, outcomes[0].value, outcomes[1].value)
			case "changed input":
				require.ErrorIs(t, outcomes[1].err, mutationreceipts.ErrConflict)
			case "competing revision":
				require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
			}
			var points, receipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM estimate_points WHERE estimate_id=$1), (SELECT count(*) FROM mutation_receipts WHERE tool='project_estimate' AND action='create_points')`, first.ResourceID).Scan(&points, &receipts))
			require.Equal(t, 2, points)
			require.Equal(t, 1, receipts)
			snapshot := estimateStorageSnapshot(t, f)
			replayCommand := first
			if outcomes[0].value.RequestID != nil && *outcomes[0].value.RequestID == *second.RequestID && mode == "competing revision" {
				replayCommand = second
			}
			// Changed-input races can be won by either batch; retry the accepted body.
			if mode == "changed input" {
				var accepted []map[string]any
				require.NoError(t, json.Unmarshal(outcomes[0].value.Record, &accepted))
				if accepted[0]["value"] == "3" {
					replayCommand = second
				}
			}
			replay, err := repository.MutateEstimate(t.Context(), replayCommand)
			require.NoError(t, err)
			require.NotNil(t, replay.Replayed)
			require.True(t, *replay.Replayed)
			require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		})
	}
}

func TestEstimateCancelledBatchLeavesReceiptReusable(t *testing.T) {
	f, repository, command := estimatePointWriteFixture(t)
	snapshot := estimateStorageSnapshot(t, f)
	cancelScopeWriteWhileLocked(t, f, projectMutationLock, command.ProjectID, func(ctx context.Context) error { _, err := repository.MutateEstimate(ctx, command); return err })
	require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
	command.Fields = map[string]any{"points": `[{"value":"retry"}]`}
	accepted, err := repository.MutateEstimate(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, accepted.Replayed)
	require.False(t, *accepted.Replayed)
	snapshot = estimateStorageSnapshot(t, f)
	replay, err := repository.MutateEstimate(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	require.Equal(t, accepted.Revision, replay.Revision)
	require.JSONEq(t, string(accepted.Record), string(replay.Record))
	require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
}
