package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCloneDatabaseIdentifiesCanceledAdminConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	pool, err := pgxpool.New(t.Context(), "postgresql://127.0.0.1/postgres")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	pair := &Pair{seedURL: "postgresql://127.0.0.1/seed", adminPool: pool}
	clone, err := pair.CloneDatabase(ctx)
	if clone != nil || !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "connect clone admin database") {
		t.Fatalf("clone=%v error=%v; want canceled admin connection stage", clone, err)
	}
}

func TestDatabaseCloseIdentifiesCanceledAdminConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	database := &Database{adminURL: "postgresql://127.0.0.1/postgres", name: "owned_clone"}
	err := database.Close(ctx)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "connect drop admin database") {
		t.Fatalf("Close error=%v; want canceled admin connection stage", err)
	}
}

func TestStartRequiresSeedMigration(t *testing.T) {
	if _, err := Start(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "seed migration is required") {
		t.Fatalf("Start() error = %v, want missing migration error", err)
	}
}

func TestUnusedTemplateDoesNotRequireDockerOrMigration(t *testing.T) {
	template := NewTemplate(nil)
	if err := template.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := template.CloneDatabase(context.Background()); err == nil {
			t.Fatal("a template without a migration must fail before starting Docker")
		}
	}
}

func TestDatabaseURLChangesOnlyDatabase(t *testing.T) {
	got, err := databaseURL(
		"postgresql://plane:secret@127.0.0.1:54321/seed?sslmode=disable&application_name=parity",
		"target",
	)
	if err != nil {
		t.Fatalf("databaseURL() error = %v", err)
	}
	for _, required := range []string{"127.0.0.1", "54321", "target", "application_name=parity"} {
		if !strings.Contains(got, required) {
			t.Fatalf("databaseURL() = %q, want %q", got, required)
		}
	}
	if strings.Contains(got, "/seed") {
		t.Fatalf("databaseURL() = %q, retained seed database", got)
	}
}

func TestDatabaseURLRejectsNonPostgresScheme(t *testing.T) {
	if _, err := databaseURL("https://127.0.0.1/seed", "target"); err == nil {
		t.Fatal("databaseURL() error = nil, want scheme rejection")
	}
}

func TestSidecarURLKeepsDatabaseAndUsesContainerLocalPostgres(t *testing.T) {
	got, err := sidecarURL(
		"postgresql://plane:secret@127.0.0.1:54321/source?sslmode=disable&application_name=parity",
	)
	if err != nil {
		t.Fatalf("sidecarURL() error = %v", err)
	}
	for _, required := range []string{"127.0.0.1:5432", "/source", "application_name=parity"} {
		if !strings.Contains(got, required) {
			t.Fatalf("sidecarURL() = %q, want %q", got, required)
		}
	}
}
