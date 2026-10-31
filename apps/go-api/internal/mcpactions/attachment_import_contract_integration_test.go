//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentImportNamesTypesAndLimits(t *testing.T) {
	cases := []struct {
		name, url, override, mime, wantName, wantType string
		size                                          int
		rejected                                      bool
	}{
		{"explicit name", "https://files.example.test/source", "chosen.txt", "text/plain; charset=utf-8", "chosen.txt", "text/plain", 17, false},
		{"path name", "https://files.example.test/report.txt?download=1#part", "", "text/plain", "report.txt", "text/plain", 17, false},
		{"trailing slash", "https://files.example.test/report.txt/", "", "text/plain", "report.txt", "text/plain", 17, false},
		{"empty path", "https://files.example.test", "", "text/plain", "attachment", "text/plain", 17, false},
		{"root path", "https://files.example.test/", "", "text/plain", "attachment", "text/plain", 17, false},
		{"escaped space", "https://files.example.test/report%20v2.txt", "", "text/plain", "report%20v2.txt", "text/plain", 17, false},
		{"escaped slash", "https://files.example.test/folder%2Freport.txt", "", "text/plain", "folder%2Freport.txt", "text/plain", 17, false},
		{"raw unicode", "https://files.example.test/café.txt", "", "text/plain", "café.txt", "text/plain", 17, false},
		{"empty header inference", "https://files.example.test/readme.md", "", "", "readme.md", "text/markdown", 17, false},
		{"octet header inference", "https://files.example.test/data.json", "", "application/octet-stream", "data.json", "application/json", 17, false},
		{"binary fallback", "https://files.example.test/unknown", "", "", "unknown", "application/octet-stream", 17, false},
		{"native name sanitization", "https://files.example.test/sanitized", "../private/report.txt", "text/plain", "report.txt", "text/plain", 17, false},
		{"empty content", "https://files.example.test/empty.txt", "", "text/plain", "", "", 0, true},
		{"upload limit", "https://files.example.test/limit.txt", "", "text/plain", "limit.txt", "text/plain", int(assets.AttachmentUploadLimit), false},
		{"upload too large", "https://files.example.test/over.txt", "", "text/plain", "", "", int(assets.AttachmentUploadLimit) + 1, true},
		{"invalid path escape", "https://files.example.test/bad%zz.txt", "", "text/plain", "", "", 17, true},
		{"native type policy", "https://files.example.test/run.exe", "", "application/x-executable", "", "", 17, true},
	}
	h := attachmentRecoveryFixture(t)
	var fetched atomic.Int64
	fetch := func(_ context.Context, location string, limit int64) ([]byte, string, error) {
		if limit != assets.AttachmentUploadLimit {
			return nil, "", fmt.Errorf("unexpected fetch limit %d", limit)
		}
		for _, tc := range cases {
			if tc.url == location {
				fetched.Add(1)
				return []byte(strings.Repeat("a", tc.size)), tc.mime, nil
			}
		}
		return nil, "", fmt.Errorf("unexpected source %q", location)
	}
	repository, err := assets.NewPostgresRepository(h.pool)
	require.NoError(t, err)
	application, err := assets.NewAttachmentApplication(repository, h.storage, fetch, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	apps := databaseApplications(t, h.pool)
	apps.Attachments = application
	h.service, err = mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": f.item(t)})
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				before, downloads := attachmentStorageSnapshot(t, f), fetched.Load()
				result := call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": tc.url, "name": tc.override})
				require.Equal(t, downloads+1, fetched.Load())
				if tc.rejected {
					milestoneFailure(t, result, "invalid_params")
					require.Equal(t, before, attachmentStorageSnapshot(t, f))
					return
				}
				created := milestoneObject(t, result)
				assert.Equal(t, tc.wantName, created["name"])
				require.Equal(t, tc.wantType, created["content_type"])
				require.Equal(t, float64(tc.size), created["size"])
				if tc.size == int(assets.AttachmentUploadLimit) {
					require.Equal(t, int64(tc.size), downloadedAttachmentSize(t, call, created["id"].(string)))
				}
			})
		}
	})
}
