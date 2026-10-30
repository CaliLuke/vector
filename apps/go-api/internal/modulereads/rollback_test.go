package modulereads

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type failedModuleCleanup struct {
	moduleMemberInsertTx
	cancel                context.CancelFunc
	insertErr, cleanupErr error
	canceled              bool
	cleanupLifetime       time.Duration
}

func (tx *failedModuleCleanup) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.cancel()
	return pgconn.CommandTag{}, tx.insertErr
}
func (tx *failedModuleCleanup) Rollback(ctx context.Context) error {
	tx.canceled = ctx.Err() != nil
	if deadline, ok := ctx.Deadline(); ok {
		tx.cleanupLifetime = time.Until(deadline)
	}
	return errors.Join(tx.moduleMemberInsertTx.Rollback(ctx), tx.cleanupErr)
}
func TestModuleReplacementKeepsInsertAndRollbackFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	tx := &failedModuleCleanup{cancel: cancel, insertErr: errors.New("insert failed"), cleanupErr: errors.New("rollback failed")}
	err := insertModuleMembersAtomically(ctx, func(context.Context) (moduleMemberTransaction, error) { return tx, nil }, "workspace", "project", "module", []string{"member"}, nil, nil, time.Now())
	if !errors.Is(err, tx.insertErr) || !errors.Is(err, tx.cleanupErr) {
		t.Fatalf("insert or rollback failure lost: %v", err)
	}
	if tx.canceled || tx.cleanupLifetime <= 0 || tx.cleanupLifetime > 5*time.Second {
		t.Fatalf("cleanup canceled=%t lifetime=%v", tx.canceled, tx.cleanupLifetime)
	}
	if tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}
