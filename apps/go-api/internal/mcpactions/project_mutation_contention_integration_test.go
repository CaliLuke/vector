//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type concurrentMutationOutcome[T any] struct {
	value T
	err   error
}

func competingProjectWrites[T any](t *testing.T, f milestoneFixture, project uuid.UUID, calls [2]func(context.Context) (T, error)) [2]concurrentMutationOutcome[T] {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
	_, err = blocker.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, project)
	require.NoError(t, err)
	done := make(chan concurrentMutationOutcome[T], 2)
	for _, call := range calls {
		go func() {
			value, callErr := call(ctx)
			done <- concurrentMutationOutcome[T]{value, callErr}
		}()
	}
	require.Eventually(t, func() bool {
		var waiting int
		queryErr := f.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
		return queryErr == nil && waiting == 2
	}, 10*time.Second, 10*time.Millisecond, "both writes must wait before the competing commits")
	require.NoError(t, blocker.Commit(ctx))
	var results [2]concurrentMutationOutcome[T]
	for i := range results {
		select {
		case results[i] = <-done:
		case <-ctx.Done():
			t.Fatal("requests did not finish after releasing the project lock")
		}
	}
	return results
}
