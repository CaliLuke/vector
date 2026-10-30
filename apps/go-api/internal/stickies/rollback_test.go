package stickies

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type stickyCleanupTransaction struct {
	pgx.Tx
	failure, cleanupFailure error
	cleanup                 context.Context
	canceled                bool
}

func (tx *stickyCleanupTransaction) QueryRow(context.Context, string, ...any) pgx.Row {
	return stickyCleanupRow{tx.failure}
}
func (tx *stickyCleanupTransaction) Rollback(ctx context.Context) error {
	tx.cleanup = ctx
	tx.canceled = ctx.Err() != nil
	return tx.cleanupFailure
}

type stickyCleanupRow struct{ err error }

func (row stickyCleanupRow) Scan(...any) error { return row.err }
func TestSortAllocationRetainsClassifiedFailureAfterCanceledCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	tx := &stickyCleanupTransaction{failure: errors.New("workspace lock failed"), cleanupFailure: errors.New("rollback failed")}
	repo, err := NewSortRepository(transactionStarterFunc(func(context.Context) (pgx.Tx, error) { cancel(); return tx, nil }))
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290401")
	_, err = repo.CreateWithAllocatedSort(ctx, StickyID(id), Scope{Operation: OperationCreate, WorkspaceID: WorkspaceID(id), UserID: UserID(id), Ownership: AssignOwner, AssignCreator: true})
	if !errors.Is(err, ErrStickyCreateFailed) || !errors.Is(err, tx.failure) || !errors.Is(err, tx.cleanupFailure) {
		t.Fatalf("classified creation or cleanup error lost: %v", err)
	}
	if tx.cleanup == nil || tx.canceled {
		t.Fatal("cleanup did not survive cancellation")
	}
	deadline, ok := tx.cleanup.Deadline()
	if !ok || time.Until(deadline) > 5*time.Second {
		t.Fatal("cleanup was unbounded")
	}
}
