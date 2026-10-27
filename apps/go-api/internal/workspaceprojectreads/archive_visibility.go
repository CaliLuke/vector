package workspaceprojectreads

import (
	"context"

	"github.com/google/uuid"
)

type ProjectWriteVisibility struct{ Archive, Unarchive bool }

// VisibleProjectWrites reports capability availability under the v1 API policy.
// Each invocation still authorizes its specific project in the write transaction.
// A zero workspace selects all current memberships for context-free OAuth discovery.
func (r *LifecycleRepository) VisibleProjectWrites(ctx context.Context, actor, workspace uuid.UUID, slug string) (ProjectWriteVisibility, error) {
	var visible ProjectWriteVisibility
	err := r.writer.QueryRow(ctx, `WITH scopes AS MATERIALIZED (
 SELECT w.id,wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id
 JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active
 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR w.id=$2) AND ($3='' OR w.slug=$3)
 ) SELECT EXISTS(SELECT 1 FROM scopes WHERE role IN(15,20)),
 EXISTS(SELECT 1 FROM scopes s JOIN projects p ON p.workspace_id=s.id JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=s.id
 WHERE p.deleted_at IS NULL AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) AND (s.role=20 OR pm.role=20))`, actor, workspace, slug).Scan(&visible.Archive, &visible.Unarchive)
	return visible, err
}
