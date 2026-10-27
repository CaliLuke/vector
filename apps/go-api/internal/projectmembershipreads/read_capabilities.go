package projectmembershipreads

import (
	"context"
	"uuid"
)

// ReadCapabilities describes available membership scopes, not a transport
// catalogue. Workspace permits scoped role reads and project collection reads;
// each target's current visibility must still be checked at invocation.
type ReadCapabilities struct{ Workspace, WorkspaceMembers, ProjectMembers bool }

// ReadCapabilities evaluates current primary-database permissions. A zero
// workspace selects all active memberships for discovery without a selected
// workspace. Supplying an immutable ID and slug requires both to match.
func (r *DirectoryReader) ReadCapabilities(ctx context.Context, actor, workspace uuid.UUID, slug string) (ReadCapabilities, error) {
	var capabilities ReadCapabilities
	err := r.primary.QueryRow(ctx, `WITH scopes AS MATERIALIZED (
 SELECT w.id,wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id
 JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active
 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR w.id=$2) AND ($3='' OR w.slug=$3)
 ) SELECT EXISTS(SELECT 1 FROM scopes),
 EXISTS(SELECT 1 FROM scopes WHERE role IN(15,20)),
 EXISTS(SELECT 1 FROM scopes s JOIN projects p ON p.workspace_id=s.id
 JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=s.id
 WHERE p.deleted_at IS NULL AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL)`, actor, workspace, slug).Scan(&capabilities.Workspace, &capabilities.WorkspaceMembers, &capabilities.ProjectMembers)
	return capabilities, err
}
