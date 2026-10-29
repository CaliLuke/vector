package pythoncompat_test

import (
	"math"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func TestDecimalFloat64PreservesDecimalSyntaxAndRangeSemantics(t *testing.T) {
	for _, sample := range []struct {
		input string
		want  float64
	}{
		{"1_234.5e-1", 123.45},
		{" １２.５ ", 12.5},
		{".5", 0.5},
		{"-0", math.Copysign(0, -1)},
		{"1e999", math.Inf(1)},
		{"-1e999", math.Inf(-1)},
		{"-1e-999", math.Copysign(0, -1)},
	} {
		t.Run(sample.input, func(t *testing.T) {
			got, err := pythoncompat.DecimalFloat64(sample.input)
			if err != nil || got != sample.want || math.Signbit(got) != math.Signbit(sample.want) {
				t.Fatalf("DecimalFloat64(%q)=(%v,%v), want %v with signbit %t", sample.input, got, err, sample.want, math.Signbit(sample.want))
			}
		})
	}
}

func TestDecimalFloat64RejectsInvalidDecimalSyntax(t *testing.T) {
	for _, input := range []string{"", "1__2", "1e", "0x1p0", "NaN", "١foo", "1.2.3", "²", "１２．５"} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.DecimalFloat64(input); err == nil {
				t.Fatalf("DecimalFloat64(%q) accepted invalid decimal syntax", input)
			}
		})
	}
}

func TestDecimalIntegerPreservesSignsUnicodeDigitsAndSeparators(t *testing.T) {
	for _, sample := range []struct {
		input string
		want  string
	}{
		{"  +1_234 ", "1234"},
		{"-１２_３", "-123"},
		{"-0", "0"},
	} {
		t.Run(sample.input, func(t *testing.T) {
			got, err := pythoncompat.DecimalInteger(sample.input)
			if err != nil {
				t.Fatalf("DecimalInteger(%q): %v", sample.input, err)
			}
			if got.String() != sample.want {
				t.Fatalf("DecimalInteger(%q)=%s, want %s", sample.input, got, sample.want)
			}
		})
	}
}

func TestDecimalIntegerRejectsMissingDigitsAndInvalidSeparators(t *testing.T) {
	for _, input := range []string{"", "  ", "+", "-", "_1", "1_", "1__2", "+_1", "²"} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.DecimalInteger(input); err == nil {
				t.Fatalf("DecimalInteger(%q) accepted invalid decimal syntax", input)
			}
		})
	}
}

func TestDecimalIntegerDigitLimitExcludesSignsAndSeparators(t *testing.T) {
	input := "+" + strings.Repeat("１_", 4299) + "１"
	got, err := pythoncompat.DecimalInteger(input)
	if err != nil {
		t.Fatalf("4300 digits with sign and separators were rejected: %v", err)
	}
	if want := strings.Repeat("1", 4300); got.String() != want {
		t.Fatal("4300-digit integer was not preserved")
	}
	if _, err := pythoncompat.DecimalInteger(input + "１"); err == nil {
		t.Fatal("4301-digit integer was accepted")
	}
}
