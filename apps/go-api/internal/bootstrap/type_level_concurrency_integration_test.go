//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestTypeLevelMigrationDoesNotOverwriteConcurrentRepair(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, validTypeLevelsDown)
	id := uuid.New().String()
	f.exec(t, insertNumericType, id, f.workspace, "NaN", nil)
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	defer testsupport.Rollback(t, tx, f.ctx)
	_, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	connection, err := pgx.Connect(ctx, f.url)
	require.NoError(t, err)
	pid := connection.PgConn().PID()
	done := make(chan error, 1)
	finished := false
	go func() { _, migrationErr := connection.Exec(ctx, validTypeLevelsUp); done <- migrationErr }()
	t.Cleanup(func() {
		cancel()
		if !finished {
			if migrationErr := <-done; migrationErr != nil {
				t.Logf("cancelled migration during test cleanup: %v", migrationErr)
			}
		}
		testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	})
	var waitingErr error
	require.Eventually(t, func() bool {
		var waiting bool
		waitingErr = tx.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND NOT granted)`, pid).Scan(&waiting)
		return waitingErr != nil || waiting
	}, 5*time.Second, 10*time.Millisecond, "migration must wait for the concurrent repair")
	require.NoError(t, waitingErr)
	_, err = tx.Exec(f.ctx, `UPDATE issue_types SET level=1 WHERE id=$1`, id)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(f.ctx))
	select {
	case err := <-done:
		finished = true
		require.NoError(t, err)
	case <-f.ctx.Done():
		t.Fatal("queued migration did not finish")
	}
	var level int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT level::integer FROM issue_types WHERE id=$1`, id).Scan(&level))
	require.Equal(t, 1, level, "migration must re-read candidates after acquiring workspace locks")
}
