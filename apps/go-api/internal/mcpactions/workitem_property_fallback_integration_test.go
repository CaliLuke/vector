//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The pinned handler returns the fallback response directly. It must be the
// same representation as requesting that successful scope on its own.
func TestMCPPropertyEmptyTypeListReturnsProjectFlatRepresentation(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		linked := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		empty := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		created := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": linked["id"], "display_name": uuid.NewString(), "property_type": "DATETIME", "display_format": "yyyy/MM/dd"}))
		scoped := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_type_id": linked["id"]}))
		require.Len(t, scoped, 1)
		require.Equal(t, created["id"], scoped[0]["id"])
		require.Equal(t, linked["id"], scoped[0]["issue_type"])
		flat := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject}))
		require.NotEmpty(t, flat)
		fallback := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_type_id": empty["id"]}))
		require.Equal(t, flat, fallback, "falling back must not relabel flat properties as belonging to the empty requested type")
	})
}

func TestMCPPropertyUnimportedTypeListReturnsWorkspaceRepresentation(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		workspaceType := milestoneObject(t, root("workitem_type", map[string]any{"action": "create", "name": uuid.NewString()}))
		created := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": workspaceType["id"], "display_name": uuid.NewString(), "property_type": "TEXT"}))
		second := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": workspaceType["id"], "display_name": uuid.NewString(), "property_type": "TEXT"}))
		flat := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject}))
		require.Empty(t, flat, "fixture has no project-visible property associations")
		workspace := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "workitem_type_id": workspaceType["id"]}))
		require.Len(t, workspace, 2)
		require.ElementsMatch(t, []any{created["id"], second["id"]}, []any{workspace[0]["id"], workspace[1]["id"]})
		require.Nil(t, workspace[0]["project"])
		input := map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_type_id": workspaceType["id"], "per_page": 1}
		fallback := catalogueArray(t, root("workitem_property", input))
		require.Equal(t, workspace, fallback, "workspace fallback must retain the complete, unpaged workspace representation")
		_, err := f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		denied := root("workitem_property", input)
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
		require.NoError(t, err)
		milestoneFailure(t, denied, "forbidden")
	})
}

func TestMCPPropertyEmptyPageUsesFallbackResponse(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		selected := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		other := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": selected["id"], "display_name": uuid.NewString(), "property_type": "TEXT"}))
		milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": other["id"], "display_name": uuid.NewString(), "property_type": "TEXT"}))
		flatPage := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "per_page": 1, "cursor": "1:1:0"}))
		require.Len(t, flatPage, 1)
		fallback := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_type_id": selected["id"], "per_page": 1, "cursor": "1:1:0"}))
		require.Equal(t, flatPage, fallback, "an empty requested type page falls back to the same project-flat page")
		workspace := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "workitem_type_id": selected["id"]}))
		require.Len(t, workspace, 1)
		exhausted := catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "project_id": testsupport.LabelProject, "workitem_type_id": selected["id"], "per_page": 1, "cursor": "1:999:0"}))
		require.Equal(t, workspace, exhausted, "empty project pages fall back to the unpaged workspace type list")
	})
}
