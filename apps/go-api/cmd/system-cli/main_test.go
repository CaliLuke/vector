package main

import (
	"bytes"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIExecutesGeneratedClientAndReportsFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health/live", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"status":"ok"}`)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	originalArgs, originalFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = originalArgs, originalFlags })
	invoke := func(args ...string) (int, string, string) {
		os.Args = append([]string{"system-cli"}, args...)
		flag.CommandLine = flag.NewFlagSet("system-cli", flag.ContinueOnError)
		var stdout, stderr bytes.Buffer
		flag.CommandLine.SetOutput(&stderr)
		status := run(&stdout, &stderr)
		return status, stdout.String(), stderr.String()
	}
	status, output, diagnostic := invoke("-url", server.URL, "system", "live")
	require.Zero(t, status, diagnostic)
	require.JSONEq(t, `{"status":"ok"}`, output)
	status, _, diagnostic = invoke("-host", "unknown", "system", "live")
	require.Equal(t, 1, status)
	require.Contains(t, diagnostic, "invalid host")
	status, _, diagnostic = invoke("-url", "://bad", "system", "live")
	require.Equal(t, 1, status)
	require.Contains(t, diagnostic, "invalid URL")
	status, _, diagnostic = invoke("-url", "ftp://localhost", "system", "live")
	require.Equal(t, 1, status)
	require.Contains(t, diagnostic, "http|https")
	status, _, diagnostic = invoke("no-such-service")
	require.Equal(t, 1, status)
	require.Contains(t, diagnostic, "--help")
	server.Close()
	status, _, diagnostic = invoke("-url", server.URL, "system", "live")
	require.Equal(t, 1, status)
	require.NotEmpty(t, diagnostic)
}
