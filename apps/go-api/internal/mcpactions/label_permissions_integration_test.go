//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPLabelMutationsRequireCurrentProjectMembership(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		for _, access := range []struct {
			name, change string
			allowed      bool
		}{
			{"workspace guest and project admin", `UPDATE workspace_members SET role=5`, true},
			{"workspace admin and project guest", `UPDATE project_members SET role=5`, true},
			{"guest in both scopes", `UPDATE workspace_members SET role=5; UPDATE project_members SET role=5`, true},
			{"member in both scopes", `UPDATE workspace_members SET role=15; UPDATE project_members SET role=15`, true},
			{"inactive project member", `UPDATE project_members SET is_active=false`, false},
			{"deleted project member", `UPDATE project_members SET deleted_at=now()`, false},
			{"unknown project role", `UPDATE project_members SET role=7`, false},
			{"archived project", `UPDATE projects SET archived_at=now()`, false},
			{"deleted project", `UPDATE projects SET deleted_at=now()`, false},
		} {
			t.Run(access.name, func(t *testing.T) {
				input := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
				created := milestoneObject(t, call("label", input))
				id := created["label_id"].(string)
				_, err := f.pool.Exec(t.Context(), access.change)
				require.NoError(t, err)
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
					defer cancel()
					_, restoreErr := f.pool.Exec(ctx, `UPDATE workspace_members SET role=20;
UPDATE project_members SET role=20,is_active=true,deleted_at=NULL;
UPDATE projects SET archived_at=NULL,deleted_at=NULL`)
					require.NoError(t, restoreErr)
				})
				before := labelMutationSnapshot(t, f)
				replay := call("label", input)
				freshInput := map[string]any{"action": "create", "name": uuid.NewString(), "request_id": uuid.NewString()}
				updateInput := map[string]any{"action": "update", "label_id": id, "sort_order": 0,
					"expected_revision": created["revision"], "request_id": uuid.NewString()}
				deleteInput := map[string]any{"action": "delete", "label_id": id, "request_id": uuid.NewString()}
				if !access.allowed {
					milestoneFailure(t, replay, "forbidden")
					milestoneFailure(t, call("label", freshInput), "forbidden")
					milestoneFailure(t, call("label", updateInput), "forbidden")
					milestoneFailure(t, call("label", deleteInput), "forbidden")
					require.Equal(t, before, labelMutationSnapshot(t, f))
					return
				}
				// API v1 labels use ProjectMemberPermission, including active guests;
				// the stricter native app editing threshold does not apply to MCP.
				require.Equal(t, true, milestoneObject(t, replay)["replayed"])
				require.Equal(t, before, labelMutationSnapshot(t, f))
				fresh := milestoneObject(t, call("label", freshInput))
				require.Equal(t, false, fresh["replayed"])
				require.Equal(t, testsupport.LabelProject, fresh["result"].(map[string]any)["project"])
				updated := milestoneObject(t, call("label", updateInput))
				require.Equal(t, float64(0), updated["result"].(map[string]any)["sort_order"])
				deleteInput["expected_revision"] = updated["revision"]
				deleted := milestoneObject(t, call("label", deleteInput))
				require.Nil(t, deleted["result"])
				var softDeleted bool
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM labels WHERE id=$1`, id).Scan(&softDeleted))
				require.True(t, softDeleted)
			})
		}
	})
}
