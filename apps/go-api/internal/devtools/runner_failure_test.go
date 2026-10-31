package devtools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunnerStopsWhenFrontendOutputFails(t *testing.T) {
	root := runnerFailureCommands(t, "printf 'frontend ready\\n'\nexec /bin/sleep 30", "exec /bin/sleep 30")
	assertRunnerFailure(t, root, failedRunnerOutput{}, io.Discard)
}

func TestRunnerStopsWhenChildLogReadFails(t *testing.T) {
	// The scanner's existing one-MiB bound is a failure, not silent EOF.
	root := runnerFailureCommands(t, "printf '%s\\n' '"+strings.Repeat("x", (1<<20)+1)+"'\nexec /bin/sleep 30", "exec /bin/sleep 30")
	assertRunnerFailure(t, root, io.Discard, io.Discard)
}

func TestRunnerStopsWhenDiagnosticOutputFails(t *testing.T) {
	root := runnerFailureCommands(t, "exec /bin/sleep 30", "exit 0")
	assertRunnerFailure(t, root, io.Discard, failedRunnerOutput{})
}

type failedRunnerOutput struct{}

func (failedRunnerOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func runnerFailureCommands(t *testing.T, frontend, docker string) string {
	t.Helper()
	root := t.TempDir()
	for name, script := range map[string]string{"pnpm": frontend, "docker": docker} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("#!/bin/sh\n"+script+"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", root)
	return root
}

func assertRunnerFailure(t *testing.T, root string, stdout, stderr io.Writer) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan int, 1)
	go func() { finished <- runLocalDevelopment(ctx, root, stdout, stderr, server.URL) }()
	select {
	case code := <-finished:
		if code != 1 {
			t.Fatalf("runner exit code = %d, want failure", code)
		}
	case <-time.After(3 * time.Second):
		cancel()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Fatal("runner did not shut down after cancellation")
		}
		t.Fatal("runner ignored failed output and left its children running")
	}
}
