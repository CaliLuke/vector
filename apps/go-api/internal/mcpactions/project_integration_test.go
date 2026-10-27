//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const mcpProjectReadSchema = `
ALTER TABLE projects ADD COLUMN name text, ADD COLUMN identifier text, ADD COLUMN description text, ADD COLUMN emoji text, ADD COLUMN icon_prop jsonb,
 ADD COLUMN cover_image text, ADD COLUMN cover_image_asset_id uuid, ADD COLUMN archived_at timestamptz, ADD COLUMN created_at timestamptz,
 ADD COLUMN updated_at timestamptz, ADD COLUMN network int, ADD COLUMN archive_in int, ADD COLUMN close_in int, ADD COLUMN created_by_id uuid,
 ADD COLUMN cycle_view boolean, ADD COLUMN default_assignee_id uuid, ADD COLUMN default_state_id uuid, ADD COLUMN description_html text,
 ADD COLUMN description_text text, ADD COLUMN estimate_id uuid, ADD COLUMN external_id text, ADD COLUMN external_source text,
 ADD COLUMN guest_view_all_features boolean, ADD COLUMN intake_view boolean, ADD COLUMN is_issue_type_enabled boolean,
 ADD COLUMN is_time_tracking_enabled boolean, ADD COLUMN issue_views_view boolean, ADD COLUMN logo_props jsonb, ADD COLUMN module_view boolean,
 ADD COLUMN page_view boolean, ADD COLUMN project_lead_id uuid, ADD COLUMN timezone text, ADD COLUMN updated_by_id uuid;
ALTER TABLE project_members ADD COLUMN created_at timestamptz DEFAULT now(), ADD COLUMN sort_order double precision;
CREATE TABLE deploy_boards(anchor text,entity_name text,entity_identifier uuid,workspace_id uuid,deleted_at timestamptz,created_at timestamptz);
CREATE TABLE user_favorites(user_id uuid,entity_identifier uuid,entity_type text,project_id uuid,deleted_at timestamptz);
CREATE TABLE issue_sequences(project_id uuid,sequence bigint,deleted_at timestamptz);
CREATE TABLE project_user_properties(project_id uuid,workspace_id uuid,user_id uuid,deleted_at timestamptz,created_at timestamptz,sort_order double precision);
CREATE TABLE cycles(project_id uuid,deleted_at timestamptz);
CREATE TABLE modules(project_id uuid,deleted_at timestamptz);
UPDATE projects SET name='Hidden private project',identifier='HIDDEN',network=0;
UPDATE projects SET name='SDK project',identifier='SDK',network=2,icon_prop='{"name":"folder","color":"#444"}',description_html='<p>Project</p>' WHERE id='30000000-0000-4000-8000-000000000001';
UPDATE projects SET name='Foreign workspace project',identifier='FOREIGN',network=2 WHERE id='30000000-0000-4000-8000-000000000003';
INSERT INTO projects(id,workspace_id,name,identifier,network,archived_at) VALUES('30000000-0000-4000-8000-000000000004','20000000-0000-4000-8000-000000000001','Archived project','ZZ',2,'2024-01-02T03:04:05Z');
`

func addMCPProjectFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(t.Context(), mcpProjectReadSchema)
	require.NoError(t, err)
}

func TestProjectReadsUsePrimaryDataAcrossCredentialsAndProtocols(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addMCPProjectFixture(t, pool)
	pat := f.handler(t, nil)
	oauth := oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := func(input string) rpcResult {
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "project", input, 200)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", projectCall(input), version)
				}
				first := call(`{"action":"list","per_page":1,"order_by":"name"}`)
				require.False(t, first.IsError, first.Content)
				require.NotContains(t, string(first.StructuredContent), "Hidden private")
				require.NotContains(t, string(first.StructuredContent), "Foreign workspace")
				// Explicit tags keep the wire names independent of Go field spelling.
				var wire struct {
					TotalResults int    `json:"total_results"`
					Count        int    `json:"count"`
					NextCursor   string `json:"next_cursor"`
					Results      []struct {
						ID, Name   string
						ArchivedAt *string `json:"archived_at"`
					}
				}
				require.NoError(t, json.Unmarshal(first.StructuredContent, &wire))
				require.Equal(t, 2, wire.TotalResults)
				require.Equal(t, 1, wire.Count)
				require.Equal(t, "1:1:0", wire.NextCursor)
				require.Equal(t, "Archived project", wire.Results[0].Name)
				require.NotNil(t, wire.Results[0].ArchivedAt)
				next := call(`{"action":"list","per_page":1,"order_by":"name","cursor":"1:1:0"}`)
				require.False(t, next.IsError, next.Content)
				require.NoError(t, json.Unmarshal(next.StructuredContent, &wire))
				require.Equal(t, "SDK project", wire.Results[0].Name)
				detail := call(`{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`)
				require.False(t, detail.IsError, detail.Content)
				var expected map[string]any
				require.NoError(t, json.Unmarshal([]byte(projectDetailJSON), &expected))
				expected["total_cycles"], expected["total_modules"] = 0, 0
				expected["member_role"], expected["is_deployed"] = 15, false
				expected["workspace"] = "20000000-0000-4000-8000-000000000001"
				body, err := json.Marshal(expected)
				require.NoError(t, err)
				require.JSONEq(t, string(body), string(detail.StructuredContent))
				for _, id := range []string{"30000000-0000-4000-8000-000000000002", "30000000-0000-4000-8000-000000000003", "30000000-0000-4000-8000-000000000099"} {
					hidden := call(`{"action":"retrieve","project_id":"` + id + `"}`)
					require.True(t, hidden.IsError)
					require.Contains(t, hidden.Content[0].Text, "[not_found]")
					require.Empty(t, hidden.StructuredContent)
				}
			})
		}
	}
	_, err := pool.Exec(t.Context(), `UPDATE projects SET name='Changed on primary' WHERE id='30000000-0000-4000-8000-000000000001'`)
	require.NoError(t, err)
	changed := rpc(t, pat, "test-one", "tools/call", projectCall(`{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`))
	require.False(t, changed.IsError, changed.Content)
	require.Contains(t, string(changed.StructuredContent), "Changed on primary")
}

func TestProjectOAuthRequiresWorkspaceAndRechecksNarrowedScopes(t *testing.T) {
	f, pool, auth, issuer := oauthFixture(t)
	addMCPProjectFixture(t, pool)
	h := oauthHandler(t, f, auth.Authenticate)
	missing := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", "project", `{"action":"list"}`, 200)
	require.True(t, missing.IsError)
	require.Contains(t, missing.Content[0].Text, "[invalid_params]")
	selected := oauthToolCall(t, h, "2026-07-28", "oauth-user", "", "project", `{"action":"list","workspace_slug":"workspace-one"}`, 200)
	require.False(t, selected.IsError, selected.Content)
	oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", "project", `{"action":"list","workspace_slug":"workspace-two"}`, 400)
	narrowed := oauthHandler(t, f, func(ctx context.Context, r *http.Request) (context.Context, error) {
		verified, err := auth.Authenticate(ctx, r)
		issuer.mu.Lock()
		issuer.scope = ""
		issuer.mu.Unlock()
		return verified, err
	})
	result := oauthToolCall(t, narrowed, "2026-07-28", "oauth-user", "workspace-one", "project", `{"action":"list"}`, 200)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].Text, "[forbidden]")
	require.Empty(t, result.StructuredContent)
}

func TestProjectActionsRejectRevocationAfterHTTPAuthentication(t *testing.T) {
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, input := range []string{`{"action":"list"}`, `{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`} {
			t.Run(mode+input, func(t *testing.T) {
				f, pool, auth, _ := oauthFixture(t)
				revoke := func() {
					_, err := pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`)
					require.NoError(t, err)
				}
				var result rpcResult
				if mode == "OAuth" {
					h := oauthHandler(t, f, func(ctx context.Context, r *http.Request) (context.Context, error) {
						verified, err := auth.Authenticate(ctx, r)
						revoke()
						return verified, err
					})
					result = oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", "project", input, 200)
				} else {
					h := f.handler(t, func(ctx context.Context, r *http.Request) (context.Context, error) {
						verified, err := f.auth.Authenticate(ctx, r)
						revoke()
						return verified, err
					})
					result = rpc(t, h, "test-one", "tools/call", projectCall(input))
				}
				require.True(t, result.IsError)
				require.Contains(t, result.Content[0].Text, "[forbidden]")
				require.Empty(t, result.StructuredContent)
			})
		}
	}
}
