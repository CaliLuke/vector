//go:build integration

package backgroundtasks

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresExporterHistoryWorkflowUsesDurableAdmission(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deleter := &exporterObjectDeleterRecorder{}
	store, err := NewPostgresExporterHistoryStore(pool, deleter)
	if err != nil {
		t.Fatal(err)
	}
	workflowApp := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-exporter-history-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterExporterHistoryWorkflow(workflowApp, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := workflowApp.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := workflowApp.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown exporter history runtime: %v", err)
		}
	}()

	scheduledAt := time.Now().UTC()
	instanceID, err := dispatcher.Admit(ctx, scheduledAt)
	if err != nil {
		t.Fatalf("admit scheduled exporter cleanup: %v", err)
	}
	duplicateID, err := dispatcher.Admit(ctx, scheduledAt)
	if err != nil {
		t.Fatalf("reconcile duplicate exporter cleanup admission: %v", err)
	}
	if instanceID != duplicateID {
		t.Fatalf("duplicate exporter admission ID %q, want %q", duplicateID, instanceID)
	}
	result, err := selvedge.WaitWorkflow[ExporterHistoryBatchOutput](ctx, workflowApp, instanceID)
	if err != nil {
		t.Fatalf("wait for exporter history workflow: %v", err)
	}
	if result.Status != "completed" || result.Output.Processed != 0 {
		t.Fatalf("exporter history result status=%q output=%+v error=%v", result.Status, result.Output, result.Error)
	}
	if deleter.calls.Load() != 0 {
		t.Fatalf("deleted %d objects with no expired rows", deleter.calls.Load())
	}
}

func TestPostgresCleanupAndExporterBeatSchedulesPersistTimers(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-cleanup-beat-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	cleanupStore, err := NewPostgresCleanupStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	cleanup, err := RegisterCleanupWorkflows(app, cleanupStore, nil)
	if err != nil {
		t.Fatal(err)
	}
	exporterStore, err := NewPostgresExporterHistoryStore(pool, &exporterObjectDeleterRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	exporter, err := RegisterExporterHistoryWorkflow(app, exporterStore, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown Beat schedule test runtime: %v", err)
		}
	}()
	if err := cleanup.StartBeatSchedules(ctx); err != nil {
		t.Fatalf("start cleanup Beat schedules: %v", err)
	}
	if err := exporter.StartBeatSchedules(ctx); err != nil {
		t.Fatalf("start exporter history Beat schedules: %v", err)
	}
	want := make([]string, 0, len(cleanupBeatSchedules)+len(exporterHistoryBeatSchedules))
	for _, spec := range cleanupBeatSchedules {
		want = append(want, "plane-beat:"+spec.name)
	}
	for _, spec := range exporterHistoryBeatSchedules {
		want = append(want, "plane-beat:"+spec.name)
	}
	for _, scheduleID := range want {
		waitForBeatTimer(t, ctx, pool, scheduleID)
	}
	if err := cleanup.StartBeatSchedules(ctx); err != nil {
		t.Fatalf("reconcile cleanup Beat schedules after repeat start: %v", err)
	}
	if err := exporter.StartBeatSchedules(ctx); err != nil {
		t.Fatalf("reconcile exporter history Beat schedules after repeat start: %v", err)
	}
	for _, scheduleID := range want {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM plane_execution.workflow_timer_subscriptions WHERE instance_id=$1`, scheduleID).Scan(&count); err != nil {
			t.Fatalf("inspect repeated schedule timer %s: %v", scheduleID, err)
		}
		if count != 1 {
			t.Fatalf("schedule %s has %d durable timers after repeated start, want 1", scheduleID, count)
		}
	}
}

func waitForBeatTimer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, scheduleID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM plane_execution.workflow_timer_subscriptions WHERE instance_id=$1`, scheduleID).Scan(&count); err != nil {
			t.Fatalf("inspect durable Beat timer %s: %v", scheduleID, err)
		}
		if count == 1 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for durable Beat timer %s: %v", scheduleID, ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("Beat schedule %s did not arm a PostgreSQL timer", scheduleID)
}

type exporterObjectDeleterRecorder struct{ calls atomic.Int64 }

func (recorder *exporterObjectDeleterRecorder) Delete(_ context.Context, _ string) error {
	recorder.calls.Add(1)
	return nil
}

var _ exporterObjectDeleter = (*exporterObjectDeleterRecorder)(nil)

func TestPostgresExporterHistoryStoreValidatesDependencies(t *testing.T) {
	if _, err := NewPostgresExporterHistoryStore((*pgxpool.Pool)(nil), &exporterObjectDeleterRecorder{}); err == nil {
		t.Fatal("expected PostgreSQL pool validation")
	}
	if _, err := NewPostgresExporterHistoryStore(nil, nil); err == nil {
		t.Fatal("expected PostgreSQL pool and object storage validation")
	}
}
