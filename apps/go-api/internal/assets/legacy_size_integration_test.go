//go:build integration

package assets

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/stretchr/testify/require"
)

func TestLegacyNonfiniteSizeCannotWriteObjectsOrActivity(t *testing.T) {
	f, issue, _ := newPublishedAttachmentFixture(t)
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(http.StatusBadRequest) }))
	t.Cleanup(server.Close)
	storage, err := NewStorage(StorageConfig{AccessKeyID: "test", SecretAccessKey: "test", Bucket: "test", Region: "us-east-1", Endpoint: server.URL, PublicEndpoint: server.URL, PathStyle: true})
	require.NoError(t, err)
	service := &LifecycleService{assets: &Service{repository: f.repository, storage: storage, fileLimit: AttachmentUploadLimit}}
	for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e309"} {
		t.Run(value, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("size", value))
			part, err := writer.CreateFormFile("asset", "notes.txt")
			require.NoError(t, err)
			_, err = io.WriteString(part, "contents")
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			ctx := context.WithValue(t.Context(), assetUserKey{}, f.actor)
			ctx = WithRequestContentType(ctx, writer.FormDataContentType())
			row, err := service.createLegacyIssueAttachment(ctx, "drafts", f.project.String(), issue.String(), io.NopCloser(bytes.NewReader(body.Bytes())))
			var failure *djangoapi.HTTPError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, http.StatusBadRequest, failure.Status)
			require.Equal(t, map[string]any{"size": []string{"A valid number is required."}}, failure.Body)
			require.Nil(t, row)
			var count, effects int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM file_assets),(SELECT count(*) FROM work_item_effects)`).Scan(&count, &effects))
			require.Equal(t, 1, count)
			require.Zero(t, effects)
			require.Zero(t, requests.Load())
		})
	}
}
