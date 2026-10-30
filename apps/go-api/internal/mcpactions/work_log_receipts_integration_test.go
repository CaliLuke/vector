//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkLogReceiptsAndNativeRevisions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		input := map[string]any{"action": "create", "duration": 30, "description": "Initial", "request_id": uuid.NewString()}
		created := milestoneObject(t, call("work_log", input))
		id, ok := created["work_log_id"].(string)
		require.True(t, ok, created)
		record, ok := created["result"].(map[string]any)
		require.True(t, ok, created)
		require.Equal(t, id, record["id"])
		require.Equal(t, testsupport.LabelActor, record["logged_by"])
		require.Equal(t, testsupport.LabelActor, record["created_by"])
		require.Equal(t, testsupport.LabelWorkspace, record["workspace_id"])
		require.Equal(t, testsupport.LabelProject, record["project_id"])
		require.Equal(t, false, created["replayed"])
		require.NotEmpty(t, created["replay_expires_at"])
		before := workLogStorageSnapshot(t, f)
		replay := milestoneObject(t, call("work_log", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, created["revision"], replay["revision"])
		require.Equal(t, before, workLogStorageSnapshot(t, f))
		conflicting := maps.Clone(input)
		conflicting["duration"] = 31
		milestoneFailure(t, call("work_log", conflicting), "request_conflict")
		require.Equal(t, before, workLogStorageSnapshot(t, f))

		// The pinned handler omits zero duration and empty descriptions on update.
		unchanged := milestoneObject(t, call("work_log", map[string]any{"action": "update", "work_log_id": id, "duration": 0, "description": "", "expected_revision": created["revision"]}))
		require.Equal(t, created["result"], unchanged["result"])
		require.Equal(t, created["revision"], unchanged["revision"])
		require.Equal(t, before, workLogStorageSnapshot(t, f))
		rows := catalogueArray(t, call("work_log", map[string]any{"action": "list", "include_revision": true}))
		require.Len(t, rows, 1)
		require.Equal(t, created["revision"], rows[0]["revision"])

		_, err := f.pool.Exec(t.Context(), `UPDATE work_logs SET duration=45,description='Native edit' WHERE id=$1`, id)
		require.NoError(t, err)
		native := workLogStorageSnapshot(t, f)
		stale := map[string]any{"action": "update", "work_log_id": id, "duration": 60, "expected_revision": created["revision"], "request_id": uuid.NewString()}
		milestoneFailure(t, call("work_log", stale), "revision_conflict")
		require.Equal(t, native, workLogStorageSnapshot(t, f))
		rows = catalogueArray(t, call("work_log", map[string]any{"action": "list", "include_revision": true}))
		require.NotEqual(t, created["revision"], rows[0]["revision"])
		require.Equal(t, float64(45), rows[0]["duration"])
		totals := catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}))
		require.Contains(t, totals, map[string]any{"issue_id": item, "duration": float64(2700)})
		stale["expected_revision"] = rows[0]["revision"]
		updated := milestoneObject(t, call("work_log", stale))
		require.Equal(t, false, updated["replayed"], "rejected CAS must leave request ID reusable")
		require.Equal(t, float64(60), updated["result"].(map[string]any)["duration"])
		after := workLogStorageSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call("work_log", stale))["replayed"])
		require.Equal(t, after, workLogStorageSnapshot(t, f))

		deletion := map[string]any{"action": "delete", "work_log_id": id, "expected_revision": updated["revision"], "request_id": uuid.NewString()}
		deleted := milestoneObject(t, call("work_log", deletion))
		require.Nil(t, deleted["result"])
		require.NotEqual(t, updated["revision"], deleted["revision"])
		after = workLogStorageSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call("work_log", deletion))["replayed"])
		require.Equal(t, after, workLogStorageSnapshot(t, f))
		require.Empty(t, catalogueArray(t, call("work_log", map[string]any{"action": "list"})))
		totals = catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}))
		for _, total := range totals {
			require.NotEqual(t, item, total["issue_id"])
		}
		var effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$1 AND payload->>'type' LIKE 'work_log.activity.%'`, item).Scan(&effects))
		require.Equal(t, 3, effects, "create, changed update and delete each enqueue one activity")
	})
}

func workLogStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'logs',(SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM work_logs l),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY work_log_id) FROM work_log_edit_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,action,request_id) FROM mutation_receipts r WHERE tool='work_log'),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e WHERE payload->>'type' LIKE 'work_log.activity.%'))`).Scan(&snapshot))
	return snapshot
}
