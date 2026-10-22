package workitems

import "time"

// Serializer timestamps use the request timezone; arbitrary JSON fields do not.
func timestampProjection(value any, zone string) any {
	location, _ := time.LoadLocation(zone)
	if location == nil {
		location = time.UTC
	}
	switch typed := value.(type) {
	case []any:
		for _, v := range typed {
			timestampProjection(v, zone)
		}
	case []map[string]any:
		for _, v := range typed {
			timestampProjection(v, zone)
		}
	case map[string]any:
		for _, key := range []string{"created_at", "updated_at", "deleted_at", "completed_at", "snoozed_till"} {
			if raw, ok := typed[key].(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
					format := time.RFC3339
					if t.Nanosecond() != 0 {
						format = "2006-01-02T15:04:05.000000Z07:00"
					}
					typed[key] = t.In(location).Format(format)
				}
			}
		}
		for key, v := range typed {
			switch key {
			case "description_json", "description_html", "icon_prop", "logo_props", "meta", "metadata", "source_data", "view_props", "progress_snapshot":
				continue
			}
			timestampProjection(v, zone)
		}
	}
	return value
}
