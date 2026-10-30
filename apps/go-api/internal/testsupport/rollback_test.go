package testsupport_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type testTransactionCleanup struct {
	calls                   int
	result, errorAtRollback error
}

func (tx *testTransactionCleanup) Rollback(ctx context.Context) error {
	tx.calls++
	tx.errorAtRollback = ctx.Err()
	return tx.result
}
func TestRollbackReportsUnexpectedCleanupFailures(t *testing.T) {
	for _, test := range []struct {
		name        string
		failure     error
		wantReports int
	}{
		{name: "open transaction"},
		{name: "already committed", failure: pgx.ErrTxClosed},
		{name: "rollback failed", failure: errors.New("injected rollback failure"), wantReports: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			reporter := &cleanupReporter{}
			tx := &testTransactionCleanup{result: test.failure}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			testsupport.Rollback(reporter, tx, ctx)
			require.Equal(t, 1, tx.calls)
			require.NoError(t, tx.errorAtRollback)
			require.Len(t, reporter.messages, test.wantReports)
		})
	}
}
