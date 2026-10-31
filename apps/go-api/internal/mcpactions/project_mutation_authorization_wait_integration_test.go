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
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type mutationWaitChange struct {
	name, mode, kind, code string
	apply                  func(*testing.T, context.Context, pgx.Tx, map[string]any) []byte
}

type mutationWaitScenario struct {
	changes      []mutationWaitChange
	schemaLock   bool
	tool         string
	action       string
	fixture      func(*testing.T) milestoneFixture
	createFields map[string]any
	snapshot     func(*testing.T, milestoneFixture) []byte
	prepare      func(*testing.T, milestoneFixture, catalogueCall) map[string]any
}

func runMutationAuthorizationWait(t *testing.T, scenario mutationWaitScenario) {
	t.Helper()
	changes := scenario.changes
	if changes == nil {
		changes = []mutationWaitChange{
			{"personal token revoked", "personal token", "token", "unauthorized", nil},
			{"OAuth token revoked", "OAuth", "token", "unauthorized", nil},
			{"OAuth write scope removed", "OAuth", "scope", "forbidden", nil},
			{"personal token membership revoked", "personal token", "membership", "forbidden", nil},
			{"OAuth membership revoked", "OAuth", "membership", "forbidden", nil},
		}
	}
	for _, change := range changes {
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
					call := catalogueCall(func(tool string, input map[string]any) rpcResult {
						body, err := json.Marshal(input)
						require.NoError(t, err)
						return oauthToolCall(t, handler, version, token, "", tool, string(body), http.StatusOK)
					})
					input := mutationWaitInput(t, f, scenario, call)
					projectID, _ := input["project_id"].(string)
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
					lockSQL, lockID := projectMutationLock, projectID
					if scenario.schemaLock {
						lockSQL, lockID = schemaMutationLock, testsupport.LabelWorkspace
					}
					_, err = blocker.Exec(ctx, lockSQL, lockID)
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
					}, 10*time.Second, 10*time.Millisecond, "request never waited on the held scope lock")
					if change.apply != nil {
						// Commit the simulated scope writer before releasing its lock.
						if expected := change.apply(t, ctx, blocker, input); expected != nil {
							before = expected
						}
					} else {
						switch change.kind {
						case "membership":
							if projectID == "" {
								_, err = f.pool.Exec(ctx, `UPDATE workspace_members SET is_active=false WHERE member_id=$1 AND workspace_id=$2`, testsupport.LabelActor, testsupport.LabelWorkspace)
							} else {
								_, err = f.pool.Exec(ctx, `UPDATE project_members SET is_active=false WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, projectID)
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
					}
					require.NoError(t, blocker.Commit(ctx))
					select {
					case <-done:
					case <-ctx.Done():
						t.Fatal("request did not finish after the scope lock was released")
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

// Request construction belongs to the resource fixture; the wait harness owns
// the concurrent revocation and verifies that no mutation escaped it.
func mutationWaitInput(t *testing.T, f milestoneFixture, scenario mutationWaitScenario, call catalogueCall) map[string]any {
	t.Helper()
	if scenario.prepare != nil {
		return scenario.prepare(t, f, call)
	}
	createInput := map[string]any{"action": "create", "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "name": uuid.NewString()}
	maps.Copy(createInput, scenario.createFields)
	if scenario.tool == "project" {
		delete(createInput, "project_id")
	}
	created := milestoneObject(t, call(scenario.tool, createInput))
	projectID := testsupport.LabelProject
	if scenario.tool == "project" {
		projectID = created["id"].(string)
	}
	action := scenario.action
	if action == "" {
		action = "update"
	}
	input := map[string]any{"action": action, "workspace_slug": "workspace-one", "project_id": projectID, scenario.tool + "_id": created["id"], "description": "accepted only with current authorization", "request_id": uuid.NewString()}
	if action == "delete" {
		delete(input, "description")
	}
	return input
}
