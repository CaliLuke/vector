package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5"
)

func TestRollbackPreservesOperationAndCleanupFailures(t *testing.T) {
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
			type contextKey struct{}
			caller, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "retained"))
			cancel()
			var cleanup context.Context
			calls := 0
			got := RollbackTransaction(caller, func(ctx context.Context) error {
				calls++
				cleanup = ctx
				if ctx.Value(contextKey{}) != "retained" {
					t.Fatal("cleanup lost caller context values")
				}
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

func TestRollbackReportsCleanupFailureWithoutPrivateDetails(t *testing.T) {
	for _, test := range []struct {
		name     string
		rollback error
		wantLog  bool
	}{
		{"rolled back", nil, false},
		{"closed", fmt.Errorf("private detail: %w", pgx.ErrTxClosed), false},
		{"deadline", fmt.Errorf("private detail: %w", context.DeadlineExceeded), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }))
			ctx = log.With(ctx, log.KV{K: "request_id", V: "cleanup-request"})
			result := RollbackTransaction(ctx, func(context.Context) error { return test.rollback }, errors.New("private operation"))
			if result == nil {
				t.Fatal("operation error was lost")
			}
			if !test.wantLog {
				if output.Len() != 0 {
					t.Fatalf("ordinary cleanup logged an error: %s", &output)
				}
				return
			}
			var entry map[string]any
			if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
				t.Fatalf("decode cleanup log: %v; log=%s", err, &output)
			}
			if entry["request_id"] != "cleanup-request" {
				t.Fatalf("cleanup event lost request correlation: %#v", entry)
			}
			errorType, hasErrorType := entry["error_type"].(string)
			if entry["deadline_exceeded"] != true || entry["context_canceled"] != false || !hasErrorType || errorType == "" {
				t.Fatalf("cleanup event lost safe failure classification: %#v", entry)
			}
			if strings.Contains(output.String(), "private") {
				t.Fatalf("cleanup log exposed private details: %s", &output)
			}
		})
	}
}
