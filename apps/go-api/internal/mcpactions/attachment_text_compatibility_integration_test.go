//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentReadPreservesUTF8ReplacementBoundaries(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		created := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/utf8.txt"}))
		id := created["id"].(string)
		var key string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT asset FROM file_assets WHERE id=$1`, id).Scan(&key))
		// The pinned handler uses UTF-8 decoding with errors="replace".
		// Each malformed subsequence has its own replacement character.
		for _, tc := range []struct{ name, body, want string }{
			{"valid unicode", "A😀Ω", "A😀Ω"},
			{"byte order mark", "\xef\xbb\xbftext", "\uFEFFtext"},
			{"empty text", "", ""},
			{"adjacent invalid bytes", "a\xff\xfex", "a\uFFFD\uFFFDx"},
			{"truncated sequence", "\xe2\x82", "\uFFFD"},
			{"surrogate encoding", "\xed\xa0\x80", "\uFFFD\uFFFD\uFFFD"},
			{"above unicode range", "\xf4\x90\x80\x80", "\uFFFD\uFFFD\uFFFD\uFFFD"},
			{"overlong encoding", "\xc0\xaf", "\uFFFD\uFFFD"},
			{"interrupted sequence", "\xe2(\xa1", "\uFFFD(\uFFFD"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, h.storage.Put(t.Context(), key, "text/plain", []byte(tc.body)))
				_, err := f.pool.Exec(t.Context(), `UPDATE file_assets SET size=$2::bigint,attributes=jsonb_set(attributes,'{size}',to_jsonb($2::bigint)) WHERE id=$1`, id, len(tc.body))
				require.NoError(t, err)
				result := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
				require.False(t, result.IsError, result.Content)
				require.Empty(t, result.StructuredContent)
				require.Len(t, result.Content, 1)
				require.Equal(t, "text", result.Content[0].Type)
				require.Equal(t, tc.want, result.Content[0].Text)
			})
		}
	})
}
