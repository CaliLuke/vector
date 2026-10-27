package issuequery

import "fmt"

// IssueVisibleSQL checks a related published item against the reader's current
// memberships. All arguments are trusted SQL expressions, never request text.
// Callers choose whether archived or triage items belong in their result set.
func IssueVisibleSQL(issue, actor, workspace string) string {
	return fmt.Sprintf(`(%[1]s.workspace_id=%[3]s AND %[1]s.deleted_at IS NULL AND NOT %[1]s.is_draft AND EXISTS(
 SELECT 1 FROM projects visibility_project
 JOIN workspaces visibility_workspace ON visibility_workspace.id=visibility_project.workspace_id
 JOIN project_members visibility_member ON visibility_member.project_id=visibility_project.id AND visibility_member.workspace_id=visibility_workspace.id
 JOIN workspace_members visibility_workspace_member ON visibility_workspace_member.workspace_id=visibility_workspace.id AND visibility_workspace_member.member_id=visibility_member.member_id
 WHERE visibility_project.id=%[1]s.project_id AND visibility_project.workspace_id=%[1]s.workspace_id
 AND visibility_project.deleted_at IS NULL AND visibility_workspace.deleted_at IS NULL
 AND visibility_member.member_id=%[2]s AND visibility_member.is_active AND visibility_member.deleted_at IS NULL
 AND visibility_workspace_member.is_active AND visibility_workspace_member.deleted_at IS NULL
 AND (visibility_member.role IN (15,20) OR visibility_member.role=5 AND (visibility_project.guest_view_all_features OR %[1]s.created_by_id=%[2]s))))`, issue, actor, workspace)
}

// VisibleParentSQL masks the ID as well as any later expansion of a parent.
func VisibleParentSQL(issue, actor string) string {
	return `(SELECT visible_parent.id FROM issues visible_parent WHERE visible_parent.id=` + issue + `.parent_id AND ` + IssueVisibleSQL("visible_parent", actor, issue+".workspace_id") + `)`
}
