//go:build integration

package mcpactions_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentInlineTypesAndSizeBoundaries(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		for _, tc := range []struct {
			name, mime, filename, kind string
			size                       int
			rejected                   bool
		}{
			{"plain text", "text/plain", "file.txt", "text", 17, false},
			{"markdown", "text/markdown", "file.md", "text", 17, false},
			{"csv", "text/csv", "file.csv", "text", 17, false},
			{"html", "text/html", "file.html", "text", 17, false},
			{"text XML", "text/xml", "file.xml", "text", 17, false},
			{"text YAML", "text/yaml", "file.yaml", "text", 17, false},
			{"JSON", "application/json", "file.json", "text", 17, false},
			{"application XML", "application/xml", "file.xml", "text", 17, false},
			{"application YAML", "application/yaml", "file.yaml", "text", 17, false},
			{"legacy YAML", "application/x-yaml", "file.yaml", "text", 17, false},
			{"PNG", "image/png", "file.png", "image", 17, false},
			{"JPEG", "image/jpeg", "file.jpg", "image", 17, false},
			{"GIF", "image/gif", "file.gif", "image", 17, false},
			{"WEBP", "image/webp", "file.webp", "image", 17, false},
			{"infer empty MIME", "", "file.txt", "text", 17, false},
			{"infer octet MIME", "application/octet-stream", "file.md", "text", 17, false},
			{"text limit", "text/plain", "file.txt", "text", int(assets.AttachmentTextLimit), false},
			{"text too large", "text/plain", "file.txt", "text", int(assets.AttachmentTextLimit) + 1, true},
			{"image limit", "image/png", "file.png", "image", int(assets.AttachmentUploadLimit), false},
			{"image too large", "image/png", "file.png", "image", int(assets.AttachmentUploadLimit) + 1, true},
			{"PDF download only", "application/pdf", "file.pdf", "", 17, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": f.item(t)})
				created := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/content.txt"}))
				id := created["id"].(string)
				var key string
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT asset FROM file_assets WHERE id=$1`, id).Scan(&key))
				body := []byte(strings.Repeat("a", tc.size))
				require.NoError(t, h.storage.Put(t.Context(), key, tc.mime, body))
				_, err := f.pool.Exec(t.Context(), `UPDATE file_assets SET size=$2::bigint,attributes=jsonb_build_object('name',$3::text,'type',$4::text,'size',$2::bigint) WHERE id=$1`, id, tc.size, tc.filename, tc.mime)
				require.NoError(t, err)
				result := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
				if tc.rejected {
					milestoneFailure(t, result, "invalid_params")
					require.Equal(t, int64(tc.size), downloadedAttachmentSize(t, call, id))
					return
				}
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.StructuredContent)
				require.Len(t, result.Content, 1)
				require.Equal(t, tc.kind, result.Content[0].Type)
				if tc.kind == "text" {
					require.True(t, string(body) == result.Content[0].Text, "inline text bytes changed")
				} else {
					decoded, decodeErr := base64.StdEncoding.DecodeString(result.Content[0].Data)
					require.NoError(t, decodeErr)
					require.True(t, bytes.Equal(body, decoded), "inline image bytes changed")
					require.Equal(t, tc.mime, result.Content[0].MIMEType)
				}
			})
		}
	})
}

func downloadedAttachmentSize(t *testing.T, call catalogueCall, id string) int64 {
	t.Helper()
	download := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "download_url", "attachment_id": id}))
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, download["download_url"].(string), nil)
	require.NoError(t, err)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	require.NoError(t, err)
	length, readErr := io.Copy(io.Discard, response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, readErr)
	require.Equal(t, http.StatusOK, response.StatusCode)
	return length
}
