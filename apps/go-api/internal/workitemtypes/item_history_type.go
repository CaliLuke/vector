package workitemtypes

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func recordItemTypeChange(ctx context.Context, tx pgx.Tx, actor string, before *ItemHistoryState, after ItemHistoryState) error {
	if before == nil || before.TypeID == after.TypeID {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_identifier,new_identifier,old_value,new_value,comment,attachments)
 VALUES(gen_random_uuid(),clock_timestamp(),clock_timestamp(),$1,$2,$3,$4,'updated','type',NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,
 COALESCE((SELECT name FROM issue_types WHERE id=NULLIF($5,'')::uuid AND workspace_id=$1),''),
 COALESCE((SELECT name FROM issue_types WHERE id=NULLIF($6,'')::uuid AND workspace_id=$1),''),'',ARRAY[]::text[])`, after.WorkspaceID, after.ProjectID, after.ItemID, actor, before.TypeID, after.TypeID)
	return err
}
