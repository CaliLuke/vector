package pythoncompat_test

import (
	"math"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func TestNormalizeJSONNumbersMatchesPythonNumericValues(t *testing.T) {
	// Numeric expectations are fixed json.dumps(json.loads(input), allow_nan=False)
	// results. Keep them literal so the test does not repeat the Go implementation.
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"integer above binary64 precision", "9007199254740993", "9007199254740993"},
		{"negative integer above binary64 precision", "-9007199254740993", "-9007199254740993"},
		{"integer above uint64 range", "123456789012345678901234567890123456789", "123456789012345678901234567890123456789"},
		{"exponent rounds through binary64", "9007199254740993e0", "9007199254740992.0"},
		{"decimal rounds through binary64", "9007199254740993.0", "9007199254740992.0"},
		{"decimal rounds to whole float", "1.0000000000000001", "1.0"},
		{"shortest decimal", "123456789.123456789", "123456789.12345679"},
		{"redundant decimal zeroes", "1.25000", "1.25"},
		{"positive underflow", "1e-999", "0.0"},
		{"negative underflow", "-1e-999", "-0.0"},
		{"integer negative zero", "-0", "0"},
		{"float negative zero", "-0.0", "-0.0"},
		{"exponent negative zero", "-0e10", "-0.0"},
		{"uppercase exponent and leading exponent zeroes", "1E+0002", "100.0"},
		{"small fixed boundary", "1e-4", "0.0001"},
		{"small scientific boundary", "1e-5", "1e-05"},
		{"large fixed boundary", "1e15", "1000000000000000.0"},
		{"large scientific boundary", "1e16", "1e+16"},
		{"smallest subnormal", "5e-324", "5e-324"},
		{"below smallest subnormal", "2e-324", "0.0"},
		{"largest finite float", "1.7976931348623157e308", "1.7976931348623157e+308"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pythoncompat.NormalizeJSONNumbers([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("NormalizeJSONNumbers(%s) = %s, want %s", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeJSONNumbersPreservesNonNumericJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"null", "null", "null"},
		{"empty object", "{}", "{}"},
		{"empty array", "[]", "[]"},
		{"empty string", `""`, `""`},
		{"numeric string", `"9007199254740993e0"`, `"9007199254740993e0"`},
		{"escaped quotes and backslashes", `{"quote\"1e999":"slash\\-0","unicode":"\u0031e999"}`, `{"quote\"1e999":"slash\\-0","unicode":"\u0031e999"}`},
		{
			"nested values preserve keys order whitespace and empties",
			" {\n\t\"9007199254740993e0\": [9007199254740993, 9007199254740993e0, {\"-0\": -0, \"zero\": -1e-999}], \"empty\": [null, {}, [], \"\", true, false], \"text\": \"1e999 -0\"\n} ",
			" {\n\t\"9007199254740993e0\": [9007199254740993, 9007199254740992.0, {\"-0\": 0, \"zero\": -0.0}], \"empty\": [null, {}, [], \"\", true, false], \"text\": \"1e999 -0\"\n} ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pythoncompat.NormalizeJSONNumbers([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("NormalizeJSONNumbers(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeJSONNumbersRejectsOverflow(t *testing.T) {
	for _, input := range []string{"1e999", "-1e999", "1.7976931348623159e308", `{"nested":[1,1e999]}`} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.NormalizeJSONNumbers([]byte(input)); err == nil {
				t.Fatalf("NormalizeJSONNumbers(%s) accepted a non-finite float", input)
			}
		})
	}
}

func TestNormalizeDecodedJSONNumbersPreservesOverflowAndNormalizesSiblings(t *testing.T) {
	input := ` {"positive":1e309,"negative":-1.7976931348623159e308,"finite":9007199254740993.0,"integer":9007199254740993,"nested":[1.0000000000000001,-9007199254740993e0,-0.0],"controls":{"positive":1e309,"negative":-1e999,"finite":9007199254740993.0}} `
	want := ` {"positive":1e309,"negative":-1.7976931348623159e308,"finite":9007199254740992.0,"integer":9007199254740993,"nested":[1.0,-9007199254740992.0,-0.0],"controls":{"positive":1e309,"negative":-1e999,"finite":9007199254740992.0}} `

	got, err := pythoncompat.NormalizeDecodedJSONNumbers([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("NormalizeDecodedJSONNumbers(%q) = %q, want %q", input, got, want)
	}
}

func TestNormalizeDecodedJSONNumbersRejectsInvalidJSON(t *testing.T) {
	for _, input := range []string{"", `{"value":1`, `{"value":1e+}`, `NaN`, `Infinity`} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.NormalizeDecodedJSONNumbers([]byte(input)); err == nil {
				t.Fatalf("NormalizeDecodedJSONNumbers(%q) accepted invalid JSON", input)
			}
		})
	}
}

func TestNormalizeJSONNumbersRejectsInvalidJSON(t *testing.T) {
	for _, input := range []string{
		"", " ", "01", "+1", ".1", "1.", "1e", "1e+", "NaN", "Infinity", "-Infinity",
		"1 2", "[1,]", `{"value":}`, `{"value":1`, `"unterminated`, `"\x31"`,
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.NormalizeJSONNumbers([]byte(input)); err == nil {
				t.Fatalf("NormalizeJSONNumbers(%q) accepted invalid JSON", input)
			}
		})
	}
}

func TestFloatStringMatchesPythonRenderingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{
		{0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{100, "100.0"},
		{-100, "-100.0"},
		{1.25, "1.25"},
		{0.0001234, "0.0001234"},
		{1e-4, "0.0001"},
		{1e-5, "1e-05"},
		{-1e-5, "-1e-05"},
		{1.234e15, "1234000000000000.0"},
		{1e16, "1e+16"},
		{-1e16, "-1e+16"},
		{9007199254740992, "9007199254740992.0"},
		{math.SmallestNonzeroFloat64, "5e-324"},
		{math.MaxFloat64, "1.7976931348623157e+308"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			if got := pythoncompat.FloatString(tc.value); got != tc.want {
				t.Fatalf("FloatString(%g) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
