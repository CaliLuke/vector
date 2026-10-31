package devtools

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestRunnerShutdownReleasesInheritedOutputPipe(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "holder.pid")
	t.Setenv("VECTOR_RUNNER_PIPE_PID", pidFile)
	t.Setenv("VECTOR_RUNNER_TEST_BINARY", os.Args[0])
	root := runnerFailureCommands(t, `"$VECTOR_RUNNER_TEST_BINARY" -test.run '^TestRunnerInheritedPipeProcess$' &
while [ ! -f "$VECTOR_RUNNER_PIPE_PID" ]; do /bin/sleep 0.01; done
exit 0`, "exec /bin/sleep 30")
	ctx, cancel := context.WithCancel(t.Context())
	finished := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		finished <- runLocalDevelopment(ctx, root, io.Discard, io.Discard, "http://127.0.0.1:1")
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		stopRunnerPipeHolder(t, pidFile)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("runner remained blocked after pipe-holder cleanup")
		}
	})
	select {
	case code := <-finished:
		if code != 0 {
			t.Fatalf("runner exit code = %d, want successful frontend exit", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runner shutdown waits for an inherited output pipe instead of releasing its reader")
	}
}

// The subprocess retains the frontend's output descriptor in a separate process
// group, as a detached tool can. The parent test owns and kills this fixture.
func TestRunnerInheritedPipeProcess(t *testing.T) {
	pidFile := os.Getenv("VECTOR_RUNNER_PIPE_PID")
	if pidFile == "" {
		return
	}
	if _, err := syscall.Setsid(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Second)
}

func stopRunnerPipeHolder(t *testing.T, pidFile string) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Error(err)
		return
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Error(err)
		return
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Error(err)
	}
}
