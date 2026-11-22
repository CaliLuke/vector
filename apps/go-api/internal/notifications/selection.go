package notifications

import (
	"net/url"
	"strings"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

// Selection is the same recipient-scoped notification view for listing and
// bulk actions. Its zero value selects the ordinary inbox.
type Selection struct {
	snoozed, archived, mentioned  bool
	read                          *bool
	assigned, subscribed, created bool
}

func listSelection(query url.Values) (Selection, error) {
	snoozed, archived := last(query, "snoozed", "false"), last(query, "archived", "false")
	if (snoozed != "true" && snoozed != "false") || (archived != "true" && archived != "false") {
		return Selection{}, Failure(400, "error", "The required key does not exist.")
	}
	selection := Selection{
		snoozed:  snoozed == "true",
		archived: archived == "true",
		// The existing list endpoint treats any nonempty value as present.
		mentioned: last(query, "mentioned", "") != "",
	}
	switch last(query, "read", "") {
	case "true", "false":
		read := last(query, "read", "") == "true"
		selection.read = &read
	}
	err := selection.setTypes(last(query, "type", ""))
	return selection, err
}

// ParseBulkSelection validates fields before a mutation can select any rows.
func ParseBulkSelection(fields map[string]any) (Selection, error) {
	var selection Selection
	for key, value := range fields {
		switch key {
		case "snoozed", "archived", "mentioned", "read":
			enabled, message := requestinput.BooleanField(value)
			if message != "" {
				return Selection{}, Failure(400, "error", key+": "+message)
			}
			switch key {
			case "snoozed":
				selection.snoozed = enabled
			case "archived":
				selection.archived = enabled
			case "mentioned":
				selection.mentioned = enabled
			case "read":
				selection.read = &enabled
			}
		case "type":
			types, ok := value.(string)
			if !ok || types == "" {
				return Selection{}, Failure(400, "error", "type must select assigned, subscribed, created, or all.")
			}
			if err := selection.setTypes(types); err != nil {
				return Selection{}, err
			}
		default:
			return Selection{}, Failure(400, "error", "Unknown notification selection field.")
		}
	}
	return selection, nil
}

func (selection *Selection) setTypes(value string) error {
	switch value {
	case "", "all":
		return nil
	}
	for kind := range strings.SplitSeq(value, ",") {
		switch kind {
		case "assigned":
			selection.assigned = true
		case "subscribed", "watching": // Watching is the legacy bulk name.
			selection.subscribed = true
		case "created":
			selection.created = true
		default:
			return Failure(400, "error", "type must select assigned, subscribed, created, or all.")
		}
	}
	return nil
}

const assigned = `n.entity_identifier IN(SELECT ia.issue_id FROM issue_assignees ia WHERE ia.deleted_at IS NULL AND ia.workspace_id=n.workspace_id AND ia.assignee_id=$1)`
const subscribed = `n.entity_identifier IN(SELECT s.issue_id FROM issue_subscribers s WHERE s.deleted_at IS NULL AND s.workspace_id=n.workspace_id AND s.subscriber_id=$1 AND NOT EXISTS(SELECT 1 FROM issues i WHERE i.deleted_at IS NULL AND i.id=s.issue_id AND i.created_by_id=$1) AND NOT EXISTS(SELECT 1 FROM issue_assignees ia WHERE ia.deleted_at IS NULL AND ia.id=s.issue_id AND ia.assignee_id=$1))`
const created = `n.entity_identifier IN(SELECT i.id FROM issues i WHERE i.deleted_at IS NULL AND i.workspace_id=n.workspace_id AND i.created_by_id=$1)`
const notGuest = `NOT EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=n.workspace_id AND wm.member_id=$1 AND wm.deleted_at IS NULL AND wm.is_active AND wm.role<15)`

func (selection Selection) where() string {
	filters := []string{scope, "n.entity_name='issue'"}
	switch {
	case selection.snoozed:
		filters = append(filters, "n.snoozed_till IS NOT NULL")
	case selection.archived:
		filters = append(filters, "(n.snoozed_till IS NULL OR n.snoozed_till>=CURRENT_TIMESTAMP)")
	default:
		filters = append(filters, "n.snoozed_till IS NULL")
	}
	if selection.archived {
		filters = append(filters, "n.archived_at IS NOT NULL")
	} else {
		filters = append(filters, "n.archived_at IS NULL")
	}
	if selection.read != nil {
		if *selection.read {
			filters = append(filters, "n.read_at IS NOT NULL")
		} else {
			filters = append(filters, "n.read_at IS NULL")
		}
	}
	mentioned := "UPPER(n.sender) LIKE '%MENTIONED%'"
	if !selection.mentioned {
		mentioned = "NOT (" + mentioned + ")"
	}
	filters = append(filters, mentioned)
	var types []string
	if selection.subscribed {
		types = append(types, subscribed)
	}
	if selection.assigned {
		types = append(types, assigned)
	}
	if selection.created {
		filters = append(filters, notGuest)
		types = append(types, created)
	}
	if len(types) > 0 {
		filters = append(filters, "("+strings.Join(types, " OR ")+")")
	}
	return strings.Join(filters, " AND ")
}
