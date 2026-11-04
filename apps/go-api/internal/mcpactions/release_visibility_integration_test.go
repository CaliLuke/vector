//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPReleaseReadsPreserveWorkspaceOwnershipAndPrivateItemCounts(t *testing.T) {
	f := templateFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	foreign, projectRelease := uuid.NewString(), uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,NULL,'Foreign'),($3,$4,$5,'Project only')`, foreign, otherWorkspace, projectRelease, testsupport.LabelWorkspace, testsupport.LabelProject)
	require.NoError(t, err)
	runCatalogueCredentials(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		created := milestoneObject(t, call("release", map[string]any{"action": "create", "name": uuid.NewString()}))
		id := created["id"]
		page := milestoneObject(t, call("release", map[string]any{"action": "list"}))
		for _, value := range page["results"].([]any) {
			row := value.(map[string]any)
			require.NotEqual(t, foreign, row["id"])
			require.NotEqual(t, projectRelease, row["id"])
		}
		for _, hidden := range []string{foreign, projectRelease} {
			milestoneFailure(t, call("release", map[string]any{"action": "retrieve", "release_id": hidden}), "not_found")
			before := releaseSnapshot(t, f)
			milestoneFailure(t, call("release", map[string]any{"action": "update", "release_id": hidden, "name": "Forbidden", "request_id": uuid.NewString()}), "not_found")
			require.Equal(t, before, releaseSnapshot(t, f))
		}
		item := f.item(t)
		catalogueNull(t, call("release", map[string]any{"action": "manage_workitems", "release_id": id, "add_ids": item}))
		list := map[string]any{"action": "list_workitems", "release_id": id, "per_page": 1}
		page = milestoneObject(t, call("release", list))
		require.EqualValues(t, 1, page["total_count"])
		require.Equal(t, item, page["results"].([]any)[0].(map[string]any)["id"])
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		page = milestoneObject(t, call("release", list))
		require.Empty(t, page["results"])
		require.EqualValues(t, 0, page["total_count"])
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		// A restricted guest can see only their own submission; counts obey the same rule.
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		page = milestoneObject(t, call("release", list))
		require.EqualValues(t, 1, page["total_count"])
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET created_by_id=NULL WHERE id=$1`, item)
		require.NoError(t, err)
		page = milestoneObject(t, call("release", list))
		require.Empty(t, page["results"])
		require.EqualValues(t, 0, page["total_count"])
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
	})
}
