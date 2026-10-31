//go:build integration

package mcpactions_test

import "testing"

func customerAssociationTargetChanges() []customerRequestLinkChange {
	var changes []customerRequestLinkChange
	for _, change := range customerRequestLinkChanges() {
		switch change.name {
		case "draft item", "deleted item":
			for _, field := range []string{"link_ids", "unlink_ids"} {
				target := change
				target.name += "/" + field
				target.arguments = func(input map[string]any) []any { return []any{input[field]} }
				changes = append(changes, target)
			}
		default:
			changes = append(changes, change)
		}
	}
	return changes
}

func TestMCPCustomerAssociationReceiptsRequireCurrentTargetAccess(t *testing.T) {
	runCustomerLinkTargetReplay(t, "customer", customerAssociationWriteInput, customerAssociationTargetChanges())
}

func TestMCPCustomerAssociationsReauthorizeTargetsAfterWaiting(t *testing.T) {
	runCustomerLinkTargetWait(t, "customer", customerAssociationWriteInput, customerAssociationTargetChanges())
}
