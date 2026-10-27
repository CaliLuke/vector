package workitems

// Bound reads obtain annotations and relation ownership in the same snapshot as
// item visibility. Later serialization does not make fresh, unbound lookups.
func collaborationReadRow(q Request, kind string) string {
	if q.ExpectedWorkspaceID == nil || kind != "comment" {
		return "to_jsonb(t)"
	}
	return `to_jsonb(t)||jsonb_build_object('is_member',true,
 'description_id',(SELECT d.id FROM descriptions d WHERE d.id=t.description_id AND d.project_id=t.project_id AND d.workspace_id=t.workspace_id AND d.deleted_at IS NULL),
 'parent_id',(SELECT parent.id FROM issue_comments parent WHERE parent.id=t.parent_id AND parent.issue_id=t.issue_id AND parent.project_id=t.project_id AND parent.workspace_id=t.workspace_id AND parent.deleted_at IS NULL))`
}
