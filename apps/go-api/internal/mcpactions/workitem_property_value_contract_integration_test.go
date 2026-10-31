//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyScalarValuesKeepTheirWireTypes(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Scalar property values", "type_id": typeRow["id"]}))
		for _, example := range []struct {
			kind  string
			value any
		}{
			{"TEXT", "007"}, {"DECIMAL", float64(0)}, {"BOOLEAN", false}, {"EMAIL", "person@example.test"}, {"FILE", "report-007.csv"}, {"URL", "https://example.test/report?code=007"},
		} {
			property := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": typeRow["id"], "display_name": uuid.NewString(), "property_type": example.kind}))
			call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"]})
			accepted := milestoneObject(t, call("workitem_property", map[string]any{"action": "set_value", "value": example.value, "external_source": "contract-test", "external_id": uuid.NewString()}))
			require.Equal(t, example.value, accepted["value"], example.kind)
			require.Equal(t, example.kind, accepted["value_type"], example.kind)
			require.Equal(t, property["id"], accepted["property_id"])
			require.Equal(t, item["id"], accepted["issue_id"])
			require.Equal(t, accepted, milestoneObject(t, call("workitem_property", map[string]any{"action": "get_value"})))
			before := propertyStorageSnapshot(t, f)
			milestoneFailure(t, call("workitem_property", map[string]any{"action": "set_value", "value": example.value, "description_html": "<p>Ambiguous</p>"}), "invalid_params")
			require.Equal(t, before, propertyStorageSnapshot(t, f), "ambiguous value inputs must not mutate storage")
			catalogueNull(t, call("workitem_property", map[string]any{"action": "delete_value"}))
			milestoneFailure(t, call("workitem_property", map[string]any{"action": "get_value"}), "not_found")
		}
	})
}

func TestMCPPropertyMultiValueReplacementRetainsSurvivingIdentity(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		property := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": typeRow["id"], "display_name": uuid.NewString(), "property_type": "OPTION", "is_multi": true, "options": `[{"name":"First"},{"name":"Second"},{"name":"Third"}]`}))
		options := property["options"].([]any)
		first, second, third := options[0].(map[string]any)["id"], options[1].(map[string]any)["id"], options[2].(map[string]any)["id"]
		item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Multiple property values", "type_id": typeRow["id"]}))
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"]})
		original := catalogueArray(t, call("workitem_property", map[string]any{"action": "set_value", "value": []any{first, second}, "external_source": "original"}))
		require.Len(t, original, 2)
		originalByValue := map[any]map[string]any{}
		for _, row := range original {
			originalByValue[row["value"]] = row
		}
		require.Contains(t, originalByValue, first)
		require.Contains(t, originalByValue, second)
		replaced := catalogueArray(t, call("workitem_property", map[string]any{"action": "set_value", "value": []any{second, third}}))
		require.Len(t, replaced, 2)
		replacementByValue := map[any]map[string]any{}
		for _, row := range replaced {
			replacementByValue[row["value"]] = row
		}
		require.NotContains(t, replacementByValue, first)
		require.Contains(t, replacementByValue, second)
		require.Contains(t, replacementByValue, third)
		require.Equal(t, originalByValue[second]["id"], replacementByValue[second]["id"])
		require.Equal(t, "original", replacementByValue[second]["external_source"])
		require.NotEqual(t, originalByValue[first]["id"], replacementByValue[third]["id"])
		require.NotEqual(t, originalByValue[second]["id"], replacementByValue[third]["id"])
		require.Equal(t, replaced, catalogueArray(t, call("workitem_property", map[string]any{"action": "get_value"})))
		catalogueNull(t, call("workitem_property", map[string]any{"action": "delete_value"}))
		milestoneFailure(t, call("workitem_property", map[string]any{"action": "get_value"}), "not_found")
	})
}
