package accountauthentication

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type accountCleanupTransaction struct {
	pgx.Tx
	cancel   context.CancelFunc
	failure  error
	rollback func(context.Context) error
}

func (tx *accountCleanupTransaction) Begin(context.Context) (pgx.Tx, error) {
	tx.cancel()
	return tx, nil
}
func (tx *accountCleanupTransaction) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, tx.failure
}
func (tx *accountCleanupTransaction) QueryRow(context.Context, string, ...any) pgx.Row {
	return accountCleanupRow{tx.failure}
}
func (tx *accountCleanupTransaction) Rollback(ctx context.Context) error { return tx.rollback(ctx) }

type accountCleanupRow struct{ failure error }

func (row accountCleanupRow) Scan(...any) error { return row.failure }

func TestAccountWritesPreserveFailureAndBoundRollbackAfterCancellation(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(context.Context, *Repository) error
	}{
		{"create user", func(ctx context.Context, r *Repository) error {
			_, err := r.CreateUser(ctx, "test@example.test", "test", "hash", false, false)
			return err
		}},
		{"ensure profile", func(ctx context.Context, r *Repository) error {
			return r.EnsureProfile(ctx, "00000000-0000-4000-8000-000000000001")
		}},
		{"initial admin", func(ctx context.Context, r *Repository) error {
			_, err := r.CreateInitialAdmin(ctx, "test@example.test", "Test", "User", "Company", "hash", "127.0.0.1", "test", false, time.Now())
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			operationErr, cleanupErr := errors.New("operation failed"), errors.New("rollback failed")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			var cleanup context.Context
			tx := &accountCleanupTransaction{cancel: cancel, failure: operationErr, rollback: func(ctx context.Context) error {
				calls++
				cleanup = ctx
				require.NoError(t, ctx.Err(), "cleanup inherits cancellation")
				deadline, ok := ctx.Deadline()
				require.True(t, ok, "cleanup needs a deadline")
				require.Positive(t, time.Until(deadline))
				require.LessOrEqual(t, time.Until(deadline), 5*time.Second)
				return cleanupErr
			}}
			repository, err := NewRepository(tx, nil, false)
			require.NoError(t, err)
			err = operation.run(ctx, repository)
			require.Equal(t, 1, calls)
			require.ErrorIs(t, err, operationErr)
			require.ErrorIs(t, err, cleanupErr)
			require.ErrorIs(t, cleanup.Err(), context.Canceled, "cleanup context must be released")
		})
	}
}
