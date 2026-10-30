package workitems

import (
	"context"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
)

// Keep relationship edits atomic with the model write: type availability and
// other database constraints may reject that write after validation has run.
func (r *Repository) updateIssueWithRelations(ctx context.Context, actor string, row, old map[string]any, changes issueChanges) (_ map[string]any, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	writer := *r
	writer.db = tx
	if err = writer.replaceIssueRelations(ctx, old, changes, false); err != nil {
		return nil, err
	}
	result, err := writer.saveIssue(ctx, actor, row, old)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit(ctx)
}
