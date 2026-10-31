//go:build integration

package mcpactions_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentNativeParentLifecycle(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	redis := testsupport.StartRedis(t)
	t.Setenv("REDIS_URL", "redis://"+redis.Options().Addr)
	var ids, keys []string
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		for _, parent := range []string{"workitem", "project"} {
			t.Run(parent, func(t *testing.T) {
				id, key := attachmentParentLifecycle(t, h, f, root, parent)
				ids, keys = append(ids, id), append(keys, key)
			})
		}
	})
	require.Len(t, ids, 8)
	var deletedBeforeWorker int
	require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT count(*) FROM file_assets WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, ids).Scan(&deletedBeforeWorker))
	require.Zero(t, deletedBeforeWorker)
	// Native deletion effects, rather than fixture SQL, must mark child assets.
	// Restorable soft deletion retains their object bytes through the retention period.
	runtime, err := backgroundtasks.NewRuntime(t.Context(), h.pool.Config().ConnString(), selvedge.WithActivityPollInterval(20*time.Millisecond), selvedge.WithWorkflowResumptionInterval(20*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 20*time.Second)
		defer cancel()
		require.NoError(t, runtime.Shutdown(ctx))
	})
	require.NoError(t, runtime.Start(t.Context()))
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var deleted int
		queryErr := h.pool.QueryRow(t.Context(), `SELECT count(*) FROM file_assets WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, ids).Scan(&deleted)
		assert.NoError(collect, queryErr)
		assert.Equal(collect, len(ids), deleted)
	}, 45*time.Second, 25*time.Millisecond)
	for _, key := range keys {
		_, headErr := h.storage.Head(t.Context(), key)
		require.NoError(t, headErr)
	}
}

func attachmentParentLifecycle(t *testing.T, h attachmentRecoveryHarness, f milestoneFixture, root catalogueCall, parent string) (string, string) {
	t.Helper()
	project := milestoneObject(t, root("project", map[string]any{"action": "create", "name": "Attachment lifecycle " + uuid.NewString()[:8], "identifier": strings.ToUpper(uuid.NewString()[:8])}))
	projectCall := catalogueScoped(root, map[string]any{"project_id": project["id"]})
	state := milestoneObject(t, projectCall("state", map[string]any{"action": "create", "name": "Attachment completed", "color": "#123456", "group": "completed"}))
	item := milestoneObject(t, projectCall("workitem", map[string]any{"action": "create", "name": "Attachment parent", "state": state["id"]}))
	call := catalogueScoped(projectCall, map[string]any{"workitem_id": item["id"]})
	input := map[string]any{"action": "upload_from_url", "url": "https://files.example.test/parent.txt", "request_id": uuid.NewString()}
	created := milestoneObject(t, call("workitem_attachment", input))
	id := created["attachment_id"].(string)
	assertVisible := func() {
		rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list"}))
		require.Len(t, rows, 1)
		require.Equal(t, id, rows[0]["id"])
		read := call("workitem_attachment", map[string]any{"action": "read", "attachment_id": id})
		require.False(t, read.IsError, read.Content)
		require.Len(t, read.Content, 1)
		require.Equal(t, "Durable attachment contents\n", read.Content[0].Text)
		require.Equal(t, id, milestoneObject(t, call("workitem_attachment", map[string]any{"action": "download_url", "attachment_id": id}))["attachment_id"])
		require.Equal(t, true, milestoneObject(t, call("workitem_attachment", input))["replayed"])
	}
	if parent == "workitem" {
		require.Equal(t, true, milestoneObject(t, call("workitem", map[string]any{"action": "archive"}))["archived"])
		assertVisible()
		second := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/archived.txt"}))
		catalogueNull(t, call("workitem_attachment", map[string]any{"action": "delete", "attachment_id": second["id"]}))
		require.Equal(t, false, milestoneObject(t, call("workitem", map[string]any{"action": "archive", "archive": false}))["archived"])
	} else {
		catalogueNull(t, projectCall("project", map[string]any{"action": "archive"}))
		assertAttachmentParentDenied(t, h, f, call, input, id, "forbidden")
		catalogueNull(t, projectCall("project", map[string]any{"action": "unarchive"}))
	}
	assertVisible()
	var key string
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT asset FROM file_assets WHERE id=$1`, id).Scan(&key))
	if parent == "workitem" {
		catalogueNull(t, call("workitem", map[string]any{"action": "delete"}))
		assertAttachmentParentDenied(t, h, f, call, input, id, "not_found")
	} else {
		catalogueNull(t, projectCall("project", map[string]any{"action": "delete"}))
		assertAttachmentParentDenied(t, h, f, call, input, id, "forbidden")
	}
	return id, key
}

func assertAttachmentParentDenied(t *testing.T, h attachmentRecoveryHarness, f milestoneFixture, call catalogueCall, replay map[string]any, id, writeCode string) {
	t.Helper()
	before, fetched := attachmentStorageSnapshot(t, f), h.fetched.Load()
	for _, action := range []string{"list", "read", "download_url"} {
		input := map[string]any{"action": action}
		if action != "list" {
			input["attachment_id"] = id
		}
		milestoneFailure(t, call("workitem_attachment", input), "forbidden")
	}
	milestoneFailure(t, call("workitem_attachment", replay), writeCode)
	milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/denied.txt"}), writeCode)
	milestoneFailure(t, call("workitem_attachment", map[string]any{"action": "delete", "attachment_id": id}), writeCode)
	require.Equal(t, before, attachmentStorageSnapshot(t, f))
	require.Equal(t, fetched, h.fetched.Load())
}
