package projectstatereads

import (
	"context"

	"github.com/google/uuid"
)

type StateCapabilities struct{ Read, Write bool }

// VisibleActions describes current workspace reads and writes available in at
// least one project. Each invocation still admits its own scope and target.
func (r *WriteRepository) VisibleActions(ctx context.Context, actor, workspace uuid.UUID, slug string) (StateCapabilities, error) {
	if actor == uuid.Nil || workspace == uuid.Nil || slug == "" {
		return StateCapabilities{}, nil
	}
	var result StateCapabilities
	err := r.database.QueryRow(ctx, `WITH scope AS MATERIALIZED (
 SELECT w.id FROM workspaces w WHERE w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=$1 AND u.is_active)
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))
)
 SELECT EXISTS(SELECT 1 FROM scope),EXISTS(SELECT 1 FROM projects p JOIN scope w ON w.id=p.workspace_id
 WHERE p.deleted_at IS NULL AND p.archived_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id
 AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(15,20)))`, actor, workspace, slug).Scan(&result.Read, &result.Write)
	return result, err
}
