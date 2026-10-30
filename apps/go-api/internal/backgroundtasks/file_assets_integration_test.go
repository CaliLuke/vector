//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	assetdomain "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func TestPostgresFileAssetMetadataWorkflowUsesStableAdmission(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	assetID := uuid.New()
	key := "background-test/" + assetID.String() + ".png"
	if _, err := pool.Exec(ctx, `
INSERT INTO file_assets (
 id, created_at, updated_at, attributes, asset, is_deleted, is_archived, size,
 is_uploaded, storage_metadata, deleted_at
) VALUES ($1, clock_timestamp(), clock_timestamp(), '{}'::jsonb, $2, false, false, 68, true, '{}'::jsonb, NULL)`, assetID, key); err != nil {
		t.Fatalf("insert FileAsset fixture: %v", err)
	}

	repository, err := assetdomain.NewPostgresRepository(pool)
	if err != nil {
		t.Fatal(err)
	}
	workflowApp := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-file-asset-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	workflow, err := assetdomain.RegisterStorageMetadataWorkflow(workflowApp, repository, backgroundMetadataReader{}, retry.NoRetry())
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewFileAssetMetadataDispatcher(workflowApp, workflow)
	if err != nil {
		t.Fatal(err)
	}
	if startErr := workflowApp.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := workflowApp.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown FileAsset metadata runtime: %v", shutdownErr)
		}
	}()

	eventID := uuid.NewString()
	instanceID, err := dispatcher.Admit(ctx, assetID.String(), eventID)
	if err != nil {
		t.Fatalf("admit FileAsset metadata workflow: %v", err)
	}
	duplicateID, err := dispatcher.Admit(ctx, assetID.String(), eventID)
	if err != nil {
		t.Fatalf("reconcile duplicate FileAsset metadata admission: %v", err)
	}
	if duplicateID != instanceID {
		t.Fatalf("duplicate metadata admission returned %q, want stable ID %q", duplicateID, instanceID)
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, workflowApp, instanceID)
	if err != nil {
		t.Fatalf("wait for FileAsset metadata workflow: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("FileAsset metadata workflow status = %q, error = %v", result.Status, result.Error)
	}
	var rawMetadata []byte
	if err := pool.QueryRow(ctx, `SELECT storage_metadata FROM file_assets WHERE id=$1`, assetID).Scan(&rawMetadata); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(rawMetadata, &got); err != nil {
		t.Fatalf("decode persisted FileAsset metadata %s: %v", rawMetadata, err)
	}
	if got["ContentType"] != "image/png" || got["ContentLength"] != float64(68) || got["ETag"] != `"metadata-etag"` {
		t.Fatalf("persisted S3 metadata = %#v", got)
	}
}

type backgroundMetadataReader struct{}

func (backgroundMetadataReader) Head(_ context.Context, key string) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(68),
		ETag:          aws.String(`"metadata-etag"`),
		Metadata:      map[string]string{"source": "integration"},
		LastModified:  aws.Time(time.Date(2026, 10, 21, 12, 30, 0, 0, time.UTC)),
	}, nil
}
