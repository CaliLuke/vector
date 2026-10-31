//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerSDKPagesRetainShapeNavigationAndWorkspace(t *testing.T) {
	for _, tool := range []string{"customer", "customer_property"} {
		t.Run(tool, func(t *testing.T) {
			f := customerIdentityFixture(t)
			foreign, _ := addPrivateProjectInOtherWorkspace(t, f)
			_, err := f.pool.Exec(t.Context(), `INSERT INTO customers(workspace_id,name) VALUES($1,'Catalogue foreign buyer');`, foreign)
			require.NoError(t, err)
			_, err = f.pool.Exec(t.Context(), `INSERT INTO customer_properties(workspace_id,name,display_name,property_type) VALUES($1,'foreign','Catalogue foreign property','TEXT')`, foreign)
			require.NoError(t, err)
			runCatalogueFixtureMatrix(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
				input := map[string]any{"action": "create", "name": "Catalogue " + uuid.NewString()}
				idField := "customer_id"
				if tool == "customer_property" {
					input = map[string]any{"action": "create", "display_name": "Catalogue " + uuid.NewString(), "property_type": "TEXT"}
					idField = "property_id"
				}
				matching := []any{}
				for range 2 {
					created := milestoneObject(t, call(tool, input))
					matching = append(matching, created)
					if tool == "customer" {
						input["name"] = "Catalogue " + uuid.NewString()
					} else {
						input["display_name"] = "Catalogue " + uuid.NewString()
					}
				}
				removed := milestoneObject(t, call(tool, input))
				catalogueNull(t, call(tool, map[string]any{"action": "delete", idField: removed["id"]}))
				list := map[string]any{"action": "list", "per_page": 1}
				if tool == "customer" {
					milestoneObject(t, call(tool, map[string]any{"action": "create", "name": "Unrelated " + uuid.NewString()}))
					list["query"] = "cAtAlOgUe"
				}
				first := milestoneObject(t, call(tool, list))
				assertCustomerSDKPage(t, first, false, true)
				list["cursor"] = first["next_cursor"]
				second := milestoneObject(t, call(tool, list))
				assertCustomerSDKPage(t, second, true, false)
				items := append(slices.Clone(first["results"].([]any)), second["results"].([]any)...)
				require.ElementsMatch(t, matching, items, "pages exclude deleted rows and another workspace even when the actor administers both")
				list["cursor"] = second["prev_cursor"]
				require.Equal(t, first, milestoneObject(t, call(tool, list)))
				delete(list, "cursor")
				delete(list, "per_page")
				complete := milestoneObject(t, call(tool, list))
				require.ElementsMatch(t, matching, complete["results"])
				require.Equal(t, float64(2), complete["count"])
				require.Equal(t, float64(1), complete["total_pages"])
				require.Equal(t, false, complete["next_page_results"])
				for _, raw := range matching {
					catalogueNull(t, call(tool, map[string]any{"action": "delete", idField: raw.(map[string]any)["id"]}))
				}
				empty := milestoneObject(t, call(tool, list))
				require.Equal(t, slices.Sorted(maps.Keys(first)), slices.Sorted(maps.Keys(empty)))
				assertCustomerSDKEmptyPage(t, empty)
			})
		})
	}
}
