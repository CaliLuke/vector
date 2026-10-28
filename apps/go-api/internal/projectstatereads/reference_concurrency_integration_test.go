//go:build integration

package projectstatereads

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func stateReferenceInsert(table string) string {
	if table == "issues" {
		return `INSERT INTO issues(id,workspace_id,project_id,state_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Issue','{}','','none',0,1,false,now(),now())`
	}
	return `INSERT INTO draft_issues(id,workspace_id,project_id,state_id,name,description_json,description_html,priority,sort_order,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Draft','{}','','none',0,now(),now())`
}

func (f stateMutationFixture) waitForBlockedBy(t *testing.T, ctx context.Context, blocker uint32) uint32 {
	t.Helper()
	var waiting uint32
	require.Eventually(t, func() bool {
		// This observer uses its own statement transaction, not the controller's
		// cached statistics snapshot. The actual blocking PID proves the wait.
		err := f.db.QueryRow(ctx, `SELECT coalesce((SELECT pid FROM pg_stat_activity
 WHERE datname=current_database() AND $1::integer=ANY(pg_blocking_pids(pid)) LIMIT 1),0)`, blocker).Scan(&waiting)
		return err == nil && waiting != 0
	}, 5*time.Second, 10*time.Millisecond)
	return waiting
}

func TestStateDeletionWinningRaceRejectsConcurrentIssueAndDraft(t *testing.T) {
	for _, table := range []string{"issues", "draft_issues"} {
		t.Run(table, func(t *testing.T) {
			f := newStateMutationFixture(t)
			state := f.create(t, "Deleting")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			_, err := f.db.Exec(ctx, `CREATE FUNCTION hold_state_deletion() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN PERFORM pg_advisory_xact_lock(94612001); RETURN NEW; END $$;
 CREATE TRIGGER hold_state_deletion BEFORE UPDATE OF deleted_at ON states FOR EACH ROW EXECUTE FUNCTION hold_state_deletion()`)
			require.NoError(t, err)
			controller, err := f.db.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = controller.Rollback(context.WithoutCancel(ctx)) }()
			_, err = controller.Exec(ctx, `SELECT pg_advisory_xact_lock(94612001)`)
			require.NoError(t, err)
			input := f.input
			input.StateID = state
			deleted := make(chan error, 1)
			go func() { deleted <- f.repository.Delete(ctx, input) }()
			deletingPID := f.waitForBlockedBy(t, ctx, controller.Conn().PgConn().PID())
			assigned := make(chan error, 1)
			go func() {
				_, failure := f.db.Exec(ctx, stateReferenceInsert(table), uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, state)
				assigned <- failure
			}()
			f.waitForBlockedBy(t, ctx, deletingPID)
			require.NoError(t, controller.Rollback(ctx))
			require.NoError(t, <-deleted)
			var failure *pgconn.PgError
			require.ErrorAs(t, <-assigned, &failure)
			require.Equal(t, "23514", failure.Code)
			require.Equal(t, "item_state_must_be_live", failure.ConstraintName)
			var effects int
			require.NoError(t, f.db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects))
			require.Equal(t, 1, effects)
		})
	}
}

func TestStateAssignmentWinningRaceBlocksNativeDeletion(t *testing.T) {
	for _, table := range []string{"issues", "draft_issues"} {
		t.Run(table, func(t *testing.T) {
			f := newStateMutationFixture(t)
			state := f.create(t, "Occupied")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			tx, err := f.db.Begin(ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
			_, err = tx.Exec(ctx, stateReferenceInsert(table), uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, state)
			require.NoError(t, err)
			_, err = tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`)
			require.NoError(t, err)
			input := f.input
			input.StateID = state
			deleted := make(chan error, 1)
			go func() { deleted <- f.repository.Delete(ctx, input) }()
			f.waitForBlockedBy(t, ctx, tx.Conn().PgConn().PID())
			require.NoError(t, tx.Commit(ctx))
			var failure *WriteFailure
			require.ErrorAs(t, <-deleted, &failure)
			require.Equal(t, 400, failure.Status)
			var live bool
			var effects int
			require.NoError(t, f.db.QueryRow(ctx, `SELECT deleted_at IS NULL,(SELECT count(*) FROM work_item_effects) FROM states WHERE id=$1`, state).Scan(&live, &effects))
			require.True(t, live)
			require.Zero(t, effects)
		})
	}
}
