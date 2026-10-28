package instancemanagement

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

type telemetryStatus struct {
	Status      string  `json:"status"`
	Destination *string `json:"destination"`
}

// Telemetry reports configured exporters, not delivery health. Only the
// authenticated instance administration endpoint may expose this projection.
func (r *Repository) Telemetry(ctx context.Context) (json.RawMessage, error) {
	var metricsEnabled bool
	err := r.db.QueryRow(ctx, `SELECT is_telemetry_enabled FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`).Scan(&metricsEnabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	values, err := r.configurationValues(ctx, []string{"POSTHOG_HOST", "POSTHOG_API_KEY"})
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"POSTHOG_HOST", "POSTHOG_API_KEY"} {
		if _, exists := values[key]; !exists {
			values[key] = os.Getenv(key)
		}
	}
	logs := exporterStatus(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), !strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true"), false)
	metrics := exporterStatus(os.Getenv("OTLP_ENDPOINT"), metricsEnabled, true)
	protocol := strings.ToLower(strings.TrimSpace(envOr("OTLP_METRICS_PROTOCOL", "grpc")))
	if metrics.Status == "enabled" && protocol != "http" && protocol != "grpc" {
		metrics.Status = "invalid_configuration"
	}
	events := exporterStatus(values["POSTHOG_HOST"], true, false)
	if values["POSTHOG_API_KEY"] == "" {
		events.Status = "not_configured"
	}
	return json.Marshal(map[string]telemetryStatus{"logs_and_traces": logs, "instance_metrics": metrics, "analytics_events": events})
}

func exporterStatus(endpoint string, enabled, allowBareHost bool) telemetryStatus {
	status := telemetryStatus{Status: "not_configured"}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		if allowBareHost && !strings.Contains(endpoint, "://") {
			endpoint = "https://" + endpoint
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			status.Status = "invalid_configuration"
		} else {
			// Show only the destination server. Credentials may appear in userinfo,
			// paths, queries or fragments; none belong in the administration UI.
			destination := parsed.Scheme + "://" + parsed.Host
			status.Destination = &destination
			status.Status = "enabled"
		}
	}
	if !enabled {
		status.Status = "disabled"
	}
	return status
}
