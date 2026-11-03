package workitems

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanningRecordsRejectMalformedActivityData(t *testing.T) {
	for name, row := range map[string]map[string]any{
		"timestamp": {"id": "record", "created_at": "not-a-timestamp"},
		"identity":  {"id": json.RawMessage(`{"broken":`)},
	} {
		t.Run(name, func(t *testing.T) {
			value, err := serializePlanningRecords("cycle", []map[string]any{row})
			require.Error(t, err)
			require.Empty(t, value, "invalid activity data must not be admitted as a valid snapshot")
		})
	}
}
