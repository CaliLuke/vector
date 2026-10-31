//go:build integration

package mcpactions_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateDeletionWorkerPreservesHistoryAndReplay(t *testing.T) {
	f := newMilestoneFixture(t)
	configureCatalogueWorker(t)
	var pointItems, systemItems, remainingPoints []string
	var pointRequests, systemRequests []map[string]any
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		estimate := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": uuid.NewString(), "type": "points"}))
		id := estimate["id"].(string)
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		points := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"key":0,"value":"1"},{"key":1,"value":"2"}]`}))
		item, otherItem := f.item(t), f.item(t)
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET estimate_point_id=CASE WHEN id=$1 THEN $3::uuid ELSE $4::uuid END WHERE id IN($1,$2)`, item, otherItem, points[0]["id"], points[1]["id"])
		require.NoError(t, err)
		pointRequest := map[string]any{"action": "delete_point", "estimate_id": id, "estimate_point_id": points[0]["id"], "request_id": uuid.NewString()}
		deleted := milestoneObject(t, call("project_estimate", pointRequest))
		require.Equal(t, false, deleted["replayed"])
		require.Nil(t, deleted["point"])
		var cleared bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT estimate_point_id IS NULL FROM issues WHERE id=$1`, item).Scan(&cleared))
		require.True(t, cleared, "point removal clears current values in the receipt transaction")
		remaining := catalogueArray(t, call("project_estimate", map[string]any{"action": "list_points", "estimate_id": id}))
		require.Len(t, remaining, 1)
		require.Equal(t, float64(0), remaining[0]["key"])
		systemRequest := map[string]any{"action": "delete", "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call("project_estimate", systemRequest))["replayed"])
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT estimate_point_id IS NULL FROM issues WHERE id=$1`, otherItem).Scan(&cleared))
		require.False(t, cleared, "the durable deletion worker owns system-child cleanup")
		pointItems, systemItems = append(pointItems, item), append(systemItems, otherItem)
		remainingPoints = append(remainingPoints, points[1]["id"].(string))
		pointRequests, systemRequests = append(pointRequests, pointRequest), append(systemRequests, systemRequest)
	})
	require.Len(t, pointItems, 4)
	var activities int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[]) AND field='estimate_points'`, pointItems).Scan(&activities))
	require.Zero(t, activities)
	startCatalogueWorker(t, f)
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var history, cleared, deleted int
		queryErr := f.pool.QueryRow(t.Context(), `SELECT
 (SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[]) AND field='estimate_points' AND verb='removed' AND old_value='1' AND new_value IS NULL AND old_identifier IS NOT NULL AND new_identifier IS NULL AND actor_id=$4),
 (SELECT count(*) FROM issues WHERE id=ANY($2::uuid[]) AND estimate_point_id IS NULL),
 (SELECT count(*) FROM estimate_points WHERE id=ANY($3::uuid[]) AND deleted_at IS NOT NULL)`, pointItems, systemItems, remainingPoints, testsupport.LabelActor).Scan(&history, &cleared, &deleted)
		assert.NoError(collect, queryErr)
		assert.Equal(collect, 4, history)
		assert.Equal(collect, 4, cleared)
		assert.Equal(collect, 4, deleted)
	}, 45*time.Second, 25*time.Millisecond)
	index := 0
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		require.Equal(t, true, milestoneObject(t, call("project_estimate", pointRequests[index]))["replayed"])
		require.Equal(t, true, milestoneObject(t, call("project_estimate", systemRequests[index]))["replayed"])
		index++
	})
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[]) AND field='estimate_points'`, pointItems).Scan(&activities))
	require.Equal(t, 4, activities)
}
