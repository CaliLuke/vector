package assets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type canceledAssetDatabase struct {
	pgx.Tx
	cancel                           context.CancelFunc
	operationFailure, cleanupFailure error
	cleanup                          context.Context
	cleanupWasCanceled               bool
	cleanupDeadline                  time.Time
	cleanupBounded                   bool
}

func (db *canceledAssetDatabase) Begin(context.Context) (pgx.Tx, error) { db.cancel(); return db, nil }
func (db *canceledAssetDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, db.operationFailure
}
func (db *canceledAssetDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	return assetFailedRow{err: db.operationFailure}
}
func (db *canceledAssetDatabase) Rollback(ctx context.Context) error {
	db.cleanup = ctx
	db.cleanupWasCanceled = ctx.Err() != nil
	db.cleanupDeadline, db.cleanupBounded = ctx.Deadline()
	return db.cleanupFailure
}

type assetFailedRow struct{ err error }

func (row assetFailedRow) Scan(...any) error { return row.err }

func TestAssetTransactionsRetainFailureWhenRequestIsCanceled(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	for _, test := range []struct {
		name   string
		invoke func(context.Context, *PostgresRepository) error
	}{
		{"attachment read", func(ctx context.Context, r *PostgresRepository) error {
			_, err := (&AttachmentApplication{repository: r}).Read(ctx, AttachmentQuery{})
			return err
		}},
		{"upload completion", func(ctx context.Context, r *PostgresRepository) error {
			return (&AttachmentApplication{repository: r}).CompleteAttachmentUpload(ctx, id.String())
		}},
		{"conditional draft upload", func(ctx context.Context, r *PostgresRepository) error {
			return r.CreateWorkspaceEntityUpload(ctx, id, id, id, uuid.Nil, "object", []byte(`{}`), 1, "DRAFT_ISSUE_DESCRIPTION", id.String(), false)
		}},
		{"public upload", func(ctx context.Context, r *PostgresRepository) error {
			return r.CreatePublicUpload(ctx, id, id, id, id, "object", []byte(`{}`), 1, "COMMENT_DESCRIPTION", id.String())
		}},
		{"draft mutation", func(ctx context.Context, r *PostgresRepository) error {
			return r.MutateDraftAsset(ctx, id, id, id, "delete", nil, false)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			db := &canceledAssetDatabase{cancel: cancel, operationFailure: errors.New("database operation failed"), cleanupFailure: errors.New("database rollback failed")}
			err := test.invoke(ctx, &PostgresRepository{database: db})
			if !errors.Is(err, db.operationFailure) {
				t.Errorf("operation error was lost: %v", err)
			}
			if !errors.Is(err, db.cleanupFailure) {
				t.Errorf("cleanup error was lost: %v", err)
			}
			if db.cleanup == nil {
				t.Fatal("transaction was not rolled back")
			}
			if db.cleanupWasCanceled {
				t.Error("caller cancellation prevented cleanup")
			}
			if !db.cleanupBounded || time.Until(db.cleanupDeadline) > 5*time.Second {
				t.Error("cleanup did not have a short deadline")
			}
			if !errors.Is(db.cleanup.Err(), context.Canceled) {
				t.Error("cleanup context was not released")
			}
		})
	}
}
