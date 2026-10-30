//go:build integration

package mcpactions_test

import "testing"

func TestMCPLabelWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	runProjectMutationAuthorizationWait(t, projectMutationWaitScenario{
		tool: "label", fixture: newMilestoneFixture, snapshot: labelMutationSnapshot,
	})
}
