//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestStateMCPDiscoveryAndCallsRecheckProjectPermissions(t *testing.T) {
	f := newMilestoneFixture(t)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := stateToolCall(f, t, mode, version)
				pat, oauth := f.handler(t, nil), oauthHandler(t, f.fixture, f.oauth.Authenticate)
				discover := func() []string {
					var result rpcResult
					if mode == "OAuth" {
						result = oauthRPC(t, oauth, version, "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
					} else {
						result = rpcVersion(t, pat, "test-one", "tools/list", map[string]any{}, version)
					}
					return discoveredActions(t, result)["state"]
				}
				for _, tc := range []struct {
					name, sql          string
					projectRead, write bool
				}{
					{"member", "SELECT 1", true, true},
					{"guest", "UPDATE project_members SET role=5", true, false},
					{"unknown role", "UPDATE project_members SET role=99", false, false},
					{"revoked membership", "UPDATE project_members SET is_active=false", false, false},
					{"archived project", "UPDATE projects SET archived_at=now()", false, false},
					{"deleted project", "UPDATE projects SET deleted_at=now()", false, false},
				} {
					t.Run(tc.name, func(t *testing.T) {
						_, err := f.pool.Exec(t.Context(), tc.sql)
						require.NoError(t, err)
						actions := discover()
						expected := []string{"list", "retrieve"}
						if tc.write {
							expected = append(expected, "create", "update", "delete")
						}
						require.ElementsMatch(t, expected, actions)
						workspace := milestoneObject(t, call(map[string]any{"action": "list"}))
						require.Empty(t, workspace["results"])
						project := call(map[string]any{"action": "list", "project_id": testsupport.LabelProject})
						require.Equal(t, !tc.projectRead, project.IsError, project.Content)
						if !tc.write {
							milestoneFailure(t, call(map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Denied", "color": "#123456"}), "forbidden")
						}
						_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=15,is_active=true; UPDATE projects SET archived_at=NULL,deleted_at=NULL`)
						require.NoError(t, err)
					})
				}
				if mode == "OAuth" {
					f.issuer.mu.Lock()
					f.issuer.scope = "vector:read"
					f.issuer.mu.Unlock()
					require.ElementsMatch(t, []string{"list", "retrieve"}, discover())
					milestoneFailure(t, call(map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Denied", "color": "#123456"}), "forbidden")
					f.issuer.mu.Lock()
					f.issuer.scope = "vector:read vector:write"
					f.issuer.mu.Unlock()
				}
			})
		}
	}
	var states, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states),(SELECT count(*) FROM mutation_receipts)`).Scan(&states, &receipts))
	require.Zero(t, states)
	require.Zero(t, receipts)
}

func TestStateMCPCursorAndClosedActionValidation(t *testing.T) {
	f := newMilestoneFixture(t)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		call := stateToolCall(f, t, "personal token", version)
		for _, name := range []string{"First", "Second", "Third"} {
			milestoneObject(t, call(map[string]any{"action": "create", "name": name + version, "color": "#123456", "project_id": testsupport.LabelProject}))
		}
		first := milestoneObject(t, call(map[string]any{"action": "list", "per_page": 2}))
		second := milestoneObject(t, call(map[string]any{"action": "list", "per_page": 2, "cursor": first["next_cursor"]}))
		require.Equal(t, float64(3), first["total_count"])
		require.Len(t, first["results"], 2)
		require.Len(t, second["results"], 1)
		require.True(t, first["next_page_results"].(bool))
		require.False(t, second["next_page_results"].(bool))
		for _, input := range []map[string]any{
			{"action": "list", "per_page": 1001}, {"action": "list", "per_page": -1}, {"action": "list", "cursor": "bad"},
			{"action": "list", "include_revision": true}, {"action": "list", "project_id": "bad"},
			{"action": "list", "project_id": "00000000-0000-0000-0000-000000000000"},
			{"action": "create", "project_id": testsupport.LabelProject, "name": "Wrong", "color": "#123456", "expected_revision": "x"},
			{"action": "create", "name": "Wrong", "color": "#123456", "group": "backlog", "default": false},
			{"action": "create", "name": "Wrong", "color": "#123456", "group": "backlog", "sequence": 0},
			{"action": "create", "name": "Wrong", "color": "#123456"},
			{"action": "create", "name": "Wrong", "color": "#123456", "group": "triage"},
			{"action": "list", "name": "unexpected"}, {"action": "unknown"},
		} {
			require.True(t, call(input).IsError, input)
		}
		_, err := f.pool.Exec(t.Context(), `UPDATE states SET deleted_at=now()`)
		require.NoError(t, err)
	}
}
