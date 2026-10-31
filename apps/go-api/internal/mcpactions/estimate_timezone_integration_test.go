//go:build integration

package mcpactions_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateTimestampsFollowUserTimezone(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'`)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		receipt := milestoneObject(t, call("project_estimate", input))
		estimate := receipt["estimate"].(map[string]any)
		assertEstimateTimezone(t, estimate)
		require.Equal(t, estimate, milestoneObject(t, call("project_estimate", input))["estimate"])
		id := estimate["id"]
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		require.Equal(t, estimate, milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve"})))
		assertEstimateTimezone(t, milestoneObject(t, call("project_estimate", map[string]any{"action": "update", "name": uuid.NewString()})))
		pointsInput := map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"value":"1"}]`, "request_id": uuid.NewString()}
		pointsReceipt := milestoneObject(t, call("project_estimate", pointsInput))
		point := pointsReceipt["points"].([]any)[0].(map[string]any)
		assertEstimateTimezone(t, point)
		require.Equal(t, pointsReceipt["points"], milestoneObject(t, call("project_estimate", pointsInput))["points"])
		rows := catalogueArray(t, call("project_estimate", map[string]any{"action": "list_points", "estimate_id": id}))
		require.Equal(t, []map[string]any{point}, rows)
		assertEstimateTimezone(t, milestoneObject(t, call("project_estimate", map[string]any{"action": "update_point", "estimate_id": id, "estimate_point_id": point["id"], "value": "2"})))
	})
}

func assertEstimateTimezone(t *testing.T, record map[string]any) {
	t.Helper()
	for _, field := range []string{"created_at", "updated_at"} {
		stamp, err := time.Parse(time.RFC3339Nano, record[field].(string))
		require.NoError(t, err)
		_, offset := stamp.Zone()
		require.Equal(t, 19800, offset, field)
	}
}
