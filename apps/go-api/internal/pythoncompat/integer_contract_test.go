package pythoncompat_test

import (
	"math"
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
	for _, input := range []string{"", "1__2", "1e", "0x1p0", "NaN", "١foo", "1.2.3"} {
		t.Run(input, func(t *testing.T) {
			if _, err := pythoncompat.DecimalFloat64(input); err == nil {
				t.Fatalf("DecimalFloat64(%q) accepted invalid decimal syntax", input)
			}
		})
	}
}
