package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type metadataStoreFake struct {
	mu       sync.Mutex
	key      string
	found    bool
	metadata any
	saves    int
}

func (store *metadataStoreFake) AssetKey(context.Context, uuid.UUID) (string, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.key, store.found, nil
}

func (store *metadataStoreFake) SaveMetadataForKey(_ context.Context, _ uuid.UUID, key string, metadata any) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if key != store.key {
		return errors.New("unexpected object key")
	}
	store.metadata = metadata
	store.saves++
	return nil
}

func (store *metadataStoreFake) snapshot() (int, any) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saves, store.metadata
}

type metadataReaderFunc func(context.Context, string) (*s3.HeadObjectOutput, error)

func (reader metadataReaderFunc) Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	return reader(ctx, key)
}

func TestMetadataFromHeadMatchesSourceShape(t *testing.T) {
	modified := time.Date(2026, 10, 19, 12, 13, 14, 123000000, time.UTC)
	actual := metadataFromHead(&s3.HeadObjectOutput{
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(68),
		LastModified:  &modified,
		ETag:          aws.String("\"etag\""),
		Metadata:      map[string]string{"source": "fixture"},
	})
	want := map[string]any{
		"ContentType":   "image/png",
		"ContentLength": int64(68),
		"LastModified":  "2026-10-19T12:13:14.123000+00:00",
		"ETag":          "\"etag\"",
		"Metadata":      map[string]string{"source": "fixture"},
	}
	if !equalJSONValue(actual, want) {
		t.Fatalf("metadata = %#v, want %#v", actual, want)
	}
	missing := metadataFromHead(&s3.HeadObjectOutput{})
	if missing["ContentType"] != nil || missing["ContentLength"] != nil || missing["LastModified"] != nil || missing["ETag"] != nil {
		t.Fatalf("missing S3 headers should remain null: %#v", missing)
	}
	if got := missing["Metadata"]; got == nil || len(got.(map[string]string)) != 0 {
		t.Fatalf("missing user metadata = %#v, want empty object", got)
	}
}

func TestStorageMetadataWorkflowMissingAssetDoesNotCallS3(t *testing.T) {
	store := &metadataStoreFake{key: "missing", found: false}
	var heads atomic.Int32
	result := runMetadataWorkflow(t, store, metadataReaderFunc(func(context.Context, string) (*s3.HeadObjectOutput, error) {
		heads.Add(1)
		return nil, errors.New("S3 should not be called for missing assets")
	}), retry.NoRetry(), "missing")
	saves, _ := store.snapshot()
	if result.Status != "completed" || saves != 0 || heads.Load() != 0 {
		t.Fatalf("status=%s saves=%d heads=%d; want completed no-op", result.Status, saves, heads.Load())
	}
}

func TestStorageMetadataWorkflowRetriesTransientS3Failure(t *testing.T) {
	store := &metadataStoreFake{key: "fixture/object.png", found: true}
	var attempts atomic.Int32
	reader := metadataReaderFunc(func(context.Context, string) (*s3.HeadObjectOutput, error) {
		if attempts.Add(1) == 1 {
			return nil, &smithy.GenericAPIError{Code: "InternalError", Fault: smithy.FaultServer}
		}
		return &s3.HeadObjectOutput{ContentType: aws.String("image/png"), ContentLength: aws.Int64(4)}, nil
	})
	policy := retry.Fixed(3, time.Millisecond)
	result := runMetadataWorkflow(t, store, reader, policy, "transient")
	saves, _ := store.snapshot()
	if result.Status != "completed" || attempts.Load() != 2 || saves != 1 {
		t.Fatalf("status=%s attempts=%d saves=%d; want completed on second attempt", result.Status, attempts.Load(), saves)
	}
}

func TestStorageMetadataWorkflowBoundsBlockedS3Read(t *testing.T) {
	store := &metadataStoreFake{key: "fixture/blocked.png", found: true}
	var attempts, cancelled atomic.Int32
	activityCancelled := make(chan time.Duration, 1)
	startedAt := time.Now()
	reader := metadataReaderFunc(func(ctx context.Context, _ string) (*s3.HeadObjectOutput, error) {
		attempts.Add(1)
		<-ctx.Done()
		cancelled.Add(1)
		activityCancelled <- time.Since(startedAt)
		return nil, ctx.Err()
	})
	app := selvedge.NewApp(
		selvedge.WithDatabase("file:"+filepath.Join(t.TempDir(), "workflow.db")),
		selvedge.WithWaitPollInterval(5*time.Millisecond),
		selvedge.WithActivityPollInterval(5*time.Millisecond),
		selvedge.WithTimerCheckInterval(5*time.Millisecond),
	)
	workflow, err := registerStorageMetadataWorkflow(app, store, reader, retry.NoRetry(), 250*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if startErr := app.Start(context.Background()); startErr != nil {
		t.Fatal(startErr)
	}
	t.Cleanup(func() { _ = app.Shutdown(context.Background()) })
	runID := "asset-metadata:blocked-read"
	if _, startWorkflowErr := selvedge.StartWorkflow(ctx, app, workflow,
		StorageMetadataWorkflowInput{AssetID: uuid.NewString(), EventID: "blocked-read"},
		selvedge.WithInstanceID(runID),
	); startWorkflowErr != nil {
		t.Fatal(startWorkflowErr)
	}
	select {
	case elapsed := <-activityCancelled:
		t.Logf("blocked S3 reader context canceled after %s", elapsed)
	case <-time.After(time.Second):
		t.Fatal("blocked S3 reader was not canceled within one second")
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, runID)
	if err != nil {
		activityTasks, taskErr := selvedge.ListActivityTasks(context.Background(), app, runID)
		workflowResult, resultErr := selvedge.GetWorkflowResult[struct{}](context.Background(), app, runID)
		t.Fatalf("wait for bounded S3 read attempts=%d cancelled=%d: %v; tasks=%+v task_error=%v result=%+v result_error=%v", attempts.Load(), cancelled.Load(), err, activityTasks, taskErr, workflowResult, resultErr)
	}
	if result.Status != "failed" || !errors.Is(result.Error, context.DeadlineExceeded) {
		t.Fatalf("blocked read result = status %s, error %v; want deadline failure after retry exhaustion", result.Status, result.Error)
	}
	if attempts.Load() != 1 || cancelled.Load() != 1 {
		t.Fatalf("blocked reads attempts=%d cancelled=%d; want one deadline-cancelled attempt", attempts.Load(), cancelled.Load())
	}
	saves, _ := store.snapshot()
	if saves != 0 {
		t.Fatalf("blocked read saved metadata %d times; want no write", saves)
	}
}

func TestClassifyStorageMetadataErrorKeepsPermanentFailuresOutOfRetry(t *testing.T) {
	permanentS3 := classifyStorageMetadataError(&smithy.GenericAPIError{Code: "AccessDenied", Fault: smithy.FaultClient})
	if !errors.Is(permanentS3, ErrPermanentStorageMetadata) {
		t.Fatalf("AccessDenied classification = %v, want permanent", permanentS3)
	}
	throttledS3 := classifyStorageMetadataError(&smithy.GenericAPIError{Code: "ThrottlingException", Fault: smithy.FaultClient})
	if errors.Is(throttledS3, ErrPermanentStorageMetadata) {
		t.Fatalf("ThrottlingException classification = %v, want retryable", throttledS3)
	}
	permanentDB := classifyStorageMetadataError(&pgconn.PgError{Code: "23505", Message: "unique violation"})
	if !errors.Is(permanentDB, ErrPermanentStorageMetadata) {
		t.Fatalf("unique violation classification = %v, want permanent", permanentDB)
	}
	transientDB := classifyStorageMetadataError(&pgconn.PgError{Code: "40001", Message: "serialization failure"})
	if errors.Is(transientDB, ErrPermanentStorageMetadata) {
		t.Fatalf("serialization failure classification = %v, want retryable", transientDB)
	}
}

func TestStorageMetadataWorkflowStoresNullForMissingObject(t *testing.T) {
	store := &metadataStoreFake{key: "fixture/missing.png", found: true}
	reader := metadataReaderFunc(func(context.Context, string) (*s3.HeadObjectOutput, error) {
		return nil, &smithy.GenericAPIError{Code: "NoSuchKey", Fault: smithy.FaultClient}
	})
	result := runMetadataWorkflow(t, store, reader, retry.NoRetry(), "missing-object")
	saves, metadata := store.snapshot()
	if result.Status != "completed" || saves != 1 || metadata != nil {
		t.Fatalf("status=%s saves=%d metadata=%#v; want completed with JSON null", result.Status, saves, metadata)
	}
}

func TestStorageMetadataWorkflowDuplicateRunIDIsRejected(t *testing.T) {
	store := &metadataStoreFake{key: "fixture/object.png", found: true}
	reader := metadataReaderFunc(func(context.Context, string) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{}, nil
	})
	databaseURL := "file:" + filepath.Join(t.TempDir(), "workflow.db")
	app := selvedge.NewApp(selvedge.WithDatabase(databaseURL), selvedge.WithWaitPollInterval(5*time.Millisecond))
	workflow, err := RegisterStorageMetadataWorkflow(app, store, reader, retry.NoRetry())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Shutdown(context.Background()) }()
	input := StorageMetadataWorkflowInput{AssetID: uuid.NewString(), EventID: "duplicate"}
	if _, err := selvedge.StartWorkflow(ctx, app, workflow, input, selvedge.WithInstanceID("asset-metadata:duplicate")); err != nil {
		t.Fatal(err)
	}
	if _, err := selvedge.StartWorkflow(ctx, app, workflow, input, selvedge.WithInstanceID("asset-metadata:duplicate")); !errors.Is(err, selvedge.ErrWorkflowAlreadyStarted) {
		t.Fatalf("duplicate start error = %v, want ErrWorkflowAlreadyStarted", err)
	}
}

func TestStorageMetadataAdmissionJoinsSelvedgeTransaction(t *testing.T) {
	store := &metadataStoreFake{found: false}
	app := selvedge.NewApp(
		selvedge.WithDatabase("file:"+filepath.Join(t.TempDir(), "workflow.db")),
		selvedge.WithWaitPollInterval(5*time.Millisecond),
		selvedge.WithActivityPollInterval(5*time.Millisecond),
	)
	workflow, err := RegisterStorageMetadataWorkflow(app, store, metadataReaderFunc(func(context.Context, string) (*s3.HeadObjectOutput, error) {
		return nil, errors.New("missing asset must not reach S3")
	}), retry.NoRetry())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Shutdown(context.Background()) })

	storage := app.Storage()
	if _, err := storage.Conn(ctx).ExecContext(ctx, `CREATE TABLE admission_probe (event_id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	for _, commit := range []bool{false, true} {
		eventID := "rollback"
		if commit {
			eventID = "commit"
		}
		runID := "asset-metadata:atomic:" + eventID
		txCtx, err := storage.BeginTransaction(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := storage.Conn(txCtx).ExecContext(txCtx, `INSERT INTO admission_probe(event_id) VALUES(?)`, eventID); err != nil {
			_ = storage.RollbackTransaction(txCtx)
			t.Fatal(err)
		}
		if _, err := selvedge.StartWorkflow(txCtx, app, workflow,
			StorageMetadataWorkflowInput{AssetID: uuid.NewString(), EventID: eventID},
			selvedge.WithInstanceID(runID),
		); err != nil {
			_ = storage.RollbackTransaction(txCtx)
			t.Fatal(err)
		}
		if commit {
			if err := storage.CommitTransaction(txCtx); err != nil {
				t.Fatal(err)
			}
			result, err := selvedge.WaitWorkflow[struct{}](ctx, app, runID)
			if err != nil || result.Status != "completed" {
				t.Fatalf("committed admission status=%v err=%v", result, err)
			}
		} else if err := storage.RollbackTransaction(txCtx); err != nil {
			t.Fatal(err)
		}
		var eventRows, workflowRows int
		if err := storage.Conn(ctx).QueryRowContext(ctx, `SELECT COUNT(*) FROM admission_probe WHERE event_id=?`, eventID).Scan(&eventRows); err != nil {
			t.Fatal(err)
		}
		if err := storage.Conn(ctx).QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_instances WHERE instance_id=?`, runID).Scan(&workflowRows); err != nil {
			t.Fatal(err)
		}
		want := 0
		if commit {
			want = 1
		}
		if eventRows != want || workflowRows != want {
			t.Fatalf("commit=%t event_rows=%d workflow_rows=%d; want both %d", commit, eventRows, workflowRows, want)
		}
	}
}

func runMetadataWorkflow(t *testing.T, store storageMetadataStore, reader objectMetadataReader, policy *retry.Policy, suffix string) *selvedge.WorkflowResult[struct{}] {
	t.Helper()
	app := selvedge.NewApp(
		selvedge.WithDatabase("file:"+filepath.Join(t.TempDir(), "workflow.db")),
		selvedge.WithWaitPollInterval(5*time.Millisecond),
		selvedge.WithActivityPollInterval(5*time.Millisecond),
	)
	workflow, err := RegisterStorageMetadataWorkflow(app, store, reader, policy)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		defer shutdownCancel()
		_ = app.Shutdown(shutdownCtx)
	})
	runID := "asset-metadata:" + suffix
	if _, startWorkflowErr := selvedge.StartWorkflow(ctx, app, workflow, StorageMetadataWorkflowInput{AssetID: uuid.NewString(), EventID: suffix}, selvedge.WithInstanceID(runID)); startWorkflowErr != nil {
		t.Fatal(startWorkflowErr)
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, runID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != nil {
		t.Fatalf("workflow status=%s: %v", result.Status, result.Error)
	}
	return result
}

func equalJSONValue(got, want any) bool {
	gotJSON, gotErr := json.Marshal(got)
	wantJSON, wantErr := json.Marshal(want)
	return gotErr == nil && wantErr == nil && bytes.Equal(gotJSON, wantJSON)
}
