//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateLinkUsesProjectReceiptsAndRevisions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		first := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "First sizing " + uuid.NewString()}))
		second := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "Second sizing " + uuid.NewString()}))
		project := milestoneObject(t, call("project", map[string]any{"action": "retrieve", "include_revision": true}))
		input := map[string]any{"action": "link", "estimate_id": first["id"], "request_id": uuid.NewString(), "expected_revision": project["revision"]}
		linked := milestoneObject(t, call("project_estimate", input))
		require.Equal(t, false, linked["replayed"])
		require.Equal(t, first["id"], linked["result"].(map[string]any)["estimate"])
		require.NotEqual(t, project["revision"], linked["revision"])
		catalogue := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		require.NotEqual(t, catalogue["revision"], linked["revision"], "link consumes the project token, not the estimate catalogue token")
		snapshot := estimateStorageSnapshot(t, f)
		changed := maps.Clone(input)
		changed["estimate_id"] = second["id"]
		milestoneFailure(t, call("project_estimate", changed), "request_conflict")
		changed["request_id"] = uuid.NewString()
		changed["expected_revision"] = catalogue["revision"]
		milestoneFailure(t, call("project_estimate", changed), "revision_conflict")
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f))
		// The native project selection column advances both revision namespaces.
		_, err := f.pool.Exec(t.Context(), `UPDATE projects SET estimate_id=$1 WHERE id=$2`, second["id"], testsupport.LabelProject)
		require.NoError(t, err)
		current := milestoneObject(t, call("project", map[string]any{"action": "retrieve", "include_revision": true}))
		selected := milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve", "include_revision": true}))
		require.Equal(t, second["id"], selected["id"])
		require.NotEqual(t, linked["revision"], current["revision"])
		require.NotEqual(t, catalogue["revision"], selected["revision"])
		snapshot = estimateStorageSnapshot(t, f)
		changed["expected_revision"] = linked["revision"]
		milestoneFailure(t, call("project_estimate", changed), "revision_conflict")
		replay := milestoneObject(t, call("project_estimate", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, linked["result"], replay["result"])
		require.Equal(t, linked["revision"], replay["revision"])
		require.Equal(t, snapshot, estimateStorageSnapshot(t, f), "replay must not restore the old selection")
		changed["estimate_id"], changed["expected_revision"] = first["id"], current["revision"]
		require.Equal(t, false, milestoneObject(t, call("project_estimate", changed))["replayed"])
	})
}
