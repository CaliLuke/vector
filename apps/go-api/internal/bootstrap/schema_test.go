package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestSchemaValidationRejectsMissingOrChangedDefinitions(t *testing.T) {
	expected := map[string]string{"table:users": "r", "column:users.id": "uuid"}
	if err := validateSchema(expected, map[string]string{"table:users": "r", "column:users.id": "uuid", "table:extension": "r"}); err != nil {
		t.Fatal(err)
	}
	for _, actual := range []map[string]string{{"table:users": "r"}, {"table:users": "r", "column:users.id": "text"}} {
		if err := validateSchema(expected, actual); err == nil {
			t.Fatal("accepted incomplete or incompatible schema")
		}
	}
}

func TestPostgresSchemaBootstrapFreshAdoptAndRejectPartial(t *testing.T) {
	databaseURL := os.Getenv("PLANE_SCHEMA_BOOTSTRAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set PLANE_SCHEMA_BOOTSTRAP_TEST_DATABASE_URL to a dedicated disposable database")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || !strings.Contains(parsed.Path, "bootstrap_test") {
		t.Fatal("schema bootstrap test requires its own bootstrap_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	var tableCount int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname='public'`).Scan(&tableCount); err != nil || tableCount != 0 {
		t.Fatalf("dedicated schema test database must be empty: tables=%d error=%v", tableCount, err)
	}
	if err := MigrateSchema(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"auth_permission": 424, "django_content_type": 106, "django_migrations": 164} {
		var got int
		if err := connection.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{"public", table}.Sanitize()).Scan(&got); err != nil || got != want {
			t.Fatalf("source metadata %s: count=%d want=%d error=%v", table, got, want, err)
		}
	}
	if err := MigrateSchema(ctx, databaseURL); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	// Reconstruct the exact source FK policy before testing legacy adoption.
	// The later empty-ledger case covers recovery of an already migrated schema.
	down, err := schemaFiles.ReadFile("schema_migrations/000002_native_deletion.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	// An unrelated support ledger and user data must survive Django adoption.
	if _, err := connection.Exec(ctx, `CREATE TABLE public.schema_migrations(version bigint, dirty boolean); INSERT INTO public.schema_migrations VALUES(9,false); INSERT INTO public.django_migrations(app,name,applied) VALUES('bootstrap_test','preserved',CURRENT_TIMESTAMP); DROP TABLE public.plane_schema_migrations`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(ctx, databaseURL); err != nil {
		t.Fatalf("adopt source schema: %v", err)
	}
	var saved string
	if err := connection.QueryRow(ctx, `SELECT name FROM public.django_migrations WHERE app='bootstrap_test'`).Scan(&saved); err != nil || saved != "preserved" {
		t.Fatalf("adoption changed data: %q %v", saved, err)
	}
	var version int
	if err := connection.QueryRow(ctx, `SELECT version FROM public.schema_migrations`).Scan(&version); err != nil || version != 9 {
		t.Fatalf("support ledger changed: %d %v", version, err)
	}
	// Recover safely if a previous adopter created its empty ledger then stopped.
	if _, err := connection.Exec(ctx, `TRUNCATE public.plane_schema_migrations`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(ctx, databaseURL); err != nil {
		t.Fatalf("recover empty ledger: %v", err)
	}
	if _, err := connection.Exec(ctx, `DROP TABLE public.plane_schema_migrations; ALTER TABLE public.users DROP COLUMN last_login_ip`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(ctx, databaseURL); err == nil || !strings.Contains(err.Error(), "column:users.last_login_ip") {
		t.Fatalf("partial schema adoption error = %v", err)
	}
	var ledger bool
	if err := connection.QueryRow(ctx, `SELECT to_regclass('public.plane_schema_migrations') IS NOT NULL`).Scan(&ledger); err != nil || ledger {
		t.Fatalf("rejected adoption wrote migration ledger: exists=%t error=%v", ledger, err)
	}
}
