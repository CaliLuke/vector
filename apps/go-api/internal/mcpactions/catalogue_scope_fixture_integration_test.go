//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// addPrivateProjectInOtherWorkspace grants the fixture actor current membership
// in both workspaces, so scope denials cannot be explained by missing access.
func addPrivateProjectInOtherWorkspace(t *testing.T, f milestoneFixture) (string, string) {
	t.Helper()
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
	return otherWorkspace, otherProject
}
