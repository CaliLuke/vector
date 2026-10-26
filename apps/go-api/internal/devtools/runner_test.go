package devtools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunnerMergesAndRedactsChildOutputAndUsesRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	cwdFile := filepath.Join(root, "pnpm-cwd")
	dockerArgsFile := filepath.Join(root, "docker-args")
	writeExecutable(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\npwd > \"$PNPM_CWD_FILE\"\nprintf 'password=private words\\n' >&2\nexec sleep 20\n")
	writeExecutable(t, filepath.Join(bin, "docker"), "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$DOCKER_ARGS_FILE\"\nprintf 'go-migrator-1 | 2026-10-22T00:00:00Z ERROR token=private\\nproxy-1 | 2026-10-22T00:00:00Z ERROR token=private\\n'\nexec sleep 20\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PNPM_CWD_FILE", cwdFile)
	t.Setenv("DOCKER_ARGS_FILE", dockerArgsFile)
	var recordsMu sync.Mutex
	var exported []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		recordsMu.Lock()
		exported = append(exported, body...)
		recordsMu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	var stdout, stderr strings.Builder
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan int, 1)
	go func() { finished <- runLocalDevelopment(ctx, root, &stdout, &stderr, server.URL) }()
	deadline := time.After(3 * time.Second)
	for {
		if _, err := os.Stat(cwdFile); err == nil {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("frontend child did not start")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := os.WriteFile(filepath.Join(root, "cancel-now"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(650 * time.Millisecond)
	cancel()
	select {
	case code := <-finished:
		if code != 0 {
			t.Fatalf("runner exit code = %d, stderr=%s", code, stderr.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not stop its child process groups")
	}
	if strings.TrimSpace(readFile(t, cwdFile)) != root {
		t.Fatalf("frontend started in %q, want repository root %q", readFile(t, cwdFile), root)
	}
	if args := readFile(t, dockerArgsFile); strings.Contains(args, "go-api") {
		t.Fatalf("runner still tails the Go API, which exports its own logs: %q", args)
	}
	recordsMu.Lock()
	defer recordsMu.Unlock()
	if len(exported) == 0 {
		t.Fatal("runner did not export child stderr/stdout records")
	}
	if strings.Contains(string(exported), "private") || !strings.Contains(string(exported), "plane-frontend") || !strings.Contains(string(exported), "plane-go-migrator") || !strings.Contains(string(exported), "plane-proxy") {
		t.Fatalf("unexpected exported logs: %s", exported)
	}
}

func TestRunnerRetriesWhenDockerLogChildCannotRestart(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	dockerPath := filepath.Join(bin, "docker")
	writeExecutable(t, dockerPath, "#!/bin/sh\n/bin/rm \"$0\"\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\nexec /bin/sleep 20\n")
	t.Setenv("PATH", bin)
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan int, 1)
	var stderr strings.Builder
	go func() { finished <- runLocalDevelopment(ctx, root, io.Discard, &stderr, "http://127.0.0.1:1") }()
	time.Sleep(2300 * time.Millisecond)
	cancel()
	select {
	case code := <-finished:
		if code != 0 {
			t.Fatalf("runner exit code = %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner stalled after Docker log capture restart failure")
	}
	if !strings.Contains(stderr.String(), "Restart Docker log capture") {
		t.Fatalf("expected restart failure diagnostic, got %q", stderr.String())
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
