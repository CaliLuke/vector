//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type pinnedAction struct {
	Action             string
	Required, Optional []string
}

// Compare the expanded catalogue with the pinned source, independently of the
// implementation's dispatch tables. Target-specific authorization remains in
// the application; this actor has workspace and project administration rights.
func TestMCPDiscoveryMatchesPinnedActionCatalogue(t *testing.T) {
	data, err := os.ReadFile("../../mcp/testdata/plane-mcp/catalogue.json")
	require.NoError(t, err)
	var pinned struct {
		ToolCount      int `json:"tool_count"`
		OperationCount int `json:"operation_count"`
		Tools          []struct {
			Tool      string
			HasAction bool `json:"has_action_parameter"`
			Actions   []pinnedAction
		}
	}
	require.NoError(t, json.Unmarshal(data, &pinned))
	f := newMilestoneFixture(t)
	_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20; UPDATE workspace_features SET initiatives=true,customers=true`)
	require.NoError(t, err)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				var listed rpcResult
				if mode == "OAuth" {
					listed = oauthRPC(t, oauthHandler(t, f.fixture, f.oauth.Authenticate), version, "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
				} else {
					listed = rpcVersion(t, f.handler(t, nil), "test-one", "tools/list", map[string]any{}, version)
				}
				require.Len(t, listed.Tools, pinned.ToolCount)
				total := 0
				for _, expected := range pinned.Tools {
					t.Run(expected.Tool, func(t *testing.T) {
						tool := discoveredTool(t, listed, expected.Tool)
						branches := map[string]map[string]json.RawMessage{}
						if !expected.HasAction {
							branches[expected.Actions[0].Action] = tool.InputSchema.Properties
						} else {
							for _, raw := range tool.InputSchema.OneOf {
								var branch struct{ Properties map[string]json.RawMessage }
								require.NoError(t, json.Unmarshal(raw, &branch))
								var discriminator struct{ Enum []string }
								require.NoError(t, json.Unmarshal(branch.Properties["action"], &discriminator))
								for _, action := range discriminator.Enum {
									require.NotContains(t, branches, action, "duplicate action schema")
									branches[action] = branch.Properties
								}
							}
						}
						require.Len(t, branches, len(expected.Actions))
						for _, action := range expected.Actions {
							require.Contains(t, branches, action.Action)
							for _, name := range append(action.Required, action.Optional...) {
								require.Contains(t, branches[action.Action], name, "%s.%s parameter", expected.Tool, action.Action)
							}
						}
						total += len(branches)
					})
				}
				require.Equal(t, pinned.OperationCount, total)
			})
		}
	}
}
