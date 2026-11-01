package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnicodeEscapePreservesThreeDigitOctalCodePoints(t *testing.T) {
	for _, sample := range []struct{ input, want string }{
		{`\377`, "\u00ff"},
		{`\400`, "\u0100"},
		{`\777`, "\u01ff"},
		{`\7777`, "\u01ff7"},
	} {
		t.Run(sample.input, func(t *testing.T) {
			require.Equal(t, sample.want, decodePythonUnicodeEscape([]byte(sample.input), false))
			require.Equal(t, sample.input, decodePythonUnicodeEscape([]byte(sample.input), true))
		})
	}
}
