package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestBootstrapRollbackPreservesOperationAndCleanupFailures(t *testing.T) {
	operationErr := errors.New("operation failed")
	cleanupErr := errors.New("connection lost during rollback")
	for _, test := range []struct {
		name        string
		operation   error
		rollback    error
		wantCleanup bool
	}{
		{"already committed", nil, pgx.ErrTxClosed, false},
		{"failed operation", operationErr, nil, false},
		{"already closed after failure", operationErr, fmt.Errorf("closed: %w", pgx.ErrTxClosed), false},
		{"both failures", operationErr, cleanupErr, true},
		{"cleanup failure", nil, cleanupErr, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller, cancel := context.WithCancel(t.Context())
			cancel()
			var cleanup context.Context
			calls := 0
			got := rollbackBootstrapTransaction(caller, func(ctx context.Context) error {
				calls++
				cleanup = ctx
				if ctx.Err() != nil {
					t.Fatalf("rollback inherited caller cancellation: %v", ctx.Err())
				}
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 5*time.Second {
					t.Fatal("rollback lacks a bounded cleanup deadline")
				}
				return test.rollback
			}, test.operation)
			if calls != 1 || !errors.Is(cleanup.Err(), context.Canceled) {
				t.Fatalf("rollback calls=%d cleanup error=%v", calls, cleanup.Err())
			}
			if test.wantCleanup {
				if !errors.Is(got, cleanupErr) || test.operation != nil && !errors.Is(got, test.operation) {
					t.Fatalf("rollback lost a failure: %v", got)
				}
			} else if !errors.Is(got, test.operation) {
				t.Fatalf("rollback replaced the operation result: got %v, want %v", got, test.operation)
			}
		})
	}
}
