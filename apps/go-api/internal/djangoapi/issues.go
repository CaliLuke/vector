package djangoapi

import (
	"strings"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func IssueOrderSQL(raw string) (string, string) {
	fallback := "created_at"
	name := strings.TrimPrefix(raw, "-")
	allowed := map[string]string{"created_at": "i.created_at", "updated_at": "i.updated_at", "sequence_id": "i.sequence_id", "sort_order": "i.sort_order", "target_date": "i.target_date", "start_date": "i.start_date", "completed_at": "i.completed_at", "archived_at": "i.archived_at", "priority": "i.priority", "state__name": "ordering_state.name", "state__group": "ordering_state.\"group\"", "assignees__first_name": "ordering_user.first_name", "labels__name": "ordering_label.name", "issue_module__module__name": "ordering_module.name"}
	joins := ""
	if _, ok := allowed[name]; !ok {
		raw = fallback
		name = strings.TrimPrefix(raw, "-")
	}
	direction := " ASC"
	if strings.HasPrefix(raw, "-") {
		direction = " DESC"
	}
	switch name {
	case "state__name", "state__group":
		joins = " LEFT JOIN states ordering_state ON ordering_state.id=i.state_id"
	case "assignees__first_name":
		joins = " LEFT JOIN issue_assignees ordering_assignee ON ordering_assignee.issue_id=i.id LEFT JOIN users ordering_user ON ordering_user.id=ordering_assignee.assignee_id"
	case "labels__name":
		joins = " LEFT JOIN issue_labels ordering_link ON " + issuequery.LiveIssueLabelPredicate("i", "ordering_link") + " LEFT JOIN labels ordering_label ON ordering_label.id=ordering_link.label_id"
	case "issue_module__module__name":
		joins = " LEFT JOIN module_issues ordering_link ON ordering_link.issue_id=i.id LEFT JOIN modules ordering_module ON ordering_module.id=ordering_link.module_id"
	}
	return allowed[name] + direction, joins
}

func IssueRowSQL(alias string) string {
	return `to_jsonb(` + alias + `)||jsonb_build_object('description_binary',replace(encode(` + alias + `.description_binary,'base64'),chr(10),''),'assignees',COALESCE((SELECT jsonb_agg(ia.assignee_id ORDER BY ia.created_at DESC) FROM issue_assignees ia WHERE ia.issue_id=` + alias + `.id AND ia.deleted_at IS NULL),'[]'::jsonb),'labels',COALESCE((SELECT jsonb_agg(il.label_id ORDER BY il.created_at DESC) FROM issue_labels il WHERE ` + issuequery.LiveIssueLabelPredicate(alias, "il") + `),'[]'::jsonb))`
}
