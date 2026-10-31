//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type attachmentRecoveryHarness struct {
	milestoneFixture
	application *assets.AttachmentApplication
	worker      *assets.AttachmentApplication
	storage     *assets.Storage
	healthy     *atomic.Bool
	fetched     *atomic.Int64
}

func attachmentRecoveryFixture(t *testing.T) attachmentRecoveryHarness {
	t.Helper()
	objectStore := testsupport.StartRustFS(t)
	target, err := url.Parse(objectStore.Endpoint)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	healthy := new(atomic.Bool)
	healthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy.Load() {
			proxy.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		if _, writeErr := io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>injected storage outage</Message></Error>`); writeErr != nil {
			t.Errorf("write storage failure response: %v", writeErr)
		}
	}))
	t.Cleanup(server.Close)
	for key, value := range map[string]string{
		"PLANE_ASSETS_S3_BUCKET": "attachment-recovery", "PLANE_ASSETS_S3_REGION": "us-east-1",
		"PLANE_ASSETS_S3_ACCESS_KEY_ID": objectStore.AccessKey, "PLANE_ASSETS_S3_SECRET_ACCESS_KEY": objectStore.SecretKey,
		"PLANE_ASSETS_S3_INTERNAL_ENDPOINT": server.URL, "PLANE_ASSETS_S3_PUBLIC_ENDPOINT": server.URL,
		"PLANE_ASSETS_S3_PATH_STYLE": "true", "SIGNED_URL_EXPIRATION": "3600",
		"AWS_CONFIG_FILE": "/dev/null", "AWS_SHARED_CREDENTIALS_FILE": "/dev/null", "AWS_PROFILE": "", "AWS_EC2_METADATA_DISABLED": "true",
		"SECRET_KEY": "attachment-recovery-test-secret", "HARD_DELETE_AFTER_DAYS": "60", "WEB_URL": "https://vector.example",
	} {
		t.Setenv(key, value)
	}
	storage, err := assets.NewStorage(assets.StorageConfig{AccessKeyID: objectStore.AccessKey, SecretAccessKey: objectStore.SecretKey, Bucket: "attachment-recovery", Region: "us-east-1", Endpoint: server.URL, PublicEndpoint: server.URL, PathStyle: true})
	require.NoError(t, err)
	require.NoError(t, storage.EnsureBucket(t.Context()))
	f := newMilestoneFixture(t)
	repository, err := assets.NewPostgresRepository(f.pool)
	require.NoError(t, err)
	fetched := new(atomic.Int64)
	fetch := func(context.Context, string, int64) ([]byte, string, error) {
		fetched.Add(1)
		return []byte("Durable attachment contents\n"), "text/plain", nil
	}
	application, err := assets.NewAttachmentApplication(repository, storage, fetch, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	apps := databaseApplications(t, f.pool)
	apps.Attachments = application
	f.service, err = mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	// A fresh worker application can recover using only the persisted intent.
	recovery, err := assets.NewAttachmentApplication(repository, storage, nil, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	return attachmentRecoveryHarness{milestoneFixture: f, application: application, worker: recovery, storage: storage, healthy: healthy, fetched: fetched}
}

func TestMCPAttachmentRecoveryUsesDurableSourceBytes(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		before := h.fetched.Load()
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/recovery.txt", "request_id": uuid.NewString()}
		h.healthy.Store(false)
		milestoneFailure(t, call("workitem_attachment", input), "indeterminate")
		var id string
		var contents []byte
		var completed, uploaded bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT a.id::text,i.contents,i.completed_at IS NOT NULL,a.is_uploaded FROM file_asset_upload_intents i JOIN file_assets a ON a.id=i.asset_id WHERE a.issue_id=$1`, item).Scan(&id, &contents, &completed, &uploaded))
		require.Equal(t, "Durable attachment contents\n", string(contents))
		require.False(t, completed)
		require.False(t, uploaded)
		require.Equal(t, before+1, h.fetched.Load())
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id}), "not_found")
		// The receipt is committed even though the object store is unavailable.
		milestoneFailure(t, call("workitem_attachment", input), "indeterminate")
		require.Equal(t, before+1, h.fetched.Load())
		h.healthy.Store(true)
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, id, replay["attachment_id"])
		require.Equal(t, before+1, h.fetched.Load())
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT i.contents,i.completed_at IS NOT NULL,a.is_uploaded FROM file_asset_upload_intents i JOIN file_assets a ON a.id=i.asset_id WHERE a.id=$1`, id).Scan(&contents, &completed, &uploaded))
		require.Empty(t, contents)
		require.True(t, completed)
		require.True(t, uploaded)
		inline := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
		require.False(t, inline.IsError, inline.Content)
		require.Len(t, inline.Content, 1)
		require.Equal(t, "Durable attachment contents\n", inline.Content[0].Text)
		var activities, metadata int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->>'issue_id'=$1 AND payload->>'type'='attachment.activity.created'),(SELECT count(*) FROM work_item_effects WHERE kind='asset_storage_metadata' AND payload->>'asset_id'=$2)`, item, id).Scan(&activities, &metadata))
		require.Equal(t, 1, activities)
		require.Equal(t, 1, metadata)
		var receipt json.RawMessage
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT result FROM mutation_receipts WHERE request_id=$1`, input["request_id"]).Scan(&receipt))
		require.NotEmpty(t, receipt)
	})
}

func TestMCPAttachmentNativeWorkerRecoversPendingUpload(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	redis := testsupport.StartRedis(t)
	t.Setenv("REDIS_URL", "redis://"+redis.Options().Addr)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		before := h.fetched.Load()
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/worker.txt", "request_id": uuid.NewString()}
		h.healthy.Store(false)
		milestoneFailure(t, call("workitem_attachment", input), "indeterminate")
		var assetID, effectID string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT a.id::text,e.id::text FROM file_assets a JOIN work_item_effects e ON e.kind='asset_storage_upload' AND e.payload->>'asset_id'=a.id::text WHERE a.issue_id=$1`, item).Scan(&assetID, &effectID))
		h.healthy.Store(true)
		runtime, err := backgroundtasks.NewRuntime(t.Context(), f.pool.Config().ConnString(), selvedge.WithActivityPollInterval(20*time.Millisecond), selvedge.WithWorkflowResumptionInterval(20*time.Millisecond))
		require.NoError(t, err)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 20*time.Second)
			defer cancel()
			require.NoError(t, runtime.Shutdown(ctx))
		})
		require.NoError(t, runtime.Start(t.Context()))
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			var uploaded, completed, admitted bool
			var activities int
			queryErr := f.pool.QueryRow(t.Context(), `SELECT a.is_uploaded,i.completed_at IS NOT NULL,e.admitted_at IS NOT NULL,(SELECT count(*) FROM issue_activities WHERE issue_id=$3 AND field='attachment' AND verb='created') FROM file_assets a JOIN file_asset_upload_intents i ON i.asset_id=a.id JOIN work_item_effects e ON e.id=$2 WHERE a.id=$1`, assetID, effectID, item).Scan(&uploaded, &completed, &admitted, &activities)
			assert.NoError(collect, queryErr)
			assert.True(collect, uploaded)
			assert.True(collect, completed)
			assert.True(collect, admitted)
			assert.Equal(collect, 1, activities)
		}, 45*time.Second, 25*time.Millisecond)
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), assetID))
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, assetID, replay["attachment_id"])
		require.Equal(t, before+1, h.fetched.Load())
		inline := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": assetID})
		require.False(t, inline.IsError, inline.Content)
		require.Len(t, inline.Content, 1)
		require.Equal(t, "Durable attachment contents\n", inline.Content[0].Text)

		var key string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT asset FROM file_assets WHERE id=$1`, assetID).Scan(&key))
		_, headErr := h.storage.Head(t.Context(), key)
		require.NoError(t, headErr)
		deletion := map[string]any{"action": "delete", "attachment_id": assetID, "request_id": uuid.NewString()}
		require.Nil(t, milestoneObject(t, call("workitem_attachment", deletion))["result"])
		require.Equal(t, true, milestoneObject(t, call("workitem_attachment", deletion))["replayed"])
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			var created, deleted int
			queryErr := f.pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE verb='created'),count(*) FILTER (WHERE verb='deleted') FROM issue_activities WHERE issue_id=$1 AND field='attachment'`, item).Scan(&created, &deleted)
			assert.NoError(collect, queryErr)
			assert.Equal(collect, 1, created)
			assert.Equal(collect, 1, deleted)
		}, 45*time.Second, 50*time.Millisecond)

		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "read", "attachment_id": assetID}), "not_found")
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "download_url", "attachment_id": assetID}), "not_found")
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
		// Modern attachment deletion retains the stored object with its soft-deleted row.
		_, headErr = h.storage.Head(t.Context(), key)
		require.NoError(t, headErr)
	})
}

func TestMCPAttachmentDeletionCancelsPendingUpload(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		before := h.fetched.Load()
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/cancelled.txt", "request_id": uuid.NewString()}
		h.healthy.Store(false)
		milestoneFailure(t, call("workitem_attachment", input), "indeterminate")
		var id, key string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id::text,asset FROM file_assets WHERE issue_id=$1`, item).Scan(&id, &key))
		deletion := map[string]any{"action": "delete", "attachment_id": id, "request_id": uuid.NewString()}
		require.Nil(t, milestoneObject(t, call("workitem_attachment", deletion))["result"])
		h.healthy.Store(true)
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		var completed, uploaded, deleted bool
		var contents []byte
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT i.completed_at IS NOT NULL,i.contents,a.is_uploaded,a.is_deleted OR a.deleted_at IS NOT NULL FROM file_asset_upload_intents i JOIN file_assets a ON a.id=i.asset_id WHERE a.id=$1`, id).Scan(&completed, &contents, &uploaded, &deleted))
		require.True(t, completed)
		require.Empty(t, contents)
		require.False(t, uploaded)
		require.True(t, deleted)
		_, err := h.storage.Head(t.Context(), key)
		var missing smithy.APIError
		require.ErrorAs(t, err, &missing)
		require.Contains(t, []string{"NotFound", "NoSuchKey", "NoSuchObject", "404"}, missing.ErrorCode())
		milestoneFailure(t, call("workitem_attachment", input), "not_found")
		require.Equal(t, true, milestoneObject(t, call("workitem_attachment", deletion))["replayed"])
		require.Equal(t, before+1, h.fetched.Load())
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
		var creations int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->>'issue_id'=$1 AND payload->>'type'='attachment.activity.created'`, item).Scan(&creations))
		require.Zero(t, creations)
	})
}
