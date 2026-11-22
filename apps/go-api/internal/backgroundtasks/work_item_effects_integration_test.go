//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/protobuf/proto"
)

func TestWorkItemEffectQuarantineTelemetryAndIdentityReplay(t *testing.T) {
	ctx := t.Context()
	_, pool := openBackgroundTaskClone(t, ctx)
	invalidID, transientID := uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO work_item_effects(id,kind,payload,created_at) VALUES
		 ($1,'email_task','{"name":[]}',clock_timestamp()-interval '2 hours'),
		 ($2,'email_task','{"name":"temporary_unavailable","args":[]}',clock_timestamp()-interval '2 minutes')
	`, invalidID, transientID)
	require.NoError(t, err)

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	mailer := &templatedEmailAdmissionSpy{err: errors.New("admission temporarily unavailable")}
	store := &PostgresWorkItemEffectStore{pool: pool, emails: mailer}
	require.NoError(t, store.configureTelemetry(provider.Meter("test")))
	count, err := store.Drain(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, []string{transientID.String()}, mailer.events)

	var retained, quarantined, delayed bool
	var attempts int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT payload='{"name":[]}', quarantined_at IS NOT NULL AND next_attempt_at='infinity',
		       admitted_at IS NULL AND last_error LIKE 'invalid work item effect payload:%', attempts
		FROM work_item_effects WHERE id=$1
	`, invalidID).Scan(&retained, &quarantined, &delayed, &attempts))
	require.True(t, retained)
	require.True(t, quarantined)
	require.True(t, delayed)
	require.Equal(t, 1, attempts)
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT payload<>'{}', quarantined_at IS NOT NULL, admitted_at IS NULL AND next_attempt_at>clock_timestamp(), attempts
		FROM work_item_effects WHERE id=$1
	`, transientID).Scan(&retained, &quarantined, &delayed, &attempts))
	require.True(t, retained)
	require.False(t, quarantined)
	require.True(t, delayed)
	require.Equal(t, 1, attempts)

	count, err = store.Drain(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "quarantined effects do not spin")
	outcomes, age := collectWorkItemEffectMetrics(t, reader)
	require.Equal(t, map[string]int64{"quarantined": 1, "retry": 1}, outcomes)
	require.InDelta(t, 120, age, 5, "quarantined rows are out of the pending-age gauge")

	// An operator fixes the payload and requeues the same admission identity.
	_, err = pool.Exec(ctx, `UPDATE work_item_effects SET payload='{"name":"repaired","args":[]}' WHERE id=$1`, invalidID)
	require.NoError(t, err)
	requeued, err := ReplayQuarantinedWorkItemEffect(ctx, pool, invalidID)
	require.NoError(t, err)
	require.True(t, requeued)
	requeued, err = ReplayQuarantinedWorkItemEffect(ctx, pool, invalidID)
	require.NoError(t, err)
	require.False(t, requeued)

	_, err = pool.Exec(ctx, `UPDATE work_item_effects SET next_attempt_at=clock_timestamp() WHERE id=$1`, transientID)
	require.NoError(t, err)
	mailer.err = nil
	count, err = store.Drain(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, []string{transientID.String(), invalidID.String(), transientID.String()}, mailer.events)
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM work_item_effects
		WHERE admitted_at IS NOT NULL AND quarantined_at IS NULL AND attempts=2 AND payload='{}' AND last_error=''
	`).Scan(&count))
	require.Equal(t, 2, count)
	outcomes, age = collectWorkItemEffectMetrics(t, reader)
	require.Equal(t, map[string]int64{"quarantined": 1, "retry": 1, "admitted": 2}, outcomes)
	require.Zero(t, age)
}

func TestWorkItemEffectWorkerExportsOperationalMetrics(t *testing.T) {
	ctx := t.Context()
	_, pool := openBackgroundTaskClone(t, ctx)
	store := &PostgresWorkItemEffectStore{pool: pool}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "false")
	shutdown, err := store.startTelemetry(ctx)
	require.NoError(t, err)
	require.Nil(t, shutdown)

	exports := make(chan *collectormetrics.ExportMetricsServiceRequest, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, readErr := io.ReadAll(r.Body)
		if readErr != nil || r.URL.Path != "/v1/metrics" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		report := new(collectormetrics.ExportMetricsServiceRequest)
		if decodeErr := proto.Unmarshal(raw, report); decodeErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case exports <- report:
		default:
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_SERVICE_NAME", "test-vector-outbox")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	shutdown, err = store.startTelemetry(ctx)
	require.NoError(t, err)
	require.Nil(t, shutdown)
	t.Setenv("OTEL_SDK_DISABLED", "false")

	worker := &workItemEffectWorker{store: store, startTelemetry: store.startTelemetry}
	require.NoError(t, worker.Start(ctx))
	require.NoError(t, worker.Shutdown(ctx))
	require.NoError(t, worker.Shutdown(ctx))
	select {
	case report := <-exports:
		require.Len(t, report.GetResourceMetrics(), 1)
		scope := report.GetResourceMetrics()[0].GetScopeMetrics()
		require.Len(t, scope, 1)
		require.Equal(t, "vector.work_item_effects", scope[0].GetScope().GetName())
		require.Len(t, scope[0].GetMetrics(), 1)
		require.Equal(t, "vector.work_item_effect.oldest_pending_age", scope[0].GetMetrics()[0].GetName())
	default:
		t.Fatal("worker shutdown did not flush operational metrics")
	}
}

func collectWorkItemEffectMetrics(t *testing.T, reader *sdkmetric.ManualReader) (map[string]int64, float64) {
	t.Helper()
	var metrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &metrics))
	outcomes := map[string]int64{}
	age := -1.0
	for _, scope := range metrics.ScopeMetrics {
		for _, sample := range scope.Metrics {
			switch sample.Name {
			case "vector.work_item_effect.admissions":
				for _, point := range sample.Data.(metricdata.Sum[int64]).DataPoints {
					require.Equal(t, 2, point.Attributes.Len())
					kind, ok := point.Attributes.Value(attribute.Key("effect.kind"))
					require.True(t, ok)
					require.Equal(t, "email_task", kind.AsString())
					outcome, ok := point.Attributes.Value(attribute.Key("effect.outcome"))
					require.True(t, ok)
					outcomes[outcome.AsString()] = point.Value
				}
			case "vector.work_item_effect.oldest_pending_age":
				points := sample.Data.(metricdata.Gauge[float64]).DataPoints
				require.Len(t, points, 1)
				require.Zero(t, points[0].Attributes.Len())
				age = points[0].Value
			}
		}
	}
	require.GreaterOrEqual(t, age, 0.0)
	return outcomes, age
}
