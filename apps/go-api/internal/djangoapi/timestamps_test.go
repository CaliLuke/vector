package djangoapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONTimestampsPreserveEmptyResults(t *testing.T) {
	result, err := JSONTimestamps(nil, "Asia/Kolkata")
	require.NoError(t, err)
	require.Nil(t, result, "an absent result must not become a JSON null body")
}

func TestJSONTimestampsRejectMalformedResults(t *testing.T) {
	result, err := JSONTimestamps(json.RawMessage(`{"created_at":`), "Asia/Kolkata")
	require.Error(t, err)
	require.Nil(t, result)
}

func TestJSONTimestampsKeepOpaqueJSONAndNumberPrecision(t *testing.T) {
	result, err := JSONTimestamps(json.RawMessage(`{"created_at":"2024-01-01T00:00:00.123456Z","count":9007199254740993,"logo_props":{"created_at":"2024-01-01T00:00:00Z"}}`), "Asia/Kolkata")
	require.NoError(t, err)
	require.JSONEq(t, `{"created_at":"2024-01-01T05:30:00.123456+05:30","count":9007199254740993,"logo_props":{"created_at":"2024-01-01T00:00:00Z"}}`, string(result))
	require.Contains(t, string(result), `"count":9007199254740993`, "decoding must retain integers above float64's exact range")
}

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

func TestTimestampsPreserveAttachmentJSON(t *testing.T) {
	for _, field := range []string{"attributes", "storage_metadata"} {
		t.Run(field, func(t *testing.T) {
			opaque := []any{map[string]any{"created_at": "2024-01-01T00:00:00Z", "nested": map[string]any{"updated_at": "2024-02-01T00:00:00Z"}}}
			before, err := json.Marshal(opaque)
			require.NoError(t, err)
			body := map[string]any{"created_at": "2024-01-01T00:00:00Z", field: opaque}
			result := Timestamps(body, "Asia/Kolkata").(map[string]any)
			require.Equal(t, "2024-01-01T05:30:00+05:30", result["created_at"])
			after, err := json.Marshal(result[field])
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}
