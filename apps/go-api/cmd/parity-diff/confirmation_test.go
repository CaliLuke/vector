package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func confirmationPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	return reader, writer
}

func assertConfirmationInputStillOpen(t *testing.T, reader, writer *os.File) {
	t.Helper()
	const value = "input remains owned by caller\n"
	if _, err := io.WriteString(writer, value); err != nil {
		t.Fatal(err)
	}
	if err := reader.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(value))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != value {
		t.Errorf("remaining input=%q, want %q", got, value)
	}
}

type confirmationPrompt struct{ ready chan struct{} }

func (output confirmationPrompt) Write(p []byte) (int, error) {
	close(output.ready)
	return len(p), nil
}

func TestResponseActionInterruptsFileInputWithoutClosingIt(t *testing.T) {
	for _, kind := range []string{"cancel", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			input, writer := confirmationPipe(t)
			ctx, cancel := context.WithCancel(t.Context())
			want := context.Canceled
			if kind == "deadline" {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 50*time.Millisecond)
				want = context.DeadlineExceeded
			}
			defer cancel()
			prompt := confirmationPrompt{ready: make(chan struct{})}
			directory := t.TempDir()
			action, err := responseAction(func(ctx context.Context) error { return readConfirmation(ctx, input) }, prompt, filepath.Join(directory, "source.json"), filepath.Join(directory, "target.json"))
			if err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { _, err := action(ctx); result <- err }()
			select {
			case <-prompt.ready:
			case <-time.After(2 * time.Second):
				t.Fatal("prompt was not written")
			}
			if kind == "cancel" {
				cancel()
			}
			<-ctx.Done()
			select {
			case err := <-result:
				if !errors.Is(err, want) {
					t.Fatalf("action error=%v; want %v", err, want)
				}
			case <-time.After(2 * time.Second):
				// Release and join even against the original blocking implementation.
				if _, err := io.WriteString(writer, "\n"); err != nil {
					t.Error(err)
				}
				lateErr := <-result
				t.Fatalf("prompt stayed blocked after context ended; eventual error: %v", lateErr)
			}
			assertConfirmationInputStillOpen(t, input, writer)
		})
	}
}

func TestReadConfirmationAcceptsEnterAndKeepsInputOpen(t *testing.T) {
	input, writer := confirmationPipe(t)
	if _, err := io.WriteString(writer, "confirm\n"); err != nil {
		t.Fatal(err)
	}
	if confirmationErr := readConfirmation(t.Context(), input); confirmationErr != nil {
		t.Fatal(confirmationErr)
	}
	assertConfirmationInputStillOpen(t, input, writer)
}

func TestReadConfirmationAcceptsFileEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("confirm"), 0o600); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	if confirmationErr := readConfirmation(t.Context(), input); confirmationErr != nil {
		t.Fatal(confirmationErr)
	}
	if _, seekErr := input.Seek(0, io.SeekStart); seekErr != nil {
		t.Fatal(seekErr)
	}
	got, err := io.ReadAll(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "confirm" {
		t.Errorf("input=%q", got)
	}
}

func TestReadConfirmationAlreadyCancelledDoesNotConsumeInput(t *testing.T) {
	input, writer := confirmationPipe(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := readConfirmation(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	assertConfirmationInputStillOpen(t, input, writer)
}

func TestReadConfirmationReportsMissingOrClosedInput(t *testing.T) {
	if err := readConfirmation(t.Context(), nil); err == nil || !strings.Contains(err.Error(), "input is missing") {
		t.Fatalf("missing input error=%v", err)
	}
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if err := readConfirmation(t.Context(), input); err == nil {
		t.Fatal("closed input was accepted")
	}
}

// Exercise the same helper entry point when os.Executable is the test binary.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == confirmationReadFlag {
		os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

func TestReadConfirmationReportsHelperInputFailure(t *testing.T) {
	input, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	err = readConfirmation(t.Context(), input)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || !strings.Contains(err.Error(), "read confirmation input") {
		t.Fatalf("helper failure=%v; want exit error and input diagnostic", err)
	}
	if _, err := input.Stat(); err != nil {
		t.Fatalf("caller input closed: %v", err)
	}
}

func TestConfirmationHelperReportsReadFailure(t *testing.T) {
	var diagnostic bytes.Buffer
	if code := run([]string{confirmationReadFlag}, nil, io.Discard, &diagnostic); code != 1 {
		t.Fatalf("exit=%d; want 1", code)
	}
	if !strings.Contains(diagnostic.String(), "confirmation input is missing") {
		t.Fatalf("diagnostic=%q", diagnostic.String())
	}
}
