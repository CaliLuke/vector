//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPropertyCompetingMutationsCommitOnce(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
		for _, mode := range []string{"same receipt", "competing revision"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				h := newPropertyRecoveryCase(t, action)
				keys := [2]uuid.UUID{h.key, h.key}
				if mode == "competing revision" {
					keys[1] = uuid.New()
				}
				var calls [2]func(context.Context) (projectmutations.Result, error)
				for i, key := range keys {
					calls[i] = func(ctx context.Context) (projectmutations.Result, error) { return h.mutate(ctx, key) }
				}
				outcomes := competingScopeWrites(t, h.milestoneFixture, schemaMutationLock, uuid.MustParse(testsupport.LabelWorkspace), calls)
				if outcomes[0].err != nil {
					outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
				}
				require.NoError(t, outcomes[0].err)
				if mode == "same receipt" {
					require.NoError(t, outcomes[1].err)
					assertConcurrentReceiptReplay(t, outcomes[0].value, outcomes[1].value)
				} else {
					require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
					require.Equal(t, projectmutations.Result{}, outcomes[1].value)
				}
				var receipts int
				require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool=$1`, h.tool).Scan(&receipts))
				require.Equal(t, 1, receipts)
				require.NotNil(t, outcomes[0].value.RequestID)
				assertPropertyAcceptedReplay(t, h, *outcomes[0].value.RequestID, outcomes[0].value)
			})
		}
	}
}

func TestPropertyCancelledMutationsLeaveReceiptReusable(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
		t.Run(action, func(t *testing.T) {
			h := newPropertyRecoveryCase(t, action)
			before := propertyStorageSnapshot(t, h.milestoneFixture)
			var cancelledResult projectmutations.Result
			cancelScopeWriteWhileLocked(t, h.milestoneFixture, schemaMutationLock, uuid.MustParse(testsupport.LabelWorkspace), func(ctx context.Context) error {
				var err error
				cancelledResult, err = h.mutate(ctx, h.key)
				return err
			})
			require.Equal(t, projectmutations.Result{}, cancelledResult)
			require.Equal(t, before, propertyStorageSnapshot(t, h.milestoneFixture))
			accepted, err := h.mutate(t.Context(), h.key)
			require.NoError(t, err)
			require.NotNil(t, accepted.Replayed)
			require.False(t, *accepted.Replayed)
			require.NotEqual(t, before, propertyStorageSnapshot(t, h.milestoneFixture))
			assertPropertyAcceptedReplay(t, h, h.key, accepted)
		})
	}
}

func TestPropertyRejectedCommitsRollbackMutationAndReceipt(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
		t.Run(action, func(t *testing.T) {
			h := newPropertyRecoveryCase(t, action)
			_, err := h.pool.Exec(t.Context(), `CREATE FUNCTION reject_property_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected property commit rejection'; END $$;
CREATE CONSTRAINT TRIGGER reject_property_commit AFTER INSERT OR UPDATE OR DELETE ON `+pgx.Identifier{h.table}.Sanitize()+` DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_property_commit()`)
			require.NoError(t, err)
			before := propertyStorageSnapshot(t, h.milestoneFixture)
			result, err := h.mutate(t.Context(), h.key)
			require.ErrorIs(t, err, mutationreceipts.ErrIndeterminate)
			require.Equal(t, projectmutations.Result{}, result)
			require.Equal(t, before, propertyStorageSnapshot(t, h.milestoneFixture))
			_, err = h.pool.Exec(t.Context(), `DROP TRIGGER reject_property_commit ON `+pgx.Identifier{h.table}.Sanitize()+`; DROP FUNCTION reject_property_commit()`)
			require.NoError(t, err)
			accepted, err := h.mutate(t.Context(), h.key)
			require.NoError(t, err)
			require.NotNil(t, accepted.Replayed)
			require.False(t, *accepted.Replayed)
			require.NotEqual(t, before, propertyStorageSnapshot(t, h.milestoneFixture))
			assertPropertyAcceptedReplay(t, h, h.key, accepted)
		})
	}
}

func assertPropertyAcceptedReplay(t *testing.T, h propertyRecoveryCase, key uuid.UUID, accepted projectmutations.Result) {
	t.Helper()
	saved := propertyStorageSnapshot(t, h.milestoneFixture)
	replay, err := h.mutate(t.Context(), key)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	require.Equal(t, &key, replay.RequestID)
	accepted.RequestID, accepted.Replayed = &key, replay.Replayed
	require.Equal(t, accepted, replay)
	require.Equal(t, saved, propertyStorageSnapshot(t, h.milestoneFixture))
}
