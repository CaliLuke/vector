package workitems

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/requestinput"
	"github.com/stretchr/testify/require"
)

func TestIssueProjectionsRejectInvalidSortOrderWithoutInventingZero(t *testing.T) {
	for _, token := range []string{"", "invalid", "1e999", "NaN", "0x1p0"} {
		t.Run(token, func(t *testing.T) {
			row := map[string]any{"id": "owned-item", "sort_order": json.Number(token)}
			projected, err := projectFields(row, []string{"id", "sort_order"})
			require.Error(t, err)
			require.Nil(t, projected)
			projected, err = appWriteProjection(row, requestinput.Input{Fields: map[string]any{}})
			require.Error(t, err)
			require.Nil(t, projected)
			require.Equal(t, json.Number(token), row["sort_order"])
		})
	}
}

func TestIssueProjectionsKeepNativeNumericValues(t *testing.T) {
	for _, sample := range []struct{ token, want string }{
		{"1", "1"},
		{"-0", "-0"},
		{"1e-5", "1e-5"},
		{"9007199254740993.0", "9007199254740993.0"},
		{"0.0001", "0.0001"},
	} {
		t.Run(sample.token, func(t *testing.T) {
			row := map[string]any{"sort_order": json.Number(sample.token)}
			projected, err := projectFields(row, []string{"sort_order"})
			require.NoError(t, err)
			require.Equal(t, json.Number(sample.want), projected["sort_order"])
			projected, err = appWriteProjection(row, requestinput.Input{Fields: map[string]any{}})
			require.NoError(t, err)
			require.Equal(t, json.Number(sample.want), projected["sort_order"])
		})
	}
}
