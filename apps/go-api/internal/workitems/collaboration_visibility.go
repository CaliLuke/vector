package workitems

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

// The parent and its child collection are authorized in one PostgreSQL snapshot.
// Comment-reaction routes resolve their item through the path's comment ID.
func collaborationVisibleParentSQL(q Request, kind string, args *[]any) string {
	anchor := q.IssueID
	if kind == "commentReaction" {
		anchor = q.Params["comment_id"]
	}
	values := []any{q.ActorID, q.Slug, q.ProjectID, anchor}
	parameters := make([]string, len(values))
	for index, value := range values {
		*args = append(*args, value)
		parameters[index] = "$" + strconv.Itoa(len(*args))
	}
	actor, slug, project, item := parameters[0], parameters[1], parameters[2], parameters[3]
	from := `issues visible_issue JOIN workspaces visible_workspace ON visible_workspace.id=visible_issue.workspace_id`
	anchorExpression := `visible_issue.id`
	if kind == "commentReaction" {
		from += ` JOIN issue_comments visible_comment ON visible_comment.issue_id=visible_issue.id
            AND visible_comment.project_id=visible_issue.project_id AND visible_comment.workspace_id=visible_issue.workspace_id
            AND visible_comment.deleted_at IS NULL`
		anchorExpression = `visible_comment.id`
	}
	return `SELECT visible_issue.id,visible_issue.workspace_id,visible_issue.project_id,` + anchorExpression + ` AS anchor_id
        FROM ` + from + ` WHERE visible_workspace.slug=` + slug + ` AND visible_issue.project_id=` + project + `::uuid
        AND ` + anchorExpression + `=` + item + `::uuid AND ` + djangoapi.IssueVisibleSQL("visible_issue", actor+"::uuid", "visible_workspace.id")
}

func (r *Repository) visibleCollaborationRows(ctx context.Context, q Request, kind, statement string, args ...any) ([]map[string]any, error) {
	if collaborationPublic(q) {
		return r.rows(ctx, statement, args...)
	}
	parent := collaborationVisibleParentSQL(q, kind, &args)
	payload, err := r.row(ctx, `WITH visible_parent AS MATERIALIZED (`+parent+`)
        SELECT jsonb_build_object('visible',EXISTS(SELECT 1 FROM visible_parent),'rows',
        COALESCE((SELECT jsonb_agg(record) FROM (`+statement+`) selected(record)
        WHERE EXISTS(SELECT 1 FROM visible_parent)),'[]'::jsonb))`, args...)
	if err != nil {
		return nil, err
	}
	if payload["visible"] != true {
		return nil, collaborationMissing(q)
	}
	values, ok := payload["rows"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid collaboration collection %T", payload["rows"])
	}
	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		row, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid collaboration row %T", value)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func collaborationWriteVisibility(q Request, table string, args *[]any) string {
	if table == "descriptions" || table == "project_public_members" {
		return "TRUE"
	}
	if collaborationPublic(q) {
		if table == "issue_comments" {
			return collaborationCommentParentWriteSQL(q, "v", args)
		}
		return "TRUE"
	}
	kind, parentColumn := "issue", "issue_id"
	if table == "comment_reactions" {
		kind, parentColumn = "commentReaction", "comment_id"
	}
	parent := collaborationVisibleParentSQL(q, kind, args)
	visibility := `EXISTS(SELECT 1 FROM (` + parent + `) permitted_parent
        WHERE permitted_parent.anchor_id=v.` + parentColumn + `
        AND permitted_parent.project_id=v.project_id AND permitted_parent.workspace_id=v.workspace_id)`
	if table == "issue_comments" {
		visibility += ` AND ` + collaborationCommentParentWriteSQL(q, "v", args)
	}
	return visibility
}

func collaborationCommentParentWriteSQL(q Request, row string, args *[]any) string {
	*args = append(*args, q.Slug)
	slug := "$" + strconv.Itoa(len(*args))
	predicate := `(` + row + `.parent_id IS NULL OR EXISTS(SELECT 1 FROM issue_comments parent
JOIN issues parent_issue ON parent_issue.id=parent.issue_id AND parent_issue.project_id=parent.project_id AND parent_issue.workspace_id=parent.workspace_id
JOIN workspaces parent_workspace ON parent_workspace.id=parent.workspace_id
WHERE parent.id=` + row + `.parent_id AND parent.id<>` + row + `.id AND parent.issue_id=` + row + `.issue_id
AND parent.project_id=` + row + `.project_id AND parent.workspace_id=` + row + `.workspace_id
AND parent_workspace.slug=` + slug + ` AND parent.deleted_at IS NULL AND `
	if collaborationPublic(q) {
		return predicate + `parent.access='EXTERNAL' AND ` + djangoapi.PublicRelatedIssueSQL("parent_issue", "parent_issue") + `))`
	}
	*args = append(*args, q.ActorID)
	actor := "$" + strconv.Itoa(len(*args)) + "::uuid"
	return predicate + djangoapi.IssueVisibleSQL("parent_issue", actor, "parent_workspace.id") + `))`
}

// Reject hidden parents before serializer/duplicate checks can disclose child
// existence. The write statement independently rechecks visibility at mutation.
func (r *Repository) authorizeCollaborationParent(ctx context.Context, q Request, kind string) error {
	if collaborationPublic(q) {
		return nil
	}
	args := []any{}
	parent := collaborationVisibleParentSQL(q, kind, &args)
	var visible bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(`+parent+`)`, args...).Scan(&visible); err != nil {
		return err
	}
	if !visible {
		return collaborationMissing(q)
	}
	return nil
}
