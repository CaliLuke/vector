package issuequery

// MilestoneReadScopeSQL admits a live, unarchived project for the native
// milestone domain. Arguments must be authored SQL expressions or placeholders,
// never request text. Callers additionally bind the authenticated workspace.
func MilestoneReadScopeSQL(project, workspace, actor string) string {
	return `EXISTS(SELECT 1 FROM projects milestone_project
 JOIN workspaces milestone_workspace ON milestone_workspace.id=milestone_project.workspace_id
 JOIN workspace_members milestone_wm ON milestone_wm.workspace_id=milestone_workspace.id
 JOIN users milestone_actor ON milestone_actor.id=milestone_wm.member_id
 JOIN project_members milestone_pm ON milestone_pm.project_id=milestone_project.id
 AND milestone_pm.workspace_id=milestone_workspace.id AND milestone_pm.member_id=milestone_actor.id
 WHERE milestone_project.id=` + project + ` AND milestone_workspace.id=` + workspace + `
 AND milestone_actor.id=` + actor + ` AND milestone_actor.is_active
 AND milestone_workspace.deleted_at IS NULL AND milestone_project.deleted_at IS NULL
 AND milestone_project.archived_at IS NULL
 AND milestone_wm.deleted_at IS NULL AND milestone_wm.is_active AND milestone_wm.role IN (5,15,20)
 AND milestone_pm.deleted_at IS NULL AND milestone_pm.is_active
 AND (milestone_pm.role IN (15,20) OR (milestone_pm.role=5 AND milestone_project.guest_view_all_features)))`
}
