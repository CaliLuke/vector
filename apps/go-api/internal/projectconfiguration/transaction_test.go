package projectconfiguration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type rollbackProbeDatabase struct {
	pgx.Tx
	cancel          context.CancelFunc
	queryErr        error
	rollbackErr     error
	cleanup         context.Context
	rollbacks       int
	commits         int
	cleanupCanceled bool
	cleanupBounded  bool
}

func (db *rollbackProbeDatabase) Begin(context.Context) (pgx.Tx, error) {
	db.cancel()
	return db, nil
}
func (db *rollbackProbeDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	return rollbackProbeRow{err: db.queryErr}
}
func (db *rollbackProbeDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, db.queryErr
}
func (db *rollbackProbeDatabase) Rollback(ctx context.Context) error {
	db.rollbacks++
	db.cleanup = ctx
	db.cleanupCanceled = ctx.Err() != nil
	deadline, ok := ctx.Deadline()
	db.cleanupBounded = ok && time.Until(deadline) > 0 && time.Until(deadline) <= 5*time.Second
	return db.rollbackErr
}
func (db *rollbackProbeDatabase) Commit(context.Context) error {
	db.commits++
	return nil
}

type rollbackProbeRow struct{ err error }

func (row rollbackProbeRow) Scan(...any) error { return row.err }

func newRollbackFailureRepository(t *testing.T, db *rollbackProbeDatabase) (context.Context, *Repository) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	db.cancel = cancel
	return ctx, &Repository{db: db}
}

func assertRollbackReleased(t *testing.T, db *rollbackProbeDatabase) {
	t.Helper()
	require.False(t, db.cleanupCanceled, "cleanup must not inherit request cancellation")
	require.True(t, db.cleanupBounded, "cleanup must have its own bounded lifetime")
	require.ErrorIs(t, db.cleanup.Err(), context.Canceled, "cleanup context must be released")
}

func TestTransactionFailuresRetainCleanupAndOperationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(context.Context, *Repository) error
	}{
		{"create estimate", func(ctx context.Context, repo *Repository) error {
			_, err := repo.CreateEstimate(ctx, "user", "workspace", "project", strings.NewReader(`{"estimate":{"name":"Estimate"}}`), "application/json")
			return err
		}},
		{"delete estimate", func(ctx context.Context, repo *Repository) error {
			return repo.DeleteEstimate(ctx, "user", "workspace", "project", "estimate")
		}},
		{"create estimate point", func(ctx context.Context, repo *Repository) error {
			_, err := repo.CreateEstimatePoint(ctx, "user", "workspace", "project", "estimate", strings.NewReader(`{"key":1,"value":"1"}`), "application/json")
			return err
		}},
		{"edit estimate point", func(ctx context.Context, repo *Repository) error {
			_, err := repo.PatchEstimatePoint(ctx, "user", "workspace", "project", "estimate", "00000000-0000-4000-8000-000000000001", strings.NewReader(`{"value":"1"}`), "application/json")
			return err
		}},
		{"delete estimate point", func(ctx context.Context, repo *Repository) error {
			_, err := repo.DeleteEstimatePoint(ctx, "user", "workspace", "project", "estimate", "point", strings.NewReader(`{}`), "application/json")
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := &rollbackProbeDatabase{queryErr: errors.New("query interrupted"), rollbackErr: errors.New("rollback interrupted")}
			ctx, repo := newRollbackFailureRepository(t, db)

			err := test.run(ctx, repo)

			require.Equal(t, 1, db.rollbacks, "the failing operation must reach transaction cleanup")
			require.Zero(t, db.commits)
			require.ErrorIs(t, err, db.queryErr)
			require.ErrorIs(t, err, db.rollbackErr)
			assertRollbackReleased(t, db)
		})
	}
}

func TestLabelValidationFailureRetainsRollbackError(t *testing.T) {
	db := &rollbackProbeDatabase{queryErr: errors.New("unexpected query"), rollbackErr: errors.New("rollback interrupted")}
	ctx, repo := newRollbackFailureRepository(t, db)

	_, err := repo.WriteAPILabel(ctx, "user", "workspace", "project", "", "POST", strings.NewReader(`[]`), "application/json")

	require.Equal(t, 1, db.rollbacks, "validation rejection must still close the opened transaction")
	require.Zero(t, db.commits)
	require.ErrorIs(t, err, db.rollbackErr)
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, 400, httpErr.Status)
	body, marshalErr := json.Marshal(httpErr.Body)
	require.NoError(t, marshalErr)
	require.JSONEq(t, `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}`, string(body))
	assertRollbackReleased(t, db)
}
