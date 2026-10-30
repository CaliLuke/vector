//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningLifecycleReceiptsDoNotRepeatEffects(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		create, created := createPlanningReceipt(t, call, kind)
		id := created[kind+"_id"].(string)
		ready := map[string]any{"action": "update", kind + "_id": id}
		if kind == "cycle" {
			ready["end_date"] = "2000-01-01"
		} else {
			ready["status"] = "completed"
		}
		milestoneObject(t, call(kind, ready))
		archive := map[string]any{"action": "archive", kind + "_id": id, "request_id": uuid.NewString(), "expected_revision": planningDetailRevision(t, call, kind, id)}
		archived := milestoneObject(t, call(kind, archive))
		require.Nil(t, archived["result"])
		var storedArchived bool
		require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT archived_at IS NOT NULL FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", id).Scan(&storedArchived))
		require.True(t, storedArchived)
		restore := map[string]any{"action": "unarchive", kind + "_id": id, "request_id": uuid.NewString(), "expected_revision": archived["revision"]}
		restored := milestoneObject(t, call(kind, restore))
		require.NotEqual(t, archived["revision"], restored["revision"])
		require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT archived_at IS NOT NULL FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", id).Scan(&storedArchived))
		require.False(t, storedArchived)
		before := planningMutationSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call(kind, archive))["replayed"])
		require.Equal(t, before, planningMutationSnapshot(t, f), "replaying an old archive must preserve the subsequent restore")

		deletion := map[string]any{"action": "delete", kind + "_id": id, "request_id": uuid.NewString(), "expected_revision": restored["revision"]}
		deleted := milestoneObject(t, call(kind, deletion))
		require.Nil(t, deleted["result"])
		require.NotEqual(t, restored["revision"], deleted["revision"])
		var softDeleted bool
		require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT deleted_at IS NOT NULL FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", id).Scan(&softDeleted))
		require.True(t, softDeleted)
		milestoneFailure(t, call(kind, map[string]any{"action": "retrieve", kind + "_id": id}), "not_found")
		before = planningMutationSnapshot(t, f)
		for _, input := range []map[string]any{create, archive, restore, deletion} {
			require.Equal(t, true, milestoneObject(t, call(kind, input))["replayed"])
		}
		require.Equal(t, before, planningMutationSnapshot(t, f), "historical receipts must neither resurrect the resource nor repeat effects")
	})
}
