//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerDefaultReferencesReauthorizeAfterWaiting(t *testing.T) {
	for _, change := range customerValueReferenceChanges {
		for _, action := range []string{"create", "update"} {
			if change.kind == "OPTION" && action == "create" {
				continue // A new property's inline options have no pre-existing IDs.
			}
			t.Run(change.kind+"/"+change.name+"/"+action, func(t *testing.T) {
				runCustomerReferenceAuthorizationWait(t, change, func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					return customerDefaultReferenceWaitInput(t, f, call, change.kind, action)
				})
			})
		}
	}
}

func customerDefaultReferenceWaitInput(t *testing.T, f milestoneFixture, call catalogueCall, kind, action string) map[string]any {
	t.Helper()
	if kind != "OPTION" {
		target, _ := customerReferenceTarget(t, f, kind)
		return customerDefaultReplayInput(t, call, action, kind, target)
	}
	require.Equal(t, "update", action)
	property := milestoneObject(t, call("customer_property", map[string]any{
		"action": "create", "workspace_slug": "workspace-one", "display_name": uuid.NewString(),
		"property_type": "OPTION", "options": `[{"name":"Choice"}]`,
	}))
	optionID := property["options"].([]any)[0].(map[string]any)["id"]
	return map[string]any{"action": "update", "workspace_slug": "workspace-one", "property_id": property["id"], "default_value": optionID, "request_id": uuid.NewString()}
}
