package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func (s *PostgresWorkItemEffectStore) startTelemetry(ctx context.Context) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") == "" ||
		strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return nil, nil
	}
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("configure work item outbox metrics exporter: %w", err)
	}
	serviceName := envOrDefault("OTEL_SERVICE_NAME", "vector-background-tasks")
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", serviceName),
		attribute.String("app.component", "work_item_effect_outbox"),
	))
	if err != nil {
		return nil, errors.Join(err, exporter.Shutdown(ctx))
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Minute), sdkmetric.WithTimeout(10*time.Second))),
	)
	if err := s.configureTelemetry(provider.Meter("vector.work_item_effects")); err != nil {
		return nil, errors.Join(err, provider.Shutdown(ctx))
	}
	return provider.Shutdown, nil
}

func (s *PostgresWorkItemEffectStore) configureTelemetry(meter metric.Meter) error {
	admissions, err := meter.Int64Counter("vector.work_item_effect.admissions",
		metric.WithDescription("Committed work item outbox admission outcomes."))
	if err != nil {
		return err
	}
	_, err = meter.Float64ObservableGauge("vector.work_item_effect.oldest_pending_age",
		metric.WithUnit("s"),
		metric.WithDescription("Age of the oldest unadmitted and non-quarantined work item effect."),
		metric.WithFloat64Callback(func(ctx context.Context, observer metric.Float64Observer) error {
			var age float64
			if scanErr := s.pool.QueryRow(ctx, `
				SELECT COALESCE(GREATEST(0, EXTRACT(epoch FROM clock_timestamp()-(
					SELECT created_at FROM work_item_effects
					WHERE admitted_at IS NULL AND quarantined_at IS NULL ORDER BY created_at LIMIT 1
				))),0)
			`).Scan(&age); scanErr != nil {
				return fmt.Errorf("observe oldest pending work item effect: %w", scanErr)
			}
			observer.Observe(age)
			return nil
		}),
	)
	if err != nil {
		return err
	}
	s.admissions = admissions
	return nil
}
