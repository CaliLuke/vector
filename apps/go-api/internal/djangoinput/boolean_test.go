package djangoinput

import (
	"encoding/json"
	"testing"
)

func TestBooleanFieldDRFVocabulary(t *testing.T) {
	for _, test := range []struct {
		input any
		want  bool
	}{{true, true}, {false, false}, {"ON", true}, {"yes", true}, {"F", false}, {"off", false}, {json.Number("1.00"), true}, {json.Number("0e10"), false}} {
		got, message := BooleanField(test.input)
		if got != test.want || message != "" {
			t.Errorf("input=%#v got=%v message=%q", test.input, got, message)
		}
	}
	for _, value := range []any{" true ", "", json.Number("2"), []any{}, map[string]any{}} {
		if _, message := BooleanField(value); message != "Must be a valid boolean." {
			t.Errorf("input=%#v message=%q", value, message)
		}
	}
	if _, message := BooleanField(nil); message != "This field may not be null." {
		t.Errorf("null message=%q", message)
	}
}
