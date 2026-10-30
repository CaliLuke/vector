package djangoinput

import (
	"encoding/json"
	"math"
	"testing"
)

func TestFloatFieldRejectsNonfiniteValues(t *testing.T) {
	for _, value := range []any{
		"NaN", "+nan", "-NaN", "inf", "+INF", "-inf", "Infinity", "-Infinity",
		"  NaN  ", "1e309", "-1e309", json.Number("1e309"), json.Number("-1e309"),
	} {
		t.Run(ScalarString(value), func(t *testing.T) {
			number, failure := FloatField(value)
			if failure != "A valid number is required." {
				t.Fatalf("FloatField(%v) = (%v, %q), want numeric validation failure", value, number, failure)
			}
		})
	}
}

func TestFloatFieldPreservesFiniteInputFormats(t *testing.T) {
	for _, test := range []struct {
		name  string
		input any
		want  float64
	}{
		{"JSON integer", json.Number("123"), 123},
		{"JSON fraction", json.Number("-12.5"), -12.5},
		{"JSON exponent", json.Number("1e3"), 1000},
		{"trimmed string", "  12.5  ", 12.5},
		{"Unicode decimal digits", "١٢", 12},
		{"digit separator", "1_000", 1000},
		{"true", true, 1}, {"false", false, 0},
		{"zero", "0", 0}, {"negative zero", "-0", math.Copysign(0, -1)},
		{"largest finite", "1.7976931348623157e308", math.MaxFloat64},
		{"smallest finite", "5e-324", math.SmallestNonzeroFloat64},
	} {
		t.Run(test.name, func(t *testing.T) {
			number, failure := FloatField(test.input)
			if failure != "" || number != test.want || math.Signbit(number) != math.Signbit(test.want) {
				t.Fatalf("FloatField(%v) = (%v, %q), want %v", test.input, number, failure, test.want)
			}
		})
	}
}
