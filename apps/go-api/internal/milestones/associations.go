package milestones

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type linkedItem struct {
	id                               uuid.UUID
	draft, deleted, archived, intake bool
}

func normalizedIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	result := append([]uuid.UUID{}, ids...)
	for _, id := range result {
		if id == uuid.Nil {
			return nil, ErrInvalidInput
		}
	}
	slices.SortFunc(result, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(result), nil
}

func lockItems(ctx context.Context, tx pgx.Tx, opts WriteOptions, ids []uuid.UUID) ([]linkedItem, error) {
	locks, err := tx.Query(ctx, `SELECT id FROM issues WHERE id=ANY($1::uuid[]) AND project_id=$2 AND workspace_id=$3 ORDER BY id FOR UPDATE`, ids, opts.ProjectID, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	locked, err := pgx.CollectRows(locks, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, err
	}
	// Read derived eligibility in a new statement after every row-lock wait.
	rows, err := tx.Query(ctx, `SELECT i.id,i.is_draft,i.deleted_at IS NOT NULL,i.archived_at IS NOT NULL,
 EXISTS(SELECT 1 FROM intake_issues bridge JOIN intakes target ON target.id=bridge.intake_id
 WHERE bridge.issue_id=i.id AND bridge.project_id=i.project_id AND bridge.workspace_id=i.workspace_id
 AND target.project_id=i.project_id AND target.workspace_id=i.workspace_id AND bridge.deleted_at IS NULL AND target.deleted_at IS NULL)
 FROM issues i WHERE i.id=ANY($1::uuid[]) AND i.project_id=$2 AND i.workspace_id=$3 ORDER BY i.id`, locked, opts.ProjectID, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []linkedItem{}
	for rows.Next() {
		var item linkedItem
		if err := rows.Scan(&item.id, &item.draft, &item.deleted, &item.archived, &item.intake); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func assignments(ctx context.Context, tx pgx.Tx, opts WriteOptions, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT issue_id,milestone_id FROM milestone_issues WHERE issue_id=ANY($1::uuid[]) AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, ids, opts.ProjectID, opts.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[uuid.UUID]uuid.UUID{}
	for rows.Next() {
		var issue, milestone uuid.UUID
		if err := rows.Scan(&issue, &milestone); err != nil {
			return nil, err
		}
		result[issue] = milestone
	}
	return result, rows.Err()
}

func (r *Repository) ManageWorkItems(ctx context.Context, opts WriteOptions, id uuid.UUID, add, remove []uuid.UUID) (MutationResult, error) {
	if id == uuid.Nil || len(add)+len(remove) == 0 {
		return MutationResult{}, ErrInvalidInput
	}
	add, err := normalizedIDs(add)
	if err != nil {
		return MutationResult{}, err
	}
	remove, err = normalizedIDs(remove)
	if err != nil {
		return MutationResult{}, err
	}
	all, err := normalizedIDs(append(slices.Clone(add), remove...))
	if err != nil {
		return MutationResult{}, err
	}
	adding, removing := idSet(add), idSet(remove)
	return r.write(ctx, opts, "manage_workitems", map[string]any{"milestone_id": id.String(), "add_work_item_ids": add, "remove_work_item_ids": remove}, func(tx pgx.Tx) (MutationResult, error) {
		if err := lockMilestone(ctx, tx, opts, id); err != nil {
			return MutationResult{}, err
		}
		record, err := retrieve(ctx, tx, opts.Scope, id)
		if err != nil {
			return MutationResult{}, err
		}
		items, err := lockItems(ctx, tx, opts, all)
		if err != nil {
			return MutationResult{}, err
		}
		if len(items) != len(all) {
			return MutationResult{}, ErrNotFound
		}
		if err = opts.revalidate(ctx); err != nil {
			return MutationResult{}, err
		}
		for _, item := range items {
			if item.draft || item.deleted {
				return MutationResult{}, ErrNotFound
			}
			if adding[item.id] && (item.archived || item.intake) {
				return MutationResult{}, ErrInvalidInput
			}
		}
		before, err := assignments(ctx, tx, opts, all)
		if err != nil {
			return MutationResult{}, err
		}
		for _, item := range add {
			if assigned := before[item]; assigned != uuid.Nil && assigned != id {
				return MutationResult{}, ErrConflict
			}
		}
		for _, item := range items {
			wasLinked := before[item.id] == id
			willLink := (wasLinked || adding[item.id]) && !removing[item.id]
			if wasLinked == willLink {
				continue
			}
			if err = changeLink(ctx, tx, opts, id, item.id, record.Title, willLink); err != nil {
				return MutationResult{}, err
			}
		}
		return mutationResult(ctx, tx, id, nil)
	})
}

func idSet(ids []uuid.UUID) map[uuid.UUID]bool {
	result := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func changeLink(ctx context.Context, tx pgx.Tx, opts WriteOptions, milestone, item uuid.UUID, title string, add bool) error {
	var err error
	if add {
		_, err = tx.Exec(ctx, `INSERT INTO milestone_issues(milestone_id,issue_id,project_id,workspace_id,created_by_id,updated_by_id,created_at,updated_at)
 VALUES($1,$2,$3,$4,$5,$5,statement_timestamp(),statement_timestamp())`, milestone, item, opts.ProjectID, opts.WorkspaceID, opts.ActorID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE milestone_issues SET deleted_at=statement_timestamp(),updated_at=statement_timestamp(),updated_by_id=$5
 WHERE milestone_id=$1 AND issue_id=$2 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, milestone, item, opts.ProjectID, opts.WorkspaceID, opts.ActorID)
	}
	if err != nil {
		return err
	}
	return admitLinkEffects(ctx, tx, opts, milestone, item, title, add)
}
