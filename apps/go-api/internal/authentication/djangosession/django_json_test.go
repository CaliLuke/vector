package djangosession

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestDjangoJSONCanonicalizesLikePython(t *testing.T) {
	input := []byte(`{ "x" : 1, "x": 2, "nan": NaN, "positive": Infinity, "negative": -Infinity, "text": "<>&\u2028\ud800", "pair": "\ud83d\ude00", "floats": [1.0, 1e15, 1e16, 1e-4, 1e-5] }`)
	values, canonical, err := decodeDjangoJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"x":2,"nan":NaN,"positive":Infinity,"negative":-Infinity,"text":"<>&\u2028\ud800","pair":"\ud83d\ude00","floats":[1.0,1000000000000000.0,1e+16,0.0001,1e-05]}`
	if string(canonical) != want {
		t.Fatalf("canonical=%q want=%q", canonical, want)
	}
	if values["x"] == nil || !math.IsNaN(values["nan"].(float64)) || !math.IsInf(values["positive"].(float64), 1) || !math.IsInf(values["negative"].(float64), -1) {
		t.Fatalf("values=%#v", values)
	}
}

func TestDjangoJSONDecodesLatin1BeforeCanonicalizing(t *testing.T) {
	input := append([]byte(`{"note":"`), 0xe9)
	input = append(input, []byte(`"}`)...)
	values, canonical, err := decodeDjangoJSON(input)
	if err != nil || values["note"] != "é" || string(canonical) != `{"note":"\u00e9"}` {
		t.Fatalf("values=%#v canonical=%q err=%v", values, canonical, err)
	}
}

func TestDjangoJSONEscapesDELAndEnforcesPythonIntegerLimit(t *testing.T) {
	_, canonical, err := decodeDjangoJSON([]byte("{\"control\":\"\x7f\"}"))
	if err != nil || string(canonical) != `{"control":"\u007f"}` {
		t.Fatalf("canonical=%q err=%v", canonical, err)
	}
	if _, _, err := decodeDjangoJSON([]byte(`{"value":` + strings.Repeat("9", 4300) + `}`)); err != nil {
		t.Fatalf("4300-digit integer rejected: %v", err)
	}
	if _, _, err := decodeDjangoJSON([]byte(`{"value":` + strings.Repeat("9", 4301) + `}`)); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("4301-digit integer error=%v", err)
	}
}

func TestDjangoJSONDistinguishesValidNonObjectPayloads(t *testing.T) {
	for _, input := range []string{`[]`, `["_auth_user_id"]`, `null`, `true`, `1`, `"session"`} {
		if _, _, err := decodeDjangoJSON([]byte(input)); !errors.Is(err, ErrInvalidSessionShape) || errors.Is(err, ErrInvalidData) {
			t.Fatalf("decodeDjangoJSON(%s) error=%v", input, err)
		}
	}
	if _, _, err := decodeDjangoJSON([]byte(`[] trailing`)); !errors.Is(err, ErrInvalidData) || errors.Is(err, ErrInvalidSessionShape) {
		t.Fatalf("invalid JSON error=%v", err)
	}
}
