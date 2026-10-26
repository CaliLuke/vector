package djangoapi

import (
	"context"
	"fmt"
	"maps"
)

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

// PageVisibleSQL applies the page-description reader's page access, project-link,
// and membership rules to a specific project context.
func PageVisibleSQL(page, actor, workspace, project string) string {
	return fmt.Sprintf(`(%[1]s.workspace_id=%[3]s AND %[1]s.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM workspaces asset_page_workspace WHERE asset_page_workspace.id=%[3]s AND asset_page_workspace.deleted_at IS NULL)
 AND (%[1]s.owned_by_id=%[2]s OR %[1]s.access=0)
 AND EXISTS(SELECT 1 FROM project_pages page_visibility_link
 JOIN projects page_visibility_project ON page_visibility_project.id=page_visibility_link.project_id
  AND page_visibility_project.workspace_id=%[1]s.workspace_id AND page_visibility_project.deleted_at IS NULL
 JOIN project_members page_visibility_member ON page_visibility_member.project_id=page_visibility_project.id
  AND page_visibility_member.workspace_id=page_visibility_project.workspace_id
  AND page_visibility_member.member_id=%[2]s AND page_visibility_member.role IN (5,15,20)
 WHERE page_visibility_link.page_id=%[1]s.id AND page_visibility_link.project_id=%[4]s
	  AND page_visibility_link.deleted_at IS NULL
	  AND page_visibility_member.is_active AND page_visibility_member.deleted_at IS NULL))`, page, actor, workspace, project)
}

// VisibleParentSQL masks the ID as well as any later expansion of a parent.
func VisibleParentSQL(issue, actor string) string {
	return `(SELECT visible_parent.id FROM issues visible_parent WHERE visible_parent.id=` + issue + `.parent_id AND ` + IssueVisibleSQL("visible_parent", actor, issue+".workspace_id") + `)`
}

// PublicRelatedIssueSQL limits related-item metadata to the already authorized
// public board. Publication of one project never publishes another project's links.
func PublicRelatedIssueSQL(related, anchor string) string {
	return fmt.Sprintf(`(%[1]s.project_id=%[2]s.project_id AND %[1]s.workspace_id=%[2]s.workspace_id
 AND %[1]s.deleted_at IS NULL AND %[1]s.archived_at IS NULL AND NOT %[1]s.is_draft
 AND NOT EXISTS(SELECT 1 FROM states public_related_state WHERE public_related_state.id=%[1]s.state_id AND public_related_state."group"='triage'))`, related, anchor)
}

func PublicParentSQL(issue string) string {
	return `(SELECT public_parent.id FROM issues public_parent WHERE public_parent.id=` + issue + `.parent_id AND ` + PublicRelatedIssueSQL("public_parent", issue) + `)`
}

// MaskIssueParent returns a separate response row. The original is retained for
// validation and activity snapshots, including relationships the reader lost access to.
func MaskIssueParent(ctx context.Context, db Database, row map[string]any, actor string) (map[string]any, error) {
	if row["parent_id"] == nil {
		return row, nil
	}
	var visible bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues related WHERE related.id=$1 AND `+IssueVisibleSQL("related", "NULLIF($2,'')::uuid", "$3::uuid")+`)`, row["parent_id"], actor, row["workspace_id"]).Scan(&visible)
	if err != nil || visible {
		return row, err
	}
	result := maps.Clone(row)
	result["parent_id"] = nil
	return result, nil
}
