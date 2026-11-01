package pythoncompat_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/stretchr/testify/require"
)

func TestDumpsRejectsInvalidJSONNumberTokens(t *testing.T) {
	for _, token := range []string{"", "bogus", "1.e", "1e", "01", "+1", "true", "1_2"} {
		t.Run(token, func(t *testing.T) {
			result, err := pythoncompat.Dumps(json.Number(token), nil)
			require.Error(t, err)
			require.Empty(t, result)
		})
	}
}

func TestDumpsKeepsPythonFloatOverflowAndIntegerPrecision(t *testing.T) {
	for _, sample := range []struct{ token, want string }{
		{"1e999", "Infinity"},
		{"-1e999", "-Infinity"},
		{"-1e-999", "-0.0"},
		{"123456789012345678901234567890", "123456789012345678901234567890"},
	} {
		t.Run(sample.token, func(t *testing.T) {
			result, err := pythoncompat.Dumps(json.Number(sample.token), nil)
			require.NoError(t, err)
			require.Equal(t, sample.want, result)
		})
	}
}
