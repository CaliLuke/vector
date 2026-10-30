//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type projectMutationWaitScenario struct {
	tool         string
	action       string
	fixture      func(*testing.T) milestoneFixture
	createFields map[string]any
	snapshot     func(*testing.T, milestoneFixture) []byte
}

func runProjectMutationAuthorizationWait(t *testing.T, scenario projectMutationWaitScenario) {
	t.Helper()
	for _, change := range []struct{ name, mode, kind, code string }{
		{"personal token revoked", "personal token", "token", "unauthorized"},
		{"OAuth token revoked", "OAuth", "token", "unauthorized"},
		{"OAuth write scope removed", "OAuth", "scope", "forbidden"},
		{"personal token project membership revoked", "personal token", "membership", "forbidden"},
		{"OAuth project membership revoked", "OAuth", "membership", "forbidden"},
	} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			for _, replay := range []bool{false, true} {
				phase := "new write"
				if replay {
					phase = "receipt replay"
				}
				t.Run(change.name+"/"+version+"/"+phase, func(t *testing.T) {
					f := scenario.fixture(t)
					ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
					defer cancel()
					handler, token := f.handler(t, nil), "test-one"
					if change.mode == "OAuth" {
						handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
					}
					createInput := map[string]any{"action": "create", "workspace_slug": "workspace-one",
						"project_id": testsupport.LabelProject, "name": uuid.NewString()}
					maps.Copy(createInput, scenario.createFields)
					if scenario.tool == "project" {
						delete(createInput, "project_id")
					}
					create, err := json.Marshal(createInput)
					require.NoError(t, err)
					created := milestoneObject(t, oauthToolCall(t, handler, version, token, "", scenario.tool, string(create), http.StatusOK))
					projectID := testsupport.LabelProject
					if scenario.tool == "project" {
						projectID = created["id"].(string)
					}
					action := scenario.action
					if action == "" {
						action = "update"
					}
					input := map[string]any{"action": action, "workspace_slug": "workspace-one", "project_id": projectID,
						scenario.tool + "_id": created["id"], "description": "accepted only with current authorization", "request_id": uuid.NewString()}
					if action == "delete" {
						delete(input, "description")
					}
					arguments, err := json.Marshal(input)
					require.NoError(t, err)
					if replay {
						initial := milestoneObject(t, oauthToolCall(t, handler, version, token, "", scenario.tool, string(arguments), http.StatusOK))
						require.Equal(t, false, initial["replayed"])
					}
					before := scenario.snapshot(t, f)
					blocker, err := f.pool.Begin(ctx)
					require.NoError(t, err)
					defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
					_, err = blocker.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, projectID)
					require.NoError(t, err)
					request := oauthRequest(t, version, token, "", "tools/call", map[string]any{"name": scenario.tool, "arguments": json.RawMessage(arguments)}).WithContext(ctx)
					response := httptest.NewRecorder()
					done := make(chan struct{})
					go func() { handler.ServeHTTP(response, request); close(done) }()
					require.Eventually(t, func() bool {
						var waiting bool
						queryErr := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity
 WHERE datname=current_database() AND $1=ANY(pg_blocking_pids(pid)))`, blocker.Conn().PgConn().PID()).Scan(&waiting)
						return queryErr == nil && waiting
					}, 10*time.Second, 10*time.Millisecond, "request never waited on the held project lock")
					switch change.kind {
					case "membership":
						_, err = f.pool.Exec(ctx, `UPDATE project_members SET is_active=false WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, projectID)
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
						t.Fatal("request did not finish after the project lock was released")
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
					require.Equal(t, before, scenario.snapshot(t, f))
				})
			}
		}
	}
}
