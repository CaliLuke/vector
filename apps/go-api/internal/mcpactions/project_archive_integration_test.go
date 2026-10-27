//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func projectLifecycle(t *testing.T, pool *pgxpool.Pool) *projects.LifecycleRepository {
	t.Helper()
	lifecycle, err := projects.NewLifecycleRepository(pool, rosterPublisher{}, "https://vector.example", func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) { return false, nil })
	require.NoError(t, err)
	return lifecycle
}

func (rosterPublisher) SoftDelete(context.Context, string, string, string) error { return nil }

func addArchiveFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	addMCPProjectFixture(t, pool)
	_, err := pool.Exec(t.Context(), `ALTER TABLE user_favorites ADD COLUMN workspace_id uuid;
 UPDATE project_members SET role=20 WHERE member_id='10000000-0000-4000-8000-000000000001';
 INSERT INTO user_favorites(user_id,entity_identifier,entity_type,project_id,workspace_id)
 VALUES('10000000-0000-4000-8000-000000000001','30000000-0000-4000-8000-000000000001','project','30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001');`)
	require.NoError(t, err)
	for _, migration := range []string{"000016_mutation_receipts", "000017_project_edit_revisions"} {
		sql, readErr := os.ReadFile("../bootstrap/schema_migrations/" + migration + ".up.sql")
		require.NoError(t, readErr)
		_, err = pool.Exec(t.Context(), string(sql))
		require.NoError(t, err)
	}
}

func TestProjectArchiveRecoveryAcrossCredentialsAndProtocols(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				f, pool, auth, issuer := oauthFixture(t)
				addArchiveFixture(t, pool)
				issuer.mu.Lock()
				issuer.scope = "vector:read vector:write"
				issuer.mu.Unlock()
				pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
				call := func(input string) rpcResult {
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "project", input, 200)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", projectCall(input), version)
				}
				const projectID = "30000000-0000-4000-8000-000000000001"
				for _, target := range []string{"30000000-0000-4000-8000-000000000003", "30000000-0000-4000-8000-000000000099"} {
					for _, action := range []string{"archive", "unarchive"} {
						denied := call(fmt.Sprintf(`{"action":%q,"project_id":%q,"request_id":%q}`, action, target, uuid.NewString()))
						require.True(t, denied.IsError)
						require.Contains(t, denied.Content[0].Text, "[not_found]")
						require.Empty(t, denied.StructuredContent)
					}
				}
				detail := call(`{"action":"retrieve","project_id":"` + projectID + `"}`)
				require.False(t, detail.IsError, detail.Content)
				require.NotContains(t, string(detail.StructuredContent), `"revision"`)
				readRevision := func() string {
					result := call(`{"action":"retrieve","project_id":"` + projectID + `","include_revision":true}`)
					require.False(t, result.IsError, result.Content)
					var body struct{ Revision string }
					require.NoError(t, json.Unmarshal(result.StructuredContent, &body))
					require.NotEmpty(t, body.Revision)
					return body.Revision
				}
				initial := readRevision()
				requestID := uuid.NewString()
				input := fmt.Sprintf(`{"action":"archive","project_id":%q,"request_id":%q,"expected_revision":%q}`, projectID, requestID, initial)
				first := call(input)
				require.False(t, first.IsError, first.Content)
				var receipt struct {
					ProjectID       string `json:"project_id"`
					RequestID       string `json:"request_id"`
					Revision        string
					ReplayExpiresAt time.Time `json:"replay_expires_at"`
					Replayed        bool
				}
				require.NoError(t, json.Unmarshal(first.StructuredContent, &receipt))
				require.Equal(t, projectID, receipt.ProjectID)
				require.Equal(t, requestID, receipt.RequestID)
				require.False(t, receipt.Replayed)
				require.Greater(t, time.Until(receipt.ReplayExpiresAt), 24*time.Hour)
				require.NotEqual(t, initial, receipt.Revision)
				after := readRevision()
				require.Equal(t, receipt.Revision, after)
				replay := call(input)
				require.False(t, replay.IsError, replay.Content)
				originalExpiry := receipt.ReplayExpiresAt
				require.NoError(t, json.Unmarshal(replay.StructuredContent, &receipt))
				require.True(t, receipt.Replayed)
				require.Equal(t, originalExpiry, receipt.ReplayExpiresAt)
				require.Equal(t, after, readRevision())
				changed := call(fmt.Sprintf(`{"action":"archive","project_id":%q,"request_id":%q}`, projectID, requestID))
				require.True(t, changed.IsError)
				require.Contains(t, changed.Content[0].Text, "[request_conflict]")
				stale := call(fmt.Sprintf(`{"action":"unarchive","project_id":%q,"expected_revision":%q}`, projectID, initial))
				require.True(t, stale.IsError)
				require.Contains(t, stale.Content[0].Text, "[revision_conflict]")
				require.Equal(t, after, readRevision())
				var archived, favoriteDeleted bool
				require.NoError(t, pool.QueryRow(t.Context(), `SELECT archived_at IS NOT NULL FROM projects WHERE id=$1`, projectID).Scan(&archived))
				require.True(t, archived)
				require.NoError(t, pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM user_favorites WHERE project_id=$1`, projectID).Scan(&favoriteDeleted))
				require.True(t, favoriteDeleted)
				for _, action := range []string{"unarchive", "archive", "unarchive"} {
					result := call(fmt.Sprintf(`{"action":%q,"project_id":%q}`, action, projectID))
					require.False(t, result.IsError, result.Content)
					require.Empty(t, result.Content)
					require.Empty(t, result.StructuredContent)
				}
				require.NoError(t, pool.QueryRow(t.Context(), `SELECT archived_at IS NOT NULL FROM projects WHERE id=$1`, projectID).Scan(&archived))
				require.False(t, archived)
				// A receipt never bypasses fresh authorization.
				_, err := pool.Exec(t.Context(), `UPDATE workspace_members SET role=5 WHERE member_id='10000000-0000-4000-8000-000000000001'`)
				require.NoError(t, err)
				denied := call(input)
				require.True(t, denied.IsError)
				require.Contains(t, denied.Content[0].Text, "[forbidden]")
				require.Empty(t, denied.StructuredContent)
			})
		}
	}
}

func TestProjectDiscoveryTracksScopesAndDomainPermissionsWithoutSharedMutation(t *testing.T) {
	f, pool, auth, issuer := oauthFixture(t)
	addArchiveFixture(t, pool)
	pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
	assertActions := func(result rpcResult, archive, unarchive bool) {
		t.Helper()
		require.GreaterOrEqual(t, len(result.Tools), 2)
		schema := discoveredTool(t, result, "project").InputSchema
		actions := string(schema.Properties["action"])
		require.Contains(t, actions, `"list"`)
		require.Contains(t, actions, `"retrieve"`)
		for action, allowed := range map[string]bool{"archive": archive, "unarchive": unarchive} {
			if allowed {
				require.Contains(t, actions, `"`+action+`"`)
			} else {
				require.NotContains(t, actions, `"`+action+`"`)
			}
		}
		if !archive && !unarchive {
			require.NotContains(t, schema.Properties, "request_id")
			require.NotContains(t, schema.Properties, "expected_revision")
			require.Len(t, schema.OneOf, 2)
		} else {
			require.Contains(t, schema.Properties, "request_id")
		}
	}
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, writable := range []bool{false, true, false, true} {
			issuer.mu.Lock()
			issuer.scope = "vector:read"
			if writable {
				issuer.scope += " vector:write"
			}
			issuer.mu.Unlock()
			listed := oauthRPC(t, oauth, version, "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
			assertActions(listed, writable, writable)
			if !writable {
				denied := oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", "project", `{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001"}`, 200)
				require.True(t, denied.IsError)
				require.Contains(t, denied.Content[0].Text, "[forbidden]")
			}
		}
		for _, token := range []string{"test-one", "test-two", "test-one"} {
			listed := rpcVersion(t, pat, token, "tools/list", map[string]any{}, version)
			assertActions(listed, true, token == "test-one")
		}
	}
	_, err := pool.Exec(t.Context(), `UPDATE workspace_members SET role=5 WHERE member_id='10000000-0000-4000-8000-000000000002'`)
	require.NoError(t, err)
	assertActions(rpc(t, pat, "test-two", "tools/list", map[string]any{}), false, false)
	assertActions(rpc(t, pat, "test-one", "tools/list", map[string]any{}), true, true)
}

func TestProjectOAuthMutationRequiresWorkspaceAndDiscoveryRechecksReadScope(t *testing.T) {
	f, pool, auth, issuer := oauthFixture(t)
	addArchiveFixture(t, pool)
	issuer.mu.Lock()
	issuer.scope = "vector:read vector:write"
	issuer.mu.Unlock()
	result := oauthToolCall(t, oauthHandler(t, f, auth.Authenticate), "2026-07-28", "oauth-user", "", "project", `{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001"}`, 200)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].Text, "[invalid_params]")
	h := oauthHandler(t, f, func(ctx context.Context, r *http.Request) (context.Context, error) {
		verified, err := auth.Authenticate(ctx, r)
		issuer.mu.Lock()
		issuer.scope = ""
		issuer.mu.Unlock()
		return verified, err
	})
	listed := oauthRPC(t, h, "2026-07-28", "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
	require.Empty(t, listed.Tools)
}
