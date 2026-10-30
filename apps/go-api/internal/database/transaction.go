package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5"
)

// RollbackTransaction gives cleanup its own bounded lifetime after a failed
// operation or canceled request. A successful commit already closes the
// transaction. Other rollback failures are retained alongside the operation
// error and logged without database details, which public error mappings omit.
func RollbackTransaction(ctx context.Context, rollback func(context.Context) error, operationErr error) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if rollbackErr := rollback(cleanup); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		log.Error(cleanup, nil,
			log.KV{K: log.MessageKey, V: "Transaction rollback failed"},
			log.KV{K: "error_type", V: fmt.Sprintf("%T", rollbackErr)},
			log.KV{K: "context_canceled", V: errors.Is(rollbackErr, context.Canceled)},
			log.KV{K: "deadline_exceeded", V: errors.Is(rollbackErr, context.DeadlineExceeded)},
		)
		return errors.Join(operationErr, fmt.Errorf("rollback transaction: %w", rollbackErr))
	}
	return operationErr
}
