//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/worklogs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func workLogMutationFixture(t *testing.T) (milestoneFixture, *worklogs.Repository, projectmutations.Command) {
	t.Helper()
	f := newMilestoneFixture(t)
	repository := worklogs.NewRepository(f.pool)
	command := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), ParentID: uuid.MustParse(f.item(t)), WorkspaceSlug: "workspace-one", Action: "create", Fields: map[string]any{"duration": 30}}
	created, err := repository.MutateResource(t.Context(), command)
	require.NoError(t, err)
	command.Action, command.ResourceID, command.ExpectedRevision = "update", uuid.MustParse(created.ID), &created.Revision
	key := uuid.New()
	command.RequestID, command.Fields = &key, map[string]any{"duration": 45}
	return f, repository, command
}

func TestWorkLogConcurrentMutationsCommitOnce(t *testing.T) {
	for _, mode := range []string{"same receipt", "conflicting receipt", "competing revision"} {
		t.Run(mode, func(t *testing.T) {
			f, repository, first := workLogMutationFixture(t)
			second := first
			var rejected error
			if mode != "same receipt" {
				second.Fields = map[string]any{"duration": 60}
				rejected = mutationreceipts.ErrConflict
			}
			if mode == "competing revision" {
				key := uuid.New()
				second.RequestID = &key
				rejected = projectmutations.ErrRevision
			}
			results := competingProjectWrites(t, f, first.ProjectID, [2]func(context.Context) (projectmutations.Result, error){
				func(ctx context.Context) (projectmutations.Result, error) {
					return repository.MutateResource(ctx, first)
				},
				func(ctx context.Context) (projectmutations.Result, error) {
					return repository.MutateResource(ctx, second)
				},
			})
			accepted := make([]projectmutations.Result, 0, 2)
			for _, result := range results {
				if result.err == nil {
					accepted = append(accepted, result.value)
				} else {
					require.ErrorIs(t, result.err, rejected)
				}
			}
			if rejected == nil {
				require.Len(t, accepted, 2)
				require.NotNil(t, accepted[0].Replayed)
				require.NotNil(t, accepted[1].Replayed)
				require.NotEqual(t, *accepted[0].Replayed, *accepted[1].Replayed)
				require.Equal(t, accepted[0].Revision, accepted[1].Revision)
				require.JSONEq(t, string(accepted[0].Record), string(accepted[1].Record))
			} else {
				require.Len(t, accepted, 1)
			}
			var committed struct{ Duration int }
			require.NoError(t, json.Unmarshal(accepted[0].Record, &committed))
			var duration, revision, receipts, effects int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT l.duration,r.revision,(SELECT count(*) FROM mutation_receipts WHERE tool='work_log' AND action='update'),(SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$2 AND payload->>'type'='work_log.activity.updated') FROM work_logs l JOIN work_log_edit_revisions r ON r.work_log_id=l.id WHERE l.id=$1`, first.ResourceID, first.ParentID.String()).Scan(&duration, &revision, &receipts, &effects))
			require.Equal(t, committed.Duration, duration)
			require.Equal(t, 2, revision)
			require.Equal(t, 1, receipts)
			require.Equal(t, 1, effects)
		})
	}
}

func TestWorkLogCancelledLockWaitLeavesReceiptReusable(t *testing.T) {
	f, repository, command := workLogMutationFixture(t)
	before := workLogStorageSnapshot(t, f)
	cancelProjectWriteWhileLocked(t, f, command.ProjectID, func(ctx context.Context) error { _, err := repository.MutateResource(ctx, command); return err })
	require.Equal(t, before, workLogStorageSnapshot(t, f))
	result, err := repository.MutateResource(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, result.Replayed)
	require.False(t, *result.Replayed)
	var duration int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT duration FROM work_logs WHERE id=$1`, command.ResourceID).Scan(&duration))
	require.Equal(t, 45, duration)
	after := workLogStorageSnapshot(t, f)
	result, err = repository.MutateResource(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, result.Replayed)
	require.True(t, *result.Replayed)
	require.Equal(t, after, workLogStorageSnapshot(t, f))
}

func TestWorkLogRejectsInvalidDurationAndIdentityFields(t *testing.T) {
	f, repository, base := workLogMutationFixture(t)
	for _, invalid := range []struct {
		name, action string
		fields       map[string]any
	}{
		{"zero duration", "create", map[string]any{"duration": 0}},
		{"negative duration", "create", map[string]any{"duration": -1}},
		{"fractional minutes", "create", map[string]any{"duration": 1.5}},
		{"duration overflow", "update", map[string]any{"duration": int64(2147483648)}},
		{"missing create duration", "create", nil},
		{"caller supplied author", "update", map[string]any{"logged_by": uuid.NewString()}},
		{"delete cannot edit fields", "delete", map[string]any{"duration": 30}},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			command := base
			command.Action, command.Fields = invalid.action, invalid.fields
			if command.Action == "create" {
				command.ResourceID = uuid.Nil
				command.ExpectedRevision = nil
			}
			before := workLogStorageSnapshot(t, f)
			_, err := repository.MutateResource(t.Context(), command)
			require.ErrorIs(t, err, projectmutations.ErrInvalid)
			require.Equal(t, before, workLogStorageSnapshot(t, f))
		})
	}
}
