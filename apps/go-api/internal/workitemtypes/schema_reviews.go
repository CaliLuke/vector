package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Catalogs must use a single snapshot for definitions and their shared revision.
func readSchemaCatalog[T any](ctx context.Context, r *Repository, actor, slug string, read func(context.Context, pgx.Tx, string, bool) (T, error)) (T, error) {
	var zero T
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return zero, err
	}
	workspace, admin, err := authorize(ctx, tx, actor, slug, false)
	if err != nil {
		return zero, err
	}
	result, err := read(ctx, tx, workspace, admin)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

// Schema writers serialize before reading the graph or taking row locks. Keep
// READ COMMITTED so a waiter observes the transaction which released the lock.
func withSchemaWrite[T any](ctx context.Context, r *Repository, actor, slug string, write func(pgx.Tx, string) (T, error)) (T, error) {
	var zero T
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	workspace, _, err := lockWorkspaceSchema(ctx, tx, actor, slug)
	if err != nil {
		return zero, err
	}
	result, err := write(tx, workspace)
	if err != nil {
		return zero, err
	}
	return result, tx.Commit(ctx)
}

// Resolve authorization again after waiting, without switching to a workspace
// whose schema lock this transaction does not own.
func lockWorkspaceSchema(ctx context.Context, tx pgx.Tx, actor, slug string) (string, bool, error) {
	workspace, _, err := authorize(ctx, tx, actor, slug, true)
	if err != nil {
		return "", false, err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace); err != nil {
		return "", false, err
	}
	current, admin, err := authorize(ctx, tx, actor, slug, true)
	if err != nil {
		return "", false, err
	}
	if current != workspace {
		return "", false, reject(409, "The workspace changed. Reload and try again.")
	}
	return workspace, admin, nil
}

type schemaReviewStatus struct {
	Fingerprint        string
	Committed, Expired bool
}

func saveSchemaReview(ctx context.Context, tx pgx.Tx, workspace, actor string, command any, fingerprint string) (string, error) {
	raw, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	_, err = tx.Exec(ctx, `INSERT INTO work_item_schema_reviews(id,workspace_id,actor_id,command,fingerprint) VALUES($1,$2,$3,$4,$5)`, id, workspace, actor, raw, fingerprint)
	return id, err
}
func loadSchemaReview(ctx context.Context, tx pgx.Tx, workspace, actor, id string, command any) (schemaReviewStatus, error) {
	status := schemaReviewStatus{}
	if _, err := uuid.Parse(id); err != nil {
		return status, reject(422, "Choose a valid review.")
	}
	raw, err := json.Marshal(command)
	if err != nil {
		return status, err
	}
	var matches bool
	err = tx.QueryRow(ctx, `SELECT fingerprint,committed_at IS NOT NULL,created_at<clock_timestamp()-interval '30 minutes',command=$4::jsonb FROM work_item_schema_reviews WHERE id=$1 AND workspace_id=$2 AND actor_id=$3 FOR UPDATE`, id, workspace, actor, raw).Scan(&status.Fingerprint, &status.Committed, &status.Expired, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return status, reject(404, "Review not found.")
	}
	if err != nil {
		return status, err
	}
	if !matches {
		return status, reject(409, "The proposed change no longer matches this review.")
	}
	return status, nil
}
func finishSchemaReview(ctx context.Context, tx pgx.Tx, workspace, id string) error {
	if _, err := tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, workspace); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE work_item_schema_reviews SET committed_at=clock_timestamp() WHERE id=$1 AND workspace_id=$2`, id, workspace)
	return err
}
