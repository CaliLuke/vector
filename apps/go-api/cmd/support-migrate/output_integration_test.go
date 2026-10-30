//go:build integration

package main

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestRunOutputFailurePreservesAppliedSupportSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	// Start with empty disposable databases: the command owns the migration.
	pair, err := postgresfixture.Start(ctx, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	migrationPath, err := filepath.Abs("../../migrations/support")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--database-url", pair.SourceURL, "--migrations-path", migrationPath}
	var stderr bytes.Buffer
	if code := run(append(args, "up"), failingOutput{}, &stderr); code != 1 {
		t.Fatalf("exit code = %d; want output failure", code)
	}
	if !strings.Contains(stderr.String(), io.ErrClosedPipe.Error()) {
		t.Fatal("missing output failure diagnostic")
	}
	connection, err := pgx.Connect(ctx, pair.SourceURL)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	var tableExists bool
	if err := connection.QueryRow(ctx, "SELECT to_regclass('public.plane_go_uuid_probes') IS NOT NULL").Scan(&tableExists); err != nil {
		t.Fatal(err)
	}
	if !tableExists {
		t.Fatal("output failure lost the applied schema")
	}
	var stdout bytes.Buffer
	if code := run(append(args, "version"), &stdout, io.Discard); code != 0 {
		t.Fatalf("version exit code = %d; want readable committed state", code)
	}
	if !strings.Contains(stdout.String(), "dirty=false") {
		t.Fatalf("migration left dirty state: %q", stdout.String())
	}
}
