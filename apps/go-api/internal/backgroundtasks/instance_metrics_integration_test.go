//go:build integration

package backgroundtasks

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresInstanceMetricsCollectionAndDurableSchedule(t *testing.T) {
	t.Setenv("SERVICE_NAME", "")
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerID, workspaceID, projectID, instanceRowID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	instanceID := "metrics-" + uuid.NewString()
	username := "metrics-test-" + ownerID.String()
	now := time.Now().UTC()
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM instances WHERE id=$1`, instanceRowID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID)
	}()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,email,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,$3,'Metrics Test','Metrics','Test','',$4,$4,$4,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, ownerID, username, username+"@example.test", now); err != nil {
		t.Fatalf("insert metric user fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Metrics Test',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, "metrics-"+ownerID.String()[:8], now); err != nil {
		t.Fatalf("insert metric workspace fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Metrics Test','',0,$2,$3,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$4,$4)`, projectID, workspaceID, "MT"+ownerID.String()[:3], now); err != nil {
		t.Fatalf("insert metric project fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO instances (created_at,updated_at,id,instance_name,instance_id,current_version,last_checked_at,is_telemetry_enabled,is_support_required,is_setup_done,is_signup_screen_visited,is_verified,domain,latest_version,edition,is_test,is_current_version_deprecated) VALUES ($1,$1,$2,'Metrics Test',$3,'v1',$1,true,false,true,false,true,'plane.example.test','v2','PLANE_COMMUNITY',true,false)`, now, instanceRowID, instanceID); err != nil {
		t.Fatalf("insert metric instance fixture: %v", err)
	}
	exporter := &recordingInstanceMetricsExporter{}
	store, err := NewPostgresInstanceMetricsStore(pool, exporter)
	if err != nil {
		t.Fatal(err)
	}
	if pushErr := store.Push(ctx); pushErr != nil {
		t.Fatalf("collect and export PostgreSQL instance metrics: %v", pushErr)
	}
	snapshot := exporter.snapshot()
	if snapshot.ServiceName != "vector-ce-api" || snapshot.InstanceID != instanceID {
		t.Fatalf("metric resource service=%q instance=%q", snapshot.ServiceName, snapshot.InstanceID)
	}
	if len(snapshot.Observations) < 15 {
		t.Fatalf("collected %d metric observations, want 9 instance and at least 6 workspace metrics", len(snapshot.Observations))
	}
	if metricValue(snapshot, "plane_instance_users_total", "") < 1 || metricValue(snapshot, "plane_instance_workspaces_total", "") < 1 || metricValue(snapshot, "plane_instance_projects_total", "") < 1 {
		t.Fatalf("instance-level counts did not include inserted rows: %+v", snapshot.Observations[:9])
	}
	if got := metricValue(snapshot, "plane_workspace_projects_total", workspaceID.String()); got != 1 {
		t.Fatalf("workspace project count = %d, want 1", got)
	}
	for _, observation := range snapshot.Observations {
		if strings.HasPrefix(observation.Name, "plane_workspace_") && observation.Attributes["workspace_id"] == workspaceID.String() && observation.Attributes["workspace_slug"] != "metrics-"+ownerID.String()[:8] {
			t.Fatalf("workspace metric %q lacks the source slug attribute: %+v", observation.Name, observation.Attributes)
		}
	}

	if _, execErr := pool.Exec(ctx, `UPDATE instances SET is_telemetry_enabled=false WHERE id=$1`, instanceRowID); execErr != nil {
		t.Fatalf("disable telemetry fixture: %v", execErr)
	}
	if pushErr := store.Push(ctx); pushErr != nil {
		t.Fatalf("skip disabled telemetry: %v", pushErr)
	}
	if got := exporter.calls(); got != 1 {
		t.Fatalf("exporter called %d times after telemetry disabled, want 1", got)
	}

	t.Setenv("METRICS_PUSH_INTERVAL_MINUTES", "5")
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-instance-metrics-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterInstanceMetricsWorkflow(app, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatalf("start instance metrics workflow app: %v", startErr)
	}
	scheduleID, err := dispatcher.StartBeatSchedule(ctx)
	if err != nil {
		t.Fatalf("admit instance metrics Beat schedule: %v", err)
	}
	waitForMetricBeatTimer(t, ctx, pool, scheduleID)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
		shutdownCancel()
		t.Fatalf("stop instance metrics worker: %v", shutdownErr)
	}
	shutdownCancel()

	restarted := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-instance-metrics-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	restartedDispatcher, err := RegisterInstanceMetricsWorkflow(restarted, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := restarted.Start(ctx); startErr != nil {
		t.Fatalf("restart instance metrics worker: %v", startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := restarted.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown restarted instance metrics worker: %v", shutdownErr)
		}
	}()
	restartedID, err := restartedDispatcher.StartBeatSchedule(ctx)
	if err != nil {
		t.Fatalf("reconcile instance metrics Beat schedule after restart: %v", err)
	}
	if restartedID != scheduleID {
		t.Fatalf("restarted schedule ID=%q, want stable ID %q", restartedID, scheduleID)
	}
	waitForMetricBeatTimer(t, ctx, pool, scheduleID)
}

type recordingInstanceMetricsExporter struct {
	mu         sync.Mutex
	value      InstanceMetricsSnapshot
	callsCount int
}

func (exporter *recordingInstanceMetricsExporter) Export(_ context.Context, snapshot InstanceMetricsSnapshot) error {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	exporter.value = snapshot
	exporter.callsCount++
	return nil
}

func (exporter *recordingInstanceMetricsExporter) snapshot() InstanceMetricsSnapshot {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	return exporter.value
}

func (exporter *recordingInstanceMetricsExporter) calls() int {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	return exporter.callsCount
}

func metricValue(snapshot InstanceMetricsSnapshot, name, workspaceID string) int64 {
	for _, observation := range snapshot.Observations {
		if observation.Name == name && (workspaceID == "" || observation.Attributes["workspace_id"] == workspaceID) {
			return observation.Value
		}
	}
	return -1
}

func waitForMetricBeatTimer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, scheduleID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM plane_execution.workflow_timer_subscriptions WHERE instance_id=$1`, scheduleID).Scan(&count); err != nil {
			t.Fatalf("inspect instance metrics Beat timer: %v", err)
		}
		if count == 1 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for instance metrics Beat timer: %v", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("durable instance metrics Beat schedule did not arm a PostgreSQL timer")
}
