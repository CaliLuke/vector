//go:build integration

package projectstatereads

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStateMutationRollsBackDomainRevisionEffectAndReceipt(t *testing.T) {
	for _, table := range []string{"work_item_effects", "mutation_receipts"} {
		t.Run(table, func(t *testing.T) {
			f := newStateMutationFixture(t)
			state := f.create(t, "Atomic deletion")
			before := f.detail(t, state, true)["revision"].(string)
			_, err := f.db.Exec(t.Context(), `CREATE FUNCTION reject_state_admission() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'test admission failure'; END $$;
 CREATE TRIGGER reject_state_admission BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION reject_state_admission()`)
			require.NoError(t, err)
			cmd := f.command("delete", state)
			cmd.RequestID, cmd.ExpectedRevision = stateValue(uuid.New()), &before
			_, err = f.repository.MutateState(t.Context(), cmd)
			require.Error(t, err)
			require.Equal(t, before, f.detail(t, state, true)["revision"])
			var effects, receipts int
			require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&effects, &receipts))
			require.Zero(t, effects)
			require.Zero(t, receipts)
			_, err = f.db.Exec(t.Context(), `DROP TRIGGER reject_state_admission ON `+table)
			require.NoError(t, err)
			_, err = f.repository.MutateState(t.Context(), cmd)
			require.NoError(t, err)
			require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&effects, &receipts))
			require.Equal(t, 1, effects)
			require.Equal(t, 1, receipts)
		})
	}
}

func TestStateMutationRevalidatesCredentialAfterProjectLockWait(t *testing.T) {
	f := newStateMutationFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	controller, err := f.db.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = controller.Rollback(context.WithoutCancel(ctx)) }()
	_, err = controller.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, f.input.ProjectID)
	require.NoError(t, err)
	var revoked atomic.Bool
	credentialError := errors.New("credential revoked")
	cmd := f.command("create", "")
	cmd.RequestID = stateValue(uuid.New())
	cmd.Changes = StateChanges{Name: stateValue("Guarded"), Color: stateValue("#abc")}
	cmd.Revalidate = func(context.Context) error {
		if revoked.Load() {
			return credentialError
		}
		return nil
	}
	finished := make(chan error, 1)
	go func() { _, failure := f.repository.MutateState(ctx, cmd); finished <- failure }()
	f.waitForBlockedBy(t, ctx, controller.Conn().PgConn().PID())
	revoked.Store(true)
	require.NoError(t, controller.Rollback(ctx))
	require.ErrorIs(t, <-finished, credentialError)
	var count int
	require.NoError(t, f.db.QueryRow(ctx, `SELECT (SELECT count(*) FROM states)+(SELECT count(*) FROM mutation_receipts)`).Scan(&count))
	require.Zero(t, count)
	revoked.Store(false)
	_, err = f.repository.MutateState(ctx, cmd)
	require.NoError(t, err)
}

func TestStateConcurrentReceiptCommitsOneResource(t *testing.T) {
	f := newStateMutationFixture(t)
	cmd := f.command("create", "")
	cmd.RequestID = stateValue(uuid.New())
	cmd.Changes = StateChanges{Name: stateValue("Once"), Color: stateValue("#abc")}
	type outcome struct {
		value MutationResult
		err   error
	}
	finished := make(chan outcome, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			value, err := f.repository.MutateState(t.Context(), cmd)
			finished <- outcome{value, err}
		}()
	}
	close(start)
	first, second := <-finished, <-finished
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, first.value.StateID, second.value.StateID)
	require.NotEqual(t, *first.value.Replayed, *second.value.Replayed)
	var states, receipts int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states),(SELECT count(*) FROM mutation_receipts)`).Scan(&states, &receipts))
	require.Equal(t, 1, states)
	require.Equal(t, 1, receipts)
}

func TestStateConcurrentRevisionAllowsOnlyOneUpdate(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Before")
	revision := f.detail(t, state, true)["revision"].(string)
	finished := make(chan error, 2)
	start := make(chan struct{})
	for _, name := range []string{"First", "Second"} {
		cmd := f.command("update", state)
		cmd.ExpectedRevision = &revision
		cmd.Changes.Name = &name
		go func() { <-start; _, err := f.repository.MutateState(t.Context(), cmd); finished <- err }()
	}
	close(start)
	require.ElementsMatch(t, []error{nil, ErrRevisionConflict}, []error{<-finished, <-finished})
}
