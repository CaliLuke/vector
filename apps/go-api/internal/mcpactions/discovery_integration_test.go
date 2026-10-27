//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func discoveredActions(t *testing.T, result rpcResult) map[string][]string {
	t.Helper()
	actions := map[string][]string{}
	for _, tool := range result.Tools {
		for _, raw := range tool.InputSchema.OneOf {
			var branch struct{ Properties map[string]json.RawMessage }
			require.NoError(t, json.Unmarshal(raw, &branch))
			var discriminator struct{ Enum []string }
			require.NoError(t, json.Unmarshal(branch.Properties["action"], &discriminator))
			actions[tool.Name] = append(actions[tool.Name], discriminator.Enum...)
		}
	}
	return actions
}

func TestDiscoveryUsesCurrentMemberReadPermissionsAcrossCredentialsAndProtocols(t *testing.T) {
	for _, tc := range []struct {
		name, change      string
		directory, roster bool
	}{
		{"member", "", true, true},
		{"guest", `UPDATE workspace_members SET role=5 WHERE member_id='10000000-0000-4000-8000-000000000001'`, false, true},
		{"admin without project membership", `UPDATE workspace_members SET role=20 WHERE member_id='10000000-0000-4000-8000-000000000001'; DELETE FROM project_members WHERE member_id='10000000-0000-4000-8000-000000000001'`, true, false},
		{"inactive project membership", `UPDATE project_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`, true, false},
		{"deleted project", `UPDATE projects SET deleted_at=now() WHERE id='30000000-0000-4000-8000-000000000001'`, true, false},
		{"foreign project membership", `UPDATE project_members SET workspace_id='20000000-0000-4000-8000-000000000002' WHERE member_id='10000000-0000-4000-8000-000000000001'`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, pool, auth, _ := oauthFixture(t)
			addMCPProjectFixture(t, pool)
			if tc.change != "" {
				_, err := pool.Exec(t.Context(), tc.change)
				require.NoError(t, err)
			}
			pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
			for _, version := range []string{"2025-11-25", "2026-07-28"} {
				for _, mode := range []string{"PAT", "OAuth"} {
					t.Run(version+"/"+mode, func(t *testing.T) {
						var listed rpcResult
						call := func(input string) rpcResult {
							if mode == "OAuth" {
								return oauthCall(t, oauth, version, "oauth-user", "workspace-one", input, 200)
							}
							return rpcVersion(t, pat, "test-one", "tools/call", memberCall(input), version)
						}
						if mode == "OAuth" {
							listed = oauthRPC(t, oauth, version, "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
						} else {
							listed = rpcVersion(t, pat, "test-one", "tools/list", map[string]any{}, version)
						}
						actions := discoveredActions(t, listed)
						expected := []string{"me", "list_roles", "retrieve_role"}
						if tc.directory {
							expected = append(expected, "list_workspace")
						}
						if tc.roster {
							require.ElementsMatch(t, []string{"list", "retrieve"}, actions["cycle"])
							expected = append(expected, "list_project")
						}
						require.ElementsMatch(t, expected, actions["member"])
						require.Contains(t, actions["project"], "list")
						require.Contains(t, actions["project"], "retrieve")
						directory := call(`{"action":"list_workspace"}`)
						require.Equal(t, !tc.directory, directory.IsError, directory.Content)
						roster := call(`{"action":"list_project","project_id":"30000000-0000-4000-8000-000000000001"}`)
						require.Equal(t, !tc.roster, roster.IsError, roster.Content)
						if !tc.directory {
							require.NotContains(t, discoveredTool(t, listed, "member").InputSchema.Properties, "email")
							require.NotContains(t, discoveredTool(t, listed, "member").InputSchema.Properties, "is_active")
						}
						if !tc.roster {
							require.NotContains(t, actions, "cycle")
							require.NotContains(t, discoveredTool(t, listed, "member").InputSchema.Properties, "project_id")
						}
					})
				}
			}
		})
	}
}

func TestOAuthDiscoveryWithoutWorkspaceUsesEligibleScopesAndRechecksMembership(t *testing.T) {
	f, pool, auth, issuer := oauthFixture(t)
	addMCPProjectFixture(t, pool)
	issuer.mu.Lock()
	issuer.scope = "vector:read vector:write"
	issuer.mu.Unlock()
	h := oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		list := func(workspace string) map[string][]string {
			return discoveredActions(t, oauthRPC(t, h, version, "oauth-user", workspace, "tools/list", map[string]any{}, 200))
		}
		_, err := pool.Exec(t.Context(), `UPDATE workspace_members SET role=5 WHERE member_id='10000000-0000-4000-8000-000000000001'; DELETE FROM workspace_members WHERE workspace_id='20000000-0000-4000-8000-000000000002'; INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true)`)
		require.NoError(t, err)
		require.Contains(t, list("")["member"], "list_workspace")
		require.NotContains(t, list("workspace-one")["member"], "list_workspace")
		require.Contains(t, list("workspace-two")["member"], "list_workspace")
		require.NotContains(t, list("workspace-two")["member"], "list_project")
		_, err = pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`)
		require.NoError(t, err)
		require.Equal(t, map[string][]string{"member": {"me"}}, list(""))
		me := oauthCall(t, h, version, "oauth-user", "", `{"action":"me"}`, 200)
		require.False(t, me.IsError, me.Content)
		_, err = pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=true WHERE member_id='10000000-0000-4000-8000-000000000001'`)
		require.NoError(t, err)
		require.Contains(t, list("")["member"], "list_workspace")
	}
}

func TestReadDiscoveryRejectsMembershipChangeAfterAuthentication(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	h := oauthHandler(t, f, func(ctx context.Context, r *http.Request) (context.Context, error) {
		verified, err := auth.Authenticate(ctx, r)
		_, changeErr := pool.Exec(ctx, `UPDATE workspace_members SET role=5 WHERE member_id='10000000-0000-4000-8000-000000000001'`)
		require.NoError(t, changeErr)
		return verified, err
	})
	listed := oauthRPC(t, h, "2026-07-28", "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
	actions := discoveredActions(t, listed)
	require.NotContains(t, actions["member"], "list_workspace")
	require.Contains(t, actions["member"], "list_roles")
}
