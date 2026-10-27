package djangoapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimestampsConvertModeledFieldsAndPreserveOpaqueJSON(t *testing.T) {
	body := map[string]any{"results": []any{map[string]any{
		"created_at": "2024-01-01T00:00:00Z", "end_date": "2024-01-01T01:00:00.123456+00:00",
		"start_date": "not a timestamp", "deleted_at": nil, "updated_at": "",
		"progress_snapshot": map[string]any{"created_at": "2024-01-01T00:00:00Z"},
		"owner":             map[string]any{"created_at": "2024-01-01T00:00:00Z"},
	}}}
	require.Equal(t, map[string]any{"results": []any{map[string]any{
		"created_at": "2024-01-01T05:30:00+05:30", "end_date": "2024-01-01T06:30:00.123456+05:30",
		"start_date": "not a timestamp", "deleted_at": nil, "updated_at": "",
		"progress_snapshot": map[string]any{"created_at": "2024-01-01T00:00:00Z"},
		"owner":             map[string]any{"created_at": "2024-01-01T05:30:00+05:30"},
	}}}, Timestamps(body, "Asia/Kolkata"))
}
func TestTimestampsDefaultOrUnknownTimezoneUsesUTC(t *testing.T) {
	for _, zone := range []string{"", "Invalid/Timezone"} {
		body := []map[string]any{{"created_at": "2024-01-01T01:00:00+01:00"}}
		require.Equal(t, []map[string]any{{"created_at": "2024-01-01T00:00:00Z"}}, Timestamps(body, zone))
	}
	require.Equal(t, "unchanged", Timestamps("unchanged", "UTC"))
}
