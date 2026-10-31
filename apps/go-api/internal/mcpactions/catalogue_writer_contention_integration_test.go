//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/stretchr/testify/require"
)

func contendWithNativeCatalogueWriter[N, T any](t *testing.T, f milestoneFixture, trigger string, nativeWrite func(context.Context) (N, error), competingWrite func(context.Context) (T, error)) (concurrentMutationOutcome[N], concurrentMutationOutcome[T]) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
	_, err = blocker.Exec(ctx, `SELECT pg_advisory_xact_lock(720010)`)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, trigger)
	require.NoError(t, err)
	nativeDone := make(chan concurrentMutationOutcome[N], 1)
	go func() {
		value, writeErr := nativeWrite(ctx)
		nativeDone <- concurrentMutationOutcome[N]{value, writeErr}
	}()
	waitCatalogueWriters(t, ctx, f, 1)
	competingDone := make(chan concurrentMutationOutcome[T], 1)
	go func() {
		value, writeErr := competingWrite(ctx)
		competingDone <- concurrentMutationOutcome[T]{value, writeErr}
	}()
	waitCatalogueWriters(t, ctx, f, 2)
	require.NoError(t, blocker.Commit(ctx))
	var native concurrentMutationOutcome[N]
	var competing concurrentMutationOutcome[T]
	select {
	case native = <-nativeDone:
	case <-ctx.Done():
		t.Fatal("native writer did not finish")
	}
	select {
	case competing = <-competingDone:
	case <-ctx.Done():
		t.Fatal("competing writer did not finish")
	}
	return native, competing
}

func waitCatalogueWriters(t *testing.T, ctx context.Context, f milestoneFixture, count int) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting int
		err := f.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
		return err == nil && waiting == count
	}, 5*time.Second, 10*time.Millisecond, "expected %d blocked writers before releasing their locks", count)
}
