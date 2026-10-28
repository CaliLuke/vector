package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// The caller owns the transaction and project lock. The state lock serializes
// deletion with the deferred live-reference checks on issues and drafts.
func deleteState(ctx context.Context, tx pgx.Tx, input StateWriteInput) error {
	var isDefault bool
	err := tx.QueryRow(ctx, `SELECT s."default" FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.id=$1 AND s.project_id=$2 AND w.slug=$3 AND s.deleted_at IS NULL AND s."group"<>'triage' AND NOT s.is_triage FOR UPDATE OF s`, input.StateID, input.ProjectID, input.Slug).Scan(&isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return missingStateFailure(input)
	}
	if err != nil {
		return err
	}
	if isDefault {
		return &WriteFailure{Status: 400, Body: map[string]string{"error": "Default state cannot be deleted"}}
	}
	var occupied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE state_id=$1 AND deleted_at IS NULL)
 OR EXISTS(SELECT 1 FROM draft_issues WHERE state_id=$1 AND deleted_at IS NULL)`, input.StateID).Scan(&occupied); err != nil {
		return err
	}
	if occupied {
		return &WriteFailure{Status: 400, Body: map[string]string{"error": "The state is not empty, only empty states can be deleted"}}
	}
	var id string
	err = tx.QueryRow(ctx, `UPDATE states SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$4
 WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL RETURNING id::text`, input.StateID, input.ProjectID, input.Slug, input.UserID).Scan(&id)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{"app": "db", "model": "state", "id": id})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,'soft_delete',$2)`, uuid.New(), payload)
	return err
}
