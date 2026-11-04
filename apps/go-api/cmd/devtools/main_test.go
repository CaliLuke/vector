package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandValidationAndCheckoutDiscovery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	require.Equal(t, 2, run(nil, &stdout, &stderr))
	require.Contains(t, stderr.String(), "usage:")
	require.Equal(t, 2, run([]string{"unknown"}, &stdout, &stderr))
	root := t.TempDir()
	t.Chdir(root)
	stderr.Reset()
	require.Equal(t, 1, run([]string{"check-login"}, &stdout, &stderr))
	require.Contains(t, stderr.String(), "repository checkout")
	require.NoError(t, os.WriteFile(filepath.Join(root, "docker-compose-local.yml"), nil, 0600))
	require.NoError(t, os.Mkdir(filepath.Join(root, "nested"), 0700))
	t.Chdir(filepath.Join(root, "nested"))
	found, err := repositoryRoot()
	require.NoError(t, err)
	canonical, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(found)
	require.NoError(t, err)
	require.Equal(t, canonical, resolved)
	t.Setenv("PLANE_TEST_EMAIL", "")
	t.Setenv("PLANE_TEST_PASSWORD", "")
	stderr.Reset()
	require.Equal(t, 1, run([]string{"check-login"}, &stdout, &stderr))
	require.Contains(t, stderr.String(), "check-login failed:")
	require.Empty(t, stdout.String())
}
