//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCycleTransferSnapshotExcludesForeignAssociations(t *testing.T) {
	f := planningAuthorizationWaitFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=true`)
	require.NoError(t, err)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
		_, source := createPlanningReceipt(t, call, "cycle")
		_, target := createPlanningReceipt(t, call, "cycle")
		sourceID, targetID := source["cycle_id"].(string), target["cycle_id"].(string)
		estimateID, pointID := uuid.NewString(), uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Transfer points '||gen_random_uuid()::text,'','points',true,now(),now())`, estimateID, testsupport.LabelWorkspace, testsupport.LabelProject)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'2','',now(),now())`, pointID, testsupport.LabelWorkspace, testsupport.LabelProject, estimateID)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET estimate_id=$1 WHERE id=$2`, estimateID, testsupport.LabelProject)
		require.NoError(t, err)
		var local, unfinished []string
		for _, group := range []string{"backlog", "unstarted", "started", "completed", "cancelled"} {
			state := milestoneObject(t, call("state", map[string]any{"action": "create", "name": uuid.NewString(), "color": "#123456", "group": group}))
			item := f.item(t)
			_, err = f.pool.Exec(t.Context(), `UPDATE issues SET state_id=$1,estimate_point_id=$3 WHERE id=$2`, state["id"], item, pointID)
			require.NoError(t, err)
			if group == "completed" {
				_, err = f.pool.Exec(t.Context(), `UPDATE issues SET completed_at='2000-12-31' WHERE id=$1`, item)
				require.NoError(t, err)
			}
			local = append(local, item)
			if group != "completed" && group != "cancelled" {
				unfinished = append(unfinished, item)
			}
		}
		encoded, err := json.Marshal(local)
		require.NoError(t, err)
		catalogueNull(t, call("cycle", map[string]any{"action": "manage_workitems", "cycle_id": sourceID, "add_ids": string(encoded)}))
		foreignItem := f.itemInScope(t, otherWorkspace, otherProject)
		wrongScopeItem := f.item(t)
		_, err = f.pool.Exec(t.Context(), `INSERT INTO cycle_issues(id,workspace_id,project_id,cycle_id,issue_id,created_at,updated_at)
 VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now()),(gen_random_uuid(),$5,$6,$3,$7,now(),now())`, testsupport.LabelWorkspace, testsupport.LabelProject, sourceID, foreignItem, otherWorkspace, otherProject, wrongScopeItem)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE cycles SET start_date='2000-12-30',end_date='2001-01-01' WHERE id=$1`, sourceID)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET completed_at='2000-12-31',estimate_point_id=$2 WHERE id=ANY($1::uuid[])`, []string{foreignItem, wrongScopeItem}, pointID)
		require.NoError(t, err)
		foreignBefore := planningAssociationSnapshot(t, f, "cycle", []string{foreignItem, wrongScopeItem}, nil)
		input := map[string]any{"action": "transfer_workitems", "cycle_id": sourceID, "new_cycle_id": targetID, "request_id": uuid.NewString()}
		accepted := milestoneObject(t, call("cycle", input))
		require.Equal(t, false, accepted["replayed"])
		var moved []string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT array_agg(issue_id::text ORDER BY issue_id) FROM cycle_issues WHERE cycle_id=$1 AND deleted_at IS NULL`, targetID).Scan(&moved))
		require.ElementsMatch(t, unfinished, moved)
		require.Equal(t, foreignBefore, planningAssociationSnapshot(t, f, "cycle", []string{foreignItem, wrongScopeItem}, nil))
		var stored []byte
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT progress_snapshot FROM cycles WHERE id=$1`, sourceID).Scan(&stored))
		var snapshot map[string]any
		require.NoError(t, json.Unmarshal(stored, &snapshot))
		detail := milestoneObject(t, call("cycle", map[string]any{"action": "retrieve", "cycle_id": sourceID}))
		require.Equal(t, snapshot, detail["progress_snapshot"])
		for name, count := range map[string]float64{"total_issues": 5, "backlog_issues": 1, "unstarted_issues": 1, "started_issues": 1, "completed_issues": 1, "cancelled_issues": 1} {
			require.Equal(t, count, snapshot[name], name)
		}
		for _, metric := range []struct {
			field, suffix string
			multiplier    float64
		}{
			{"distribution", "issues", 1}, {"estimate_distribution", "estimates", 2},
		} {
			distribution := snapshot[metric.field].(map[string]any)
			for _, dimension := range []string{"labels", "assignees"} {
				rows := distribution[dimension].([]any)
				require.Len(t, rows, 1)
				bucket := rows[0].(map[string]any)
				require.Equal(t, 5*metric.multiplier, bucket["total_"+metric.suffix], dimension)
				require.Equal(t, metric.multiplier, bucket["completed_"+metric.suffix], dimension)
			}
			require.Equal(t, map[string]any{"2000-12-30": 5 * metric.multiplier, "2000-12-31": 4 * metric.multiplier, "2001-01-01": 4 * metric.multiplier}, distribution["completion_chart"])
		}
		beforeReplay := planningMutationSnapshot(t, f)
		replay := milestoneObject(t, call("cycle", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, accepted["revision"], replay["revision"])
		require.Equal(t, beforeReplay, planningMutationSnapshot(t, f))
	})
}
