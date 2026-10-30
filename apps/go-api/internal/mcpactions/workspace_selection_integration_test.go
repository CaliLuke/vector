//go:build integration

package mcpactions_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceSelectionAndListingAcrossCredentialsAndProtocols(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addMCPProjectFixture(t, pool)
	_, err := pool.Exec(t.Context(), `
 INSERT INTO workspace_members(workspace_id,member_id,role,is_active)
 VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true);
 UPDATE workspaces SET name=slug;
 INSERT INTO workspaces(id,slug,name) VALUES('20000000-0000-4000-8000-000000000003','other-user-workspace','Other user workspace');
 INSERT INTO workspace_members(workspace_id,member_id,role,is_active)
 VALUES('20000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000002',15,true);
 UPDATE api_tokens SET workspace_id='20000000-0000-4000-8000-000000000001';`)
	require.NoError(t, err)
	for _, mode := range []struct {
		name, token string
		handler     http.Handler
	}{
		{"personal token", "test-one", f.handler(t, nil)},
		{"OAuth", "oauth-user", oauthHandler(t, f, auth.Authenticate)},
	} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode.name+"/"+version, func(t *testing.T) {
				call := func(tool, input string) rpcResult {
					return oauthToolCall(t, mode.handler, version, mode.token, "", tool, input, 200)
				}
				listed := call("workspace", `{"action":"list"}`)
				require.False(t, listed.IsError, listed.Content)
				require.JSONEq(t, `{"results":[{"id":"20000000-0000-4000-8000-000000000001","name":"workspace-one","slug":"workspace-one"},{"id":"20000000-0000-4000-8000-000000000002","name":"workspace-two","slug":"workspace-two"}]}`, string(listed.StructuredContent))
				first := call("project", `{"action":"list","workspace_slug":"workspace-one"}`)
				require.False(t, first.IsError, first.Content)
				require.Contains(t, string(first.StructuredContent), "SDK project")
				require.NotContains(t, string(first.StructuredContent), "Foreign workspace project")
				second := call("project", `{"action":"list","workspace_slug":"workspace-two"}`)
				require.False(t, second.IsError, second.Content)
				require.Contains(t, string(second.StructuredContent), "Foreign workspace project")
				require.NotContains(t, string(second.StructuredContent), "SDK project")
				me := call("member", `{"action":"me"}`)
				require.False(t, me.IsError, me.Content)
				for _, input := range []struct{ tool, arguments string }{
					{"workspace", `{"action":"retrieve"}`},
					{"project", `{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`},
				} {
					missing := call(input.tool, input.arguments)
					require.True(t, missing.IsError)
					require.Contains(t, missing.Content[0].Text, "[invalid_params]")
				}
				denied := call("project", `{"action":"list","workspace_slug":"unavailable"}`)
				require.True(t, denied.IsError)
				require.Contains(t, denied.Content[0].Text, "[forbidden]")
				// Existing header-scoped clients retain their default and conflict behavior.
				legacy := oauthToolCall(t, mode.handler, version, mode.token, "workspace-one", "project", `{"action":"list"}`, 200)
				require.False(t, legacy.IsError, legacy.Content)
				oauthToolCall(t, mode.handler, version, mode.token, "workspace-one", "project", `{"action":"list","workspace_slug":"workspace-two"}`, 400)
				actions := discoveredActions(t, oauthRPC(t, mode.handler, version, mode.token, "", "tools/list", map[string]any{}, 200))
				require.Contains(t, actions["workspace"], "list")
				require.Contains(t, actions["state"], "list")
				require.Contains(t, actions["workitem_type"], "list")
			})
		}
	}
}

func TestWorkspaceReadsRecheckLiveAccessAcrossCredentialsAndProtocols(t *testing.T) {
	for _, change := range []struct{ name, statement string }{
		{"inactive membership", `UPDATE workspace_members SET is_active=false WHERE workspace_id='20000000-0000-4000-8000-000000000001' AND member_id='10000000-0000-4000-8000-000000000001'`},
		{"deleted membership", `UPDATE workspace_members SET deleted_at=now() WHERE workspace_id='20000000-0000-4000-8000-000000000001' AND member_id='10000000-0000-4000-8000-000000000001'`},
		{"removed membership", `DELETE FROM workspace_members WHERE workspace_id='20000000-0000-4000-8000-000000000001' AND member_id='10000000-0000-4000-8000-000000000001'`},
		{"invalid membership role", `UPDATE workspace_members SET role=0 WHERE workspace_id='20000000-0000-4000-8000-000000000001' AND member_id='10000000-0000-4000-8000-000000000001'`},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=now() WHERE id='20000000-0000-4000-8000-000000000001'`},
	} {
		for _, mode := range []string{"personal token", "OAuth"} {
			for _, version := range []string{"2025-11-25", "2026-07-28"} {
				t.Run(change.name+"/"+mode+"/"+version, func(t *testing.T) {
					f, pool, auth, _ := oauthFixture(t)
					addMCPProjectFixture(t, pool)
					_, err := pool.Exec(t.Context(), `UPDATE workspaces SET name=slug;
 INSERT INTO workspace_members(workspace_id,member_id,role,is_active)
 VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',15,true);
 INSERT INTO workspaces(id,slug,name) VALUES('20000000-0000-4000-8000-000000000003','other-user-workspace','Other user workspace');
 INSERT INTO workspace_members(workspace_id,member_id,role,is_active)
 VALUES('20000000-0000-4000-8000-000000000003','10000000-0000-4000-8000-000000000002',15,true);
 UPDATE api_tokens SET workspace_id='20000000-0000-4000-8000-000000000001';`)
					require.NoError(t, err)
					handler, token := f.handler(t, nil), "test-one"
					if mode == "OAuth" {
						handler, token = oauthHandler(t, f, auth.Authenticate), "oauth-user"
					}
					call := func(input string) rpcResult {
						return oauthToolCall(t, handler, version, token, "", "workspace", input, 200)
					}
					before := call(`{"action":"list"}`)
					require.False(t, before.IsError, before.Content)
					require.JSONEq(t, `{"results":[{"id":"20000000-0000-4000-8000-000000000001","name":"workspace-one","slug":"workspace-one"},{"id":"20000000-0000-4000-8000-000000000002","name":"workspace-two","slug":"workspace-two"}]}`, string(before.StructuredContent))
					_, err = pool.Exec(t.Context(), change.statement)
					require.NoError(t, err)
					// Another user's surviving membership and token metadata cannot restore access.
					after := call(`{"action":"list"}`)
					require.False(t, after.IsError, after.Content)
					require.JSONEq(t, `{"results":[{"id":"20000000-0000-4000-8000-000000000002","name":"workspace-two","slug":"workspace-two"}]}`, string(after.StructuredContent))
					denied := call(`{"action":"retrieve","workspace_slug":"workspace-one"}`)
					require.True(t, denied.IsError, denied.Content)
					require.Empty(t, denied.StructuredContent)
					require.Contains(t, denied.Content[0].Text, "[forbidden]")
					project := oauthToolCall(t, handler, version, token, "", "project", `{"action":"list","workspace_slug":"workspace-one"}`, 200)
					require.True(t, project.IsError, project.Content)
					require.Empty(t, project.StructuredContent)
					require.Contains(t, project.Content[0].Text, "[forbidden]")
					allowed := call(`{"action":"retrieve","workspace_slug":"workspace-two"}`)
					require.False(t, allowed.IsError, allowed.Content)
					require.Contains(t, string(allowed.StructuredContent), `"slug":"workspace-two"`)
					require.NotContains(t, string(allowed.StructuredContent), "workspace-one")
				})
			}
		}
	}
}

func TestPersonalTokenUnscopedCallsRevalidateAfterAuthentication(t *testing.T) {
	for _, change := range []struct {
		name, statement string
		emptyList       bool
	}{
		{"token", `UPDATE api_tokens SET is_active=false`, false},
		{"account", `UPDATE users SET is_active=false`, false},
		{"membership", `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`, true},
	} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(change.name+"/"+version, func(t *testing.T) {
				f, pool := databaseFixture(t)
				h := f.handler(t, func(ctx context.Context, r *http.Request) (context.Context, error) {
					verified, err := f.auth.Authenticate(ctx, r)
					require.NoError(t, err)
					_, err = pool.Exec(ctx, change.statement)
					require.NoError(t, err)
					return verified, nil
				})
				result := oauthToolCall(t, h, version, "test-one", "", "workspace", `{"action":"list"}`, 200)
				if change.emptyList {
					require.False(t, result.IsError, result.Content)
					require.JSONEq(t, `{"results":[]}`, string(result.StructuredContent))
				} else {
					require.True(t, result.IsError, result.Content)
					require.Empty(t, result.StructuredContent)
					require.Contains(t, result.Content[0].Text, "[unauthorized]")
				}
			})
		}
	}
}

func TestPersonalTokenUnscopedDiscoveryUsesCurrentPermissions(t *testing.T) {
	f, pool := databaseFixture(t)
	addMCPProjectFixture(t, pool)
	h := f.handler(t, nil)
	list := func() map[string][]string {
		return discoveredActions(t, oauthRPC(t, h, "2026-07-28", "test-one", "", "tools/list", map[string]any{}, 200))
	}
	require.Contains(t, list()["state"], "create")
	_, err := pool.Exec(t.Context(), `UPDATE project_members SET role=5; UPDATE workspace_members SET role=5`)
	require.NoError(t, err)
	require.NotContains(t, list()["state"], "create")
	_, err = pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false`)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"member": {"me"}, "workspace": {"list"}}, list())
}
