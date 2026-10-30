//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAPIActivityLogAdmissionRollback(t *testing.T) {
	databaseURL, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := NewPostgresAPIActivityLogStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	app, workflow, cleanup := newAPIActivityLogTestApp(t, databaseURL, executionURL, store, nil)
	defer cleanup()
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}

	transactionContext, err := app.Storage().BeginTransaction(ctx)
	if err != nil {
		t.Fatal(err)
	}
	instanceID := "api-activity-log:rollback:" + uuid.NewString()
	input := testAPIActivityLogInput()
	if _, err := selvedge.StartWorkflow(transactionContext, app, workflow, input, selvedge.WithInstanceID(instanceID)); err != nil {
		t.Fatalf("admit inside PostgreSQL transaction: %v", dbutil.RollbackTransaction(transactionContext, app.Storage().RollbackTransaction, err))
	}
	if err := app.Storage().RollbackTransaction(transactionContext); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetInstance(ctx, instanceID); !errors.Is(err, selvedge.ErrInstanceNotFound) {
		t.Fatalf("workflow survived transaction rollback: instance lookup error = %v", err)
	}
}

func TestPostgresAPIActivityLogRestartAndRetryAfterCommit(t *testing.T) {
	databaseURL, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	store, err := NewPostgresAPIActivityLogStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	input := testAPIActivityLogInput()
	instanceID := "api-activity-log:restart:" + input.ID

	producer, producerWorkflow, stopProducer := newAPIActivityLogTestApp(t, databaseURL, executionURL, store, []selvedge.Option{
		selvedge.WithActivityQueues("background-test-producer-does-not-work"),
	})
	if startErr := producer.Start(ctx); startErr != nil {
		stopProducer()
		t.Fatal(startErr)
	}
	t.Cleanup(stopProducer)
	if _, startWorkflowErr := selvedge.StartWorkflow(ctx, producer, producerWorkflow, input, selvedge.WithInstanceID(instanceID)); startWorkflowErr != nil {
		stopProducer()
		t.Fatalf("admit API activity log: %v", startWorkflowErr)
	}
	waitForQueuedActivity(t, ctx, pool, instanceID)
	stopProducer()

	flakyStore := &insertThenFailOnce{store: store}
	worker, workerWorkflow, stopWorker := newAPIActivityLogTestApp(t, databaseURL, executionURL, flakyStore, nil)
	defer stopWorker()
	if workerWorkflow.Name() != producerWorkflow.Name() {
		t.Fatalf("workflow names differ after restart: producer=%q worker=%q", producerWorkflow.Name(), workerWorkflow.Name())
	}
	if startErr := worker.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, worker, instanceID)
	if err != nil {
		t.Fatalf("wait after worker restart: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("recovered workflow status = %q, error = %v", result.Status, result.Error)
	}
	if !flakyStore.failedAfterCommit.Load() {
		t.Fatal("the first activity attempt did not commit and then fail as arranged")
	}

	var attempts, failures, records int
	if err := pool.QueryRow(ctx, `SELECT attempts,failures FROM plane_execution.workflow_activity_tasks WHERE instance_id=$1 AND activity_name=$2`, instanceID, APIActivityLogActivityName).Scan(&attempts, &failures); err != nil {
		t.Fatal(err)
	}
	if attempts < 2 || failures < 1 {
		t.Fatalf("activity attempts=%d failures=%d, want a retried committed write", attempts, failures)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_activity_logs WHERE id=$1::uuid`, input.ID).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if records != 1 {
		t.Fatalf("API activity log rows = %d, want exactly one after replay", records)
	}
}

type insertThenFailOnce struct {
	store             *PostgresAPIActivityLogStore
	failedAfterCommit atomic.Bool
}

func (store *insertThenFailOnce) InsertAPIActivityLog(ctx context.Context, input APIActivityLogInput) error {
	if err := store.store.InsertAPIActivityLog(ctx, input); err != nil {
		return err
	}
	if store.failedAfterCommit.CompareAndSwap(false, true) {
		return errors.New("temporary failure after committed API activity log insert")
	}
	return nil
}

func testAPIActivityLogInput() APIActivityLogInput {
	return APIActivityLogInput{
		ID: uuid.NewString(), TokenIdentifier: "integration-test-token", Path: "/api/v1/background-test/",
		Method: "GET", QueryParams: "source=integration", Headers: "{'X-Api-Key': '[REDACTED]'}",
		ResponseCode: 200, ResponseBody: `{"ok":true}`, IPAddress: "192.0.2.42", UserAgent: "Plane background task integration test",
	}
}

func newAPIActivityLogTestApp(t *testing.T, databaseURL, executionURL string, store apiActivityLogStore, options []selvedge.Option) (*selvedge.App, *selvedge.WorkflowFunc[APIActivityLogInput, struct{}], func()) {
	t.Helper()
	appOptions := []selvedge.Option{
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-background-task-integration-test"),
		selvedge.WithActivityPollInterval(20 * time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20 * time.Millisecond),
		selvedge.WithTimerCheckInterval(20 * time.Millisecond),
	}
	appOptions = append(appOptions, options...)
	app := selvedge.NewApp(appOptions...)
	workflow, err := RegisterAPIActivityLogWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown test worker for %s: %v", databaseURL, err)
		}
	}
	return app, workflow, stop
}

func waitForQueuedActivity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, instanceID string) {
	t.Helper()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM plane_execution.workflow_activity_tasks WHERE instance_id=$1 AND activity_name=$2 AND status='queued'`, instanceID, APIActivityLogActivityName).Scan(&count)
		if err != nil {
			t.Fatalf("inspect admitted PostgreSQL activity: %v", err)
		}
		if count == 1 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("workflow %s did not leave a queued activity before timeout", instanceID)
		case <-ticker.C:
		}
	}
}
