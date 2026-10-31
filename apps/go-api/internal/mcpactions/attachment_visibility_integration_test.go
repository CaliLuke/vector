//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentScopeAndParentVisibility(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	f := h.milestoneFixture
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		item, otherItem := f.item(t), f.itemInScope(t, otherWorkspace, otherProject)
		call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": item})
		foreign := milestoneObject(t, root("workitem_attachment", map[string]any{"action": "upload_from_url", "workspace_slug": "other", "project_id": otherProject, "workitem_id": otherItem, "url": "https://files.example.test/foreign.txt"}))
		input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/visible.txt", "request_id": uuid.NewString()}
		created := milestoneObject(t, call("workitem_attachment", input))
		id := created["attachment_id"].(string)
		rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"}))
		require.Len(t, rows, 1)
		require.Equal(t, id, rows[0]["id"])
		before, fetched := attachmentStorageSnapshot(t, f), h.fetched.Load()
		for _, action := range []string{"read", "download_url", "delete"} {
			milestoneFailure(t, call("workitem_attachment", map[string]any{"action": action, "attachment_id": foreign["id"]}), "not_found")
			milestoneFailure(t, call("workitem_attachment", map[string]any{"action": action, "workitem_id": f.item(t), "attachment_id": id}), "not_found")
		}
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "list", "workitem_id": otherItem}), "forbidden")
		milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "list", "project_id": otherProject, "workitem_id": otherItem}), "forbidden")
		require.Equal(t, before, attachmentStorageSnapshot(t, f))
		require.Equal(t, fetched, h.fetched.Load())
		for _, condition := range []struct{ name, set, restore string }{
			{"draft parent", "is_draft=true", "is_draft=false"},
			{"deleted parent", "deleted_at=now()", "deleted_at=NULL"},
		} {
			t.Run(condition.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), "UPDATE issues SET "+condition.set+" WHERE id=$1", item)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), "UPDATE issues SET "+condition.restore+" WHERE id=$1", item)
					require.NoError(t, restoreErr)
				})
				hidden := attachmentStorageSnapshot(t, f)
				for _, action := range []string{"list", "read", "download_url"} {
					arguments := map[string]any{"action": action}
					if action != "list" {
						arguments["attachment_id"] = id
					}
					milestoneFailure(t, call("workitem_attachment", arguments), "forbidden")
				}
				milestoneFailure(t, call("workitem_attachment", input), "not_found")
				milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/new.txt"}), "not_found")
				milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "delete", "attachment_id": id}), "not_found")
				require.Equal(t, hidden, attachmentStorageSnapshot(t, f))
				require.Equal(t, fetched, h.fetched.Load())
			})
		}
		inline := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
		require.False(t, inline.IsError, inline.Content)
		require.Len(t, inline.Content, 1)
		require.Equal(t, "Durable attachment contents\n", inline.Content[0].Text)
	})
}
