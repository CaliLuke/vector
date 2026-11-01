package testsupport_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestJSONEqualRejectsEncodingFailuresAndPreservesValidComparisons(t *testing.T) {
	for _, test := range []struct {
		name             string
		actual, expected any
		equal            bool
	}{
		{"both invalid", make(chan int), func() {}, false},
		{"actual invalid", make(chan int), nil, false},
		{"expected invalid", nil, make(chan int), false},
		{"null", nil, nil, true},
		{"numeric projection", map[string]any{"count": 1}, map[string]any{"count": float64(1)}, true},
		{"different value", map[string]int{"count": 1}, map[string]int{"count": 2}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.equal, testsupport.JSONEqual(test.actual, test.expected))
		})
	}
}
