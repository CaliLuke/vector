//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestInstallFreshDatabaseAndRepeatWithRustFS(t *testing.T) {
	databaseURL := postgresfixture.EmptyDatabase(t)
	storage := testsupport.StartRustFS(t)
	storage.ConfigureAssets(t, "bootstrap-proof")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var existing int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public'`).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	if existing != 0 {
		t.Fatal("test requires an empty owned database")
	}
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("MACHINE_SIGNATURE", "go-bootstrap-cli-disposable-test")
	t.Setenv("SECRET_KEY", "isolated-bootstrap-cli-test-secret")
	t.Setenv("APP_VERSION", "v0.0.0-test")
	t.Setenv("IS_TEST", "1")
	t.Setenv("OTLP_METRICS_PROTOCOL", "http")
	t.Setenv("OTLP_ENDPOINT", collector.URL)
	var firstID, firstConfig string
	for attempt := 0; attempt < 2; attempt++ {
		if err := run(ctx, []string{"install"}); err != nil {
			t.Fatalf("install attempt %d: %v", attempt+1, err)
		}
		var count int
		var id, config string
		if err := conn.QueryRow(ctx, `SELECT count(*),min(id::text) FROM instances`).Scan(&count, &id); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("instances=%d, want1", count)
		}
		if err := conn.QueryRow(ctx, `SELECT json_agg(row_to_json(c) ORDER BY key)::text FROM instance_configurations c`).Scan(&config); err != nil {
			t.Fatal(err)
		}
		if attempt == 0 {
			firstID, firstConfig = id, config
		} else if id != firstID || config != firstConfig {
			t.Fatal("repeat install changed the registered instance or stored configuration")
		}
		var metricsJobs, schedules int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM plane_execution.workflow_instances WHERE workflow_name='plane.license.push_instance_metrics'`).Scan(&metricsJobs); err != nil {
			t.Fatal(err)
		}
		if metricsJobs != attempt+1 {
			t.Fatalf("durable registration jobs=%d, want%d", metricsJobs, attempt+1)
		}
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM plane_execution.workflow_instances WHERE workflow_name='plane.beat.push_instance_metrics'`).Scan(&schedules); err != nil {
			t.Fatal(err)
		}
		if schedules != 1 {
			t.Fatalf("durable metrics schedules=%d, want1", schedules)
		}

		for _, table := range []string{"plane_schema_migrations", "schema_migrations", "river_job", "plane_execution.workflow_instances"} {
			var present bool
			if err := conn.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&present); err != nil {
				t.Fatal(err)
			}
			if !present {
				t.Errorf("missing installation table %s", table)
			}
		}
	}
}
