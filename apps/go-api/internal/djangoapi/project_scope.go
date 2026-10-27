package djangoapi

import (
	"fmt"
	"uuid"
)

// ProjectMemberReadScopeSQL binds a project read to a live actor, workspace and
// current memberships in the data query. Arguments are SQL parameter positions.
// Select the final aggregate FROM scope to distinguish denial from an empty page.
func ProjectMemberReadScopeSQL(project, slug, actor, workspace int) string {
	return fmt.Sprintf(`scope AS MATERIALIZED (
 SELECT p.id,p.workspace_id FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE p.id=$%[1]d::uuid AND w.slug=$%[2]d AND w.id=$%[4]d::uuid AND p.deleted_at IS NULL AND w.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=$%[3]d::uuid AND u.is_active)
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$%[3]d::uuid
  AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id
  AND pm.member_id=$%[3]d::uuid AND pm.is_active AND pm.deleted_at IS NULL)
), `, project, slug, actor, workspace)
}

// ProjectReadRequest retains the API query while optionally requiring a bound,
// fresh project membership. Each repository restricts its supported operations.
type ProjectReadRequest struct {
	ExpectedWorkspaceID *uuid.UUID
	Operation           string
	Params              map[string]string
	Present             map[string]bool
}
