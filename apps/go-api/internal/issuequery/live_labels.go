package issuequery

import "fmt"

// LiveIssueLabelPredicate matches an active label bridge to an active label
// owned by the issue's project and workspace. Aliases are internal SQL names.
func LiveIssueLabelPredicate(issueAlias, relationAlias string) string {
	return liveLabelPredicate(issueAlias, relationAlias, "issue_id", "=")
}

// LiveDraftIssueLabelPredicate preserves draft ownership and nullable project
// scope; a projectless draft can only retain a projectless workspace label.
func LiveDraftIssueLabelPredicate(draftAlias, relationAlias string) string {
	return liveLabelPredicate(draftAlias, relationAlias, "draft_issue_id", "IS NOT DISTINCT FROM")
}

func liveLabelPredicate(ownerAlias, relationAlias, ownerColumn, projectComparison string) string {
	return fmt.Sprintf(`%[2]s.%[3]s = %[1]s.id AND %[2]s.deleted_at IS NULL AND EXISTS (
  SELECT 1 FROM labels live_label
  WHERE live_label.id = %[2]s.label_id
    AND live_label.project_id %[4]s %[1]s.project_id
    AND live_label.workspace_id = %[1]s.workspace_id
    AND live_label.deleted_at IS NULL
)`, ownerAlias, relationAlias, ownerColumn, projectComparison)
}
