//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedWorkspaceMembersAndRolesReadAuthorizedPostgresState(t *testing.T) {
	f, pool := databaseFixture(t)
	h := f.handler(t, nil)
	_, err := pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000002'`)
	require.NoError(t, err)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			call := func(input string) rpcResult {
				return rpcVersion(t, h, "test-one", "tools/call", memberCall(input), version)
			}
			page := call(`{"action":"list_workspace","first_name":"ZO","per_page":1}`)
			require.False(t, page.IsError, page.Content)
			assert.JSONEq(t, `{"count":1,"extra_stats":null,"grouped_by":null,"next_cursor":"1:1:0","next_page_results":false,"prev_cursor":"1:-1:1","prev_page_results":false,"results":[{"id":"10000000-0000-4000-8000-000000000001","first_name":"Zoë","last_name":"First","email":"","avatar":"","avatar_url":null,"display_name":"first","role":15,"role_slug":"member","is_active":true,"is_bot":false}],"sub_grouped_by":null,"total_count":1,"total_pages":1,"total_results":1}`, string(page.StructuredContent))
			require.Len(t, page.Content, 1)
			assert.JSONEq(t, string(page.StructuredContent), page.Content[0].Text)
			for _, input := range []string{
				`{"action":"list_workspace","is_active":false}`, `{"action":"list_workspace","is_active":false,"is_bot":false}`,
			} {
				inactive := call(input)
				require.False(t, inactive.IsError, inactive.Content)
				assert.Contains(t, string(inactive.StructuredContent), "second@example.test")
				assert.NotContains(t, string(inactive.StructuredContent), "Zoë")
			}
			for _, input := range []string{`{"action":"list_workspace"}`, `{"action":"list_workspace","is_active":null,"is_bot":null,"per_page":0}`} {
				all := call(input)
				require.False(t, all.IsError, all.Content)
				var page struct {
					Total  int    `json:"total_count"`
					Cursor string `json:"next_cursor"`
				}
				require.NoError(t, json.Unmarshal(all.StructuredContent, &page))
				assert.Equal(t, 2, page.Total)
				assert.Equal(t, "100:1:0", page.Cursor)
			}

			for _, emptyCase := range []struct {
				name, input  string
				total, pages int
				previous     bool
			}{
				{"empty filter", `{"action":"list_workspace","first_name":"missing","per_page":1}`, 0, 0, false},
				{"workspace past end", `{"action":"list_workspace","per_page":1,"cursor":"1:99:0"}`, 2, 2, true},
				{"roles past end", `{"action":"list_roles","per_page":1,"cursor":"1:99:0"}`, 7, 7, true},
			} {
				t.Run(emptyCase.name, func(t *testing.T) {
					result := call(emptyCase.input)
					require.False(t, result.IsError, result.Content)
					var empty struct {
						Count    int
						Total    int `json:"total_count"`
						Pages    int `json:"total_pages"`
						Results  []json.RawMessage
						Next     bool `json:"next_page_results"`
						Previous bool `json:"prev_page_results"`
					}
					require.NoError(t, json.Unmarshal(result.StructuredContent, &empty))
					assert.Zero(t, empty.Count)
					require.NotNil(t, empty.Results)
					assert.Empty(t, empty.Results)
					assert.Equal(t, emptyCase.total, empty.Total)
					assert.Equal(t, emptyCase.pages, empty.Pages)
					assert.False(t, empty.Next)
					assert.Equal(t, emptyCase.previous, empty.Previous)
					require.Len(t, result.Content, 1)
					assert.JSONEq(t, string(result.StructuredContent), result.Content[0].Text)
				})
			}
			roles := call(`{"action":"list_roles","per_page":0}`)
			require.False(t, roles.IsError, roles.Content)
			var catalogue struct {
				Total   int    `json:"total_count"`
				Cursor  string `json:"next_cursor"`
				Results []map[string]any
			}
			require.NoError(t, json.Unmarshal(roles.StructuredContent, &catalogue))
			require.Len(t, catalogue.Results, 7)
			assert.Equal(t, 7, catalogue.Total)
			assert.Equal(t, "20:1:0", catalogue.Cursor)
			for _, role := range catalogue.Results {
				result := call(`{"action":"retrieve_role","role_id":"` + role["id"].(string) + `"}`)
				require.False(t, result.IsError, result.Content)
				encoded, err := json.Marshal(role)
				require.NoError(t, err)
				assert.JSONEq(t, string(encoded), string(result.StructuredContent))
			}
			workspace := call(`{"action":"list_roles","namespace":"workspace","per_page":1,"cursor":"1:1:0"}`)
			require.False(t, workspace.IsError, workspace.Content)
			require.NoError(t, json.Unmarshal(workspace.StructuredContent, &catalogue))
			require.Len(t, catalogue.Results, 1)
			assert.Equal(t, 4, catalogue.Total)
			assert.Equal(t, "admin", catalogue.Results[0]["slug"])
			missing := call(`{"action":"retrieve_role","role_id":"00000000-0000-4000-8000-000000000000"}`)
			require.True(t, missing.IsError)
			assert.Contains(t, missing.Content[0].Text, "[not_found]")
			for _, input := range []string{
				`{"action":"list_workspace","cursor":"bad"}`, `{"action":"list_workspace","order_by":"email; SELECT secret"}`,
				`{"action":"list_roles","cursor":"1:1:2"}`, `{"action":"list_roles","per_page":-1}`, `{"action":"list_roles","per_page":1001}`,
			} {
				invalid := call(input)
				require.True(t, invalid.IsError)
				assert.Contains(t, invalid.Content[0].Text, "[invalid_params]")
				assert.NotContains(t, invalid.Content[0].Text, "secret")
			}
			decimal := call(`{"action":"list_workspace","per_page":1,"cursor":"1.0:1:2"}`)
			require.False(t, decimal.IsError, decimal.Content)
		})
	}
}

func TestGeneratedDirectoryActionsRecheckRevocationAfterAuthentication(t *testing.T) {
	for _, tc := range []struct{ name, mutation, reset, code string }{
		{"workspace membership revoked", `UPDATE workspace_members SET is_active=false`, `UPDATE workspace_members SET is_active=true`, "forbidden"},
		{"workspace deleted", `UPDATE workspaces SET deleted_at=now()`, `UPDATE workspaces SET deleted_at=NULL`, "forbidden"},
		{"account inactive", `UPDATE users SET is_active=false`, `UPDATE users SET is_active=true`, "unauthorized"},
		{"token revoked", `UPDATE api_tokens SET is_active=false`, `UPDATE api_tokens SET is_active=true`, "unauthorized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, pool := databaseFixture(t)
			h := f.handler(t, func(ctx context.Context, r *http.Request) (context.Context, error) {
				authenticated, err := f.auth.Authenticate(ctx, r)
				require.NoError(t, err)
				_, err = pool.Exec(ctx, tc.mutation)
				require.NoError(t, err)
				return authenticated, nil
			})
			for _, input := range []string{`{"action":"list_workspace"}`, `{"action":"list_roles"}`, `{"action":"retrieve_role","role_id":"00000000-0000-4000-8000-000000000000"}`} {
				_, err := pool.Exec(t.Context(), tc.reset)
				require.NoError(t, err)
				result := rpc(t, h, "test-one", "tools/call", memberCall(input))
				require.True(t, result.IsError)
				assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
				assert.Empty(t, result.StructuredContent)
			}
		})
	}
}

func TestGeneratedDirectoryDoesNotCrossWorkspaceOrGrantGuestDirectoryAccess(t *testing.T) {
	f, pool := databaseFixture(t)
	h := f.handler(t, nil)
	_, err := pool.Exec(t.Context(), `INSERT INTO workspace_members(workspace_id,member_id,role,is_active) VALUES('20000000-0000-4000-8000-000000000002','10000000-0000-4000-8000-000000000001',5,true)`)
	require.NoError(t, err)
	other := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Workspace-Slug", "workspace-two")
		h.ServeHTTP(w, r)
	})
	roles := rpc(t, other, "test-one", "tools/call", memberCall(`{"action":"list_roles"}`))
	require.False(t, roles.IsError, roles.Content)
	var page struct{ Results []struct{ ID string } }
	require.NoError(t, json.Unmarshal(roles.StructuredContent, &page))
	require.NotEmpty(t, page.Results)
	cross := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"retrieve_role","role_id":"`+page.Results[0].ID+`"}`))
	require.True(t, cross.IsError)
	assert.Contains(t, cross.Content[0].Text, "[not_found]")
	denied := rpc(t, other, "test-one", "tools/call", memberCall(`{"action":"list_workspace"}`))
	require.True(t, denied.IsError)
	assert.Contains(t, denied.Content[0].Text, "[forbidden]")
}
