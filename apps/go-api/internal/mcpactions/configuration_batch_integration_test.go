//go:build integration

package mcpactions_test

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPConfigurationCatalogueBatch(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		repo := workitemtypes.NewRepository(f.pool)
		catalog, err := repo.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
		require.NoError(t, err)
		if !catalog.Enabled {
			_, err = repo.Mutate(t.Context(), testsupport.LabelActor, "workspace-one", "enable", "", workitemtypes.Input{Revision: catalog.Revision})
			require.NoError(t, err)
		}
		t.Run("estimates", func(t *testing.T) { catalogueEstimates(t, call) })
		t.Run("types and properties", func(t *testing.T) { catalogueTypesAndProperties(t, call) })
		t.Run("relations", func(t *testing.T) { catalogueRelations(t, f, call) })
		t.Run("intake", func(t *testing.T) { catalogueIntake(t, call) })
	})
}

func catalogueEstimates(t *testing.T, root catalogueCall) {
	call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
	estimate := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "Sizing", "type": "points"}))
	id := estimate["id"].(string)
	linked := milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
	require.Equal(t, id, linked["estimate"])
	require.Equal(t, id, milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve"}))["id"])
	require.Equal(t, "Effort", milestoneObject(t, call("project_estimate", map[string]any{"action": "update", "name": "Effort"}))["name"])
	points := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"key":0,"value":"1"},{"key":1,"value":"2"}]`}))
	require.Len(t, points, 2)
	point := points[0]["id"].(string)
	updated := milestoneObject(t, call("project_estimate", map[string]any{"action": "update_point", "estimate_id": id, "estimate_point_id": point, "value": "3", "key": 0}))
	require.Equal(t, "3", updated["value"])
	require.Equal(t, float64(0), updated["key"])
	require.Len(t, catalogueArray(t, call("project_estimate", map[string]any{"action": "list_points", "estimate_id": id})), 2)
	catalogueNull(t, call("project_estimate", map[string]any{"action": "delete_point", "estimate_id": id, "estimate_point_id": point}))
	catalogueNull(t, call("project_estimate", map[string]any{"action": "delete"}))
}

func catalogueTypesAndProperties(t *testing.T, root catalogueCall) {
	name := "Type " + uuid.NewString()
	typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "create", "name": name, "is_active": true}))
	typeID := typeRow["id"].(string)
	require.Equal(t, "Updated", milestoneObject(t, root("workitem_type", map[string]any{"action": "update", "workitem_type_id": typeID, "description": "Updated"}))["description"])
	catalogueNull(t, root("workitem_type", map[string]any{"action": "import_to_project", "project_id": testsupport.LabelProject, "workitem_type_ids": typeID}))
	resolved := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": name}))
	require.Equal(t, typeID, resolved["id"])
	require.Equal(t, typeID, milestoneObject(t, root("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": typeID}))["id"])
	types := catalogueArray(t, root("workitem_type", map[string]any{"action": "list", "project_id": testsupport.LabelProject}))
	require.Contains(t, fmt.Sprint(types), typeID)
	catalogueNull(t, root("workitem_type", map[string]any{"action": "delete", "workitem_type_id": typeID}))

	active := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": "Properties " + uuid.NewString()}))
	typeID = active["id"].(string)
	optionProperty := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "display_name": "Choice " + uuid.NewString(), "property_type": "OPTION", "workitem_type_id": typeID, "options": `[{"name":"Initial"}]`}))
	propertyID := optionProperty["id"].(string)
	optionCall := catalogueScoped(root, map[string]any{"property_id": propertyID})
	option := milestoneObject(t, optionCall("workitem_property", map[string]any{"action": "create_option", "name": "First", "is_default": true}))
	optionID := option["id"].(string)
	require.Equal(t, "Renamed", milestoneObject(t, optionCall("workitem_property", map[string]any{"action": "update_option", "option_id": optionID, "name": "Renamed", "is_default": false}))["name"])
	require.Equal(t, "Renamed", milestoneObject(t, optionCall("workitem_property", map[string]any{"action": "retrieve_option", "option_id": optionID}))["name"])
	require.Len(t, catalogueArray(t, optionCall("workitem_property", map[string]any{"action": "list_options"})), 2)
	catalogueNull(t, optionCall("workitem_property", map[string]any{"action": "delete_option", "option_id": optionID}))
	catalogueNull(t, root("workitem_property", map[string]any{"action": "manage_type_properties", "workitem_type_id": typeID, "detach_ids": propertyID}))
	catalogueNull(t, root("workitem_property", map[string]any{"action": "delete", "workitem_property_id": propertyID}))
	number := catalogueCreatedWithReceipt(t, root, "workitem_property", map[string]any{"action": "create", "display_name": "Effort " + uuid.NewString(), "property_type": "DECIMAL", "workitem_type_id": typeID})
	propertyID = number["id"].(string)
	require.Equal(t, "Measured effort", milestoneObject(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": propertyID, "description": "Measured effort"}))["description"])
	require.Equal(t, propertyID, milestoneObject(t, root("workitem_property", map[string]any{"action": "retrieve", "workitem_property_id": propertyID}))["id"])
	require.Len(t, catalogueArray(t, root("workitem_property", map[string]any{"action": "list", "workitem_type_id": typeID})), 1)
	item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Typed item", "type_id": typeID}))
	valueCall := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": propertyID})
	value := milestoneObject(t, valueCall("workitem_property", map[string]any{"action": "set_value", "value": 0}))
	require.Equal(t, float64(0), value["value"])
	read := milestoneObject(t, valueCall("workitem_property", map[string]any{"action": "get_value"}))
	require.Equal(t, value["id"], read["id"])
	require.Equal(t, float64(0), read["value"])
	catalogueNull(t, valueCall("workitem_property", map[string]any{"action": "delete_value"}))
	milestoneFailure(t, valueCall("workitem_property", map[string]any{"action": "get_value"}), "not_found")
}

func catalogueRelations(t *testing.T, f milestoneFixture, root catalogueCall) {
	definition := milestoneObject(t, root("workitem_relation", map[string]any{"action": "create_definition", "name": "Dependency " + uuid.NewString(), "outward": "supports", "inward": "supported by", "is_active": true}))
	id := definition["id"].(string)
	require.Equal(t, "delivers", milestoneObject(t, root("workitem_relation", map[string]any{"action": "update_definition", "definition_id": id, "outward": "delivers"}))["outward"])
	definitions := milestoneObject(t, root("workitem_relation", map[string]any{"action": "list_definitions"}))
	require.Contains(t, fmt.Sprint(definitions["custom_definitions"]), id)
	first := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "First relation endpoint"}))["id"].(string)
	second := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Second relation endpoint"}))["id"].(string)
	call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": first})
	created := catalogueArray(t, call("workitem_relation", map[string]any{"action": "create", "workitem_ids": []string{second}, "relation_type": "blocking"}))
	require.Len(t, created, 1)
	listed := milestoneObject(t, call("workitem_relation", map[string]any{"action": "list"}))
	require.Contains(t, fmt.Sprint(listed["dependencies"].(map[string]any)["blocking"]), second)
	catalogueNull(t, call("workitem_relation", map[string]any{"action": "delete", "related_workitem_id": second, "is_dependency": true}))
	custom := catalogueArray(t, call("workitem_relation", map[string]any{"action": "create", "workitem_ids": []string{second}, "relation_definition_id": id, "relation_definition_label": "delivers"}))
	require.Len(t, custom, 1)
	catalogueNull(t, call("workitem_relation", map[string]any{"action": "delete", "related_workitem_id": second}))
	catalogueNull(t, root("workitem_relation", map[string]any{"action": "delete_definition", "definition_id": id}))
}

func catalogueIntake(t *testing.T, root catalogueCall) {
	call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
	milestoneObject(t, call("state", map[string]any{"action": "create", "name": "Accepted " + uuid.NewString(), "color": "#123456", "group": "unstarted", "default": true}))
	milestoneObject(t, call("project", map[string]any{"action": "update_features", "intakes": true}))
	created := milestoneObject(t, call("intake", map[string]any{"action": "create", "name": "Submission", "priority": "high"}))
	id := created["issue"].(string)
	require.Equal(t, id, milestoneObject(t, call("intake", map[string]any{"action": "retrieve", "workitem_id": id}))["issue"])
	require.Contains(t, fmt.Sprint(catalogueRows(t, call("intake", map[string]any{"action": "list"}))), id)
	accepted := milestoneObject(t, call("intake", map[string]any{"action": "update", "workitem_id": id, "status": 1}))
	require.Equal(t, float64(1), accepted["status"])
	catalogueNull(t, call("intake", map[string]any{"action": "delete", "workitem_id": id}))
	require.Equal(t, id, milestoneObject(t, call("workitem", map[string]any{"action": "retrieve", "workitem_id": id}))["id"], "accepted intake deletion keeps its work item")
}
