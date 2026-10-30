//go:build integration

package mcpactions_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPPQLReferenceWorksWithoutWorkspaceSelection(t *testing.T) {
	full, err := os.ReadFile("pql_full.txt")
	require.NoError(t, err)
	brief, err := os.ReadFile("pql_brief.txt")
	require.NoError(t, err)
	runSelectedWorkspaceMatrix(t, newMilestoneFixture(t), func(t *testing.T, call catalogueCall) {
		defaultReference := milestoneObject(t, call("get_pql_reference", map[string]any{}))
		require.Equal(t, map[string]any{"detail": "full", "reference": string(full)}, defaultReference)
		explicitFull := milestoneObject(t, call("get_pql_reference", map[string]any{"detail": "full"}))
		require.Equal(t, defaultReference, explicitFull)
		explicitBrief := milestoneObject(t, call("get_pql_reference", map[string]any{"detail": "brief"}))
		require.Equal(t, map[string]any{"detail": "brief", "reference": string(brief)}, explicitBrief)
		milestoneFailure(t, call("get_pql_reference", map[string]any{"detail": "unknown"}), "invalid_params")
	})
}
