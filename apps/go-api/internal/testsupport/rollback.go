package testsupport

import (
	"context"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
)

// Rollback releases a test-owned transaction with an independent deadline.
// Already committed or rolled-back transactions need no further cleanup. Other
// failures fail the test without interrupting the remaining teardown.
func Rollback(t errorReporter, transaction interface{ Rollback(context.Context) error }, ctx context.Context) {
	t.Helper()
	if err := dbutil.RollbackTransaction(ctx, transaction.Rollback, nil); err != nil {
		t.Errorf("rollback test transaction: %v", err)
	}
}
