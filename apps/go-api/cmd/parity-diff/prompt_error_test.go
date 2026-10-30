package main

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
)

type rejectedPromptWriter struct{ calls int }

func (writer *rejectedPromptWriter) Write([]byte) (int, error) {
	writer.calls++
	return 0, io.ErrClosedPipe
}

type observedPromptReader struct{ calls int }

func (reader *observedPromptReader) Read([]byte) (int, error) {
	reader.calls++
	return 0, io.EOF
}

func TestResponseActionDoesNotWaitWhenPromptCannotBeWritten(t *testing.T) {
	directory := t.TempDir()
	output, input := &rejectedPromptWriter{}, &observedPromptReader{}
	action, err := responseAction(testConfirmation(input), output, filepath.Join(directory, "source.json"), filepath.Join(directory, "target.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = action(t.Context())
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("action error = %v; want prompt write failure", err)
	}
	if output.calls != 1 {
		t.Errorf("prompt writes = %d; want 1", output.calls)
	}
	if input.calls != 0 {
		t.Errorf("stdin reads = %d; want no wait after failed prompt", input.calls)
	}
}
