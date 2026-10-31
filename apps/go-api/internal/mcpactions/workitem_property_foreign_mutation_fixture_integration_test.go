//go:build integration

package mcpactions_test

import (
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
)

func foreignPropertyMutationInputs(f propertyReadFixture) []map[string]any {
	inputs := []map[string]any{
		{"action": "create", "display_name": uuid.NewString(), "property_type": "TEXT", "workitem_type_id": f.typeID},
		{"action": "update", "workitem_property_id": f.propertyID, "description": "Foreign edit"},
		{"action": "delete", "workitem_property_id": f.propertyID},
		{"action": "manage_type_properties", "workitem_type_id": f.typeID, "detach_ids": f.propertyID},
		{"action": "create_option", "property_id": f.propertyID, "name": "Foreign option"},
		{"action": "update_option", "property_id": f.propertyID, "option_id": f.optionID, "external_id": "foreign-edit"},
		{"action": "delete_option", "property_id": f.propertyID, "option_id": f.optionID},
		{"action": "set_value", "project_id": testsupport.LabelProject, "workitem_id": f.itemID, "property_id": f.propertyID, "value": f.optionID},
		{"action": "delete_value", "project_id": testsupport.LabelProject, "workitem_id": f.itemID, "property_id": f.propertyID},
	}
	for _, input := range inputs {
		input["request_id"] = uuid.NewString()
	}
	return inputs
}
