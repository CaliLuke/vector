//go:build integration

package backgroundtasks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCleanupBatchOperations(t *testing.T) {
	_, _, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	store, err := NewPostgresCleanupStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range cleanupTasks {
		if _, deleteCleanupBatchErr := store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: task, Cutoff: time.Now().UTC()}); deleteCleanupBatchErr != nil {
			t.Fatalf("execute empty PostgreSQL cleanup batch for %s: %v", task, deleteCleanupBatchErr)
		}
	}

	cutoff := time.Now().UTC().Add(-time.Hour)
	ids := make([]uuid.UUID, 0, CleanupBatchSize+1)
	for range CleanupBatchSize + 1 {
		id := uuid.New()
		ids = append(ids, id)
		if insertAPIActivityErr := insertAPIActivityLogFixture(ctx, pool, id, cutoff.Add(-time.Second)); insertAPIActivityErr != nil {
			t.Fatalf("insert old API activity log: %v", insertAPIActivityErr)
		}
	}
	recentID := uuid.New()
	ids = append(ids, recentID)
	if insertAPIActivityErr := insertAPIActivityLogFixture(ctx, pool, recentID, cutoff.Add(time.Second)); insertAPIActivityErr != nil {
		t.Fatalf("insert recent API activity log: %v", insertAPIActivityErr)
	}
	defer func() {
		idStrings := make([]string, len(ids))
		for index, id := range ids {
			idStrings[index] = id.String()
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM api_activity_logs WHERE id = ANY($1::uuid[])`, idStrings)
	}()

	deleted, err := store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: CleanupAPIActivityLogs, Cutoff: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != CleanupBatchSize {
		t.Fatalf("first cleanup batch deleted %d rows, want %d", deleted, CleanupBatchSize)
	}
	deleted, err = store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: CleanupAPIActivityLogs, Cutoff: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("second cleanup batch deleted %d rows, want 1", deleted)
	}
	deleted, err = store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: CleanupAPIActivityLogs, Cutoff: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatalf("repeat cleanup deleted %d rows, want 0", deleted)
	}
	var remaining int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_activity_logs WHERE id=ANY($1::uuid[])`, []string{recentID.String()}).Scan(&remaining); scanErr != nil {
		t.Fatal(scanErr)
	}
	if remaining != 1 {
		t.Fatalf("recent API activity logs remaining = %d, want 1", remaining)
	}

	assetID := uuid.New()
	if insertFileAssetFixtureErr := insertFileAssetFixture(ctx, pool, assetID, cutoff.Add(-time.Second), false); insertFileAssetFixtureErr != nil {
		t.Fatalf("insert old unuploaded FileAsset: %v", insertFileAssetFixtureErr)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM file_assets WHERE id=$1`, assetID)
	}()
	deleted, err = store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: CleanupUnuploadedFileAssets, Cutoff: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("unuploaded FileAsset cleanup affected %d rows, want 1", deleted)
	}
	var deletedAt *time.Time
	if scanErr := pool.QueryRow(ctx, `SELECT deleted_at FROM file_assets WHERE id=$1`, assetID).Scan(&deletedAt); scanErr != nil {
		t.Fatalf("read soft-deleted FileAsset: %v", scanErr)
	}
	if deletedAt == nil {
		t.Fatal("unuploaded FileAsset cleanup must preserve Django's soft-delete semantics")
	}
	deleted, err = store.DeleteCleanupBatch(ctx, CleanupBatchInput{Task: CleanupUnuploadedFileAssets, Cutoff: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatalf("repeat unuploaded FileAsset cleanup affected %d rows, want 0", deleted)
	}
}

func TestPostgresCleanupWorkflowUsesDurableAdmission(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	t.Setenv("API_ACTIVITY_LOG_RETENTION_DAYS", "0")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	store, err := NewPostgresCleanupStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-cleanup-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterCleanupWorkflows(app, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown cleanup runtime: %v", shutdownErr)
		}
	}()

	scheduledAt := time.Now().UTC()
	oldID := uuid.New()
	if insertAPIActivityErr := insertAPIActivityLogFixture(ctx, pool, oldID, scheduledAt.Add(-time.Second)); insertAPIActivityErr != nil {
		t.Fatal(insertAPIActivityErr)
	}
	freshID := uuid.New()
	if insertAPIActivityErr := insertAPIActivityLogFixture(ctx, pool, freshID, scheduledAt.Add(time.Hour)); insertAPIActivityErr != nil {
		t.Fatal(insertAPIActivityErr)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM api_activity_logs WHERE id=ANY($1::uuid[])`, []string{oldID.String(), freshID.String()})
	}()

	instanceID, err := dispatcher.Admit(ctx, CleanupAPIActivityLogs, scheduledAt)
	if err != nil {
		t.Fatalf("admit scheduled cleanup: %v", err)
	}
	result, err := selvedge.WaitWorkflow[CleanupRunOutput](ctx, app, instanceID)
	if err != nil {
		t.Fatalf("wait for durable cleanup workflow: %v", err)
	}
	if result.Status != "completed" || result.Output.Deleted != 1 || result.Output.Batches != 1 {
		t.Fatalf("cleanup result = status %q, output %+v, error %v; want completed, one deletion in one batch", result.Status, result.Output, result.Error)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_activity_logs WHERE id=ANY($1::uuid[])`, []string{oldID.String(), freshID.String()}).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("old and fresh API activity rows remaining = %d, want 1", count)
	}
}

func insertAPIActivityLogFixture(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, createdAt time.Time) error {
	_, err := pool.Exec(ctx, `
INSERT INTO api_activity_logs (
 id, token_identifier, path, method, response_code,
 created_at, updated_at, created_by_id, updated_by_id, deleted_at
) VALUES ($1, $2, $3, $4, $5, $6, $6, NULL, NULL, NULL)`, id, fmt.Sprintf("integration-%s", id), "/api/v1/cleanup-test/", "GET", 200, createdAt)
	return err
}

func insertFileAssetFixture(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, createdAt time.Time, uploaded bool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO file_assets (
 id, created_at, updated_at, attributes, asset, is_deleted, is_archived, size,
 is_uploaded, storage_metadata, deleted_at
) VALUES ($1, $2, $2, '{}'::jsonb, $3, false, false, 0, $4, '{}'::jsonb, NULL)`, id, createdAt, "background-test/"+id.String(), uploaded)
	return err
}
