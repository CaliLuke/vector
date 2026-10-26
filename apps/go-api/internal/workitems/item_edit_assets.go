package workitems

import (
	"context"
	"slices"
)

func (r *Repository) associateItemEditAssets(ctx context.Context, actor string, row, old map[string]any) error {
	ids, err := r.itemAssetReferences(ctx, old["workspace_id"], row)
	if err != nil {
		return err
	}
	command := r.itemEdit.command
	column, kind := "issue_id", "ISSUE_DESCRIPTION"
	if command.Mode != "edit" {
		column, kind = "draft_issue_id", "DRAFT_ISSUE_DESCRIPTION"
	}
	assets, err := r.rows(ctx, `SELECT to_jsonb(a) FROM file_assets a WHERE id=ANY($1::uuid[]) OR `+column+`=$2 ORDER BY id FOR UPDATE`, ids, command.ItemID)
	if err != nil {
		return err
	}
	found := map[string]bool{}
	selected := []map[string]any{}
	for _, asset := range assets {
		id := stringValue(asset["id"])
		referenced := slices.Contains(ids, id)
		attached := command.Mode != "edit" && stringValue(asset["draft_issue_id"]) == command.ItemID && asset["entity_type"] == "DRAFT_ISSUE_ATTACHMENT" && asset["deleted_at"] == nil && asset["is_deleted"] != true && asset["is_archived"] != true
		if !referenced && !attached {
			continue
		}
		if !itemEditAssetAvailable(asset, old, actor, command) {
			return failure(400, "error", "A referenced image or attachment is unavailable for this work item.")
		}
		if asset["is_uploaded"] != true {
			return failure(409, "error", "Wait for referenced images and attachments to finish uploading before saving.")
		}
		found[id] = true
		selected = append(selected, asset)
		r.itemEdit.publicationAssets = append(r.itemEdit.publicationAssets, id)
	}
	for _, id := range ids {
		if !found[id] {
			return failure(400, "error", "A referenced image is unavailable. Remove it or upload it again.")
		}
	}
	// Include every existing binding in the snapshot, including removed inline
	// images. Only final references and retained attachments transfer on publish.
	r.itemEdit.assets = assets
	for _, asset := range selected {
		existing := stringValue(asset[column]) == command.ItemID
		if existing && (command.Mode == "edit" || stringValue(asset["project_id"]) == command.ProjectID) {
			continue
		}
		assetKind := kind
		if existing {
			assetKind = stringValue(asset["entity_type"])
		}
		if _, err = r.db.Exec(ctx, `UPDATE file_assets SET `+column+`=$2,project_id=NULLIF($3,'')::uuid,entity_identifier=$2::uuid::text,entity_type=$4 WHERE id=$1`, asset["id"], command.ItemID, command.ProjectID, assetKind); err != nil {
			return err
		}
	}
	return nil
}

func itemEditAssetAvailable(asset, source map[string]any, actor string, command itemEditCommand) bool {
	if asset["workspace_id"] != source["workspace_id"] || asset["deleted_at"] != nil || asset["is_deleted"] == true || asset["is_archived"] == true {
		return false
	}
	for _, field := range []string{"comment_id", "page_id", "user_id"} {
		if asset[field] != nil {
			return false
		}
	}
	issue, draft := stringValue(asset["issue_id"]), stringValue(asset["draft_issue_id"])
	if command.Mode == "edit" && issue == command.ItemID && draft == "" {
		return true
	}
	if command.Mode != "edit" && draft == command.ItemID && issue == "" && asset["created_by_id"] == actor {
		return true
	}
	if issue != "" || draft != "" || asset["created_by_id"] != actor {
		return false
	}
	project := stringValue(asset["project_id"])
	if project != "" && project != command.ProjectID && project != stringValue(source["project_id"]) {
		return false
	}
	return slices.Contains([]string{"ISSUE_DESCRIPTION", "DRAFT_ISSUE_DESCRIPTION"}, stringValue(asset["entity_type"]))
}
