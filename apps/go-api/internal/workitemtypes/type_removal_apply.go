package workitemtypes

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/jackc/pgx/v5"
)

func applyTypeRemoval(ctx context.Context, tx pgx.Tx, workspace, actor string, command typeRemovalCommand, plan typeRemovalPlan) error {
	if plan.review.InvalidItems > 0 {
		return reject(422, "Resolve affected work items before deleting this type.")
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS work_item_hierarchy_guard,draft_work_item_hierarchy_guard,type_hierarchy_guard,settings_hierarchy_guard DEFERRED`); err != nil {
		return err
	}
	for _, item := range plan.items {
		if err := applyTypeReplacementItem(ctx, tx, workspace, actor, command, item); err != nil {
			return err
		}
	}
	// Reload real usage after migration. Never bypass the deletion guard by
	// inventing a zero count, or transfer the user's creation preference silently.
	catalog, err := loadCatalog(ctx, tx, workspace, true)
	if err != nil {
		return err
	}
	if err = remove(ctx, tx, workspace, actor, command.TypeID, catalog); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT validate_workspace_work_item_hierarchy($1::uuid)`, workspace); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SET CONSTRAINTS work_item_hierarchy_guard,draft_work_item_hierarchy_guard,type_hierarchy_guard,settings_hierarchy_guard IMMEDIATE`)
	return err
}

func compactTypeRemovalHierarchy(ctx context.Context, tx pgx.Tx, workspace, actor string) error {
	catalog, err := loadHierarchyCatalog(ctx, tx, workspace, true)
	if err != nil || !catalog.HierarchyEnabled {
		return err
	}
	input := HierarchyInput{Revision: catalog.Revision, DefaultLevel: catalog.DefaultLevel}
	for _, item := range catalog.Types {
		input.Placements = append(input.Placements, HierarchyPlacement{TypeID: item.ID, Level: item.Level})
	}
	command, err := normalizeHierarchy(input)
	if err != nil {
		return err
	}
	// Unlike a hierarchy-edit review this migration must not unlink anything.
	// The shared writer compacts placements/defaults, then validates every link.
	return applyHierarchy(ctx, tx, workspace, actor, command, hierarchyPlan{})
}
func applyTypeReplacementItem(ctx context.Context, tx pgx.Tx, workspace, actor string, command typeRemovalCommand, item typeRemovalItem) error {
	table := "issues"
	if item.Draft {
		table = "draft_issues"
	}
	values, err := json.Marshal(item.Transition.Values)
	if err != nil {
		return err
	}
	description := item.Transition.Description
	if _, err = tx.Exec(ctx, `UPDATE `+table+` SET type_id=$3,custom_properties=$4,description_html=$5,description_stripped=$6,updated_at=clock_timestamp(),updated_by_id=$7 WHERE id=$1 AND workspace_id=$2`, item.ID, workspace, command.ReplacementID, values, description, strings.TrimSpace(content.StripTags(description)), actor); err != nil {
		return err
	}
	if item.Draft {
		return nil
	}
	if err = workitemsearch.Refresh(ctx, tx, workspace, item.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_identifier,new_identifier,old_value,new_value,comment,attachments) SELECT gen_random_uuid(),clock_timestamp(),clock_timestamp(),$1,$2,$3,$4,'updated','type',$5,$6,old.name,replacement.name,'Type reassigned after reviewed workspace type deletion. Former custom values preserved in the description.',ARRAY[]::text[] FROM issue_types old,issue_types replacement WHERE old.id=$5 AND replacement.id=$6`, workspace, item.Project, item.ID, actor, command.TypeID, command.ReplacementID); err != nil {
		return err
	}
	state := ItemHistoryState{WorkspaceID: workspace, ProjectID: item.Project, ItemID: item.ID}
	for _, entry := range item.Transition.Preserved {
		if err = RecordPreservedProperty(ctx, tx, actor, state, entry); err != nil {
			return err
		}
	}
	return nil
}
