//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTypeCompetingMutationsCommitOnce(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "resolve", "import_to_project"} {
		for _, mode := range []string{"same receipt", "competing revision"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				f, repository, first := typeMutationFixture(t, action)
				second := first
				if mode == "competing revision" {
					key := uuid.New()
					second.RequestID = &key
				}
				var calls [2]func(context.Context) (projectmutations.Result, error)
				for i, command := range []workitemtypes.APIMutation{first, second} {
					calls[i] = func(ctx context.Context) (projectmutations.Result, error) { return repository.MutateAPI(ctx, command) }
				}
				outcomes := competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, calls)
				if outcomes[0].err != nil {
					outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
				}
				require.NoError(t, outcomes[0].err)
				if mode == "same receipt" {
					require.NoError(t, outcomes[1].err)
					assertConcurrentReceiptReplay(t, outcomes[0].value, outcomes[1].value)
				} else {
					require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
				}
				var receipts int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool='workitem_type'`).Scan(&receipts))
				require.Equal(t, 1, receipts)
				accepted := first
				if *outcomes[0].value.RequestID == *second.RequestID {
					accepted = second
				}
				snapshot := typeStorageSnapshot(t, f)
				replay, err := repository.MutateAPI(t.Context(), accepted)
				require.NoError(t, err)
				require.NotNil(t, replay.Replayed)
				require.True(t, *replay.Replayed)
				require.Equal(t, outcomes[0].value.Record, replay.Record)
				require.Equal(t, outcomes[0].value.Revision, replay.Revision)
				require.Equal(t, snapshot, typeStorageSnapshot(t, f))
			})
		}
	}
}
