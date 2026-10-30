//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPArchivedProjectFeaturesRemainReadable(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE projects SET archived_at=NULL,cycle_view=true WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		expected := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject}))
		catalogueNull(t, call("project", map[string]any{"action": "archive", "project_id": testsupport.LabelProject}))
		native := milestoneObject(t, call("project", map[string]any{"action": "retrieve", "project_id": testsupport.LabelProject}))
		require.NotNil(t, native["archived_at"])
		require.Equal(t, true, native["cycle_view"])
		snapshot := featureMutationSnapshot(t, f)
		actual := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject}))
		require.Equal(t, expected, actual)
		require.JSONEq(t, string(snapshot), string(featureMutationSnapshot(t, f)))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		milestoneFailure(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject}), "forbidden")
		require.JSONEq(t, string(snapshot), string(featureMutationSnapshot(t, f)))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=true WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		catalogueNull(t, call("project", map[string]any{"action": "unarchive", "project_id": testsupport.LabelProject}))
	})
}

func TestMCPArchivedProjectFeaturesRejectWritesAndReplayUntilRestored(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		key := uuid.NewString()
		input := map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": true, "request_id": key}
		original := milestoneObject(t, call("project", input))
		catalogueNull(t, call("project", map[string]any{"action": "archive", "project_id": testsupport.LabelProject}))
		snapshot := featureMutationSnapshot(t, f)
		milestoneFailure(t, call("project", input), "forbidden")
		fresh := map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": false, "request_id": uuid.NewString()}
		milestoneFailure(t, call("project", fresh), "forbidden")
		require.JSONEq(t, string(snapshot), string(featureMutationSnapshot(t, f)))
		catalogueNull(t, call("project", map[string]any{"action": "unarchive", "project_id": testsupport.LabelProject}))
		restored := featureMutationSnapshot(t, f)
		replay := milestoneObject(t, call("project", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, original["result"], replay["result"])
		require.JSONEq(t, string(restored), string(featureMutationSnapshot(t, f)))
		updated := milestoneObject(t, call("project", fresh))
		require.Equal(t, false, updated["result"].(map[string]any)["cycles"])
		require.Equal(t, false, updated["replayed"])
	})
}
