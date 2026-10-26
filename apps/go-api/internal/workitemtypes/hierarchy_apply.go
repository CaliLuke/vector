package workitemtypes

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func applyHierarchy(ctx context.Context, tx pgx.Tx, workspace, actor string, command hierarchyCommand, plan hierarchyPlan) error {
	// A valid final graph may have invalid intermediate level placements. Defer
	// only graph constraints, never foreign keys or unrelated schema validation.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS work_item_hierarchy_guard,draft_work_item_hierarchy_guard,type_hierarchy_guard,settings_hierarchy_guard DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL vector.schema_migration='on'`); err != nil {
		return err
	}
	raw, err := json.Marshal(command.Placements)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `WITH placement AS (SELECT t.id,COALESCE(p.level,0) AS level FROM issue_types t LEFT JOIN jsonb_to_recordset($2::jsonb) AS p(type_id uuid,level integer) ON p.type_id=t.id WHERE t.workspace_id=$1 AND t.deleted_at IS NULL)
 UPDATE issue_types t SET level=p.level,updated_at=clock_timestamp(),updated_by_id=$3 FROM placement p WHERE t.id=p.id AND t.level IS DISTINCT FROM p.level::double precision`, workspace, raw, actor); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE project_issue_types p SET level=t.level::integer,updated_at=clock_timestamp(),updated_by_id=$2 FROM issue_types t WHERE p.issue_type_id=t.id AND t.workspace_id=$1 AND p.workspace_id=$1 AND p.level IS DISTINCT FROM t.level::integer`, workspace, actor); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_type_settings SET hierarchy_enabled=true,default_level=$2 WHERE workspace_id=$1`, workspace, command.DefaultLevel); err != nil {
		return err
	}
	for _, child := range plan.invalid {
		table := "issues"
		if child.Draft {
			table = "draft_issues"
		}
		if _, err = tx.Exec(ctx, `UPDATE `+table+` SET parent_id=NULL,updated_at=clock_timestamp(),updated_by_id=$3 WHERE workspace_id=$1 AND id=$2`, workspace, child.ID, actor); err != nil {
			return err
		}
		if child.Draft {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_identifier,new_identifier,old_value,new_value,comment,attachments) VALUES(gen_random_uuid(),clock_timestamp(),clock_timestamp(),$1,$2,$3,$4,'updated','parent',$5,NULL,'link after a hierarchy change','','Parent relationship removed after a reviewed hierarchy change.',ARRAY[]::text[])`, workspace, child.Project, child.ID, actor, child.Parent); err != nil {
			return err
		}
	}
	// The same validator protects ordinary writers and configuration-only saves.
	// Check before recording the committed receipt, then force all queued checks.
	if _, err = tx.Exec(ctx, `SELECT validate_workspace_work_item_hierarchy($1::uuid)`, workspace); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SET CONSTRAINTS work_item_hierarchy_guard,draft_work_item_hierarchy_guard,type_hierarchy_guard,settings_hierarchy_guard IMMEDIATE`)
	return err
}
