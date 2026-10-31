//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func planningAuthorizationWaitFixture(t *testing.T) milestoneFixture {
	t.Helper()
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=true,module_view=true`)
	require.NoError(t, err)
	return f
}

func TestMCPPlanningWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, kind := range []string{"cycle", "module"} {
		t.Run(kind, func(t *testing.T) {
			extra := map[string]any{}
			if kind == "cycle" {
				extra["owned_by"] = testsupport.LabelActor
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{
				tool: kind, fixture: planningAuthorizationWaitFixture, createFields: extra, snapshot: planningMutationSnapshot,
			})
		})
	}
}
