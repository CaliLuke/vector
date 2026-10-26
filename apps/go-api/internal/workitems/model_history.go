package workitems

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

// The model and its history share a transaction even for callers that do not
// already own one. Existing transactional callers receive a nested savepoint.
// Callers constructing a full proposed row must lock its workspace schema before
// reading the source. Reloading history here does not merge stale proposed edits.
func (r *Repository) saveIssue(ctx context.Context, actor string, row, old map[string]any) (map[string]any, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := *r
	writer.db = tx
	project, err := writer.project(ctx, stringValue(row["project_id"]))
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, project["workspace_id"]); err != nil {
		return nil, err
	}
	var before *workitemtypes.ItemHistoryState
	if old != nil {
		current, readErr := writer.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE id=$1 FOR UPDATE`, row["id"])
		if readErr != nil {
			return nil, readErr
		}
		before, err = loadModelHistory(ctx, tx, current)
		if err != nil {
			return nil, err
		}
	}
	saved, err := writer.saveIssueRow(ctx, actor, row, old)
	if err != nil {
		return nil, err
	}
	after, err := loadModelHistory(ctx, tx, saved)
	if err != nil {
		return nil, err
	}
	if after != nil {
		var preserved []workitemtypes.PreservedItemProperty
		if r.itemEdit != nil {
			preserved = r.itemEdit.plan.Preserved
			if r.itemEdit.command.Restore != nil {
				after.RestoredPropertyID = r.itemEdit.command.Restore.PropertyID
			}
		}
		if err = workitemtypes.RecordItemPropertyChanges(ctx, tx, actor, before, *after, preserved...); err != nil {
			return nil, err
		}
	}
	if err = workitemsearch.Refresh(ctx, tx, stringValue(saved["workspace_id"]), stringValue(saved["id"])); err != nil {
		return nil, err
	}
	return saved, tx.Commit(ctx)
}

func loadModelHistory(ctx context.Context, tx pgx.Tx, row map[string]any) (*workitemtypes.ItemHistoryState, error) {
	if truthy(row["is_draft"]) || row["deleted_at"] != nil {
		return nil, nil
	}
	values, _ := row["custom_properties"].(map[string]any)
	state, err := workitemtypes.LoadItemHistoryState(ctx, tx, workitemtypes.ItemHistoryState{
		WorkspaceID: stringValue(row["workspace_id"]), ProjectID: stringValue(row["project_id"]),
		ItemID: stringValue(row["id"]), TypeID: stringValue(row["type_id"]), Values: values,
	})
	return &state, err
}
