package workitems

import (
	"context"
	"errors"
	"io"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

// writeProjectTransaction acquires the schema lock before any item snapshot is
// read. Older full-row writers must participate in the same ordering as schema
// migrations, or a concurrent edit can restore removed fields and descriptions.
func (r *Repository) writeProjectTransaction(ctx context.Context, projectID string, authorize func(*Repository) error, operation func(*Repository) (Result, error)) (_ Result, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	writer := *r
	writer.db, writer.transactionalEffects = tx, true
	if err = authorize(&writer); err != nil {
		return Result{}, err
	}
	var workspaceID string
	if err = tx.QueryRow(ctx, `SELECT workspace_id::text FROM projects WHERE id=$1`, projectID).Scan(&workspaceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Result{}, requiredObject()
		}
		return Result{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspaceID); err != nil {
		return Result{}, err
	}
	// Membership can change while the schema lock is held by another writer.
	if err = authorize(&writer); err != nil {
		return Result{}, err
	}
	result, err := operation(&writer)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

// projectRequest keeps read authorization outside a transaction and repeats
// write authorization inside the schema-locked mutation boundary.
func (r *Repository) projectRequest(ctx context.Context, q Request, body io.Reader, contentType string,
	authorize func(*Repository, context.Context, Request) error,
	operation func(*Repository, context.Context, Request, io.Reader, string) (Result, error),
) (Result, error) {
	if q.Method == "GET" {
		if err := authorize(r, ctx, q); err != nil {
			return Result{}, err
		}
		return operation(r, ctx, q, body, contentType)
	}
	return r.writeProjectTransaction(ctx, q.ProjectID, func(writer *Repository) error {
		if err := writer.AuthorizeInitial(ctx, q); err != nil {
			return err
		}
		return authorize(writer, ctx, q)
	}, func(writer *Repository) (Result, error) {
		return operation(writer, ctx, q, body, contentType)
	})
}
