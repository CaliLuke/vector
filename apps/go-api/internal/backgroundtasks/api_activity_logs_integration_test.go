//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const backgroundTasksDatabaseEnv = "PLANE_BACKGROUND_TASKS_DATABASE_URL"

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
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}

	transactionContext, err := app.Storage().BeginTransaction(ctx)
	if err != nil {
		t.Fatal(err)
	}
	instanceID := "api-activity-log:rollback:" + uuid.NewString()
	input := testAPIActivityLogInput()
	if _, err := selvedge.StartWorkflow(transactionContext, app, workflow, input, selvedge.WithInstanceID(instanceID)); err != nil {
		_ = app.Storage().RollbackTransaction(transactionContext)
		t.Fatalf("admit inside PostgreSQL transaction: %v", err)
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
	if err := producer.Start(ctx); err != nil {
		stopProducer()
		t.Fatal(err)
	}
	t.Cleanup(stopProducer)
	if _, err := selvedge.StartWorkflow(ctx, producer, producerWorkflow, input, selvedge.WithInstanceID(instanceID)); err != nil {
		stopProducer()
		t.Fatalf("admit API activity log: %v", err)
	}
	waitForQueuedActivity(t, ctx, pool, instanceID)
	stopProducer()

	flakyStore := &insertThenFailOnce{store: store}
	worker, workerWorkflow, stopWorker := newAPIActivityLogTestApp(t, databaseURL, executionURL, flakyStore, nil)
	defer stopWorker()
	if workerWorkflow.Name() != producerWorkflow.Name() {
		t.Fatalf("workflow names differ after restart: producer=%q worker=%q", producerWorkflow.Name(), workerWorkflow.Name())
	}
	if err := worker.Start(ctx); err != nil {
		t.Fatal(err)
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

func openBackgroundTaskDatabase(t *testing.T) (string, string, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv(backgroundTasksDatabaseEnv)
	if databaseURL == "" {
		t.Skip("set " + backgroundTasksDatabaseEnv + " to an isolated database with the Plane application schema")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse %s: %v", backgroundTasksDatabaseEnv, err)
	}
	if !strings.HasPrefix(config.ConnConfig.Database, "test_plane_background_tasks_") {
		t.Fatalf("integration requires an owned test_plane_background_tasks_* database, got %q", config.ConnConfig.Database)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("connect to isolated PostgreSQL: %v", err)
	}
	var tableExists bool
	if err := pool.QueryRow(t.Context(), `SELECT to_regclass('api_activity_logs') IS NOT NULL`).Scan(&tableExists); err != nil {
		pool.Close()
		t.Fatalf("inspect existing Plane application schema: %v", err)
	}
	if !tableExists {
		pool.Close()
		t.Fatalf("isolated database %q does not contain api_activity_logs; prepare its application schema before running this proof", config.ConnConfig.Database)
	}
	if _, err := pool.Exec(t.Context(), `CREATE SCHEMA IF NOT EXISTS plane_execution`); err != nil {
		pool.Close()
		t.Fatalf("create isolated Selvedge schema: %v", err)
	}
	executionURL, err := withExecutionSearchPath(databaseURL)
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return databaseURL, executionURL, pool
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
