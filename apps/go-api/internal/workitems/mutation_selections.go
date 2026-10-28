package workitems

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Lock supplied active selections in their owning tenant. Native serializers
// retain field validation and model hooks; stale or foreign IDs cannot silently
// become partial assignment changes while this transaction waits.
func (r *Repository) lockWorkitemSelections(ctx context.Context, cmd projectmutations.Command, fields map[string]any) error {
	for _, kind := range []string{"assignees", "labels"} {
		values, present := fields[kind]
		if !present {
			continue
		}
		ids, ok := values.([]string)
		if !ok {
			return projectmutations.ErrInvalid
		}
		for _, raw := range ids {
			var id uuid.UUID
			query := `SELECT id FROM labels WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR SHARE`
			if kind == "assignees" {
				query = `SELECT u.id FROM users u JOIN project_members pm ON pm.member_id=u.id JOIN workspace_members wm ON wm.member_id=u.id AND wm.workspace_id=pm.workspace_id
 WHERE u.id=$1 AND u.is_active AND pm.project_id=$2 AND pm.workspace_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(15,20)
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,pm,wm`
			}
			err := r.db.QueryRow(ctx, query, raw, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				return projectmutations.ErrInvalid
			}
			if err != nil {
				return err
			}
		}
	}
	// State and estimate selection must retain same-project ownership even when
	// records carry inconsistent foreign keys; native validation checks the rest.
	for _, kind := range []string{"state", "estimate_point"} {
		value, ok := fields[kind]
		if !ok {
			continue
		}
		table := "states"
		if kind == "estimate_point" {
			table = "estimate_points"
		}
		var id uuid.UUID
		err := r.db.QueryRow(ctx, "SELECT id FROM "+table+" WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR SHARE", value, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrInvalid
		}
		if err != nil {
			return err
		}
	}
	return nil
}
