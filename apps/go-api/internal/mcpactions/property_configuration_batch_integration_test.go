//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyConfigurationBatch(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		repo := workitemtypes.NewRepository(f.pool)
		catalog, err := repo.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
		require.NoError(t, err)
		if !catalog.Enabled {
			_, err = repo.Mutate(t.Context(), testsupport.LabelActor, "workspace-one", "enable", "", workitemtypes.Input{Revision: catalog.Revision})
			require.NoError(t, err)
		}
		typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": "Configured " + uuid.NewString()}))
		create := func(kind string, fields map[string]any) map[string]any {
			input := map[string]any{"action": "create", "display_name": "Configured " + uuid.NewString(), "property_type": kind, "workitem_type_id": typeRow["id"]}
			for k, v := range fields {
				input[k] = v
			}
			return milestoneObject(t, root("workitem_property", input))
		}
		formula := create("FORMULA", map[string]any{"configuration": map[string]any{"expression": "2 * 3", "result": "Number"}})
		require.Equal(t, map[string]any{"expression": "2 * 3", "result": "Number"}, formula["settings"])
		fixed := create("TEXT", map[string]any{"display_format": "readonly", "configuration": map[string]any{"content": "Instructions"}})
		parent, child := uuid.NewString(), uuid.NewString()
		levels := []map[string]any{{"id": uuid.NewString(), "name": "Country", "required": false}, {"id": uuid.NewString(), "name": "City", "required": false}}
		options := []map[string]any{{"id": parent, "name": "Italy", "level": 0}, {"id": child, "name": "Rome", "parent": parent, "level": 1}}
		cascade := create("CASCADING", map[string]any{"configuration": map[string]any{"levels": levels, "options": options}})
		require.Len(t, cascade["settings"].(map[string]any)["levels"], 2)
		item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Configured values", "type_id": typeRow["id"]}))
		values := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"]})
		read := func(property any) map[string]any {
			return milestoneObject(t, values("workitem_property", map[string]any{"action": "get_value", "property_id": property}))
		}
		require.Equal(t, float64(6), read(formula["id"])["value"])
		changed := milestoneObject(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": formula["id"], "configuration": map[string]any{"expression": "0"}}))
		require.Equal(t, "Number", changed["settings"].(map[string]any)["result"])
		require.Equal(t, float64(0), read(formula["id"])["value"])
		require.Equal(t, "Instructions", read(fixed["id"])["value"])
		milestoneObject(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": fixed["id"], "configuration": map[string]any{"content": ""}}))
		require.Empty(t, read(fixed["id"])["value"])
		stored := catalogueArray(t, values("workitem_property", map[string]any{"action": "set_value", "property_id": cascade["id"], "value": []string{parent, child}}))
		require.Len(t, stored, 2)
		require.Equal(t, parent, stored[0]["value"])
		require.Equal(t, child, stored[1]["value"])
		readCascade := func() []map[string]any {
			return catalogueArray(t, values("workitem_property", map[string]any{"action": "get_value", "property_id": cascade["id"]}))
		}
		require.Equal(t, stored, readCascade())
		milestoneFailure(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": cascade["id"], "configuration": map[string]any{"options": []any{}}}), "conflict")
		require.Equal(t, stored, readCascade(), "destructive configuration cannot discard the saved path")
		milestoneFailure(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": formula["id"], "configuration": map[string]any{"content": "wrong kind"}}), "invalid_params")
		milestoneFailure(t, values("workitem_property", map[string]any{"action": "set_value", "property_id": formula["id"], "value": 7}), "invalid_params")
		unused := create("CASCADING", map[string]any{"configuration": map[string]any{"levels": levels, "options": options}})
		cleared := milestoneObject(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": unused["id"], "configuration": map[string]any{"options": []any{}}}))
		require.Empty(t, cleared["options"], "present-empty arrays clear an unused option tree")
		require.Len(t, cleared["settings"].(map[string]any)["levels"], 2, "omitted levels retain their identities")
	})
}
