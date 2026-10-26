package workitems

import (
	"context"
)

// An interrupted upload may never have reached the editor. Only pending assets
// referenced by the submitted content prevent publication; abandoned rows stay
// private and are never transferred to the published item.
func (r *Repository) validateDraftUploads(ctx context.Context, draft, row map[string]any) error {
	ids, err := r.itemAssetReferences(ctx, draft["workspace_id"], row)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	var pending bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE draft_issue_id=$1 AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND NOT is_deleted AND NOT is_archived AND NOT is_uploaded AND workspace_id=$3 AND created_by_id=$4)`, draft["id"], ids, draft["workspace_id"], draft["created_by_id"]).Scan(&pending)
	if err != nil {
		return err
	}
	if pending {
		return failure(409, "error", "Wait for referenced draft uploads to finish before publishing.")
	}
	return nil
}
