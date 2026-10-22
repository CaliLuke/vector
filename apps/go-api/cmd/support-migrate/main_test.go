package main

import (
	"bytes"
	"testing"
)

func TestRunRequiresDatabaseAndCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := run(nil, &stdout, &stderr); exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2", exitCode)
	}
}

func TestRunRejectsUnknownCommandBeforeConnection(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--database-url", "pgx5://invalid.invalid/database", "invalid"}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("run() exit code = %d, want 2; stderr = %s", exitCode, stderr.String())
	}
}

func TestSupportDatabaseURLSelectsPGXV5Driver(t *testing.T) {
	got, err := supportDatabaseURL("postgresql://plane:secret@localhost:5432/plane?sslmode=disable")
	if err != nil {
		t.Fatalf("supportDatabaseURL() error = %v", err)
	}
	want := "pgx5://plane:secret@localhost:5432/plane?sslmode=disable"
	if got != want {
		t.Fatalf("supportDatabaseURL() = %q, want %q", got, want)
	}
}
