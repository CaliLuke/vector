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
	restore                func(*testing.T, context.Context, milestoneFixture, map[string]any)
}

type mutationWaitScenario struct {
	changes []mutationWaitChange
	// Opt in only when prepare creates independent action inputs. Each credential
	// change restores its original state before the next case starts; reference
	// mutations retain their own fixtures even when this option is enabled.
	reuseCredentials bool
	schemaLock       bool
	tool             string
	action           string
	fixture          func(*testing.T) milestoneFixture
	createFields     map[string]any
	snapshot         func(*testing.T, milestoneFixture) []byte
	prepare          func(*testing.T, milestoneFixture, catalogueCall) map[string]any
}

// Every action retains both credential bindings and OAuth write-scope enforcement,
// for new writes and replay. Callers use the full matrix at each distinct domain
// authorization boundary; repeating membership revocation for actions that share
// that boundary adds no separate permission rule.
func mutationCredentialChanges() []mutationWaitChange {
	return []mutationWaitChange{
		{name: "personal token revoked", mode: "personal token", kind: "token", code: "unauthorized"},
		{name: "OAuth token revoked", mode: "OAuth", kind: "token", code: "unauthorized"},
		{name: "OAuth write scope removed", mode: "OAuth", kind: "scope", code: "forbidden"},
	}
}

func runMutationAuthorizationWait(t *testing.T, scenario mutationWaitScenario) {
	t.Helper()
	changes := scenario.changes
	if changes == nil {
		changes = append(mutationCredentialChanges(),
			mutationWaitChange{name: "personal token membership revoked", mode: "personal token", kind: "membership", code: "forbidden"},
			mutationWaitChange{name: "OAuth membership revoked", mode: "OAuth", kind: "membership", code: "forbidden"},
		)
	}
	owner := t
	var credentialFixture *milestoneFixture
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			// A rejected new write leaves domain storage unchanged. Reuse that
			// fixture and request for receipt replay after restoring the case's
			// credential or explicitly reversible reference change. Other domain
			// changes keep isolated fixtures.
			var shared milestoneFixture
			var input map[string]any
			reusable := change.apply == nil || change.restore != nil
			if reusable {
				if scenario.reuseCredentials && change.apply == nil {
					if credentialFixture == nil {
						value := scenario.fixture(owner)
						credentialFixture = &value
					}
					shared = *credentialFixture
				} else {
					shared = scenario.fixture(t)
				}
			}
			for _, replay := range []bool{false, true} {
				phase := "new write"
				if replay {
					phase = "receipt replay"
				}
				if !t.Run(phase, func(t *testing.T) {
					f := shared
					if !reusable {
						f = scenario.fixture(t)
						input = nil
					}
					input = exerciseMutationAuthorizationWait(t, f, scenario, change, replay, input)
				}) {
					return
				}
			}
		})
	}
}

func exerciseMutationAuthorizationWait(t *testing.T, f milestoneFixture, scenario mutationWaitScenario, change mutationWaitChange, replay bool, input map[string]any) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	handler, token := f.handler(t, nil), "test-one"
	if change.mode == "OAuth" {
		handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
	}
	call := catalogueCall(func(tool string, input map[string]any) rpcResult {
		body, err := json.Marshal(input)
		require.NoError(t, err)
		return toolCall(t, handler, token, "", tool, string(body), http.StatusOK)
	})
	if input == nil {
		input = mutationWaitInput(t, f, scenario, call)
	}
	projectID, _ := input["project_id"].(string)
	arguments, err := json.Marshal(input)
	require.NoError(t, err)
	if replay {
		initial := milestoneObject(t, toolCall(t, handler, token, "", scenario.tool, string(arguments), http.StatusOK))
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
	request := mcpRequest(t, token, "", "tools/call", map[string]any{"name": scenario.tool, "arguments": json.RawMessage(arguments)}).WithContext(ctx)
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
		if change.restore != nil {
			t.Cleanup(func() {
				restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				change.restore(t, restoreCtx, f, input)
			})
		}
		// Commit the simulated scope writer before releasing its lock.
		if expected := change.apply(t, ctx, blocker, input); expected != nil {
			before = expected
		}
	} else {
		restore := revokeMutationCredential(t, ctx, f, change, projectID)
		t.Cleanup(restore)
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
	return input
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
