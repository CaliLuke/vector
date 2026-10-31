//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPPropertyReceiptsBindMutationInput(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
			t.Run(action, func(t *testing.T) {
				input := preparePropertyMutationInput(t, f, call, action)
				conflict := propertyReceiptConflictInput(t, f, call, action, input)
				conflict["request_id"] = input["request_id"]
				accepted := milestoneObject(t, call("workitem_property", input))
				before := propertyStorageSnapshot(t, f)
				milestoneFailure(t, call("workitem_property", conflict), "request_conflict")
				require.Equal(t, before, propertyStorageSnapshot(t, f), "a receipt cannot authorize different input or a different target")
				accepted["replayed"] = true
				require.Equal(t, accepted, milestoneObject(t, call("workitem_property", input)))
				require.Equal(t, before, propertyStorageSnapshot(t, f))
			})
		}
	})
}

func propertyReceiptConflictInput(t *testing.T, f milestoneFixture, call catalogueCall, action string, input map[string]any) map[string]any {
	t.Helper()
	changed := maps.Clone(input)
	switch action {
	case "create":
		changed["display_name"] = "Different receipt input"
	case "update":
		changed["description"] = "Different receipt input"
	case "create_option", "update_option":
		changed["name"] = "Different receipt input"
	case "set_value":
		changed["value"] = "Different receipt input"
	default:
		return preparePropertyMutationInput(t, f, call, action)
	}
	return changed
}
