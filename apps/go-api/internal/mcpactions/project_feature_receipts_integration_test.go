//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectFeatureReceiptsBindProjectAndRespectNativeRevisions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=false,module_view=false WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		other := uuid.NewString()
		_, err = f.pool.Exec(t.Context(), `INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at)
 SELECT $1::uuid,'Other ' || $2,'',0,workspace_id,$2,false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now() FROM projects WHERE id=$3`, other, other[:8], testsupport.LabelProject)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `INSERT INTO project_members(id,workspace_id,project_id,member_id,role,created_at,updated_at,view_props,default_props,sort_order,preferences,is_active)
 SELECT gen_random_uuid(),workspace_id,$1::uuid,member_id,20,now(),now(),'{}','{}',0,'{}',true FROM project_members WHERE project_id=$2 AND member_id=$3`, other, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		key := uuid.NewString()
		firstInput := map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": true, "request_id": key}
		otherInput := map[string]any{"action": "update_features", "project_id": other, "modules": true, "request_id": key}
		first := milestoneObject(t, call("project", firstInput))
		second := milestoneObject(t, call("project", otherInput))
		require.Equal(t, testsupport.LabelProject, first["resource_id"])
		require.Equal(t, other, second["resource_id"])
		require.Equal(t, false, first["replayed"])
		require.Equal(t, false, second["replayed"])
		beforeNative := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject, "include_revision": true}))
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=false WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		native := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject, "include_revision": true}))
		otherCurrent := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": other, "include_revision": true}))
		require.Equal(t, false, native["cycles"])
		require.NotEqual(t, beforeNative["revision"], native["revision"])
		snapshot := featureMutationSnapshot(t, f)
		milestoneFailure(t, call("project", map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": true, "expected_revision": beforeNative["revision"]}), "revision_conflict")
		milestoneFailure(t, call("project", map[string]any{"action": "update_features", "project_id": other, "modules": false, "expected_revision": native["revision"]}), "revision_conflict")
		conflict := maps.Clone(firstInput)
		conflict["cycles"] = false
		milestoneFailure(t, call("project", conflict), "request_conflict")
		replayFirst := milestoneObject(t, call("project", firstInput))
		replayOther := milestoneObject(t, call("project", otherInput))
		require.Equal(t, true, replayFirst["replayed"])
		require.Equal(t, true, replayOther["replayed"])
		require.Equal(t, first["result"], replayFirst["result"])
		require.Equal(t, second["result"], replayOther["result"])
		require.JSONEq(t, string(snapshot), string(featureMutationSnapshot(t, f)))
		accepted := milestoneObject(t, call("project", map[string]any{"action": "update_features", "project_id": other, "modules": false, "expected_revision": otherCurrent["revision"], "request_id": uuid.NewString()}))
		require.Equal(t, false, accepted["result"].(map[string]any)["modules"])
		require.Equal(t, false, accepted["replayed"])
	})
}
