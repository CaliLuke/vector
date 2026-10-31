//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func attachmentImportCommand(t *testing.T, h attachmentRecoveryHarness) projectmutations.Command {
	t.Helper()
	key := uuid.New()
	return projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), ParentID: uuid.MustParse(h.item(t)), WorkspaceSlug: "workspace-one", Action: "create", RequestID: &key, Fields: map[string]any{"url": "https://files.example.test/concurrent.txt"}}
}

func attachmentDeleteCommand(t *testing.T, h attachmentRecoveryHarness) projectmutations.Command {
	t.Helper()
	command := attachmentImportCommand(t, h)
	created, err := h.application.MutateResource(t.Context(), command)
	require.NoError(t, err)
	result, err := h.application.Read(t.Context(), assets.AttachmentQuery{ActorID: command.ActorID, WorkspaceID: command.WorkspaceID, ProjectID: command.ProjectID, IssueID: command.ParentID, WorkspaceSlug: command.WorkspaceSlug, Action: "list", IncludeRevision: true})
	require.NoError(t, err)
	rows, ok := result.([]map[string]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	revision, ok := rows[0]["revision"].(string)
	require.True(t, ok)
	key := uuid.New()
	command.Action, command.ResourceID, command.Fields = "delete", uuid.MustParse(created.ID), nil
	command.RequestID, command.ExpectedRevision = &key, &revision
	return command
}

func competingAttachmentWrites(t *testing.T, h attachmentRecoveryHarness, commands [2]projectmutations.Command) [2]concurrentMutationOutcome[projectmutations.Result] {
	t.Helper()
	calls := [2]func(context.Context) (projectmutations.Result, error){}
	for i, command := range commands {
		calls[i] = func(ctx context.Context) (projectmutations.Result, error) {
			return h.application.MutateResource(ctx, command)
		}
	}
	return competingProjectWrites(t, h.milestoneFixture, commands[0].ProjectID, calls)
}

func TestAttachmentConcurrentImportsFetchAndCommitOnce(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		name := "matching receipt"
		if conflict {
			name = "changed input"
		}
		t.Run(name, func(t *testing.T) {
			h := attachmentRecoveryFixture(t)
			first := attachmentImportCommand(t, h)
			second := first
			if conflict {
				second.Fields = map[string]any{"url": "https://files.example.test/other.txt"}
			}
			outcomes := competingAttachmentWrites(t, h, [2]projectmutations.Command{first, second})
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			if conflict {
				require.ErrorIs(t, outcomes[1].err, mutationreceipts.ErrConflict)
			} else {
				require.NoError(t, outcomes[1].err)
				require.Equal(t, outcomes[0].value.ID, outcomes[1].value.ID)
				require.NotNil(t, outcomes[0].value.Replayed)
				require.NotNil(t, outcomes[1].value.Replayed)
				require.NotEqual(t, *outcomes[0].value.Replayed, *outcomes[1].value.Replayed)
			}
			require.Equal(t, int64(1), h.fetched.Load())
			var uploads, receipts, effects int
			require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM file_assets WHERE issue_id=$1 AND is_uploaded),(SELECT count(*) FROM mutation_receipts WHERE request_id=$2),(SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$1::text AND payload->>'type'='attachment.activity.created')`, first.ParentID.String(), first.RequestID).Scan(&uploads, &receipts, &effects))
			require.Equal(t, 1, uploads)
			require.Equal(t, 1, receipts)
			require.Equal(t, 1, effects)
		})
	}
}

func TestAttachmentConcurrentDeletesCommitOnce(t *testing.T) {
	for _, sameReceipt := range []bool{true, false} {
		name := "matching receipt"
		if !sameReceipt {
			name = "independent stale revision"
		}
		t.Run(name, func(t *testing.T) {
			h := attachmentRecoveryFixture(t)
			first := attachmentDeleteCommand(t, h)
			second := first
			if !sameReceipt {
				key := uuid.New()
				second.RequestID = &key
			}
			outcomes := competingAttachmentWrites(t, h, [2]projectmutations.Command{first, second})
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			if sameReceipt {
				require.NoError(t, outcomes[1].err)
				require.NotNil(t, outcomes[0].value.Replayed)
				require.NotNil(t, outcomes[1].value.Replayed)
				require.NotEqual(t, *outcomes[0].value.Replayed, *outcomes[1].value.Replayed)
			} else {
				// The first deletion makes the target unavailable before CAS is read.
				require.ErrorIs(t, outcomes[1].err, projectmutations.ErrNotFound)
			}
			var receipts, effects int
			require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM mutation_receipts WHERE tool='workitem_attachment' AND action='delete'),(SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$1 AND payload->>'type'='attachment.activity.deleted')`, first.ParentID.String()).Scan(&receipts, &effects))
			require.Equal(t, 1, receipts)
			require.Equal(t, 1, effects)
		})
	}
}

func TestAttachmentCancelledLockWaitLeavesReceiptReusable(t *testing.T) {
	for _, action := range []string{"create", "delete"} {
		t.Run(action, func(t *testing.T) {
			h := attachmentRecoveryFixture(t)
			command := attachmentImportCommand(t, h)
			if action == "delete" {
				command = attachmentDeleteCommand(t, h)
			}
			before, fetched := attachmentStorageSnapshot(t, h.milestoneFixture), h.fetched.Load()
			cancelProjectWriteWhileLocked(t, h.milestoneFixture, command.ProjectID, func(ctx context.Context) error { _, err := h.application.MutateResource(ctx, command); return err })
			require.Equal(t, before, attachmentStorageSnapshot(t, h.milestoneFixture))
			require.Equal(t, fetched, h.fetched.Load())
			result, err := h.application.MutateResource(t.Context(), command)
			require.NoError(t, err)
			require.NotNil(t, result.Replayed)
			require.False(t, *result.Replayed)
			after := attachmentStorageSnapshot(t, h.milestoneFixture)
			result, err = h.application.MutateResource(t.Context(), command)
			require.NoError(t, err)
			require.NotNil(t, result.Replayed)
			require.True(t, *result.Replayed)
			require.Equal(t, after, attachmentStorageSnapshot(t, h.milestoneFixture))
		})
	}
}
