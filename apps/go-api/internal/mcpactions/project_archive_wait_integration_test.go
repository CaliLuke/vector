//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestProjectArchiveRevalidatesCredentialAfterProjectLockWait(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, change := range []struct{ name, code string }{
			{"personal token revoked", "unauthorized"}, {"OAuth token revoked", "unauthorized"}, {"OAuth write scope removed", "forbidden"},
		} {
			for _, action := range []string{"archive", "unarchive"} {
				for _, replay := range []bool{false, true} {
					phase := "new write"
					if replay {
						phase = "receipt replay"
					}
					t.Run(version+"/"+change.name+"/"+action+"/"+phase, func(t *testing.T) {
						f, pool, auth, issuer := oauthFixture(t)
						addArchiveFixture(t, pool)
						const projectID = "30000000-0000-4000-8000-000000000001"
						if action == "unarchive" {
							_, err := pool.Exec(t.Context(), `UPDATE projects SET archived_at=clock_timestamp() WHERE id=$1`, projectID)
							require.NoError(t, err)
						}
						issuer.mu.Lock()
						issuer.scope = "vector:read vector:write"
						issuer.mu.Unlock()
						handler, token := f.handler(t, nil), "test-one"
						if change.name != "personal token revoked" {
							handler, token = oauthHandler(t, f, auth.Authenticate), "oauth-user"
						}
						input, err := json.Marshal(map[string]any{"action": action, "workspace_slug": "workspace-one", "project_id": projectID, "request_id": uuid.NewString()})
						require.NoError(t, err)
						if replay {
							milestoneObject(t, oauthToolCall(t, handler, version, token, "", "project", string(input), http.StatusOK))
						}
						before := projectArchiveSnapshot(t, pool)
						ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
						defer cancel()
						blocker, err := pool.Begin(ctx)
						require.NoError(t, err)
						defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
						_, err = blocker.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, projectID)
						require.NoError(t, err)
						request := oauthRequest(t, version, token, "", "tools/call", map[string]any{"name": "project", "arguments": json.RawMessage(input)}).WithContext(ctx)
						response := httptest.NewRecorder()
						done := make(chan struct{})
						go func() { handler.ServeHTTP(response, request); close(done) }()
						require.Eventually(t, func() bool {
							var waiting bool
							queryErr := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND $1=ANY(pg_blocking_pids(pid)))`, blocker.Conn().PgConn().PID()).Scan(&waiting)
							return queryErr == nil && waiting
						}, 10*time.Second, 10*time.Millisecond, "archive request did not wait for the held project lock")
						switch change.name {
						case "personal token revoked":
							_, err = pool.Exec(ctx, `UPDATE api_tokens SET is_active=false WHERE token='test-one'`)
							require.NoError(t, err)
						case "OAuth token revoked":
							issuer.mu.Lock()
							issuer.active = false
							issuer.mu.Unlock()
						case "OAuth write scope removed":
							issuer.mu.Lock()
							issuer.scope = "vector:read"
							issuer.mu.Unlock()
						}
						require.NoError(t, blocker.Commit(ctx))
						select {
						case <-done:
						case <-ctx.Done():
							t.Fatal("archive request did not finish after releasing the project lock")
						}
						require.Equal(t, http.StatusOK, response.Code, response.Body.String())
						var result struct {
							Result rpcResult
							Error  json.RawMessage
						}
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
						require.Empty(t, result.Error, response.Body.String())
						milestoneFailure(t, result.Result, change.code)
						require.Empty(t, result.Result.StructuredContent)
						require.Equal(t, before, projectArchiveSnapshot(t, pool))
					})
				}
			}
		}
	}
}

func projectArchiveSnapshot(t *testing.T, pool *pgxpool.Pool) []byte {
	t.Helper()
	var value []byte
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'projects',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p),
 'favorites',(SELECT jsonb_agg(to_jsonb(f) ORDER BY project_id,user_id,entity_identifier) FROM user_favorites f),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_edit_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,action,request_id) FROM mutation_receipts r))`).Scan(&value))
	return value
}
