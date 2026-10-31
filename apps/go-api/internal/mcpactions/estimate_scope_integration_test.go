//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateWorkspaceAndPointParentScope(t *testing.T) {
	f := newMilestoneFixture(t)
	_, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
		foreignCall := catalogueScoped(root, map[string]any{"workspace_slug": "other", "project_id": otherProject})
		own := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": uuid.NewString()}))
		foreign := milestoneObject(t, foreignCall("project_estimate", map[string]any{"action": "create", "name": uuid.NewString()}))
		ownID, foreignID := own["id"].(string), foreign["id"].(string)
		ownPoints := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": ownID, "points": `[{"value":"1"}]`}))
		foreignPoints := catalogueArray(t, foreignCall("project_estimate", map[string]any{"action": "create_points", "estimate_id": foreignID, "points": `[{"value":"2"}]`}))
		require.Len(t, catalogueArray(t, foreignCall("project_estimate", map[string]any{"action": "list_points", "estimate_id": foreignID})), 1)
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": ownID}))
		before := estimateStorageSnapshot(t, f)
		for _, input := range estimateActionInputs(ownID, ownPoints[0]["id"].(string)) {
			wrongProject := maps.Clone(input)
			wrongProject["project_id"] = otherProject
			rejected := call("project_estimate", wrongProject)
			require.True(t, rejected.IsError, input["action"])
			require.Empty(t, rejected.StructuredContent)
		}
		for _, action := range []string{"list_points", "create_points", "update_point", "delete_point"} {
			input := map[string]any{"action": action, "estimate_id": foreignID}
			if action == "create_points" {
				input["points"] = `[{"value":"3"}]`
			}
			if action == "update_point" || action == "delete_point" {
				input["estimate_point_id"] = foreignPoints[0]["id"]
			}
			milestoneFailure(t, call("project_estimate", input), "not_found")
		}
		milestoneFailure(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": foreignID}), "invalid_params")
		require.Equal(t, before, estimateStorageSnapshot(t, f))
		other := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": uuid.NewString()}))
		before = estimateStorageSnapshot(t, f)
		for _, action := range []string{"update_point", "delete_point"} {
			milestoneFailure(t, call("project_estimate", map[string]any{"action": action, "estimate_id": other["id"], "estimate_point_id": ownPoints[0]["id"]}), "not_found")
		}
		require.Equal(t, before, estimateStorageSnapshot(t, f))
	})
}

func TestMCPEstimateCurrentProjectVisibilityAndReplay(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		request := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
		receipt := milestoneObject(t, call("project_estimate", request))
		id := receipt["estimate"].(map[string]any)["id"].(string)
		milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		points := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"value":"1"}]`}))
		for _, condition := range []struct{ name, hide, restore string }{
			{"archived project", `UPDATE projects SET archived_at=now() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`},
			{"deleted project", `UPDATE projects SET deleted_at=now() WHERE id=$1`, `UPDATE projects SET deleted_at=NULL WHERE id=$1`},
			{"inactive membership", `UPDATE project_members SET is_active=false WHERE project_id=$1`, `UPDATE project_members SET is_active=true WHERE project_id=$1`},
			{"deleted membership", `UPDATE project_members SET deleted_at=now() WHERE project_id=$1`, `UPDATE project_members SET deleted_at=NULL WHERE project_id=$1`},
		} {
			t.Run(condition.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), condition.hide, testsupport.LabelProject)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), condition.restore, testsupport.LabelProject)
					require.NoError(t, restoreErr)
				})
				before := estimateStorageSnapshot(t, f)
				for _, input := range append(estimateActionInputs(id, points[0]["id"].(string)), request) {
					rejected := call("project_estimate", input)
					require.True(t, rejected.IsError, input["action"])
					require.Empty(t, rejected.StructuredContent)
				}
				require.Equal(t, before, estimateStorageSnapshot(t, f))
			})
		}
		require.Equal(t, id, milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve"}))["id"])
		require.Equal(t, true, milestoneObject(t, call("project_estimate", request))["replayed"])
	})
}

func estimateActionInputs(estimate, point string) []map[string]any {
	return []map[string]any{
		{"action": "retrieve"}, {"action": "list_points", "estimate_id": estimate},
		{"action": "create", "name": uuid.NewString()}, {"action": "update", "name": "Hidden update"}, {"action": "delete"},
		{"action": "link", "estimate_id": estimate}, {"action": "create_points", "estimate_id": estimate, "points": `[{"value":"new"}]`},
		{"action": "update_point", "estimate_id": estimate, "estimate_point_id": point, "value": "changed"},
		{"action": "delete_point", "estimate_id": estimate, "estimate_point_id": point},
	}
}
