//go:build integration

package mutationreceipts_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	receipts "github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type outcome struct {
	result receipts.Result
	err    error
}

func invoke(ctx context.Context, f fixture, op receipts.Operation) <-chan outcome {
	done := make(chan outcome, 1)
	go func() { result, err := receipts.Execute(ctx, f.db, f.command, op); done <- outcome{result, err} }()
	return done
}

func receive(t *testing.T, done <-chan outcome) outcome {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("mutation did not finish")
		return outcome{}
	}
}

func signal(t *testing.T, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("transaction did not reach controlled boundary")
	}
}

func waitForReceiptLock(t *testing.T, f fixture) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := f.db.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory')`).Scan(&waiting)
		return err == nil && waiting
	}, 10*time.Second, 10*time.Millisecond, "duplicate never waited for the receipt transaction")
}

func holdMutation(entered, release chan struct{}, fail bool) receipts.Operation {
	op := operation()
	apply := op.Apply
	op.Apply = func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		value, err := apply(ctx, tx)
		if err != nil {
			return nil, err
		}
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if fail {
			return nil, errDenied
		}
		return value, nil
	}
	return op
}

func TestConcurrentSameKeyCommitsOnceAndWaiterRecoversRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[fail], func(t *testing.T) {
			f := receiptDatabase(t)
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			entered, release := make(chan struct{}), make(chan struct{})
			first := invoke(ctx, f, holdMutation(entered, release, fail))
			signal(t, entered)
			second := invoke(ctx, f, operation())
			waitForReceiptLock(t, f)
			close(release)
			a, b := receive(t, first), receive(t, second)
			require.NoError(t, b.err)
			if fail {
				require.ErrorIs(t, a.err, errDenied)
				require.False(t, b.result.Replayed)
			} else {
				require.NoError(t, a.err)
				require.True(t, b.result.Replayed)
				require.Equal(t, a.result.Value, b.result.Value)
			}
			assertCounts(t, f, 1, 1, 1)
		})
	}
}

func TestCancelledDuplicateCannotExecuteOrPoisonReceipt(t *testing.T) {
	f := receiptDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	first := invoke(ctx, f, holdMutation(entered, release, false))
	signal(t, entered)
	waiting, cancelWaiting := context.WithCancel(ctx)
	second := invoke(waiting, f, operation())
	waitForReceiptLock(t, f)
	cancelWaiting()
	b := receive(t, second)
	require.Error(t, b.err)
	require.Empty(t, b.result)
	close(release)
	a := receive(t, first)
	require.NoError(t, a.err)
	replay, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, a.result.Value, replay.Value)
	assertCounts(t, f, 1, 1, 1)
}

func TestWaitingReplayUsesAuthorizationCommittedAfterItsTransactionBegan(t *testing.T) {
	f := receiptDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	first := invoke(ctx, f, holdMutation(entered, release, false))
	signal(t, entered)
	authEntered, authRelease := make(chan struct{}), make(chan struct{})
	op := operation()
	op.Authorize = func(ctx context.Context, tx pgx.Tx) error {
		close(authEntered)
		select {
		case <-authRelease:
		case <-ctx.Done():
			return ctx.Err()
		}
		return authorize(ctx, tx)
	}
	second := invoke(ctx, f, op)
	waitForReceiptLock(t, f)
	close(release)
	require.NoError(t, receive(t, first).err)
	signal(t, authEntered)
	_, err := f.db.Exec(ctx, `UPDATE receipt_test_permission SET allowed=false`)
	require.NoError(t, err)
	close(authRelease)
	denied := receive(t, second)
	require.ErrorIs(t, denied.err, errDenied)
	require.Empty(t, denied.result)
	assertCounts(t, f, 1, 1, 1)
}

func TestReceiptStorageFailureRollsBackAlreadyWrittenDomainEffects(t *testing.T) {
	f := receiptDatabase(t)
	_, err := f.db.Exec(t.Context(), `CREATE FUNCTION fail_receipt_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled receipt failure'; END $$;
CREATE TRIGGER fail_receipt_insert BEFORE INSERT ON mutation_receipts FOR EACH ROW EXECUTE FUNCTION fail_receipt_insert()`)
	require.NoError(t, err)
	result, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.Error(t, err)
	require.Empty(t, result)
	assertCounts(t, f, 0, 0, 0)
}
