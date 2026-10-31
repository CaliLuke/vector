//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentMetadataMatchesSDKShapeAndTimezone(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	_, setupErr := h.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'`)
	require.NoError(t, setupErr)
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item := f.item(t)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		created := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/report.txt"}))
		id := created["id"].(string)
		// SDK 0.3.1 WorkItemAttachment fields, plus the pinned normalizer's
		// name and content_type. Receipt and revision fields remain opt-in.
		require.ElementsMatch(t, strings.Fields("id created_at updated_at deleted_at attributes asset entity_type entity_identifier is_deleted is_archived external_id external_source size is_uploaded storage_metadata created_by updated_by user workspace draft_issue project issue comment page name content_type"), slices.Collect(maps.Keys(created)))
		require.Equal(t, "report.txt", created["name"])
		require.Equal(t, "text/plain", created["content_type"])
		require.Equal(t, float64(28), created["size"])
		require.Equal(t, map[string]any{"name": "report.txt", "type": "text/plain", "size": float64(28)}, created["attributes"])
		require.Equal(t, testsupport.LabelActor, created["created_by"])
		require.Equal(t, testsupport.LabelWorkspace, created["workspace"])
		require.Equal(t, testsupport.LabelProject, created["project"])
		require.Equal(t, item, created["issue"])
		require.Equal(t, "ISSUE_ATTACHMENT", created["entity_type"])
		require.Equal(t, true, created["is_uploaded"])
		require.Equal(t, false, created["is_deleted"])
		require.Equal(t, false, created["is_archived"])
		for _, key := range strings.Fields("deleted_at entity_identifier external_id external_source updated_by user draft_issue comment page") {
			require.Nil(t, created[key], key)
		}
		for _, key := range []string{"created_at", "updated_at"} {
			stamp, parseErr := time.Parse(time.RFC3339Nano, created[key].(string))
			require.NoError(t, parseErr)
			_, offset := stamp.Zone()
			assert.Equal(t, 19800, offset, "upload %s must use the user's timezone", key)
		}
		rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"}))
		require.Len(t, rows, 1)
		assert.Equal(t, created, rows[0], "upload and list must expose the same completed attachment metadata")

		// JSON metadata belongs to the caller. Timestamp-shaped values inside
		// those objects must not be rewritten as serializer timestamp fields.
		attributes := map[string]any{"name": "report.txt", "type": "text/plain", "size": 13, "created_at": "2024-01-01T00:00:00Z"}
		metadata := map[string]any{"created_at": "2024-02-01T00:00:00Z", "nested": map[string]any{"updated_at": "2024-02-02T00:00:00Z"}}
		attributesJSON, marshalErr := json.Marshal(attributes)
		require.NoError(t, marshalErr)
		metadataJSON, marshalErr := json.Marshal(metadata)
		require.NoError(t, marshalErr)
		_, err := f.pool.Exec(t.Context(), `UPDATE file_assets SET attributes=$2::jsonb,storage_metadata=$3::jsonb WHERE id=$1`, id, attributesJSON, metadataJSON)
		require.NoError(t, err)
		rows = catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"}))
		require.Len(t, rows, 1)
		assert.Equal(t, float64(13), rows[0]["size"], "nonzero attribute size takes precedence")
		assert.Equal(t, "2024-01-01T00:00:00Z", rows[0]["attributes"].(map[string]any)["created_at"])
		assert.Equal(t, metadata, rows[0]["storage_metadata"])
		for _, value := range []string{"0", "null"} {
			_, err = f.pool.Exec(t.Context(), `UPDATE file_assets SET attributes=jsonb_set(attributes,'{size}',$2::jsonb) WHERE id=$1`, id, value)
			require.NoError(t, err)
			rows = catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"}))
			require.Equal(t, float64(28), rows[0]["size"], "empty attribute size falls back to stored size")
		}
		_, err = f.pool.Exec(t.Context(), `UPDATE file_assets SET attributes='{}'::jsonb WHERE id=$1`, id)
		require.NoError(t, err)
		download := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "download_url", "attachment_id": id}))
		require.ElementsMatch(t, []string{"download_url", "attachment_id", "name"}, slices.Collect(maps.Keys(download)))
		require.Equal(t, id, download["name"])
		require.Equal(t, id, download["attachment_id"])
		require.NotEmpty(t, download["download_url"])

		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/receipt.txt", "request_id": uuid.NewString()}
		receipt := milestoneObject(t, call("workitem_attachment", input))
		stamp, parseErr := time.Parse(time.RFC3339Nano, receipt["result"].(map[string]any)["created_at"].(string))
		require.NoError(t, parseErr)
		_, offset := stamp.Zone()
		assert.Equal(t, 19800, offset, "receipt result must use the user's timezone")
		replay := milestoneObject(t, call("workitem_attachment", input))
		require.Equal(t, receipt["result"], replay["result"])
	})
}
