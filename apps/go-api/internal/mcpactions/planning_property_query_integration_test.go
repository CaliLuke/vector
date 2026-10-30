//go:build integration

package mcpactions_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningPropertyQueriesUseBoundDiscoverySnapshot(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
	require.NoError(t, err)
	repo := workitemtypes.NewRepository(f.pool)
	catalog, err := repo.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
	require.NoError(t, err)
	require.False(t, catalog.Enabled)
	_, err = repo.Mutate(t.Context(), testsupport.LabelActor, "workspace-one", "enable", "", workitemtypes.Input{Revision: catalog.Revision})
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, fixture milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		milestoneObject(t, call("project", map[string]any{"action": "update_features", "cycles": true, "modules": true}))
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) {
				_, parent := createPlanningReceipt(t, call, kind)
				parentID := parent[kind+"_id"].(string)
				typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": "Query " + uuid.NewString()}))
				property := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "display_name": "Query number " + uuid.NewString(), "property_type": "DECIMAL", "workitem_type_id": typeRow["id"]}))
				var items []string
				for i, value := range []int{0, 5, 0, 0} {
					item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "name": fmt.Sprintf("Property query %d", i), "type_id": typeRow["id"], "priority": "high"}))
					id := item["id"].(string)
					milestoneObject(t, call("workitem_property", map[string]any{"action": "set_value", "workitem_id": id, "property_id": property["id"], "value": value}))
					items = append(items, id)
				}
				catalogueNull(t, call(kind, map[string]any{"action": "manage_workitems", kind + "_id": parentID, "add_ids": strings.Join(items[:3], ",")}))
				before := planningMutationSnapshot(t, fixture)
				query := fmt.Sprintf(`cf["%s"] = 0 AND priority = "high"`, property["id"])
				input := map[string]any{"action": "list_workitems", kind + "_id": parentID, "pql": query,
					"order_by": "sequence_id", "per_page": 1, "fields": "id,name"}
				assertPlanningQueryPages(t, call, kind, input,
					map[string]any{"id": items[0], "name": "Property query 0"},
					map[string]any{"id": items[2], "name": "Property query 2"})
				count := milestoneObject(t, call("workitem", map[string]any{"action": "count", "pql": query + fmt.Sprintf(` AND %s = "%s"`, kind, parentID)}))
				require.Equal(t, float64(2), count["total_count"])
				require.Equal(t, before, planningMutationSnapshot(t, fixture))
			})
		}
	})
}
