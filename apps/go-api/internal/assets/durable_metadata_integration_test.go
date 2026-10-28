package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	crashProofChildEnv  = "PLANE_DURABLE_METADATA_CRASH_CHILD"
	crashProofMarkerEnv = "PLANE_DURABLE_METADATA_CRASH_MARKER"
	crashProofEventEnv  = "PLANE_DURABLE_METADATA_CRASH_EVENT_ID"
)

// TestStorageMetadataCrashAfterWriteBeforeCompletion exercises the actual
// PostgreSQL repository and object store in disposable fixtures. The child
// process stops after committing metadata so SIGKILL cannot run cleanup code.
func TestStorageMetadataCrashAfterWriteBeforeCompletion(t *testing.T) {
	if os.Getenv(crashProofChildEnv) == "1" {
		assetID, err := uuid.Parse(os.Getenv("PLANE_DURABLE_METADATA_ASSET_ID"))
		if err != nil {
			t.Fatal(err)
		}
		runMetadataUntilSavedAndBlocked(t, os.Getenv("PLANE_DURABLE_METADATA_DATABASE_URL"), assetID, os.Getenv(crashProofEventEnv))
		return
	}
	databaseURL, assetID := metadataCrashFixture(t)
	eventID := "crash-after-write-proof-" + uuid.NewString()
	runID := fmt.Sprintf("asset-metadata:%s:%s", assetID, eventID)
	t.Logf("crash proof workflow run_id=%s", runID)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to isolated database: %v", err)
	}
	defer pool.Close()
	var activeLeaders int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM system_locks WHERE lock_name='selvedge_leader' AND lock_expires_at > NOW()`).Scan(&activeLeaders); scanErr != nil {
		t.Fatalf("check for another Selvedge worker using the proof database: %v", scanErr)
	}
	if activeLeaders != 0 {
		t.Fatalf("proof database has %d live Selvedge leader lease(s); stop other workers before the crash proof", activeLeaders)
	}
	marker := filepath.Join(t.TempDir(), "metadata-saved")
	child := exec.Command(os.Args[0], "-test.run=^TestStorageMetadataCrashAfterWriteBeforeCompletion$")
	child.Env = append(os.Environ(), crashProofChildEnv+"=1", crashProofMarkerEnv+"="+marker, crashProofEventEnv+"="+eventID)
	var output bytes.Buffer
	child.Stdout = &output
	child.Stderr = &output
	if startErr := child.Start(); startErr != nil {
		t.Fatalf("start activity subprocess: %v", startErr)
	}
	childDone := make(chan error, 1)
	go func() { childDone <- child.Wait() }()
	childWaited := false
	defer func() {
		if !childWaited {
			_ = child.Process.Kill()
			if cleanupWaitErr := <-childDone; cleanupWaitErr != nil {
				var exitErr *exec.ExitError
				if !errors.As(cleanupWaitErr, &exitErr) {
					t.Errorf("wait for activity subprocess cleanup: %v", cleanupWaitErr)
				}
			}
		}
	}()
	markerTicker := time.NewTicker(20 * time.Millisecond)
	defer markerTicker.Stop()
	for {
		if _, statErr := os.Stat(marker); statErr == nil {
			break
		} else if !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("stat committed metadata marker: %v", statErr)
		}
		select {
		case childErr := <-childDone:
			childWaited = true
			t.Fatalf("activity subprocess exited before write marker: %v; output: %s", childErr, output.String())
		case <-ctx.Done():
			t.Fatalf("wait for committed metadata marker: %v; child output: %s", ctx.Err(), output.String())
		case <-markerTicker.C:
		}
	}

	metadataBefore, err := readStorageMetadata(ctx, pool, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadataBefore) == 0 || string(metadataBefore) == "null" {
		t.Fatalf("write marker appeared without persisted object metadata: %s", metadataBefore)
	}
	if killErr := child.Process.Kill(); killErr != nil {
		t.Fatalf("SIGKILL activity subprocess after database commit: %v", killErr)
	}
	childErr := <-childDone
	childWaited = true
	if childErr == nil {
		t.Fatal("activity subprocess exited successfully; expected SIGKILL during activity")
	}

	store, err := NewPostgresRepository(pool)
	if err != nil {
		t.Fatal(err)
	}
	config, err := StorageConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	objects, err := NewStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(databaseURL),
		selvedge.WithActivityPollInterval(25*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(25*time.Millisecond),
	)
	_, err = RegisterStorageMetadataWorkflow(app, store, objects, retry.Exponential(8, 10*time.Millisecond, 100*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown restarted metadata runtime: %v", shutdownErr)
		}
	})
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, runID)
	if err != nil {
		t.Fatalf("wait for restarted activity: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("restarted workflow status = %s, want completed: %v", result.Status, result.Error)
	}
	metadataAfter, err := readStorageMetadata(ctx, pool, assetID)
	if err != nil {
		t.Fatal(err)
	}
	if !equalJSONValue(json.RawMessage(metadataAfter), json.RawMessage(metadataBefore)) {
		t.Fatalf("metadata changed after activity replay: before=%s after=%s", metadataBefore, metadataAfter)
	}
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT attempts FROM workflow_activity_tasks WHERE instance_id=$1 AND activity_name=$2`, runID, StorageMetadataActivityName).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatalf("activity attempts = %d, want a replay after SIGKILL", attempts)
	}
}

// metadataCrashFixture owns both services and the committed object that the
// subprocess must observe. Its runtime startup installs the real execution schema.
func metadataCrashFixture(t *testing.T) (string, uuid.UUID) {
	t.Helper()
	databaseURL := postgresfixture.EmptyDatabase(t)
	storage := testsupport.StartRustFS(t)
	storage.ConfigureAssets(t, "metadata-crash-proof")
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, execErr := pool.Exec(ctx, `CREATE TABLE file_assets(id uuid PRIMARY KEY, asset text NOT NULL, storage_metadata jsonb, deleted_at timestamptz)`); execErr != nil {
		t.Fatal(execErr)
	}
	assetID := uuid.New()
	const key = "crash-proof/object.txt"
	if _, execErr := pool.Exec(ctx, `INSERT INTO file_assets(id,asset) VALUES($1,$2)`, assetID, key); execErr != nil {
		t.Fatal(execErr)
	}
	config, err := StorageConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	objects, err := NewStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	if bucketErr := objects.EnsureBucket(ctx); bucketErr != nil {
		t.Fatal(bucketErr)
	}
	if putErr := objects.Put(ctx, key, "text/plain", []byte("committed metadata proof")); putErr != nil {
		t.Fatal(putErr)
	}
	app := selvedge.NewApp(selvedge.WithDatabase(databaseURL))
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	if shutdownErr := app.Shutdown(ctx); shutdownErr != nil {
		t.Fatal(shutdownErr)
	}
	t.Setenv("PLANE_DURABLE_METADATA_DATABASE_URL", databaseURL)
	t.Setenv("PLANE_DURABLE_METADATA_ASSET_ID", assetID.String())
	return databaseURL, assetID
}

type metadataWriteBlocker struct {
	store  *PostgresRepository
	marker string
}

func (blocker metadataWriteBlocker) AssetKey(ctx context.Context, assetID uuid.UUID) (string, bool, error) {
	return blocker.store.AssetKey(ctx, assetID)
}

func (blocker metadataWriteBlocker) SaveMetadataForKey(ctx context.Context, assetID uuid.UUID, key string, metadata any) error {
	if err := blocker.store.SaveMetadataForKey(ctx, assetID, key, metadata); err != nil {
		return err
	}
	if err := os.WriteFile(blocker.marker, []byte("committed\n"), 0o600); err != nil {
		return fmt.Errorf("write proof marker after metadata commit: %w", err)
	}
	select {}
}

func runMetadataUntilSavedAndBlocked(t *testing.T, databaseURL string, assetID uuid.UUID, eventID string) {
	t.Helper()
	marker := os.Getenv(crashProofMarkerEnv)
	if marker == "" || eventID == "" {
		t.Fatal("crash proof marker path and event ID are required in child")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store, err := NewPostgresRepository(pool)
	if err != nil {
		t.Fatal(err)
	}
	config, err := StorageConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	objects, err := NewStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(selvedge.WithDatabase(databaseURL), selvedge.WithActivityPollInterval(25*time.Millisecond))
	workflow, err := RegisterStorageMetadataWorkflow(app, metadataWriteBlocker{store: store, marker: marker}, objects, retry.Exponential(8, time.Second, time.Minute, 2))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := selvedge.StartWorkflow(ctx, app, workflow,
		StorageMetadataWorkflowInput{AssetID: assetID.String(), EventID: eventID},
		selvedge.WithInstanceID(fmt.Sprintf("asset-metadata:%s:%s", assetID, eventID)),
	); err != nil {
		t.Fatal(err)
	}
	select {}
}

func readStorageMetadata(ctx context.Context, pool *pgxpool.Pool, assetID uuid.UUID) ([]byte, error) {
	var metadata []byte
	if err := pool.QueryRow(ctx, `SELECT storage_metadata FROM file_assets WHERE id=$1`, assetID).Scan(&metadata); err != nil {
		return nil, fmt.Errorf("read FileAsset %s storage metadata: %w", assetID, err)
	}
	return metadata, nil
}
