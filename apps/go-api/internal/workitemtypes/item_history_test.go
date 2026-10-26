package workitemtypes

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestHistoryValueEqualityPreservesMeaning(t *testing.T) {
	for _, test := range []struct {
		name          string
		kind          property.Kind
		multiple      bool
		before, after any
		equal         bool
	}{
		{"different numeric Go forms", property.Number, false, json.Number("7.0"), float64(7), true},
		{"zero is not empty", property.Number, false, float64(0), nil, false},
		{"false is not empty", property.Boolean, false, false, nil, false},
		{"empty text is empty", property.Text, false, "", nil, true},
		{"empty selection is empty", property.Dropdown, true, []any{}, nil, true},
		{"multiple selection order is immaterial", property.Dropdown, true, []any{"b", "a"}, []string{"a", "b"}, true},
		{"cascading order remains meaningful", property.Cascading, false, []any{"b", "a"}, []string{"a", "b"}, false},
		{"rich text formatting is meaningful", property.RichText, false, "<p>Text</p>", "<p><strong>Text</strong></p>", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			def := property.Definition{Kind: test.kind, Configuration: property.Configuration{Multiple: test.multiple}}
			equal, err := equalHistoryValues(def, test.before, test.after)
			if err != nil || equal != test.equal {
				t.Fatalf("equal=%v, error=%v", equal, err)
			}
		})
	}
}
