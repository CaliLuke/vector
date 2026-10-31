package workitemtypes

import (
	"context"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

func readProjectSchemaCatalog[T any](ctx context.Context, r *Repository, actor, slug, project string, read func(context.Context, pgx.Tx, string, string, bool) (T, error)) (_ T, resultErr error) {
	var zero T
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return zero, err
	}
	workspace, admin, err := authorizeProject(ctx, tx, actor, slug, project, false, false)
	if err != nil {
		return zero, err
	}
	result, err := read(ctx, tx, workspace, project, admin)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

// Reauthorize under the workspace schema lock, before reading mutable configuration.
func withProjectSchemaWrite[T any](ctx context.Context, r *Repository, actor, slug, project string, write func(pgx.Tx, string, bool) (T, error)) (_ T, resultErr error) {
	var zero T
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	workspace, _, err := authorizeProject(ctx, tx, actor, slug, project, true, false)
	if err != nil {
		return zero, err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace); err != nil {
		return zero, err
	}
	current, admin, err := authorizeProject(ctx, tx, actor, slug, project, true, true)
	if err != nil {
		return zero, err
	}
	if current != workspace {
		return zero, reject(409, "The workspace changed. Reload and try again.")
	}
	result, err := write(tx, workspace, admin)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}
