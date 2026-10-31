//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPEstimateWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "link", "create_points", "update_point", "delete_point"} {
		t.Run(action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{
				tool: "project_estimate", fixture: func(t *testing.T) milestoneFixture {
					f := newMilestoneFixture(t)
					_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
					require.NoError(t, err)
					return f
				}, snapshot: estimateStorageSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, root catalogueCall) map[string]any {
					input := map[string]any{"action": action, "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "request_id": uuid.NewString()}
					if action == "create" {
						input["name"] = uuid.NewString()
						return input
					}
					call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
					estimate := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": uuid.NewString()}))
					id := estimate["id"].(string)
					switch action {
					case "update", "delete":
						milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
						if action == "update" {
							input["name"] = "Authorized edit"
						}
					case "link":
						input["estimate_id"] = id
					case "create_points":
						input["estimate_id"], input["points"] = id, `[{"value":"1"}]`
					case "update_point", "delete_point":
						points := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "estimate_id": id, "points": `[{"value":"1"}]`}))
						input["estimate_id"], input["estimate_point_id"] = id, points[0]["id"]
						if action == "update_point" {
							input["value"] = "2"
						}
					}
					return input
				},
			})
		})
	}
}
