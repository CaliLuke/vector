package workitemsearch

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type backfillDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Backfill runs before bootstrap releases its session lock and before the new
// server starts. Upgrades must quiesce older writers: they cannot maintain this
// projection. Completed workspaces are journalled atomically and are not replayed.
func Backfill(ctx context.Context, db backfillDatabase) error {
	for {
		var workspace string
		err := db.QueryRow(ctx, `SELECT w.id::text FROM public.workspaces w WHERE NOT EXISTS(SELECT 1 FROM public.work_item_search_backfills b WHERE b.workspace_id=w.id) ORDER BY w.id LIMIT 1`).Scan(&workspace)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = backfillWorkspace(ctx, db, workspace); err != nil {
			return err
		}
	}
}
func backfillWorkspace(ctx context.Context, db backfillDatabase, workspace string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET LOCAL search_path TO public`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace); err != nil {
		return err
	}
	var done bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM work_item_search_backfills WHERE workspace_id=$1)`, workspace).Scan(&done); err != nil {
		return err
	}
	if done {
		return tx.Commit(ctx)
	}
	cursor := "00000000-0000-0000-0000-000000000000"
	for {
		ids, batchErr := backfillBatch(ctx, tx, workspace, cursor)
		if batchErr != nil {
			return batchErr
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			if err = Refresh(ctx, tx, workspace, id); err != nil {
				return err
			}
		}
		cursor = ids[len(ids)-1]
	}
	if _, err = tx.Exec(ctx, `INSERT INTO work_item_search_backfills(workspace_id) VALUES($1)`, workspace); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func backfillBatch(ctx context.Context, tx pgx.Tx, workspace, cursor string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT id::text FROM issues WHERE workspace_id=$1 AND id>$2::uuid ORDER BY id LIMIT 100 FOR UPDATE`, workspace, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
