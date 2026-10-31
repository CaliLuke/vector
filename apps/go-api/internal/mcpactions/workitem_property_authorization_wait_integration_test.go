//go:build integration

package mcpactions_test

import "testing"

func TestMCPPropertyWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option", "set_value", "delete_value"} {
		t.Run(action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{
				schemaLock: true, tool: "workitem_property", action: action,
				fixture: newMilestoneFixture, snapshot: propertyStorageSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					return preparePropertyMutationInput(t, f, call, action)
				},
			})
		})
	}
}
