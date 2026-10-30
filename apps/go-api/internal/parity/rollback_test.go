package parity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSnapshotRollbackSanitizesConnectionDetails(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	primary := errors.New("snapshot failed")
	secret := "postgresql://private:password@internal-host/database"
	for _, rollbackErr := range []error{errors.New(secret), &pgconn.PgError{Code: "08006", Message: secret}, pgx.ErrTxClosed, nil} {
		err := rollbackSnapshot(ctx, "source", func(cleanup context.Context) error {
			deadline, ok := cleanup.Deadline()
			if cleanup.Err() != nil || !ok || time.Until(deadline) > 5*time.Second {
				t.Error("rollback context was canceled or unbounded")
			}
			return rollbackErr
		}, primary)
		if !errors.Is(err, primary) {
			t.Fatalf("original snapshot failure lost: %v", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("connection details leaked: %v", err)
		}
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err.Error() != primary.Error() {
				t.Errorf("closed transaction changed failure: %v", err)
			}
		} else if !strings.Contains(err.Error(), "rollback snapshot") {
			t.Errorf("cleanup failure omitted: %v", err)
		}
	}
}
