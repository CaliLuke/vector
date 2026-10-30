package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMigrationCleanupPreservesEveryFailure(t *testing.T) {
	primary, source, database := errors.New("migration failed"), errors.New("source close failed"), errors.New("database close failed")
	for _, tc := range []struct {
		name                        string
		operation, source, database error
	}{
		{"all failures", primary, source, database},
		{"operation only", primary, nil, nil},
		{"source only", nil, source, nil},
		{"database only", nil, nil, database},
		{"both cleanup failures", nil, source, database},
		{"success", nil, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			closes := 0
			err := closeMigration("application", func() (error, error) { closes++; return tc.source, tc.database }, tc.operation)
			require.Equal(t, 1, closes)
			for _, cause := range []error{tc.operation, tc.source, tc.database} {
				if cause != nil {
					require.ErrorIs(t, err, cause)
				}
			}
			if tc.source == nil && tc.database == nil {
				require.Equal(t, tc.operation, err)
			}
		})
	}
}

func TestMigrationSourceCleanupRetainsConfigurationAndCloseCauses(t *testing.T) {
	primary, cleanup := errors.New("configure migration failed"), errors.New("source close failed")
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "close failed"}[failure], func(t *testing.T) {
			closes := 0
			err := closeMigrationSource("support", func() error {
				closes++
				if failure {
					return cleanup
				}
				return nil
			}, primary)
			require.Equal(t, 1, closes)
			require.ErrorIs(t, err, primary)
			if failure {
				require.ErrorIs(t, err, cleanup)
			} else {
				require.Same(t, primary, err)
			}
		})
	}
}

func TestSchemaConnectionCleanupSurvivesCancellationAndPreservesCauses(t *testing.T) {
	type contextKey struct{}
	primary, cleanup := errors.New("schema adoption failed"), errors.New("connection close failed")
	for _, tc := range []struct {
		name               string
		operation, cleanup error
	}{
		{"both failures", primary, cleanup},
		{"cleanup failure after success", nil, cleanup},
		{"operation failure with clean close", primary, nil},
		{"success", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "migration context"))
			cancel()
			closes := 0
			var cleanupContextErr error
			var remaining time.Duration
			var bounded bool
			var value any
			err := closeSchemaConnection(ctx, func(ctx context.Context) error {
				closes++
				cleanupContextErr = ctx.Err()
				deadline, ok := ctx.Deadline()
				bounded, remaining = ok, time.Until(deadline)
				value = ctx.Value(contextKey{})
				return tc.cleanup
			}, tc.operation)
			require.Equal(t, 1, closes)
			require.NoError(t, cleanupContextErr)
			require.True(t, bounded)
			require.Positive(t, remaining)
			require.LessOrEqual(t, remaining, 10*time.Second)
			require.Equal(t, "migration context", value)
			if tc.operation != nil {
				require.ErrorIs(t, err, tc.operation)
			}
			if tc.cleanup != nil {
				require.ErrorIs(t, err, tc.cleanup)
			} else {
				require.Equal(t, tc.operation, err)
			}
		})
	}
}
