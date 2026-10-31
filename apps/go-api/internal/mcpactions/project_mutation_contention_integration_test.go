//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type concurrentMutationOutcome[T any] struct {
	value T
	err   error
}

const projectMutationLock = `SELECT id FROM projects WHERE id=$1 FOR UPDATE`
const schemaMutationLock = `SELECT work_item_schema_lock($1::uuid)`

func competingScopeWrites[T any](t *testing.T, f milestoneFixture, lockSQL string, scope uuid.UUID, calls [2]func(context.Context) (T, error)) [2]concurrentMutationOutcome[T] {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
	_, err = blocker.Exec(ctx, lockSQL, scope)
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
			t.Fatal("requests did not finish after releasing the scope lock")
		}
	}
	return results
}

func cancelScopeWriteWhileLocked(t *testing.T, f milestoneFixture, lockSQL string, scope uuid.UUID, call func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
	_, err = blocker.Exec(ctx, lockSQL, scope)
	require.NoError(t, err)
	requestCtx, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	done := make(chan error, 1)
	go func() { done <- call(requestCtx) }()
	require.Eventually(t, func() bool {
		var waiting bool
		queryErr := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND $1=ANY(pg_blocking_pids(pid)))`, blocker.Conn().PgConn().PID()).Scan(&waiting)
		return queryErr == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "request must reach the held scope lock before cancellation")
	cancelRequest()
	select {
	case err = <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-ctx.Done():
		t.Fatal("cancelled request did not return while the scope lock remained held")
	}
}

func assertConcurrentReceiptReplay(t *testing.T, first, second projectmutations.Result) {
	t.Helper()
	require.NotNil(t, first.Replayed)
	require.NotNil(t, second.Replayed)
	require.NotEqual(t, *first.Replayed, *second.Replayed)
	require.Equal(t, first.Revision, second.Revision)
	require.JSONEq(t, string(first.Record), string(second.Record))
}
