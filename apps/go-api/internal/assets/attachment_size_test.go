package assets

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttachmentSizeUsesOnlyFiniteNonzeroNumericMetadata(t *testing.T) {
	for _, test := range []struct {
		name       string
		size, want any
	}{
		{"missing", nil, json.Number("37.5")}, {"zero", json.Number("0"), json.Number("37.5")},
		{"decimal zero", json.Number("0.0"), json.Number("37.5")}, {"exponent zero", json.Number("0e3"), json.Number("37.5")},
		{"fractional", json.Number("12.5"), json.Number("12.5")}, {"negative finite", json.Number("-1.25"), json.Number("-1.25")},
		{"float", 12.5, 12.5}, {"float zero", float64(0), json.Number("37.5")},
		{"NaN", math.NaN(), json.Number("37.5")}, {"infinity", math.Inf(1), json.Number("37.5")}, {"negative infinity", math.Inf(-1), json.Number("37.5")},
		{"overflow", json.Number("1e309"), json.Number("37.5")}, {"invalid number", json.Number("NaN"), json.Number("37.5")},
		{"numeric string", "12.5", json.Number("37.5")}, {"string infinity", "Infinity", json.Number("37.5")},
		{"boolean", true, json.Number("37.5")}, {"object", map[string]any{"value": 12.5}, json.Number("37.5")}, {"array", []any{12.5}, json.Number("37.5")},
	} {
		t.Run(test.name, func(t *testing.T) {
			attributes := map[string]any{"name": "notes.txt", "type": "text/plain", "size": test.size}
			row := map[string]any{"attributes": attributes, "size": json.Number("37.5")}
			normalizeAttachment(row)
			require.Equal(t, test.want, row["size"])
			require.Equal(t, "notes.txt", row["name"])
			require.Equal(t, "text/plain", row["content_type"])
		})
	}
}
