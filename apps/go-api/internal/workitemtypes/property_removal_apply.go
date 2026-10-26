package workitemtypes

import (
	"context"
	"html"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func preservationLabels(ctx context.Context, tx pgx.Tx, workspace, project string, kind property.Kind, value any) (map[string]string, error) {
	labels := map[string]string{}
	if kind != property.Member && kind != property.Release {
		return labels, nil
	}
	ids := []string{}
	switch saved := value.(type) {
	case string:
		ids = append(ids, saved)
	case []string:
		ids = append(ids, saved...)
	case []any:
		for _, entry := range saved {
			if id, ok := entry.(string); ok {
				ids = append(ids, id)
			}
		}
	}
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return nil, reject(422, "A stored selection is invalid. Repair it before removal.")
		}
	}
	query := `SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),NULLIF(trim(u.first_name||' '||u.last_name),''),u.username) FROM users u WHERE u.id=ANY($2::uuid[]) AND EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=$1 AND m.member_id=u.id AND m.deleted_at IS NULL)`
	args := []any{workspace, ids}
	if kind == property.Release {
		query = `SELECT id::text,name FROM work_item_releases WHERE workspace_id=$1 AND id=ANY($2::uuid[]) AND project_id=NULLIF($3,'')::uuid`
		args = append(args, project)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		labels[id] = name
	}
	return labels, rows.Err()
}

func applyPropertyRemoval(ctx context.Context, tx pgx.Tx, workspace, actor string, command propertyRemoval, plan propertyRemovalPlan) error {
	// Unrelated required fields may predate their configuration. Removing a field
	// does not make those old items invalid; ordinary subsequent edits still validate.
	if _, err := tx.Exec(ctx, `SET LOCAL vector.schema_migration='on'`); err != nil {
		return err
	}
	reason := preservationReason(command, plan.definition)
	for _, item := range plan.items {
		description := item.Description
		if !command.Permanent && item.HTML != "" {
			description += "<h3>" + html.EscapeString(reason) + "</h3>" + item.HTML
		}
		if len(description) > content.MaxContentBytes {
			return reject(422, "Preserving these values would exceed a work item's description limit.")
		}
		// Sanitize only newly preserved content; never rewrite the existing document.
		stripped := strings.TrimSpace(content.StripTags(description))
		table := "issues"
		if item.Draft {
			table = "draft_issues"
		}
		if _, err := tx.Exec(ctx, `UPDATE `+table+` SET custom_properties=custom_properties-$3,description_html=$4,description_stripped=$5,updated_at=clock_timestamp(),updated_by_id=$6 WHERE id=$1 AND workspace_id=$2`, item.ID, workspace, command.ID, description, stripped, actor); err != nil {
			return err
		}
		if item.Draft {
			continue
		}
		if err := workitemsearch.Refresh(ctx, tx, workspace, item.ID); err != nil {
			return err
		}
		previous := content.StripTags(item.HTML)
		change := "Values preserved in description."
		if command.Permanent {
			change = "Current values removed without preservation. Existing history retained."
			previous = ""
		}
		if _, err := tx.Exec(ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_value,new_value,comment,attachments) VALUES(gen_random_uuid(),clock_timestamp(),clock_timestamp(),$1,NULLIF($2,'')::uuid,$3,$4,'updated','custom_properties',$5,$6,$7,ARRAY[]::text[])`, workspace, item.Project, item.ID, actor, previous, change, reason); err != nil {
			return err
		}
		// Keep a field-specific final version even when its definition is removed.
		// History access remains subject to the owning item's visibility.
		if !command.Permanent && plan.definition.Kind == property.RichText && item.Value != nil {
			state := ItemHistoryState{WorkspaceID: workspace, ProjectID: item.Project, ItemID: item.ID}
			entry := PreservedItemProperty{Definition: plan.definition, Value: item.Value, HTML: item.HTML}
			if err := RecordPreservedProperty(ctx, tx, actor, state, entry); err != nil {
				return err
			}
		}
	}
	return nil
}
