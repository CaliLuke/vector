//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPInitiativeFeatureGatePreservesReceiptRecovery(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		setEnabled := func(enabled bool) {
			_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET initiatives=$1`, enabled)
			require.NoError(t, err)
		}
		snapshot := func() [3]int64 {
			var state [3]int64
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT
                (SELECT count(*) FROM initiatives WHERE name=$1),
                (SELECT count(*) FROM initiative_events WHERE input->'fields'->>'name'=$1),
                (SELECT count(*) FROM mutation_receipts WHERE request_id=$2)`, input["name"], input["request_id"]).Scan(&state[0], &state[1], &state[2]))
			return state
		}
		setEnabled(false)
		milestoneFailure(t, call("initiative", input), "feature_disabled")
		require.Equal(t, [3]int64{}, snapshot(), "disabled writes must not consume a receipt or leave domain/audit data")
		setEnabled(true)
		accepted := milestoneObject(t, call("initiative", input))
		require.Equal(t, false, accepted["replayed"])
		require.Equal(t, [3]int64{1, 1, 1}, snapshot())
		setEnabled(false)
		milestoneFailure(t, call("initiative", input), "feature_disabled")
		milestoneFailure(t, call("initiative", map[string]any{"action": "retrieve", "initiative_id": accepted["resource_id"]}), "feature_disabled")
		require.Equal(t, [3]int64{1, 1, 1}, snapshot(), "revoked feature access must preserve the accepted receipt and data")
		setEnabled(true)
		accepted["replayed"] = true
		require.Equal(t, accepted, milestoneObject(t, call("initiative", input)))
		require.Equal(t, [3]int64{1, 1, 1}, snapshot(), "replay must not repeat the mutation or audit")
	})
}
