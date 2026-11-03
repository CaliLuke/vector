package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
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

func TestPostgresSchemaBootstrapFreshAndRepeat(t *testing.T) {
	databaseURL := postgresfixture.EmptyDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	})
	for range 2 {
		if migrateErr := MigrateSchema(ctx, databaseURL); migrateErr != nil {
			t.Fatal(migrateErr)
		}
		assertSchemaBootstrapLockReleased(t, ctx, connection)
		var version int64
		var dirty bool
		if scanErr := connection.QueryRow(ctx, `SELECT version,dirty FROM public.plane_schema_migrations`).Scan(&version, &dirty); scanErr != nil || version < 1 || dirty {
			t.Fatalf("application migration ledger: version=%d dirty=%t error=%v", version, dirty, scanErr)
		}
	}
	var metadata bool
	if err := connection.QueryRow(ctx, `SELECT to_regclass('public.django_migrations') IS NOT NULL OR to_regclass('public.django_session') IS NOT NULL OR to_regclass('public.auth_permission') IS NOT NULL`).Scan(&metadata); err != nil || metadata {
		t.Fatalf("fresh installation requires retired framework metadata: present=%v error=%v", metadata, err)
	}
	// Older managed databases may retain metadata and must keep its data. It
	// is neither an acceptance prerequisite nor permission to delete it.
	if _, err := connection.Exec(ctx, `CREATE TABLE public.django_migrations(name text); INSERT INTO public.django_migrations VALUES('retained');`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := connection.QueryRow(ctx, `SELECT name FROM public.django_migrations`).Scan(&name); err != nil || name != "retained" {
		t.Fatalf("existing metadata changed: %q %v", name, err)
	}
}

func TestPostgresSchemaBootstrapRejectsUnmanagedApplicationTables(t *testing.T) {
	// An existing schema without the native ledger must not be silently adopted.
	baseline, err := schemaFiles.ReadFile("schema_migrations/000001_initial_schema.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		setup string
	}{
		{name: "missing ledger"},
		{name: "empty ledger", setup: `CREATE TABLE public.plane_schema_migrations(version bigint PRIMARY KEY, dirty boolean NOT NULL)`},
		{name: "partial schema", setup: `ALTER TABLE public.users DROP COLUMN last_login_ip`},
	} {
		t.Run(test.name, func(t *testing.T) {
			databaseURL := postgresfixture.EmptyDatabase(t)
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			defer cancel()
			connection, connectErr := pgx.Connect(ctx, databaseURL)
			if connectErr != nil {
				t.Fatal(connectErr)
			}
			t.Cleanup(func() {
				if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil {
					t.Error(closeErr)
				}
			})
			if _, execErr := connection.Exec(ctx, string(baseline)); execErr != nil {
				t.Fatal(execErr)
			}
			if _, execErr := connection.Exec(ctx, `CREATE TABLE public.schema_migrations(version bigint, dirty boolean); INSERT INTO public.schema_migrations VALUES(9,false); CREATE TABLE public.django_migrations(app text,name text,applied timestamptz); INSERT INTO public.django_migrations(app,name,applied) VALUES('bootstrap_test','preserved',CURRENT_TIMESTAMP);`+test.setup); execErr != nil {
				t.Fatal(execErr)
			}
			migrateErr := MigrateSchema(ctx, databaseURL)
			if migrateErr == nil || !strings.Contains(migrateErr.Error(), "without a managed migration baseline") {
				t.Fatalf("unmanaged schema error = %v", migrateErr)
			}
			var ledgerRows int
			if test.name == "empty ledger" {
				if scanErr := connection.QueryRow(ctx, `SELECT count(*) FROM public.plane_schema_migrations`).Scan(&ledgerRows); scanErr != nil || ledgerRows != 0 {
					t.Fatalf("rejected schema changed migration ledger: rows=%d error=%v", ledgerRows, scanErr)
				}
			} else {
				var ledger bool
				if scanErr := connection.QueryRow(ctx, `SELECT to_regclass('public.plane_schema_migrations') IS NOT NULL`).Scan(&ledger); scanErr != nil || ledger {
					t.Fatalf("rejected schema wrote migration ledger: exists=%t error=%v", ledger, scanErr)
				}
			}
			assertSchemaBootstrapLockReleased(t, ctx, connection)
			var saved string
			if scanErr := connection.QueryRow(ctx, `SELECT name FROM public.django_migrations WHERE app='bootstrap_test'`).Scan(&saved); scanErr != nil || saved != "preserved" {
				t.Fatalf("rejected schema changed data: %q %v", saved, scanErr)
			}
			var version int
			if scanErr := connection.QueryRow(ctx, `SELECT version FROM public.schema_migrations`).Scan(&version); scanErr != nil || version != 9 {
				t.Fatalf("support ledger changed: %d %v", version, scanErr)
			}
		})
	}
}

func assertSchemaBootstrapLockReleased(t *testing.T, ctx context.Context, connection *pgx.Conn) {
	t.Helper()
	// Closing the socket sends Terminate; PostgreSQL releases the session lock
	// when its backend processes that message. Wait for that cleanup rather
	// than requiring the client close and server release to be simultaneous.
	lockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := connection.Exec(lockCtx, `SELECT pg_advisory_lock($1)`, registrationLockID+2); err != nil {
		t.Fatalf("schema bootstrap retained its session lock: %v", err)
	}
	var released bool
	if err := connection.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, registrationLockID+2).Scan(&released); err != nil || !released {
		t.Fatalf("release test session lock: released=%t error=%v", released, err)
	}
}
