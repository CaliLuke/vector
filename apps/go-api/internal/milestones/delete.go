package milestones

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) Delete(ctx context.Context, opts WriteOptions, id uuid.UUID) (MutationResult, error) {
	if id == uuid.Nil {
		return MutationResult{}, ErrInvalidInput
	}
	return r.write(ctx, opts, "delete", map[string]any{"milestone_id": id.String()}, func(tx pgx.Tx) (MutationResult, error) {
		if err := lockMilestone(ctx, tx, opts, id); err != nil {
			return MutationResult{}, err
		}
		record, err := retrieve(ctx, tx, opts.Scope, id)
		if err != nil {
			return MutationResult{}, err
		}
		ids, err := linkedItemIDs(ctx, tx, opts, id)
		if err != nil {
			return MutationResult{}, err
		}
		items, err := lockItems(ctx, tx, opts, ids)
		if err != nil {
			return MutationResult{}, err
		}
		if err = opts.revalidate(ctx); err != nil {
			return MutationResult{}, err
		}
		// Cleanup includes hidden remnants; no hidden IDs or counts are returned.
		_, err = tx.Exec(ctx, `UPDATE milestone_issues SET deleted_at=statement_timestamp(),updated_at=statement_timestamp(),updated_by_id=$4
 WHERE milestone_id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, id, opts.ProjectID, opts.WorkspaceID, opts.ActorID)
		if err != nil {
			return MutationResult{}, err
		}
		for _, item := range items {
			if item.draft || item.deleted {
				continue
			}
			if err = admitLinkEffects(ctx, tx, opts, id, item.id, record.Title, false); err != nil {
				return MutationResult{}, err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE milestones SET deleted_at=statement_timestamp(),updated_at=statement_timestamp(),updated_by_id=$2 WHERE id=$1`, id, opts.ActorID)
		if err != nil {
			return MutationResult{}, err
		}
		if err = admitDeletionEffect(ctx, tx, opts, id); err != nil {
			return MutationResult{}, err
		}
		return mutationResult(ctx, tx, id, nil)
	})
}

func linkedItemIDs(ctx context.Context, tx pgx.Tx, opts WriteOptions, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT issue_id FROM milestone_issues WHERE milestone_id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL ORDER BY issue_id`, id, opts.ProjectID, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var item uuid.UUID
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		ids = append(ids, item)
	}
	return ids, rows.Err()
}
