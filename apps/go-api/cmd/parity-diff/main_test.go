package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestResponseActionLoadsFilesAfterActionPause(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "source.json")
	targetPath := filepath.Join(directory, "target.json")
	var output bytes.Buffer
	reader := &actionReader{beforeRead: func() {
		writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"source-after-action"}}`)
		writeResponse(t, targetPath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"target-after-action"}}`)
	}}
	action, err := responseAction(testConfirmation(reader), &output, sourcePath, targetPath)
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}
	result, err := action(context.Background())
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}
	if !bytes.Contains(result.SourceResponse.Body, []byte("source-after-action")) {
		t.Fatalf("source body = %s, want response written after callback construction", result.SourceResponse.Body)
	}
	if !bytes.Contains(result.TargetResponse.Body, []byte("target-after-action")) {
		t.Fatalf("target body = %s, want response written after callback construction", result.TargetResponse.Body)
	}
}

func TestResponseActionRejectsStaleArtifact(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "source.json")
	targetPath := filepath.Join(directory, "target.json")
	writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"stale-source"}}`)
	writeResponse(t, targetPath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"stale-target"}}`)
	action, err := responseAction(testConfirmation(bytes.NewBufferString("\n")), &bytes.Buffer{}, sourcePath, targetPath)
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}

	if _, err := action(context.Background()); err == nil || !bytes.Contains([]byte(err.Error()), []byte("source response artifact was not")) {
		t.Fatalf("action() error = %v, want stale source rejection", err)
	}
}

func TestResponseActionDoesNotTreatPrePromptChangeAsFresh(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "source.json")
	targetPath := filepath.Join(directory, "target.json")
	writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"before-snapshots"}}`)
	writeResponse(t, targetPath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"before-snapshots"}}`)
	action, err := responseAction(testConfirmation(bytes.NewBufferString("\n")), &bytes.Buffer{}, sourcePath, targetPath)
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}
	writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"before-prompt"}}`)
	writeResponse(t, targetPath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"before-prompt"}}`)

	if _, err := action(context.Background()); err == nil || !bytes.Contains([]byte(err.Error()), []byte("source response artifact was not")) {
		t.Fatalf("action() error = %v, want pre-prompt change rejection", err)
	}
}

func TestResponseActionRequiresBothArtifactsToBeFresh(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "source.json")
	targetPath := filepath.Join(directory, "target.json")
	writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"old-source"}}`)
	writeResponse(t, targetPath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"old-target"}}`)
	reader := &actionReader{beforeRead: func() {
		writeResponse(t, sourcePath, `{"status":201,"headers":{},"cookies":{},"body":{"runtime":"new-source"}}`)
	}}
	action, err := responseAction(testConfirmation(reader), &bytes.Buffer{}, sourcePath, targetPath)
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}

	if _, err := action(context.Background()); err == nil || !bytes.Contains([]byte(err.Error()), []byte("target response artifact was not")) {
		t.Fatalf("action() error = %v, want stale target rejection", err)
	}
}

func TestResponseActionAcceptsReplacedArtifactsWithIdenticalContents(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "source.json")
	targetPath := filepath.Join(directory, "target.json")
	response := `{"status":201,"headers":{},"cookies":{},"body":{"ok":true}}`
	writeResponse(t, sourcePath, response)
	writeResponse(t, targetPath, response)
	reader := &actionReader{beforeRead: func() {
		replaceResponse(t, sourcePath, response)
		replaceResponse(t, targetPath, response)
	}}
	action, err := responseAction(testConfirmation(reader), &bytes.Buffer{}, sourcePath, targetPath)
	if err != nil {
		t.Fatalf("responseAction() error = %v", err)
	}

	if _, err := action(context.Background()); err != nil {
		t.Fatalf("action() error = %v", err)
	}
}

type actionReader struct {
	beforeRead func()
	read       bool
}

func (reader *actionReader) Read(buffer []byte) (int, error) {
	if reader.read {
		return 0, io.EOF
	}
	reader.read = true
	reader.beforeRead()
	return copy(buffer, "\n"), nil
}

func TestResponseActionRejectsSharedArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "response.json")
	writeResponse(t, path, `{"status":201,"headers":{},"cookies":{},"body":null}`)
	if _, err := responseAction(testConfirmation(bytes.NewBufferString("\n")), &bytes.Buffer{}, path, path); err == nil {
		t.Fatal("responseAction() error = nil, want shared-artifact rejection")
	}
}

func writeResponse(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write response fixture: %v", err)
	}
}

func replaceResponse(t *testing.T, path string, contents string) {
	t.Helper()
	temporary := path + ".replacement"
	writeResponse(t, temporary, contents)
	if err := os.Rename(temporary, path); err != nil {
		t.Fatalf("replace response fixture: %v", err)
	}
}

func testConfirmation(reader io.Reader) func(context.Context) error {
	return func(context.Context) error {
		_, err := bufio.NewReader(reader).ReadString('\n')
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}
