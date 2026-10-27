//go:build integration

package mcpactions_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedProjectMemberActionReadsRealAuthorizedRows(t *testing.T) {
	f, pool := databaseFixture(t)
	h := f.handler(t, nil)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			result := rpcVersion(t, h, "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`), version)
			require.False(t, result.IsError, result.Content)
			require.Len(t, result.Content, 1)
			assert.Empty(t, result.StructuredContent)
			assert.JSONEq(t, `[{"id":"10000000-0000-4000-8000-000000000001","first_name":"Zoë","last_name":"First","email":null,"avatar":"","avatar_url":null,"display_name":"first","role":null,"role_slug":null,"is_active":null,"is_bot":null}]`, result.Content[0].Text)
			for _, project := range []string{projectTwo, "30000000-0000-4000-8000-000000000003"} {
				denied := rpcVersion(t, h, "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+project+`"}`), version)
				require.True(t, denied.IsError)
				assert.Contains(t, denied.Content[0].Text, "[forbidden]")
				assert.NotContains(t, denied.Content[0].Text, "second@example.test")
			}
		})
	}
	_, err := pool.Exec(t.Context(), `UPDATE users SET first_name='Updated roster' WHERE id='10000000-0000-4000-8000-000000000001'`)
	require.NoError(t, err)
	updated := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`))
	require.False(t, updated.IsError)
	assert.Contains(t, updated.Content[0].Text, "Updated roster")
}

func TestProjectMemberActionRechecksRevocationAfterRequestAuthentication(t *testing.T) {
	for _, tc := range []struct{ name, mutation, code string }{
		{"project membership removed", `DELETE FROM project_members WHERE member_id='10000000-0000-4000-8000-000000000001'`, "forbidden"},
		{"project membership deactivated", `UPDATE project_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`, "forbidden"},
		{"project deleted", `UPDATE projects SET deleted_at=CURRENT_TIMESTAMP WHERE id='` + projectOne + `'`, "forbidden"},
		{"token revoked", `UPDATE api_tokens SET is_active=false WHERE token='test-one'`, "unauthorized"},
		{"workspace membership removed", `DELETE FROM workspace_members WHERE member_id='10000000-0000-4000-8000-000000000001'`, "forbidden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, pool := databaseFixture(t)
			h := f.handler(t, func(ctx context.Context, request *http.Request) (context.Context, error) {
				verified, err := f.auth.Authenticate(ctx, request)
				require.NoError(t, err)
				_, err = pool.Exec(ctx, tc.mutation)
				require.NoError(t, err)
				return verified, nil
			})
			result := rpc(t, h, "test-one", "tools/call", memberCall(`{"action":"list_project","project_id":"`+projectOne+`"}`))
			require.True(t, result.IsError)
			assert.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			assert.NotContains(t, result.Content[0].Text, "Zoë")
			assert.Empty(t, result.StructuredContent)
		})
	}
}
