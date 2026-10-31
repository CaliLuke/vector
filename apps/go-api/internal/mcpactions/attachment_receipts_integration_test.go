//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentReceiptsAndNativeRevisions(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	f := h.milestoneFixture
	repository, err := assets.NewPostgresRepository(f.pool)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/receipt.txt", "request_id": uuid.NewString()}
		created := milestoneObject(t, call("workitem_attachment", input))
		id, ok := created["attachment_id"].(string)
		require.True(t, ok, created)
		require.Equal(t, false, created["replayed"])
		require.NotEmpty(t, created["replay_expires_at"])
		before, downloads := attachmentStorageSnapshot(t, f), h.fetched.Load()
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, created["revision"], replay["revision"])
		require.Equal(t, before, attachmentStorageSnapshot(t, f))
		for _, changed := range []map[string]any{{"url": "https://files.example.test/changed.txt"}, {"name": "changed.txt"}} {
			conflict := maps.Clone(input)
			maps.Copy(conflict, changed)
			milestoneFailure(t, call("workitem_attachment", conflict), "request_conflict")
			require.Equal(t, before, attachmentStorageSnapshot(t, f))
		}
		require.Equal(t, downloads, h.fetched.Load(), "replays and conflicts must not fetch the source")
		rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list", "include_revision": true}))
		require.Len(t, rows, 1)
		deletion := map[string]any{"action": "delete", "attachment_id": id, "request_id": uuid.NewString(), "expected_revision": rows[0]["revision"]}
		var key string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT asset FROM file_assets WHERE id=$1`, id).Scan(&key))
		// This is the metadata writer used by the native asset worker.
		require.NoError(t, repository.SaveMetadataForKey(t.Context(), uuid.MustParse(id), key, map[string]any{"ContentLength": 27, "ContentType": "text/plain"}))
		native := attachmentStorageSnapshot(t, f)
		milestoneFailure(t, call("workitem_attachment", deletion), "revision_conflict")
		require.Equal(t, native, attachmentStorageSnapshot(t, f))
		latest := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list", "include_revision": true}))
		require.Len(t, latest, 1)
		require.NotEqual(t, rows[0]["revision"], latest[0]["revision"])
		deletion["expected_revision"] = latest[0]["revision"]
		deleted := milestoneObject(t, call("workitem_attachment", deletion))
		require.Nil(t, deleted["result"])
		require.Equal(t, false, deleted["replayed"], "rejected CAS must leave the receipt key reusable")
		after := attachmentStorageSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call("workitem_attachment", deletion))["replayed"])
		require.Equal(t, after, attachmentStorageSnapshot(t, f))
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
		var activities int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$1 AND payload->>'type' LIKE 'attachment.activity.%'`, item).Scan(&activities))
		require.Equal(t, 2, activities, "upload and deletion each enqueue one activity")
	})
}

func attachmentStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'assets',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a),
 'intents',(SELECT jsonb_agg(to_jsonb(i) ORDER BY asset_id) FROM file_asset_upload_intents i),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY asset_id) FROM file_asset_edit_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,action,request_id) FROM mutation_receipts r WHERE tool='workitem_attachment'),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e WHERE kind LIKE 'asset_storage_%' OR payload->>'type' LIKE 'attachment.activity.%'))`).Scan(&snapshot))
	return snapshot
}
