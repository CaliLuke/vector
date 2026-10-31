//go:build integration

package mcpactions_test

import (
	"maps"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateReceiptsAndNativePointRevisions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		input := map[string]any{"action": "create", "name": "Receipt sizing", "request_id": uuid.NewString()}
		created := milestoneObject(t, call("project_estimate", input))
		require.Equal(t, false, created["replayed"])
		estimate := created["estimate"].(map[string]any)
		id := estimate["id"].(string)
		snapshot := estimateStorageSnapshot(t, f)
		replay := milestoneObject(t, call("project_estimate", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["estimate"], replay["estimate"])
		require.Equal(t, created["revision"], replay["revision"])
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		changed := maps.Clone(input)
		changed["name"] = "Different"
		milestoneFailure(t, call("project_estimate", changed), "request_conflict")
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		pointsInput := map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"value":"1","key":0},{"value":"2","key":1}]`, "request_id": uuid.NewString()}
		pointsReceipt := milestoneObject(t, call("project_estimate", pointsInput))
		points := pointsReceipt["points"].([]any)
		require.Len(t, points, 2)
		point := points[0].(map[string]any)["id"].(string)
		snapshot = estimateStorageSnapshot(t, f)
		replay = milestoneObject(t, call("project_estimate", pointsInput))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, pointsReceipt["points"], replay["points"])
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		before := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		repository, err := projectconfiguration.NewRepository(f.pool)
		require.NoError(t, err)
		_, err = repository.PatchEstimatePoint(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, id, point, strings.NewReader(`{"value":"3"}`), "application/json")
		require.NoError(t, err)
		update := map[string]any{"action": "update", "name": "After native edit", "expected_revision": before["revision"], "request_id": uuid.NewString()}
		snapshot = estimateStorageSnapshot(t, f)
		milestoneFailure(t, call("project_estimate", update), "revision_conflict")
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		after := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		require.NotEqual(t, before["revision"], after["revision"])
		update["expected_revision"] = after["revision"]
		updated := milestoneObject(t, call("project_estimate", update))
		require.Equal(t, false, updated["replayed"])
		require.Equal(t, "After native edit", updated["estimate"].(map[string]any)["name"])
		deletion := map[string]any{"action": "delete", "request_id": uuid.NewString(), "expected_revision": updated["revision"]}
		deleted := milestoneObject(t, call("project_estimate", deletion))
		require.Equal(t, false, deleted["replayed"])
		require.Nil(t, deleted["estimate"])
		snapshot = estimateStorageSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call("project_estimate", deletion))["replayed"])
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		// A stored result cannot restore the caller's lost write permission.
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		milestoneFailure(t, call("project_estimate", input), "forbidden")
		milestoneFailure(t, call("project_estimate", pointsInput), "forbidden")
		milestoneFailure(t, call("project_estimate", deletion), "forbidden")
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
	})
}

func estimateStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'projects',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p),
 'project_revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_edit_revisions r),
 'estimates',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM estimates e),
 'points',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM estimate_points p),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_estimate_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,tool,action,request_id) FROM mutation_receipts r WHERE tool IN ('project_estimate','project')),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`).Scan(&snapshot))
	return snapshot
}
