package backgroundtasks

import (
	"context"
	"fmt"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

// Deactivation and its notification commit together. The effect worker retries
// email admission independently of the now-disabled webhook's delivery workflow.
func (store *PostgresWebhookTaskStore) deactivate(ctx context.Context, webhook webhookRecord, currentSite string, cause error) (resultErr error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	changed, err := tx.Exec(ctx, `UPDATE webhooks SET is_active=false,updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND is_active=true`, webhook.ID, webhook.WorkspaceID)
	if err != nil {
		return fmt.Errorf("deactivate webhook after retries: %w", err)
	}
	if changed.RowsAffected() == 1 && webhook.CreatorID != nil {
		effects := projectmutations.Effects{Tx: tx}
		if err := effects.Task(ctx, webhookDeactivationEmailTask, []any{webhook.ID, *webhook.CreatorID, currentSite, cause.Error()}, nil); err != nil {
			return fmt.Errorf("queue webhook deactivation email: %w", err)
		}
	}
	return tx.Commit(ctx)
}
