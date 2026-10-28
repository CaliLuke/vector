package worklogs

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	duration, description, err := mutationFields(cmd)
	if err != nil {
		return projectmutations.Result{}, err
	}
	resource := projectmutations.Resource{Name: "work_log", Table: "work_logs", RevisionTable: "work_log_edit_revisions", RevisionColumn: "work_log_id", ParentColumn: "issue_id", SchemaLock: true}
	resource.Authorize = func(ctx context.Context, tx pgx.Tx) error { return authorizeMutation(ctx, tx, cmd) }
	return projectmutations.Execute(ctx, r.db, resource, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		id := cmd.ResourceID
		var current json.RawMessage
		if cmd.Action != "create" {
			if err = tx.QueryRow(ctx, `SELECT `+projection+` FROM work_logs l WHERE l.id=$1`, id).Scan(&current); err != nil {
				return nil, err
			}
		}
		switch cmd.Action {
		case "create":
			id = uuid.New()
			text := ""
			if description != nil {
				text = *description
			}
			if _, err = tx.Exec(ctx, `INSERT INTO work_logs(id,issue_id,project_id,workspace_id,duration,description,logged_by_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,$7,$7)`, id, cmd.ParentID, cmd.ProjectID, cmd.WorkspaceID, *duration, text, cmd.ActorID); err != nil {
				return nil, err
			}
		case "update":
			changed, execErr := tx.Exec(ctx, `UPDATE work_logs SET duration=COALESCE($2,duration),description=COALESCE($3,description),updated_by_id=$4,updated_at=now() WHERE id=$1 AND (duration IS DISTINCT FROM COALESCE($2,duration) OR description IS DISTINCT FROM COALESCE($3,description))`, id, duration, description, cmd.ActorID)
			if execErr != nil {
				return nil, execErr
			}
			if changed.RowsAffected() == 0 {
				return current, nil
			}
		default:
			if _, err = tx.Exec(ctx, `UPDATE work_logs SET deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, id, cmd.ActorID); err != nil {
				return nil, err
			}
		}
		var result json.RawMessage
		if err = tx.QueryRow(ctx, `SELECT `+projection+` FROM work_logs l WHERE l.id=$1`, id).Scan(&result); err != nil {
			return nil, err
		}
		verb := map[string]string{"create": "created", "update": "updated", "delete": "deleted"}[cmd.Action]
		effect := map[string]any{"type": "work_log.activity." + verb, "workspace_id": cmd.WorkspaceID.String(), "project_id": cmd.ProjectID.String(), "issue_id": cmd.ParentID.String(), "actor_id": cmd.ActorID.String(), "epoch": time.Now().Unix(), "requested_data": result, "current_instance": current, "notification": false}
		if err = (projectmutations.Effects{Tx: tx}).Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, effect); err != nil {
			return nil, err
		}
		if cmd.Action == "delete" {
			return json.RawMessage("null"), nil
		}
		return result, nil
	})
}
func mutationFields(cmd projectmutations.Command) (*int64, *string, error) {
	if cmd.ParentID == uuid.Nil || (cmd.Action != "create" && cmd.Action != "update" && cmd.Action != "delete") {
		return nil, nil, projectmutations.ErrInvalid
	}
	var duration *int64
	var description *string
	for key, value := range cmd.Fields {
		switch key {
		case "duration":
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, nil, projectmutations.ErrInvalid
			}
			number, err := strconv.ParseInt(string(raw), 10, 32)
			if err != nil || number <= 0 {
				return nil, nil, projectmutations.ErrInvalid
			}
			duration = &number
		case "description":
			text, ok := value.(string)
			if !ok {
				return nil, nil, projectmutations.ErrInvalid
			}
			description = &text
		default:
			return nil, nil, projectmutations.ErrInvalid
		}
	}
	if cmd.Action == "create" && duration == nil || cmd.Action == "delete" && len(cmd.Fields) > 0 {
		return nil, nil, projectmutations.ErrInvalid
	}
	return duration, description, nil
}
