//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// A completed object PUT cannot participate in PostgreSQL rollback. The durable
// source must survive a failed completion transaction so retry can safely finish.
func TestMCPAttachmentRecoveryAfterCompletionRollback(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		before := h.fetched.Load()
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/rollback.txt", "request_id": uuid.NewString()}
		_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_attachment_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.completed_at IS NOT NULL THEN RAISE EXCEPTION 'injected completion failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER reject_attachment_completion BEFORE UPDATE ON file_asset_upload_intents FOR EACH ROW EXECUTE FUNCTION reject_attachment_completion()`)
		require.NoError(t, err)
		milestoneFailure(t, call("workitem_attachment", input), "indeterminate")
		var id, key string
		var contents []byte
		var completed, uploaded bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT a.id::text,a.asset,i.contents,i.completed_at IS NOT NULL,a.is_uploaded FROM file_asset_upload_intents i JOIN file_assets a ON a.id=i.asset_id WHERE a.issue_id=$1`, item).Scan(&id, &key, &contents, &completed, &uploaded))
		require.Equal(t, "Durable attachment contents\n", string(contents))
		require.False(t, completed)
		require.False(t, uploaded)
		require.Equal(t, before+1, h.fetched.Load())
		_, err = h.storage.Head(t.Context(), key)
		require.NoError(t, err, "the object PUT succeeded before the database failure")
		require.Empty(t, catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"})))
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id}), "not_found")
		var effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE (kind='activity' AND payload->>'issue_id'=$1) OR (kind='asset_storage_metadata' AND payload->>'asset_id'=$2)`, item, id).Scan(&effects))
		require.Zero(t, effects, "failed completion must roll back its activity and metadata effects")
		_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_attachment_completion ON file_asset_upload_intents; DROP FUNCTION reject_attachment_completion()`)
		require.NoError(t, err)
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		require.NoError(t, h.worker.CompleteAttachmentUpload(t.Context(), id))
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, id, replay["attachment_id"])
		require.Equal(t, before+1, h.fetched.Load())
		inline := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
		require.False(t, inline.IsError, inline.Content)
		require.Len(t, inline.Content, 1)
		require.Equal(t, "Durable attachment contents\n", inline.Content[0].Text)
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT i.contents,i.completed_at IS NOT NULL,a.is_uploaded FROM file_asset_upload_intents i JOIN file_assets a ON a.id=i.asset_id WHERE a.id=$1`, id).Scan(&contents, &completed, &uploaded))
		require.Empty(t, contents)
		require.True(t, completed)
		require.True(t, uploaded)
		var activities, metadata int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->>'issue_id'=$1 AND payload->>'type'='attachment.activity.created'),(SELECT count(*) FROM work_item_effects WHERE kind='asset_storage_metadata' AND payload->>'asset_id'=$2)`, item, id).Scan(&activities, &metadata))
		require.Equal(t, 1, activities)
		require.Equal(t, 1, metadata)
	})
}
