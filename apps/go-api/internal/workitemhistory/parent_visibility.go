package workitemhistory

import "github.com/CaliLuke/go-plane/internal/djangoapi"

// Activity values contain historical names, so masking only the identifier would
// still reveal a parent in a project the reader cannot access today.
func parentActivityVisibilitySQL() string {
	visible := func(side string) string {
		return `(COALESCE(a.` + side + `_value,'')='' AND COALESCE(a.` + side + `_identifier::text,'')='' OR EXISTS(SELECT 1 FROM issues related WHERE related.id::text=a.` + side + `_identifier::text AND ` + djangoapi.IssueVisibleSQL("related", "$5::uuid", "a.workspace_id") + `))`
	}
	old, next := visible("old"), visible("new")
	return `||CASE WHEN a.field IN ('parent','parent_id') THEN jsonb_build_object(
 'old_identifier',CASE WHEN ` + old + ` THEN a.old_identifier ELSE NULL END,
 'new_identifier',CASE WHEN ` + next + ` THEN a.new_identifier ELSE NULL END,
 'old_value',CASE WHEN ` + old + ` THEN a.old_value ELSE 'Restricted work item' END,
 'new_value',CASE WHEN ` + next + ` THEN a.new_value ELSE 'Restricted work item' END,
 'comment',CASE WHEN ` + old + ` AND ` + next + ` THEN a.comment ELSE 'updated the parent link' END
 ) ELSE '{}'::jsonb END`
}
