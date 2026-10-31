package projectconfiguration

import (
	"context"
	"encoding/json"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

// Point writers take the catalogue lock before reading their parent or changing
// a point. MCP and native deletion use the same order, before revision triggers.
// The response is captured in the write transaction so a subsequent edit cannot
// replace the snapshot returned for this accepted write.
func (r *Repository) withEstimatePointMutation(ctx context.Context, slug, project string, mutate func(*Repository) (json.RawMessage, error)) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err = lockEstimateCatalogue(ctx, tx, slug, project); err != nil {
		return nil, err
	}
	result, err := mutate(&Repository{db: tx})
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func lockEstimateCatalogue(ctx context.Context, tx pgx.Tx, slug, project string) error {
	_, err := tx.Exec(ctx, `SELECT work_item_schema_lock(p.workspace_id) FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2`, project, slug)
	if err != nil {
		return err
	}
	// Workspace deletion does not take the schema lock. Hold its row through
	// commit, after acquiring the schema lock in the same order as MCP writes.
	_, err = tx.Exec(ctx, `SELECT w.id FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 FOR SHARE OF w`, project, slug)
	return err
}
