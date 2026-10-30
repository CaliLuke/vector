//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectFeaturesBindTheSelectedWorkspace(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := uuid.NewString(), uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
 VALUES($1::uuid,'Other',$2,'other','UTC','',now(),now())`, otherWorkspace, testsupport.LabelActor)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features)
 VALUES(gen_random_uuid(),$1,$2,20,now(),now(),'{}','{}','{}',true,'{}','{}','{}')`, otherWorkspace, testsupport.LabelActor)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at)
 VALUES($1,'Other','',0,$2,'OTHER',false,false,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, otherProject, otherWorkspace)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO project_members(id,workspace_id,project_id,member_id,role,created_at,updated_at,view_props,default_props,sort_order,preferences,is_active)
 VALUES(gen_random_uuid(),$1,$2,$3,20,now(),now(),'{}','{}',0,'{}',true)`, otherWorkspace, otherProject, testsupport.LabelActor)
	require.NoError(t, err)
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) {
				handler, token := f.handler(t, nil), "test-one"
				if mode == "OAuth" {
					handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
				}
				call := func(input map[string]any) rpcResult {
					raw, marshalErr := json.Marshal(input)
					require.NoError(t, marshalErr)
					return oauthToolCall(t, handler, version, token, "", "project", string(raw), http.StatusOK)
				}
				key := uuid.NewString()
				input := map[string]any{"action": "update_features", "workspace_slug": "other", "project_id": otherProject, "cycles": true, "request_id": key}
				accepted := milestoneObject(t, call(input))
				require.Equal(t, otherProject, accepted["resource_id"])
				require.Equal(t, false, accepted["replayed"])
				before := featureMutationSnapshot(t, f)
				otherBefore := featureMutationSnapshotForWorkspace(t, f, otherWorkspace)
				for _, scope := range []struct{ workspace, project string }{{"workspace-one", otherProject}, {"other", testsupport.LabelProject}} {
					milestoneFailure(t, call(map[string]any{"action": "get_features", "workspace_slug": scope.workspace, "project_id": scope.project}), "forbidden")
					milestoneFailure(t, call(map[string]any{"action": "update_features", "workspace_slug": scope.workspace, "project_id": scope.project, "cycles": true, "request_id": key}), "forbidden")
				}
				require.JSONEq(t, string(before), string(featureMutationSnapshot(t, f)))
				require.JSONEq(t, string(otherBefore), string(featureMutationSnapshotForWorkspace(t, f, otherWorkspace)))
				replay := milestoneObject(t, call(input))
				require.Equal(t, true, replay["replayed"])
				require.Equal(t, accepted["result"], replay["result"])
				require.JSONEq(t, string(otherBefore), string(featureMutationSnapshotForWorkspace(t, f, otherWorkspace)))
			})
		}
	}
}
