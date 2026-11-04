//go:build integration

package mcpactions_test

import "testing"

func TestMCPLabelWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	runMutationAuthorizationWait(t, mutationWaitScenario{reuseCredentials: true,
		tool: "label", fixture: newMilestoneFixture, snapshot: labelMutationSnapshot,
	})
}
