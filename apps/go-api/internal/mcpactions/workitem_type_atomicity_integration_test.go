//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestTypeCancelledMutationsLeaveReceiptReusable(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "resolve", "import_to_project"} {
		t.Run(action, func(t *testing.T) {
			f, repository, command := typeMutationFixture(t, action)
			before := typeStorageSnapshot(t, f)
			cancelScopeWriteWhileLocked(t, f, schemaMutationLock, command.WorkspaceID, func(ctx context.Context) error {
				_, err := repository.MutateAPI(ctx, command)
				return err
			})
			require.Equal(t, before, typeStorageSnapshot(t, f))
			assertTypeMutationRetry(t, f, repository, command)
		})
	}
}

func TestTypeRejectedCommitsRollbackMutationAndReceipt(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "resolve", "import_to_project"} {
		t.Run(action, func(t *testing.T) {
			f, repository, command := typeMutationFixture(t, action)
			table := "issue_types"
			if action == "resolve" || action == "import_to_project" {
				table = "project_issue_types"
			}
			_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_type_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected type commit rejection'; END $$;
 CREATE CONSTRAINT TRIGGER reject_type_commit AFTER INSERT OR UPDATE OR DELETE ON `+pgx.Identifier{table}.Sanitize()+` DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_type_commit()`)
			require.NoError(t, err)
			before := typeStorageSnapshot(t, f)
			result, err := repository.MutateAPI(t.Context(), command)
			require.ErrorIs(t, err, mutationreceipts.ErrIndeterminate)
			require.Equal(t, projectmutations.Result{}, result, "unacknowledged commit must not return a receipt or accepted representation")
			require.Equal(t, before, typeStorageSnapshot(t, f))
			_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_type_commit ON `+pgx.Identifier{table}.Sanitize()+`; DROP FUNCTION reject_type_commit()`)
			require.NoError(t, err)
			assertTypeMutationRetry(t, f, repository, command)
		})
	}
}

func assertTypeMutationRetry(t *testing.T, f milestoneFixture, repository *workitemtypes.Repository, command workitemtypes.APIMutation) {
	t.Helper()
	before := typeStorageSnapshot(t, f)
	accepted, err := repository.MutateAPI(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, accepted.Replayed)
	require.False(t, *accepted.Replayed)
	require.Equal(t, command.RequestID, accepted.RequestID)
	saved := typeStorageSnapshot(t, f)
	require.NotEqual(t, before, saved)
	replay, err := repository.MutateAPI(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	require.Equal(t, accepted.ID, replay.ID)
	require.Equal(t, accepted.Revision, replay.Revision)
	require.Equal(t, accepted.ReplayExpiresAt, replay.ReplayExpiresAt)
	require.Equal(t, accepted.Record, replay.Record)
	require.Equal(t, saved, typeStorageSnapshot(t, f))
}
