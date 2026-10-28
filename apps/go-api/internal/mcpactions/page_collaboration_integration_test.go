//go:build integration

package mcpactions_test

import (
	"errors"
	"testing"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPageReplacementRejectsOldCollaborationWrites(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		page := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Before", "description_html": "<p>Before</p>"}))
		id := page["id"].(string)
		repository := pagereads.NewPageAPIRepository(f.pool)
		native := func(operation string, fields map[string]any) error {
			_, _, err := repository.Lifecycle(t.Context(), operation, stduuid.UUID(uuid.MustParse(testsupport.LabelActor)), "workspace-one", stduuid.UUID(uuid.MustParse(testsupport.LabelProject)), stduuid.UUID(uuid.MustParse(id)), "UTC", djangoinput.Input{Fields: fields})
			return err
		}
		revision := func() int64 {
			var value int64
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT collaboration_revision FROM pages WHERE id=$1`, id).Scan(&value))
			return value
		}
		require.NoError(t, native("description_update", map[string]any{"description_html": "<p>Native before replacement</p>"}))
		input := map[string]any{"action": "update", "page_id": id, "description_html": "<p>Replacement</p>", "request_id": uuid.NewString()}
		milestoneObject(t, call("page", input))
		require.Equal(t, int64(1), revision())
		require.Equal(t, true, milestoneObject(t, call("page", input))["replayed"])
		require.Equal(t, int64(1), revision(), "receipt replay cannot create another document generation")
		for _, write := range []struct {
			operation string
			fields    map[string]any
		}{
			{"description_update", map[string]any{"description_html": "<p>Old cached body</p>"}},
			{"description_update", map[string]any{"description_html": "<p>Old cached body</p>", "collaboration_revision": 0}},
			{"update", map[string]any{"name": "Old cached title"}},
			{"update", map[string]any{"name": "Old cached title", "collaboration_revision": 0}},
		} {
			var conflict *djangoapi.HTTPError
			err := native(write.operation, write.fields)
			require.True(t, errors.As(err, &conflict), "%s: %v", write.operation, err)
			require.Equal(t, 409, conflict.Status)
		}
		after := milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": id}))
		require.Equal(t, "<p>Replacement</p>", after["description_html"])
		require.Equal(t, "Before", after["name"])
		require.NoError(t, native("description_update", map[string]any{"description_html": "<p>Current session edit</p>", "collaboration_revision": 1}))
		require.NoError(t, native("update", map[string]any{"name": "Current title", "collaboration_revision": 1}))
		milestoneObject(t, call("page", map[string]any{"action": "update", "page_id": id, "name": "MCP title"}))
		require.Equal(t, int64(2), revision(), "title replacements also invalidate cached title fragments")
		after = milestoneObject(t, call("page", map[string]any{"action": "retrieve", "page_id": id}))
		require.Equal(t, "<p>Current session edit</p>", after["description_html"])
		require.Equal(t, "MCP title", after["name"])
	})
}
