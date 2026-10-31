//go:build integration

package mcpactions_test

import "testing"

func TestMCPPropertyReceiptStorageFailureRollsBack(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
			t.Run(action, func(t *testing.T) {
				input := preparePropertyMutationInput(t, f, call, action)
				assertCatalogueStorageFailure(t, f, call, "workitem_property", input, "mutation_receipts", func() []byte { return propertyStorageSnapshot(t, f) })
			})
		}
	})
}

func TestMCPPropertyValueEffectStorageFailureRollsBack(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"set_value", "delete_value"} {
			t.Run(action, func(t *testing.T) {
				input := preparePropertyMutationInput(t, f, call, action)
				assertCatalogueStorageFailure(t, f, call, "workitem_property", input, "work_item_effects", func() []byte { return propertyStorageSnapshot(t, f) })
			})
		}
	})
}
