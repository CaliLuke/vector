package djangoapi

import (
	"strings"
	"time"
)

// Timestamps renders modeled API timestamps in the user timezone, preserving
// microsecond precision and leaving arbitrary JSON fields untouched.
func Timestamps(value any, timezone string) any {
	location := time.UTC
	if timezone != "" {
		if parsed, err := time.LoadLocation(timezone); err == nil {
			location = parsed
		}
	}
	timestampFields := [...]string{
		"created_at", "updated_at", "deleted_at", "archived_at", "completed_at", "start_date", "end_date", "target_date", "snoozed_till",
	}
	convertObject := func(item map[string]any) {
		for _, field := range timestampFields {
			raw, ok := item[field].(string)
			if !ok || raw == "" {
				continue
			}
			timestamp, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				continue
			}
			layout := time.RFC3339
			if strings.Contains(strings.SplitN(raw, "Z", 2)[0], ".") {
				layout = "2006-01-02T15:04:05.000000Z07:00"
			}
			item[field] = timestamp.In(location).Format(layout)
		}
	}
	var convertContainer func(any)
	convertContainer = func(container any) {
		switch typed := container.(type) {
		case []map[string]any:
			for _, raw := range typed {
				convertContainer(raw)
			}
		case []any:
			for _, raw := range typed {
				convertContainer(raw)
			}
		case map[string]any:
			convertObject(typed)
			for key, raw := range typed {
				// JSON fields are not serializer objects. Django leaves timestamp-looking
				// values inside them unchanged.
				switch key {
				case "comment_json", "description_json", "description_text", "description_html", "icon_prop", "logo_props", "meta", "metadata", "source_data", "view_props", "progress_snapshot", "template_data":
					continue
				}
				convertContainer(raw)
			}
		}
	}
	convertContainer(value)
	return value
}
