//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPInitiativeReadsBindWorkspaceAndFilterTargetsBeforeCounting(t *testing.T) {
	f := initiativeFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	runCatalogueCredentials(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		foreignID := uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO initiatives(id,workspace_id,name,created_by_id) VALUES($1,$2,'Foreign initiative',$3)`, foreignID, otherWorkspace, testsupport.LabelActor)
		require.NoError(t, err)
		lead, setActive := customerReferenceTarget(t, f, "USER")
		created := milestoneObject(t, call("initiative", map[string]any{"action": "create", "name": uuid.NewString(), "lead": lead}))
		id := created["id"]
		require.NotContains(t, templateIDs(catalogueArray(t, call("initiative", map[string]any{"action": "list"}))), foreignID)
		milestoneFailure(t, call("initiative", map[string]any{"action": "retrieve", "initiative_id": foreignID}), "not_found")
		input := map[string]any{"action": "update", "initiative_id": foreignID, "name": "Cross-tenant update", "request_id": uuid.NewString()}
		before := initiativeSnapshot(t, f)
		milestoneFailure(t, call("initiative", input), "not_found")
		require.Equal(t, before, initiativeSnapshot(t, f))
		item := f.item(t)
		catalogueNull(t, call("initiative", map[string]any{"action": "manage_workitems", "initiative_id": id, "add_ids": item}))
		projects := milestoneObject(t, call("initiative", map[string]any{"action": "list_projects", "initiative_id": id}))
		require.Empty(t, projects["results"], "item links do not create project links")
		catalogueNull(t, call("initiative", map[string]any{"action": "add_projects", "initiative_id": id, "project_ids": testsupport.LabelProject}))
		for _, action := range []string{"list_projects", "list_workitems"} {
			page := milestoneObject(t, call("initiative", map[string]any{"action": action, "initiative_id": id, "per_page": 1, "include_revision": true}))
			require.Len(t, page["results"], 1)
			require.EqualValues(t, 1, page["total_count"])
			require.NotEmpty(t, page["revision"])
		}
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET network=0 WHERE id=$1;`, testsupport.LabelProject)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		for _, action := range []string{"list_projects", "list_workitems"} {
			page := milestoneObject(t, call("initiative", map[string]any{"action": action, "initiative_id": id, "per_page": 1}))
			require.Empty(t, page["results"])
			require.EqualValues(t, 0, page["total_count"], "counts must not disclose private targets")
		}
		setActive(false)
		detail := milestoneObject(t, call("initiative", map[string]any{"action": "retrieve", "initiative_id": id}))
		require.Nil(t, detail["lead"], "revoked lead membership must not leak a stale reference")
		setActive(true)
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
	})
}
