//go:build integration

package bootstrap

import (
	"context"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrateRuntimeSchemasFreshAndRepeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pair, err := postgresfixture.StartUncloned(ctx, func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("terminate disposable PostgreSQL: %v", closeErr)
		}
	})

	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := MigrateRuntimeSchemas(ctx, pair.SeedURL(), pool); err != nil {
		t.Fatalf("fresh runtime migrations: %v", err)
	}
	firstRiverVersion, firstSupportVersion := assertRuntimeMigrationState(t, ctx, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO public.plane_go_runtime_probes (name) VALUES ('repeat-check')`); err != nil {
		t.Fatalf("write runtime probe after fresh migrations: %v", err)
	}

	if err := MigrateRuntimeSchemas(ctx, pair.SeedURL(), pool); err != nil {
		t.Fatalf("repeat runtime migrations: %v", err)
	}
	secondRiverVersion, secondSupportVersion := assertRuntimeMigrationState(t, ctx, pool)
	if secondRiverVersion != firstRiverVersion || secondSupportVersion != firstSupportVersion {
		t.Fatalf("repeat migration changed ledgers: River %d→%d, support %d→%d", firstRiverVersion, secondRiverVersion, firstSupportVersion, secondSupportVersion)
	}

	var probeName string
	if err := pool.QueryRow(ctx, `SELECT name FROM public.plane_go_runtime_probes WHERE name = 'repeat-check'`).Scan(&probeName); err != nil || probeName != "repeat-check" {
		t.Fatalf("runtime schema did not remain usable after repeat: name=%q error=%v", probeName, err)
	}
}

func assertRuntimeMigrationState(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (int64, int64) {
	t.Helper()

	var riverVersion, supportVersion int64
	if err := pool.QueryRow(ctx, `SELECT max(version) FROM public.river_migration WHERE line = 'main'`).Scan(&riverVersion); err != nil || riverVersion == 0 {
		t.Fatalf("River migration ledger is missing or empty: version=%d error=%v", riverVersion, err)
	}
	var dirty bool
	if err := pool.QueryRow(ctx, `SELECT version, dirty FROM public.schema_migrations`).Scan(&supportVersion, &dirty); err != nil || supportVersion == 0 || dirty {
		t.Fatalf("support migration ledger is missing, empty, or dirty: version=%d dirty=%t error=%v", supportVersion, dirty, err)
	}
	var riverJobs, runtimeProbes int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.river_job`).Scan(&riverJobs); err != nil {
		t.Fatalf("River job schema is not queryable: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.plane_go_runtime_probes`).Scan(&runtimeProbes); err != nil {
		t.Fatalf("support runtime schema is not queryable: %v", err)
	}
	return riverVersion, supportVersion
}
