package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/go-plane/internal/routing"
	"github.com/stretchr/testify/require"
)

func TestCommandWritesCanonicalPolicyAndReportsInvalidDestinations(t *testing.T) {
	var diagnostic bytes.Buffer
	require.Equal(t, 2, run(nil, &diagnostic))
	require.Contains(t, diagnostic.String(), "output")
	output := filepath.Join(t.TempDir(), "routes.caddy")
	diagnostic.Reset()
	require.Equal(t, 0, run([]string{"-output", output}, &diagnostic), diagnostic.String())
	policy, err := routing.Load()
	require.NoError(t, err)
	expected, err := routing.RenderCaddy(policy)
	require.NoError(t, err)
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, 1, run([]string{"-output", filepath.Dir(output)}, &diagnostic))
	require.Equal(t, 2, run([]string{"-unknown"}, &diagnostic))
	require.Equal(t, 0, run([]string{"-help"}, &diagnostic))
}
