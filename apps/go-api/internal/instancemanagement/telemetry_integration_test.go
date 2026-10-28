//go:build integration

package instancemanagement

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTelemetryReportsEffectiveServerConfiguration(t *testing.T) {
	ctx := t.Context()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE instances (is_telemetry_enabled boolean, created_at timestamptz DEFAULT now(), deleted_at timestamptz);
INSERT INTO instances (is_telemetry_enabled) VALUES (true);
CREATE TABLE instance_configurations (key text, value text, is_encrypted boolean, deleted_at timestamptz);
INSERT INTO instance_configurations VALUES ('POSTHOG_HOST','https://db.example.test/secret?token=secret',true,NULL), ('POSTHOG_API_KEY','test-key',true,NULL);`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repository, err := NewRepository(pool, timestampTestCipher{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, skipEnv, sdkDisabled, metricsEndpoint, protocol, key string
		wantLogs, wantMetrics, wantEvents, wantEventDestination    string
	}{
		{name: "database configuration wins", metricsEndpoint: "metrics.example.test:4317", protocol: "grpc", key: "env-key", wantLogs: "enabled", wantMetrics: "enabled", wantEvents: "enabled", wantEventDestination: "https://db.example.test"},
		{name: "environment override", skipEnv: "0", sdkDisabled: "true", wantLogs: "disabled", wantMetrics: "not_configured", wantEvents: "not_configured", wantEventDestination: "https://env.example.test"},
		{name: "invalid metrics protocol", skipEnv: "false", metricsEndpoint: "https://metrics.example.test", protocol: "invalid", key: "env-key", wantLogs: "enabled", wantMetrics: "invalid_configuration", wantEvents: "enabled", wantEventDestination: "https://env.example.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SKIP_ENV_VAR", tc.skipEnv)
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://logal:4318")
			t.Setenv("OTEL_SDK_DISABLED", tc.sdkDisabled)
			t.Setenv("OTLP_ENDPOINT", tc.metricsEndpoint)
			t.Setenv("OTLP_METRICS_PROTOCOL", tc.protocol)
			t.Setenv("POSTHOG_HOST", "https://env.example.test")
			t.Setenv("POSTHOG_API_KEY", tc.key)
			body, readErr := repository.Telemetry(t.Context())
			if readErr != nil {
				t.Fatal(readErr)
			}
			var status map[string]telemetryStatus
			if decodeErr := json.Unmarshal(body, &status); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			got := []string{status["logs_and_traces"].Status, status["instance_metrics"].Status, status["analytics_events"].Status}
			want := []string{tc.wantLogs, tc.wantMetrics, tc.wantEvents}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("status=%v want=%v", got, want)
			}
			if destination := status["analytics_events"].Destination; destination == nil || *destination != tc.wantEventDestination {
				t.Fatalf("unexpected analytics destination: %v", destination)
			}
		})
	}
	t.Setenv("OTLP_ENDPOINT", "https://metrics.example.test")
	if _, updateErr := pool.Exec(ctx, `UPDATE instances SET is_telemetry_enabled=false`); updateErr != nil {
		t.Fatal(updateErr)
	}
	body, err := repository.Telemetry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]telemetryStatus
	if decodeErr := json.Unmarshal(body, &status); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if status["instance_metrics"].Status != "disabled" {
		t.Fatalf("disabled metrics reported as %s", body)
	}
}
