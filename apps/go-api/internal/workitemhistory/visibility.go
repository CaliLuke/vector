package workitemhistory

import "github.com/CaliLuke/go-plane/internal/djangoapi"

// Guard the result in its own SQL statement: an earlier project permission read
// must not authorize values after an item's visibility or membership changes.
// Expressions here are internal positional parameters, never request input.
func visibleItemHistorySQL(query, item, slug, actor string) string {
	return `SELECT history_result.data FROM (` + query + `) AS history_result(data)
 WHERE EXISTS(SELECT 1 FROM issues history_item JOIN workspaces history_workspace ON history_workspace.id=history_item.workspace_id
 WHERE history_item.id=` + item + ` AND history_item.project_id=$2 AND history_workspace.slug=` + slug + ` AND ` + djangoapi.IssueVisibleSQL("history_item", actor, "history_workspace.id") + `)`
}
