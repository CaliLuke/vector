//go:build integration

package bootstrap_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMilestoneSchemaConcurrentAssignmentHasOneWinner(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, milestone := range []string{f.milestone, f.otherMilestone} {
		db, err := pgx.Connect(f.ctx, f.url)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close(context.WithoutCancel(f.ctx))) })
		go func() {
			<-start
			_, failure := db.Exec(f.ctx, insertMilestoneLink, milestone, f.item, f.workspace, f.project)
			results <- failure
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first != nil {
		first, second = second, first
	}
	require.NoError(t, first)
	requireMilestoneSQLState(t, second, "23505")
	var links, totalRevision int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM milestone_issues WHERE deleted_at IS NULL`).Scan(&links))
	require.Equal(t, 1, links)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT sum(revision) FROM milestone_edit_revisions`).Scan(&totalRevision))
	require.Equal(t, 3, totalRevision)
}

func TestMilestoneSchemaSnapshotCannotDeleteNewlyLinkedTarget(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	reader, err := pgx.Connect(f.ctx, f.url)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Close(context.WithoutCancel(f.ctx))) })
	tx, err := reader.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) })
	var links int
	require.NoError(t, tx.QueryRow(f.ctx, `SELECT count(*) FROM milestone_issues`).Scan(&links))
	require.Zero(t, links)
	f.exec(t, insertMilestoneLink, f.milestone, f.item, f.workspace, f.project)
	_, err = tx.Exec(f.ctx, `UPDATE milestones SET deleted_at=now() WHERE id=$1`, f.milestone)
	// Foreign-key validation detects the newer reference outside the old snapshot.
	requireMilestoneSQLState(t, err, "23503")
	require.NoError(t, tx.Rollback(f.ctx))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM milestone_issues WHERE deleted_at IS NULL`).Scan(&links))
	require.Equal(t, 1, links)
	var live bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT deleted_at IS NULL FROM milestones WHERE id=$1`, f.milestone).Scan(&live))
	require.True(t, live)
}
