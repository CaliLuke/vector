package workitemtypes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type schemaFailureTransaction struct {
	pgx.Tx
	cancel                    context.CancelFunc
	operationErr, rollbackErr error
	rollbackContext           context.Context
	canceled                  bool
	remaining                 time.Duration
}

func (tx *schemaFailureTransaction) Begin(context.Context) (pgx.Tx, error) {
	tx.cancel()
	return tx, nil
}
func (tx *schemaFailureTransaction) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, tx.operationErr
}
func (tx *schemaFailureTransaction) QueryRow(context.Context, string, ...any) pgx.Row {
	return schemaFailureRow{tx.operationErr}
}
func (tx *schemaFailureTransaction) Rollback(ctx context.Context) error {
	tx.rollbackContext = ctx
	tx.canceled = ctx.Err() != nil
	if deadline, ok := ctx.Deadline(); ok {
		tx.remaining = time.Until(deadline)
	}
	return tx.rollbackErr
}

type schemaFailureRow struct{ err error }

func (row schemaFailureRow) Scan(...any) error { return row.err }

func TestSchemaOperationsKeepRollbackErrorsAfterCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(context.Context, *Repository) error
	}{
		{"workspace catalog", func(ctx context.Context, r *Repository) error { _, err := r.Catalog(ctx, "actor", "space"); return err }},
		{"project catalog", func(ctx context.Context, r *Repository) error {
			_, err := r.ProjectCatalog(ctx, "actor", "space", "project")
			return err
		}},
		{"workspace review", func(ctx context.Context, r *Repository) error {
			_, err := r.PreviewTypeRemoval(ctx, "actor", "space", TypeRemovalInput{TypeID: "00000000-0000-4000-8000-000000000001", ReplacementID: "00000000-0000-4000-8000-000000000002"})
			return err
		}},
		{"project mutation", func(ctx context.Context, r *Repository) error {
			_, err := r.MutateProject(ctx, "actor", "space", "project", ProjectInput{})
			return err
		}},
		{"discovery", func(ctx context.Context, r *Repository) error {
			_, err := WithTypeDiscoveryRead[int](ctx, r.db, "actor", "space", "project", func(pgx.Tx, TypeIdentityCatalog) (int, error) {
				return 0, errors.New("unexpected read callback after database failure")
			})
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			tx := &schemaFailureTransaction{cancel: cancel, operationErr: errors.New("schema operation failed"), rollbackErr: errors.New("schema cleanup failed")}
			err := test.run(ctx, NewRepository(tx))
			if !errors.Is(err, tx.operationErr) || !errors.Is(err, tx.rollbackErr) {
				t.Fatalf("operation or cleanup error lost: %v", err)
			}
			if tx.rollbackContext == nil || tx.canceled || tx.remaining <= 0 || tx.remaining > 5*time.Second {
				t.Fatalf("cleanup called=%t canceled=%t lifetime=%v", tx.rollbackContext != nil, tx.canceled, tx.remaining)
			}
			if !errors.Is(tx.rollbackContext.Err(), context.Canceled) {
				t.Fatal("cleanup context was not released")
			}
		})
	}
}
