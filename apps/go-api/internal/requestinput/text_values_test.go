package requestinput

import (
	"encoding/json"
	"math"
	"testing"
)

func TestTextUsesNativeJSONAndPreservesExactNumbers(t *testing.T) {
	for _, test := range []struct {
		value any
		want  string
	}{
		{"<p>Zoë</p>", "<p>Zoë</p>"},
		{json.Number("9007199254740993"), "9007199254740993"},
		{nil, "null"}, {false, "false"},
		{[]any{"a", true, nil}, `["a",true,null]`},
		{map[string]any{"z": json.Number("9007199254740993"), "a": false}, `{"a":false,"z":9007199254740993}`},
	} {
		result, err := Text(test.value)
		if err != nil || result != test.want {
			t.Fatalf("Text(%#v)=%q, error=%v; want %q", test.value, result, err, test.want)
		}
	}
	for _, value := range []any{string([]byte{0xff}), math.NaN(), math.Inf(1), UploadedFile{Name: "upload"}} {
		if result, err := Text(value); err == nil {
			t.Fatalf("unsupported text value %#v became %q", value, result)
		}
	}
}

func TestNonemptyKeepsOptionalValuesForValidation(t *testing.T) {
	for _, value := range []any{nil, false, "", json.Number("-0.0e20"), float64(0), int(0), int64(0), []any{}, map[string]any{}} {
		if Nonempty(value) {
			t.Fatalf("empty value %#v was retained", value)
		}
	}
	for _, value := range []any{true, "false", json.Number("1e-1000"), float64(1), int(-1), int64(-1), []any{nil}, map[string]any{"x": nil}, UploadedFile{Name: "upload"}} {
		if !Nonempty(value) {
			t.Fatalf("value %#v silently disappeared", value)
		}
	}
}

func TestTextFieldRejectsInvalidUTF8Values(t *testing.T) {
	_, failures := TextField(string([]byte{0xff}), false, 255)
	if len(failures) != 1 || failures[0] != "Not a valid UTF-8 string." {
		t.Fatalf("invalid text failures=%v", failures)
	}
}
