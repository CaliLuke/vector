package projectstatereads

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type stateRollbackDatabase struct {
	pgx.Tx
	cancel           context.CancelFunc
	failure          error
	cleanupFailure   error
	cleanup          context.Context
	cleanupCancelled bool
	cleanupBounded   bool
	rollbacks        int
	commits          int
}

func (db *stateRollbackDatabase) Begin(context.Context) (pgx.Tx, error) {
	db.cancel()
	return db, nil
}
func (db *stateRollbackDatabase) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return db.Begin(ctx)
}
func (db *stateRollbackDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	return stateRollbackRow{db.failure}
}
func (db *stateRollbackDatabase) Commit(context.Context) error {
	db.commits++
	return nil
}
func (db *stateRollbackDatabase) Rollback(ctx context.Context) error {
	db.rollbacks++
	db.cleanup = ctx
	db.cleanupCancelled = ctx.Err() != nil
	deadline, present := ctx.Deadline()
	remaining := time.Until(deadline)
	db.cleanupBounded = present && remaining > 0 && remaining <= 5*time.Second
	return db.cleanupFailure
}

type stateRollbackRow struct{ failure error }

func (row stateRollbackRow) Scan(...any) error { return row.failure }

func TestStateWritesRetainRollbackFailureAfterCancellation(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(context.Context, *WriteRepository, StateWriteInput) error
	}{
		{"create", func(ctx context.Context, repository *WriteRepository, input StateWriteInput) error {
			_, err := repository.Create(ctx, input)
			return err
		}},
		{"update", func(ctx context.Context, repository *WriteRepository, input StateWriteInput) error {
			_, err := repository.Update(ctx, input)
			return err
		}},
		{"delete", func(ctx context.Context, repository *WriteRepository, input StateWriteInput) error {
			return repository.Delete(ctx, input)
		}},
		{"mark default", func(ctx context.Context, repository *WriteRepository, input StateWriteInput) error {
			return repository.MarkDefault(ctx, input)
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			db := &stateRollbackDatabase{cancel: cancel, failure: errors.New("project lock failed"), cleanupFailure: errors.New("rollback failed")}
			repository := &WriteRepository{database: db}
			input := StateWriteInput{Fields: map[string]any{"name": "Started", "color": "#abcdef"}}

			err := operation.run(ctx, repository, input)

			require.Equal(t, 1, db.rollbacks, "operation must reach transaction cleanup")
			require.Zero(t, db.commits)
			require.ErrorIs(t, err, db.failure)
			require.ErrorIs(t, err, db.cleanupFailure)
			require.False(t, db.cleanupCancelled, "cleanup must survive request cancellation")
			require.True(t, db.cleanupBounded, "cleanup must retain its own bounded lifetime")
			require.ErrorIs(t, db.cleanup.Err(), context.Canceled, "cleanup context must be released")
		})
	}
}
