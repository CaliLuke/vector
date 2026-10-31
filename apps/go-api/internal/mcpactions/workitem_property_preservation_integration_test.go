//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyDestructiveMutationsPreserveUsedValuesAndHistory(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		fixture := newPropertyReadFixture(t, f, root, "workspace-one", testsupport.LabelProject)
		kind, property, option := fixture.typeID, fixture.propertyID, fixture.optionID
		value := fixture.valueRead()
		original := milestoneObject(t, root("workitem_property", value))
		extra := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "TEXT"}))
		for _, example := range []struct {
			name, code string
			input      map[string]any
		}{
			{"delete property", "conflict", map[string]any{"action": "delete", "workitem_property_id": property}},
			{"detach property", "conflict", map[string]any{"action": "manage_type_properties", "workitem_type_id": kind, "detach_ids": property}},
			{"attach then rejected detach", "conflict", map[string]any{"action": "manage_type_properties", "workitem_type_id": kind, "attach_ids": extra["id"], "detach_ids": property}},
			{"change cardinality", "invalid_params", map[string]any{"action": "update", "workitem_property_id": property, "is_multi": true}},
			{"change kind", "invalid_params", map[string]any{"action": "update", "workitem_property_id": property, "property_type": "TEXT"}},
			{"add option", "invalid_params", map[string]any{"action": "create_option", "property_id": property, "name": "New option"}},
			{"rename option", "invalid_params", map[string]any{"action": "update_option", "property_id": property, "option_id": option, "name": "Renamed option"}},
			{"delete option", "invalid_params", map[string]any{"action": "delete_option", "property_id": property, "option_id": option}},
		} {
			t.Run(example.name, func(t *testing.T) {
				example.input["request_id"] = uuid.NewString()
				before := propertyStorageSnapshot(t, f)
				milestoneFailure(t, root("workitem_property", example.input), example.code)
				require.Equal(t, before, propertyStorageSnapshot(t, f), "rejection must preserve definitions, associations, values, history and receipts")
				require.Equal(t, original, milestoneObject(t, root("workitem_property", value)))
			})
		}
		updated := milestoneObject(t, root("workitem_property", map[string]any{"action": "update", "workitem_property_id": property, "description": "Safe metadata change"}))
		require.Equal(t, "Safe metadata change", updated["description"])
		optionMetadata := milestoneObject(t, root("workitem_property", map[string]any{"action": "update_option", "property_id": property, "option_id": option, "external_source": "preserved"}))
		require.Equal(t, "preserved", optionMetadata["external_source"])
		require.Equal(t, original, milestoneObject(t, root("workitem_property", value)))
		catalogueNull(t, root("workitem_property", map[string]any{"action": "delete_value", "project_id": testsupport.LabelProject, "workitem_id": value["workitem_id"], "property_id": property}))
		milestoneFailure(t, root("workitem_property", value), "not_found")
		before := propertyStorageSnapshot(t, f)
		milestoneFailure(t, root("workitem_property", map[string]any{"action": "delete", "workitem_property_id": property, "request_id": uuid.NewString()}), "conflict")
		require.Equal(t, before, propertyStorageSnapshot(t, f), "cleared current values do not authorize deleting their retained history")
	})
}
