//go:build integration

package mcpactions_test

import "testing"

func TestMCPPropertyWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
		t.Run(action, func(t *testing.T) {
			// Catalogue edits share admission; type attachment and value mutations retain their own matrix.
			changes := mutationCredentialChanges()
			if action == "create" || action == "manage_type_properties" || action == "set_value" || action == "delete_value" {
				changes = nil
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{
				changes:    changes,
				schemaLock: true, tool: "workitem_property", action: action,
				fixture: newMilestoneFixture, snapshot: propertyStorageSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					return preparePropertyMutationInput(t, f, call, action)
				},
			})
		})
	}
}
