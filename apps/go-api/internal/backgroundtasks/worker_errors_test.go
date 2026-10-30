package backgroundtasks

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestWorkerDatabaseErrorsPreserveCauses(t *testing.T) {
	for _, worker := range []struct {
		name      string
		classify  func(error) error
		permanent error
	}{
		{"activity", classifyAPIActivityLogError, ErrPermanentAPIActivityLog},
		{"cleanup", classifyCleanupError, ErrPermanentCleanup},
		{"exporter", classifyExporterHistoryError, ErrPermanentExporterHistory},
	} {
		t.Run(worker.name, func(t *testing.T) {
			for _, code := range []string{"", "X", "23505"} {
				t.Run("code_"+code, func(t *testing.T) {
					cause := &pgconn.PgError{Code: code, Message: "database failure"}
					original := fmt.Errorf("operation: %w", cause)
					classified := worker.classify(original)
					require.ErrorIs(t, classified, worker.permanent)
					require.ErrorIs(t, classified, original)
					var postgresError *pgconn.PgError
					require.ErrorAs(t, classified, &postgresError)
					require.Same(t, cause, postgresError)
				})
			}
			for _, code := range []string{"08006", "40001", "53300", "57P01"} {
				original := fmt.Errorf("operation: %w", &pgconn.PgError{Code: code})
				require.Same(t, original, worker.classify(original))
			}
			unrelated := errors.New("network unavailable")
			require.Same(t, unrelated, worker.classify(unrelated))
			require.NoError(t, worker.classify(nil))
		})
	}
}

func TestExistingPermanentWorkerErrorsRetainIdentity(t *testing.T) {
	cleanup := fmt.Errorf("cleanup: %w", ErrPermanentCleanup)
	require.Same(t, cleanup, classifyCleanupError(cleanup))
	exporter := fmt.Errorf("exporter: %w", ErrPermanentExporterHistory)
	require.Same(t, exporter, classifyExporterHistoryError(exporter))
}

func TestExporterDatabaseRetryStillClassifiesStorageCause(t *testing.T) {
	denied := &smithy.GenericAPIError{Code: "AccessDenied", Fault: smithy.FaultClient}
	original := errors.Join(&pgconn.PgError{Code: "40001"}, denied)
	classified := classifyExporterHistoryError(original)
	require.ErrorIs(t, classified, ErrPermanentExporterHistory)
	require.ErrorIs(t, classified, original)
	require.ErrorIs(t, classified, denied)
}
