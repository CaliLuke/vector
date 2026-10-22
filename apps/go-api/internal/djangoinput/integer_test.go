package djangoinput

import (
	"encoding/json"
	"testing"
)

func TestIntegerFieldSourceCoercionVectors(t *testing.T) {
	for _, input := range []any{"1.0", "1.000  ", json.Number("1.00"), json.Number("1e0"), "١.0", "+1", "1_0"} {
		got, err := IntegerField(input)
		if err != nil || (got.String() != "1" && got.String() != "10") {
			t.Errorf("IntegerField(%#v)=%v,%v", input, got, err)
		}
	}
	for _, input := range []any{true, nil, "1.1", json.Number("1.1"), "1e0", []any{1}} {
		if got, err := IntegerField(input); err == nil {
			t.Errorf("IntegerField(%#v) unexpectedly accepted %v", input, got)
		}
	}
}
