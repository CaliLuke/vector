//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentFetchRechecksCurrentCredential(t *testing.T) {
	for _, mode := range []string{"personal token revoked", "OAuth token revoked", "OAuth scope removed"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) { attachmentFetchRevocation(t, mode, version) })
		}
	}
}

func attachmentFetchRevocation(t *testing.T, mode, version string) {
	t.Helper()
	h := attachmentRecoveryFixture(t)
	f := h.milestoneFixture
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	entered, release := make(chan struct{}, 1), make(chan struct{})
	var downloads atomic.Int64
	fetch := func(ctx context.Context, _ string, _ int64) ([]byte, string, error) {
		downloads.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return []byte("Authorized attachment\n"), "text/plain", nil
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}
	repository, err := assets.NewPostgresRepository(f.pool)
	require.NoError(t, err)
	application, err := assets.NewAttachmentApplication(repository, h.storage, fetch, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	apps := databaseApplications(t, f.pool)
	apps.Attachments = application
	f.service, err = mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	handler, token := f.handler(t, nil), "test-one"
	if mode != "personal token revoked" {
		handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
	}
	input := map[string]any{"action": "upload_from_url", "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": f.item(t), "url": "https://files.example.test/revoked.txt", "request_id": uuid.NewString()}
	arguments, err := json.Marshal(input)
	require.NoError(t, err)
	before := attachmentStorageSnapshot(t, f)
	request := oauthRequest(t, version, token, "", "tools/call", map[string]any{"name": "workitem_attachment", "arguments": json.RawMessage(arguments)}).WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { handler.ServeHTTP(response, request); close(done) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("request never reached the source download")
	}
	code := "unauthorized"
	f.issuer.mu.Lock()
	previousScope := f.issuer.scope
	switch mode {
	case "OAuth token revoked":
		f.issuer.active = false
	case "OAuth scope removed":
		f.issuer.scope = "vector:read"
		code = "forbidden"
	}
	f.issuer.mu.Unlock()
	if mode == "personal token revoked" {
		_, err = f.pool.Exec(ctx, `UPDATE api_tokens SET is_active=false WHERE token='test-one'`)
		require.NoError(t, err)
	}
	close(release)
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("request did not finish after releasing the download")
	}
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var envelope struct {
		Result rpcResult
		Error  json.RawMessage
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Empty(t, envelope.Error)
	milestoneFailure(t, envelope.Result, code)
	require.Equal(t, before, attachmentStorageSnapshot(t, f))
	require.Equal(t, int64(1), downloads.Load())
	f.issuer.mu.Lock()
	f.issuer.active, f.issuer.scope = true, previousScope
	f.issuer.mu.Unlock()
	_, err = f.pool.Exec(ctx, `UPDATE api_tokens SET is_active=true WHERE token='test-one'`)
	require.NoError(t, err)
	retry := milestoneObject(t, oauthToolCall(t, handler, version, token, "", "workitem_attachment", string(arguments), http.StatusOK))
	require.Equal(t, false, retry["replayed"])
	require.Equal(t, int64(2), downloads.Load())
}
