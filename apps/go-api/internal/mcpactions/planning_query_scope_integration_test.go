//go:build integration

package mcpactions_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningQueriesBindParentScopeAndCurrentAccess(t *testing.T) {
	f := newMilestoneFixture(t)
	_, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET cycle_view=true,module_view=true`)
	require.NoError(t, err)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		primary := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
		foreign := catalogueScoped(root, map[string]any{"workspace_slug": "other", "project_id": otherProject})
		project := milestoneObject(t, root("project", map[string]any{"action": "create", "workspace_slug": "workspace-one",
			"name": "Query scope " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8]), "cycles": true, "modules": true}))
		sibling := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": project["id"]})
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) {
				_, local := createPlanningReceipt(t, primary, kind)
				input := map[string]any{"action": "create", "name": "Foreign query target " + uuid.NewString()}
				if kind == "cycle" {
					input["owned_by"] = testsupport.LabelActor
				}
				other := milestoneObject(t, foreign(kind, input))
				neighbor := milestoneObject(t, sibling(kind, input))
				before := planningMutationSnapshot(t, f)
				milestoneFailure(t, primary(kind, map[string]any{"action": "list_workitems", kind + "_id": neighbor["id"]}), "not_found")
				require.Empty(t, catalogueRows(t, sibling(kind, map[string]any{"action": "list_workitems", kind + "_id": neighbor["id"]})))
				milestoneFailure(t, primary(kind, map[string]any{"action": "list_workitems", kind + "_id": other["id"]}), "not_found")
				milestoneFailure(t, foreign(kind, map[string]any{"action": "list_workitems", kind + "_id": local[kind+"_id"]}), "not_found")
				milestoneFailure(t, primary(kind, map[string]any{"action": "list_workitems", kind + "_id": local[kind+"_id"], "workspace_slug": "other"}), "forbidden")
				require.Empty(t, catalogueRows(t, primary(kind, map[string]any{"action": "list_workitems", kind + "_id": local[kind+"_id"]})))
				_, updateErr := f.pool.Exec(t.Context(), `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
				require.NoError(t, updateErr)
				t.Cleanup(func() {
					cleanup, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
					defer cancel()
					_, restoreErr := f.pool.Exec(cleanup, `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor)
					require.NoError(t, restoreErr)
				})
				milestoneFailure(t, primary(kind, map[string]any{"action": "list_workitems", kind + "_id": local[kind+"_id"]}), "forbidden")
				require.Empty(t, catalogueRows(t, foreign(kind, map[string]any{"action": "list_workitems", kind + "_id": other["id"]})))
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}
