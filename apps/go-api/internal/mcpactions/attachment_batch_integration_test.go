//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Use the deployment's S3 implementation with an isolated, disposable bucket.
// Successful remote downloads use a controlled fetch boundary; private-address
// rejection below calls the production public-file fetcher.
func attachmentBatchStorage(t *testing.T) *assets.Storage {
	t.Helper()
	key, secret := "integration-access", uuid.NewString()
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "rustfs/rustfs:1.0.3@sha256:9ca9559533b537d10482fe923288b676ad08f92a7069d9a17327fbae3cab688b",
			ExposedPorts: []string{"9000/tcp"}, Cmd: []string{"/data"},
			Env:        map[string]string{"RUSTFS_ACCESS_KEY": key, "RUSTFS_SECRET_KEY": secret, "RUSTFS_ADDRESS": ":9000", "RUSTFS_CONSOLE_ENABLE": "false"},
			WaitingFor: wait.ForHTTP("/health/ready").WithPort("9000/tcp").WithStartupTimeout(90 * time.Second),
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })
	endpoint, err := container.PortEndpoint(t.Context(), "9000/tcp", "http")
	require.NoError(t, err)
	storage, err := assets.NewStorage(assets.StorageConfig{AccessKeyID: key, SecretAccessKey: secret, Bucket: "mcp-attachments", Region: "us-east-1", Endpoint: endpoint, PublicEndpoint: endpoint, PathStyle: true})
	require.NoError(t, err)
	require.NoError(t, storage.EnsureBucket(t.Context()))
	return storage
}

func TestMCPAttachmentStorageBatch(t *testing.T) {
	storage := attachmentBatchStorage(t)
	f := newMilestoneFixture(t)
	repository, err := assets.NewPostgresRepository(f.pool)
	require.NoError(t, err)
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aB9sAAAAASUVORK5CYII="
	imageBytes, err := base64.StdEncoding.DecodeString(png)
	require.NoError(t, err)
	var fetched atomic.Int64
	fetch := func(ctx context.Context, location string, limit int64) ([]byte, string, error) {
		switch location {
		case "https://files.example.test/note.txt":
			fetched.Add(1)
			return []byte("Attachment contents\n"), "text/plain", nil
		case "https://files.example.test/pixel.png":
			fetched.Add(1)
			return imageBytes, "image/png", nil
		default:
			return backgroundtasks.FetchPublicFile(ctx, location, limit)
		}
	}
	application, err := assets.NewAttachmentApplication(repository, storage, fetch, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	apps := databaseApplications(t, f.pool)
	apps.Attachments = application
	f.service, err = mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		before := fetched.Load()
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/note.txt", "request_id": uuid.NewString()}
		receipt := milestoneObject(t, call("workitem_attachment", input))
		id := receipt["attachment_id"].(string)
		require.Equal(t, true, receipt["result"].(map[string]any)["is_uploaded"])
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, receipt["result"], replay["result"])
		require.Equal(t, before+1, fetched.Load(), "receipt replay must not fetch the source again")
		rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list", "include_revision": true}))
		require.Len(t, rows, 1)
		require.Equal(t, id, rows[0]["id"])
		require.Equal(t, "note.txt", rows[0]["name"])
		inline := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
		require.False(t, inline.IsError, inline.Content)
		require.Empty(t, inline.StructuredContent)
		require.Len(t, inline.Content, 1)
		require.Equal(t, "text", inline.Content[0].Type)
		require.Equal(t, "Attachment contents\n", inline.Content[0].Text)
		download := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "download_url", "attachment_id": id}))
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, download["download_url"].(string), nil)
		require.NoError(t, err)
		response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
		require.NoError(t, err)
		body, readErr := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, readErr)
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "Attachment contents\n", string(body))
		var completed bool
		var retainedBytes int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT completed_at IS NOT NULL,octet_length(contents) FROM file_asset_upload_intents WHERE asset_id=$1`, id).Scan(&completed, &retainedBytes))
		require.True(t, completed)
		require.Zero(t, retainedBytes)
		imageRow := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/pixel.png"}))
		imageResult := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": imageRow["id"]})
		require.False(t, imageResult.IsError, imageResult.Content)
		require.Empty(t, imageResult.StructuredContent)
		require.Len(t, imageResult.Content, 1)
		require.Equal(t, "image", imageResult.Content[0].Type)
		require.Equal(t, "image/png", imageResult.Content[0].MIMEType)
		require.Equal(t, png, imageResult.Content[0].Data)
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "http://127.0.0.1/private"}), "invalid_params")
		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=5; UPDATE project_members SET role=5`)
		require.NoError(t, err)
		denied := call("workitem_attachment", input)
		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
		require.NoError(t, err)
		milestoneFailure(t, denied, "forbidden")
		deletion := map[string]any{"action": "delete", "attachment_id": id, "request_id": uuid.NewString(), "expected_revision": rows[0]["revision"]}
		deleted := milestoneObject(t, call("workitem_attachment", deletion))
		require.Nil(t, deleted["result"])
		require.Equal(t, true, milestoneObject(t, call("workitem_attachment", deletion))["replayed"])
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id}), "not_found")
		catalogueNull(t, call("workitem_attachment", map[string]any{"action": "delete", "attachment_id": imageRow["id"]}))
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
	})
}
