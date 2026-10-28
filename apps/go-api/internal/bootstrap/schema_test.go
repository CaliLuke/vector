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
		for table, want := range map[string]int{"auth_permission": 424, "django_content_type": 106, "django_migrations": 164} {
			var got int
			if scanErr := connection.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{"public", table}.Sanitize()).Scan(&got); scanErr != nil || got != want {
				t.Fatalf("source metadata %s: count=%d want=%d error=%v", table, got, want, scanErr)
			}
		}
	}
}

func TestPostgresSchemaBootstrapAdoptsDjango(t *testing.T) {
	// Build the historical source directly. Reversing selected migrations from
	// the latest schema does not represent an interrupted baseline adoption.
	baseline, err := schemaFiles.ReadFile("schema_migrations/000001_django_baseline.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		setup     string
		wantError string
	}{
		{name: "missing ledger"},
		{name: "empty ledger", setup: `CREATE TABLE public.plane_schema_migrations(version bigint PRIMARY KEY, dirty boolean NOT NULL)`},
		{name: "partial schema", setup: `ALTER TABLE public.users DROP COLUMN last_login_ip`, wantError: "column:users.last_login_ip"},
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
			if _, execErr := connection.Exec(ctx, `CREATE TABLE public.schema_migrations(version bigint, dirty boolean); INSERT INTO public.schema_migrations VALUES(9,false); INSERT INTO public.django_migrations(app,name,applied) VALUES('bootstrap_test','preserved',CURRENT_TIMESTAMP);`+test.setup); execErr != nil {
				t.Fatal(execErr)
			}
			migrateErr := MigrateSchema(ctx, databaseURL)
			if test.wantError != "" {
				if migrateErr == nil || !strings.Contains(migrateErr.Error(), test.wantError) {
					t.Fatalf("partial schema adoption error = %v", migrateErr)
				}
				var ledger bool
				if scanErr := connection.QueryRow(ctx, `SELECT to_regclass('public.plane_schema_migrations') IS NOT NULL`).Scan(&ledger); scanErr != nil || ledger {
					t.Fatalf("rejected adoption wrote migration ledger: exists=%t error=%v", ledger, scanErr)
				}
			} else if migrateErr != nil {
				t.Fatalf("adopt source schema: %v", migrateErr)
			}
			var saved string
			if scanErr := connection.QueryRow(ctx, `SELECT name FROM public.django_migrations WHERE app='bootstrap_test'`).Scan(&saved); scanErr != nil || saved != "preserved" {
				t.Fatalf("adoption changed data: %q %v", saved, scanErr)
			}
			var version int
			if scanErr := connection.QueryRow(ctx, `SELECT version FROM public.schema_migrations`).Scan(&version); scanErr != nil || version != 9 {
				t.Fatalf("support ledger changed: %d %v", version, scanErr)
			}
		})
	}
}
