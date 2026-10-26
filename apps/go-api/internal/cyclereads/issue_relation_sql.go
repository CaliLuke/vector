package cyclereads

import "github.com/CaliLuke/go-plane/internal/issuequery"

// Preserve each relation's existing bridge policy while labels also require a
// live target in the issue's project and workspace.
func issueRelationPredicate(table, alias string, softDelete bool) string {
	if table == "issue_labels" {
		return issuequery.LiveIssueLabelPredicate("issue", alias)
	}
	predicate := alias + ".issue_id=issue.id"
	if softDelete {
		predicate += " AND " + alias + ".deleted_at IS NULL"
	}
	return predicate
}
