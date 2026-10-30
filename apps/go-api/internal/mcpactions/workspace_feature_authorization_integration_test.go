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
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFeatureWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, tool := range []string{"workspace", "project"} {
		for _, change := range []struct{ name, mode, kind, code string }{
			{"personal token revoked", "personal token", "token", "unauthorized"},
			{"OAuth token revoked", "OAuth", "token", "unauthorized"},
			{"OAuth write scope removed", "OAuth", "scope", "forbidden"},
			{"personal token administrator demoted", "personal token", "role", "forbidden"},
			{"OAuth administrator demoted", "OAuth", "role", "forbidden"},
		} {
			for _, version := range []string{"2025-11-25", "2026-07-28"} {
				for _, phase := range []string{"new write", "receipt replay"} {
					t.Run(tool+"/"+change.name+"/"+version+"/"+phase, func(t *testing.T) {
						f := newMilestoneFixture(t)
						ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
						defer cancel()
						_, err := f.pool.Exec(ctx, `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20; UPDATE workspace_features SET customers=false; UPDATE projects SET cycle_view=false`)
						require.NoError(t, err)
						handler, token := f.handler(t, nil), "test-one"
						if change.mode == "OAuth" {
							handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
						}
						input := map[string]any{"action": "update_features", "workspace_slug": "workspace-one", "customers": true, "request_id": uuid.NewString()}
						if tool == "project" {
							delete(input, "customers")
							input["project_id"], input["cycles"] = testsupport.LabelProject, true
						}
						arguments, err := json.Marshal(input)
						require.NoError(t, err)
						if phase == "receipt replay" {
							initial := milestoneObject(t, oauthToolCall(t, handler, version, token, "", tool, string(arguments), http.StatusOK))
							require.Equal(t, false, initial["replayed"])
						}
						before := featureMutationSnapshot(t, f)
						blocker, err := f.pool.Begin(ctx)
						require.NoError(t, err)
						defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
						_, err = blocker.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, testsupport.LabelWorkspace)
						require.NoError(t, err)
						request := oauthRequest(t, version, token, "", "tools/call", map[string]any{"name": tool, "arguments": json.RawMessage(arguments)}).WithContext(ctx)
						response := httptest.NewRecorder()
						done := make(chan struct{})
						go func() { handler.ServeHTTP(response, request); close(done) }()
						require.Eventually(t, func() bool {
							var waiting bool
							queryErr := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory')`).Scan(&waiting)
							return queryErr == nil && waiting
						}, 10*time.Second, 10*time.Millisecond, "feature request never waited before transaction authorization")
						switch change.kind {
						case "role":
							if tool == "project" {
								_, err = f.pool.Exec(ctx, `UPDATE project_members SET role=15 WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, testsupport.LabelProject)
							} else {
								_, err = f.pool.Exec(ctx, `UPDATE workspace_members SET role=15 WHERE member_id=$1 AND workspace_id=$2`, testsupport.LabelActor, testsupport.LabelWorkspace)
							}
							require.NoError(t, err)
						case "scope":
							f.issuer.mu.Lock()
							f.issuer.scope = "vector:read"
							f.issuer.mu.Unlock()
						case "token":
							if change.mode == "OAuth" {
								f.issuer.mu.Lock()
								f.issuer.active = false
								f.issuer.mu.Unlock()
							} else {
								_, err = f.pool.Exec(ctx, `UPDATE api_tokens SET is_active=false WHERE token='test-one'`)
								require.NoError(t, err)
							}
						default:
							t.Fatal("unknown authorization change")
						}
						require.NoError(t, blocker.Commit(ctx))
						select {
						case <-done:
						case <-ctx.Done():
							t.Fatal("feature request did not finish after releasing admission")
						}
						require.Equal(t, http.StatusOK, response.Code, response.Body.String())
						var result struct {
							Result rpcResult
							Error  json.RawMessage
						}
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
						require.Empty(t, result.Error, response.Body.String())
						milestoneFailure(t, result.Result, change.code)
						require.JSONEq(t, string(before), string(featureMutationSnapshot(t, f)))
					})
				}
			}
		}
	}
}

func featureMutationSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'flags',(SELECT to_jsonb(f) FROM workspace_features f WHERE workspace_id=$1),
 'projects',COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p WHERE workspace_id=$1),'[]'::jsonb),
 'intakes',COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM intakes i WHERE workspace_id=$1),'[]'::jsonb),
 'events',COALESCE((SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM feature_events e WHERE workspace_id=$1),'[]'::jsonb),
 'receipts',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY tool,project_id,request_id) FROM mutation_receipts r WHERE workspace_id=$1 AND action='update_features'),'[]'::jsonb))`, testsupport.LabelWorkspace).Scan(&snapshot))
	return snapshot
}
