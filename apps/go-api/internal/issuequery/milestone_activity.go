package issuequery

// MilestoneActivityVisibleSQL protects retained milestone snapshots with current
// project access and item ownership. It deliberately does not require the old
// milestone to exist: deletion is itself part of the item's audit history.
// Arguments must be authored SQL expressions, never request text.
func MilestoneActivityVisibleSQL(field, item, project, workspace, actor string) string {
	return `(` + field + ` IS DISTINCT FROM 'milestone' OR (` + MilestoneReadScopeSQL(project, workspace, actor) + `
 AND EXISTS(SELECT 1 FROM issues milestone_activity_item
 WHERE milestone_activity_item.id=` + item + ` AND milestone_activity_item.project_id=` + project + `
 AND milestone_activity_item.workspace_id=` + workspace + `
 AND milestone_activity_item.deleted_at IS NULL AND NOT milestone_activity_item.is_draft)))`
}
