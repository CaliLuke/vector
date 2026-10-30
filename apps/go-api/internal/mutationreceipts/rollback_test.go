package mutationreceipts_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	receipts "github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type receiptCleanupTransaction struct {
	pgx.Tx
	commitErr, rollbackErr error
	cancel                 context.CancelFunc
	committed, rolledBack  bool
	cleanupErr             error
	cleanupBounded         bool
}

func (tx *receiptCleanupTransaction) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return tx, nil
}
func (tx *receiptCleanupTransaction) Commit(context.Context) error {
	tx.committed = true
	tx.cancel()
	return tx.commitErr
}
func (tx *receiptCleanupTransaction) Rollback(ctx context.Context) error {
	tx.rolledBack = true
	tx.cleanupErr = ctx.Err()
	deadline, ok := ctx.Deadline()
	remaining := time.Until(deadline)
	tx.cleanupBounded = ok && remaining > 0 && remaining <= 5*time.Second
	return tx.rollbackErr
}
func TestReceiptCleanupPreservesCommitCertainty(t *testing.T) {
	operationErr := errors.New("authorization failed")
	commitErr := errors.New("database commit connection secret")
	rollbackErr := errors.New("database rollback connection secret")
	for _, test := range []struct {
		name                     string
		authorizeErr, commitErr  error
		wantCommit, wantRollback bool
	}{
		{name: "authorization failure", authorizeErr: operationErr, wantRollback: true},
		{name: "lost commit acknowledgement", commitErr: commitErr, wantCommit: true, wantRollback: true},
		{name: "acknowledged commit despite late cancellation", wantCommit: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			tx := &receiptCleanupTransaction{commitErr: test.commitErr, rollbackErr: rollbackErr, cancel: cancel}
			result, err := receipts.Execute(ctx, tx, receipts.Command{ActorID: uuid.New(), WorkspaceID: uuid.New(), Tool: "test", Action: "create", NormalizedInput: json.RawMessage(`{}`)}, receipts.Operation{
				Authorize: func(context.Context, pgx.Tx) error {
					if test.authorizeErr != nil {
						cancel()
					}
					return test.authorizeErr
				},
				Apply: func(context.Context, pgx.Tx) (json.RawMessage, error) {
					return json.RawMessage(`{"created":true}`), nil
				},
			})
			if tx.committed != test.wantCommit || tx.rolledBack != test.wantRollback {
				t.Errorf("commit=%t rollback=%t", tx.committed, tx.rolledBack)
			}
			if test.wantRollback && (!tx.cleanupBounded || tx.cleanupErr != nil) {
				t.Errorf("cleanup bounded=%t context error=%v", tx.cleanupBounded, tx.cleanupErr)
			}
			switch {
			case test.authorizeErr != nil:
				if !errors.Is(err, operationErr) || !errors.Is(err, rollbackErr) || errors.Is(err, receipts.ErrIndeterminate) {
					t.Fatalf("authorization and cleanup causes: %v", err)
				}
			case test.commitErr != nil:
				if !errors.Is(err, receipts.ErrIndeterminate) || !errors.Is(err, commitErr) || !errors.Is(err, rollbackErr) {
					t.Fatalf("indeterminate and cleanup causes: %v", err)
				}
				if err.Error() != receipts.ErrIndeterminate.Error() {
					t.Fatalf("public commit error disclosed database detail: %v", err)
				}
			default:
				if err != nil || string(result.Value) != `{"created":true}` {
					t.Fatalf("acknowledged result=%s error=%v", result.Value, err)
				}
			}
		})
	}
}

func TestReceiptCleanupDoesNotMutateReusedOperationError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	command := receipts.Command{ActorID: uuid.New(), WorkspaceID: uuid.New(), Tool: "test", Action: "create", NormalizedInput: json.RawMessage(`{}`)}
	committed := &receiptCleanupTransaction{cancel: cancel, commitErr: errors.New("commit acknowledgement lost"), rollbackErr: pgx.ErrTxClosed}
	_, original := receipts.Execute(ctx, committed, command, receipts.Operation{
		Authorize: func(context.Context, pgx.Tx) error { return nil },
		Apply:     func(context.Context, pgx.Tx) (json.RawMessage, error) { return json.RawMessage(`{}`), nil },
	})
	if !errors.Is(original, receipts.ErrIndeterminate) {
		t.Fatalf("original error = %v", original)
	}
	cleanupFailure := errors.New("unrelated rollback failure")
	outer := &receiptCleanupTransaction{cancel: func() {}, rollbackErr: cleanupFailure}
	_, err := receipts.Execute(t.Context(), outer, command, receipts.Operation{
		Authorize: func(context.Context, pgx.Tx) error { return original },
		Apply:     func(context.Context, pgx.Tx) (json.RawMessage, error) { return json.RawMessage(`{}`), nil },
	})
	if errors.Is(original, cleanupFailure) {
		t.Error("cleanup modified a caller-owned error")
	}
	if !errors.Is(err, original) || !errors.Is(err, cleanupFailure) || err.Error() != receipts.ErrIndeterminate.Error() {
		t.Fatalf("returned causes or sanitized message changed: %v", err)
	}
}
