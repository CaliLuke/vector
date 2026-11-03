//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// Revocation is visible to the waiting request before its lock is released.
// Restore the precise previous credential state only after its assertions finish.
func revokeMutationCredential(t *testing.T, ctx context.Context, f milestoneFixture, change mutationWaitChange, projectID string) func() {
	t.Helper()
	if change.mode == "OAuth" && change.kind != "membership" {
		f.issuer.mu.Lock()
		active, scope := f.issuer.active, f.issuer.scope
		switch change.kind {
		case "scope":
			f.issuer.scope = "vector:read"
		case "token":
			f.issuer.active = false
		default:
			f.issuer.mu.Unlock()
			t.Fatalf("unknown authorization change %q", change.kind)
		}
		f.issuer.mu.Unlock()
		return func() {
			f.issuer.mu.Lock()
			defer f.issuer.mu.Unlock()
			f.issuer.active, f.issuer.scope = active, scope
		}
	}
	table, predicate := "api_tokens", "token=$1"
	args := []any{"test-one"}
	if change.kind == "membership" {
		table, predicate = "project_members", "member_id=$1 AND project_id=$2"
		args = []any{testsupport.LabelActor, projectID}
		if projectID == "" {
			table, predicate = "workspace_members", "member_id=$1 AND workspace_id=$2"
			args[1] = testsupport.LabelWorkspace
		}
	} else {
		require.Equal(t, "token", change.kind)
	}
	quoted := pgx.Identifier{table}.Sanitize()
	rows, err := f.pool.Query(ctx, "UPDATE "+quoted+" SET is_active=false WHERE is_active AND "+predicate+" RETURNING id::text", args...)
	require.NoError(t, err)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.NotEmpty(t, ids, "fixture credential must be active before revocation")
	return func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, err := f.pool.Exec(cleanupCtx, "UPDATE "+quoted+" SET is_active=true WHERE id::text=ANY($1::text[])", ids)
		require.NoError(t, err)
	}
}
