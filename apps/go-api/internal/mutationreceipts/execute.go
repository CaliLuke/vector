// Package mutationreceipts coordinates optional retry receipts with an owned
// domain transaction. It does not supply domain authorization or perform retries.
package mutationreceipts

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRequest = errors.New("invalid mutation receipt request")
	ErrConflict       = errors.New("request_id was already used with different input")
	ErrIndeterminate  = errors.New("mutation commit outcome is unknown; reconcile before retrying")
)

// Database must begin a transaction on the primary database.
type Database interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Command identifies an authorized operation. ProjectID is nil for workspace
// operations. RequestID is optional; a nil value creates no receipt.
// NormalizedInput is an object containing the typed domain inputs after defaults
// and normalization, including expected_revision if supplied, but never credentials
// or request_id. Object order and insignificant JSON whitespace do not affect its
// hash. The domain owns semantic normalization (including numeric representations).
type Command struct {
	ActorID         uuid.UUID
	WorkspaceID     uuid.UUID
	ProjectID       *uuid.UUID
	Tool            string
	Action          string
	RequestID       *uuid.UUID
	NormalizedInput json.RawMessage
}

// Operation runs inside the same transaction as the receipt. Authorize runs on
// every invocation, after waiting for a concurrent receipt writer and before any
// receipt is inspected. It must check current actor/workspace/project/action rights
// and lock the domain's authorization rows through commit. Destructive replay must
// not require a live target. Domains must retain any ownership scope they need
// throughout the receipt lifetime. The caller must also revalidate the
// credential in Authorize, after the receipt lock wait, on every invocation.
//
// Apply checks the optional revision, performs the domain mutation, and records
// activity/outbox rows in tx. It returns only a safe, replayable result. It must not
// commit, roll back, use another writer, or dispatch an external effect directly.
// Replay skips Apply, so it precedes revision checks. Neither callback is retried.
type Operation struct {
	Authorize func(context.Context, pgx.Tx) error
	Apply     func(context.Context, pgx.Tx) (json.RawMessage, error)
}

type Result struct {
	Value           json.RawMessage
	ReplayExpiresAt *time.Time
	Replayed        bool
}

// Execute returns a result only after an acknowledged commit. A commit error is
// indeterminate even when cancellation or a connection failure caused it. The
// original cause remains available to internal error classification via Unwrap.
// Successful commit is never replaced by a late context cancellation error.
func Execute(ctx context.Context, db Database, cmd Command, op Operation) (_ Result, resultErr error) {
	hash, err := prepare(cmd, op)
	if err != nil || db == nil {
		return Result{}, ErrInvalidRequest
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Result{}, err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		// Preserve a sanitized indeterminate message without changing errors
		// owned by an operation callback or another receipt execution.
		operationErr := resultErr
		resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, operationErr)
		if errors.Is(operationErr, ErrIndeterminate) {
			resultErr = &commitError{cause: resultErr}
		}
	}()
	result, err := execute(ctx, tx, cmd, hash, op)
	if err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, &commitError{cause: err}
	}
	committed = true
	return result, nil
}

func execute(ctx context.Context, tx pgx.Tx, cmd Command, hash [32]byte, op Operation) (Result, error) {
	if cmd.RequestID != nil {
		if err := lock(ctx, tx, cmd); err != nil {
			return Result{}, err
		}
	}
	if err := op.Authorize(ctx, tx); err != nil {
		return Result{}, err
	}
	if cmd.RequestID != nil {
		previous, err := replay(ctx, tx, cmd, hash)
		if err != nil || previous.Replayed {
			return previous, err
		}
	}
	value, err := op.Apply(ctx, tx)
	if err != nil {
		return Result{}, err
	}
	if !json.Valid(value) {
		return Result{}, ErrInvalidRequest
	}
	if cmd.RequestID != nil {
		return record(ctx, tx, cmd, hash, value)
	}
	return Result{Value: value}, nil
}

type commitError struct{ cause error }

func (*commitError) Error() string     { return ErrIndeterminate.Error() }
func (e *commitError) Unwrap() error   { return e.cause }
func (*commitError) Is(err error) bool { return err == ErrIndeterminate }
