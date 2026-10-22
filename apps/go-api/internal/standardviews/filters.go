package standardviews

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// normalizeFilters is the Go equivalent of plane.utils.issue_filters for the
// JSON filter object stored on IssueView. The API keeps the user-facing
// filter names in filters, but query is the ORM-shaped object used by issue
// queries.
func normalizeFilters(raw json.RawMessage, method string) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`{}`), nil
	}
	var params map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&params); err != nil {
		return nil, err
	}
	result := make(map[string]any)
	// Keep the same ordering as ISSUE_FILTER in Django. In particular, the
	// `type` filter intentionally overrides an explicitly supplied state_group.
	for _, key := range []string{
		"state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees", "mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "cycle", "module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date",
	} {
		value, exists := params[key]
		if !exists {
			continue
		}
		if value == nil || emptyFilter(value) {
			continue
		}
		// Django's POST implementation of filter_intake_status currently reads
		// inbox_status by mistake. Preserve that wire-level behavior.
		if key == "intake_status" && method != "GET" {
			value = params["inbox_status"]
		}
		addFilter(result, key, value, method)
	}
	return json.Marshal(result)
}

func emptyFilter(value any) bool {
	s, ok := value.(string)
	return ok && (s == "" || s == "null")
}

func addFilter(out map[string]any, key string, value any, method string) {
	switch key {
	case "state":
		out["state__in"] = value
	case "state_group", "state__group__in":
		out["state__group__in"] = value
	case "estimate_point":
		out["estimate_point__in"] = value
	case "priority":
		out["priority__in"] = value
	case "parent":
		if method == "GET" {
			addNullableIDFilter(out, "parent", value)
		} else {
			out["parent__in"] = value
		}
	case "labels":
		if method == "GET" {
			addNullableIDFilter(out, "labels", value)
		} else {
			out["labels__in"] = value
		}
		out["label_issue__deleted_at__isnull"] = true
	case "assignees":
		if method == "GET" {
			addNullableIDFilter(out, "assignees", value)
		} else {
			out["assignees__in"] = value
		}
		out["issue_assignee__deleted_at__isnull"] = true
	case "mentions":
		out["issue_mention__mention__id__in"] = value
	case "created_by":
		if method == "GET" {
			addNullableIDFilter(out, "created_by", value)
		} else {
			out["created_by__in"] = value
		}
	case "logged_by":
		if method == "GET" {
			addNullableIDFilter(out, "logged_by", value)
		} else {
			out["logged_by__in"] = value
		}
	case "name":
		out["name__icontains"] = value
	case "created_at", "updated_at", "completed_at":
		addDateFilter(out, key+"__date", value)
	case "start_date", "target_date":
		if method == "GET" {
			addDateFilter(out, key, value)
		} else {
			out[key] = value
		}
	case "type":
		groups := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
		if s, ok := value.(string); ok {
			switch s {
			case "backlog":
				groups = []string{"backlog"}
			case "active":
				groups = []string{"unstarted", "started"}
			}
		}
		out["state__group__in"] = groups
	case "project":
		out["project__in"] = value
	case "cycle":
		if method == "GET" {
			addNullableRelationFilter(out, "issue_cycle__cycle_id", value)
		} else {
			out["issue_cycle__cycle_id__in"] = value
		}
		out["issue_cycle__deleted_at__isnull"] = true
	case "module":
		if method == "GET" {
			addNullableRelationFilter(out, "issue_module__module_id", value)
		} else {
			out["issue_module__module_id__in"] = value
		}
		out["issue_module__deleted_at__isnull"] = true
	case "intake_status", "inbox_status":
		out["issue_intake__status__in"] = value
	case "sub_issue":
		if s, ok := value.(string); ok && s == "false" {
			out["parent__isnull"] = true
		}
	case "subscriber":
		out["issue_subscribers__subscriber_id__in"] = value
		out["issue_subscribers__deleted_at__isnull"] = true
	case "start_target_date":
		if s, ok := value.(string); ok && s == "true" {
			out["target_date__isnull"] = false
			out["start_date__isnull"] = false
		}
	default:
		// Unknown filter keys are ignored by issue_filters.
	}
}

func addNullableIDFilter(out map[string]any, field string, value any) {
	if containsNone(value) {
		out[field+"__isnull"] = true
	}
	if filtered := withoutNone(value); filtered != nil {
		out[field+"__in"] = filtered
	}
}

func addNullableRelationFilter(out map[string]any, field string, value any) {
	if containsNone(value) {
		out[field+"__isnull"] = true
	}
	if filtered := withoutNone(value); filtered != nil {
		out[field+"__in"] = filtered
	}
}

func containsNone(value any) bool {
	for _, item := range filterItems(value) {
		if item == "None" {
			return true
		}
	}
	return false
}

func withoutNone(value any) any {
	items := filterItems(value)
	if len(items) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item != "None" && item != "null" && item != "" {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterItems(value any) []string {
	switch v := value.(type) {
	case string:
		return strings.Split(v, ",")
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	case []string:
		return v
	default:
		return nil
	}
}

func addDateFilter(out map[string]any, field string, value any) {
	for _, query := range filterItems(value) {
		parts := strings.Split(query, ";")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		if len(parts) == 1 {
			out[field+"__contains"] = parts[0]
			continue
		}
		if len(parts) == 3 {
			if duration, unit, ok := relativeDate(parts[0]); ok {
				delta := 24 * time.Hour * time.Duration(duration)
				if unit == "weeks" {
					delta *= 7
				} else {
					delta *= 30
				}
				date := time.Now().UTC().Add(delta)
				if parts[2] != "fromnow" {
					date = time.Now().UTC().Add(-delta)
				}
				if parts[1] == "after" {
					out[field+"__gte"] = date.Format("2006-01-02")
				} else {
					out[field+"__lte"] = date.Format("2006-01-02")
				}
				continue
			}
		}
		if strings.Contains(query, "after") {
			out[field+"__gte"] = parts[0]
		} else {
			out[field+"__lte"] = parts[0]
		}
	}
}

func relativeDate(value string) (int, string, bool) {
	parts := strings.Split(value, "_")
	if len(parts) != 2 || (parts[1] != "weeks" && parts[1] != "months") {
		return 0, "", false
	}
	var duration int
	if _, err := fmt.Sscanf(parts[0], "%d", &duration); err != nil || duration < 0 {
		return 0, "", false
	}
	return duration, parts[1], true
}
