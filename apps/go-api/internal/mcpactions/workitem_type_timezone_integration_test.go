//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkitemTypeTimestampsAndReplayKeepUserTimezone(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'`)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		receipt := milestoneObject(t, call("workitem_type", input))
		record := receipt["result"].(map[string]any)
		assertCatalogueTimezone(t, record)
		require.Equal(t, record, milestoneObject(t, call("workitem_type", input))["result"])
		require.Equal(t, record, milestoneObject(t, call("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": record["id"]})))
		for _, row := range catalogueArray(t, call("workitem_type", map[string]any{"action": "list"})) {
			assertCatalogueTimezone(t, row)
		}
		assertCatalogueTimezone(t, milestoneObject(t, call("workitem_type", map[string]any{"action": "update", "workitem_type_id": record["id"], "description": "Changed"})))
		_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='UTC'`)
		require.NoError(t, err)
		require.Equal(t, record, milestoneObject(t, call("workitem_type", input))["result"], "replay retains the accepted timezone after preferences change")
		_, err = f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'`)
		require.NoError(t, err)
	})
}
