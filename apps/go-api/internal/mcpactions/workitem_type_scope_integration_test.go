//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkitemTypeWorkspaceBinding(t *testing.T) {
	f := newMilestoneFixture(t)
	_, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
	enableTypeCatalogue(t, f, "workspace-one")
	enableTypeCatalogue(t, f, "other")
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		own := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one"})
		foreign := catalogueScoped(root, map[string]any{"workspace_slug": "other"})
		created := milestoneObject(t, foreign("workitem_type", map[string]any{"action": "resolve", "project_id": foreignProject, "name": uuid.NewString()}))
		before := typeStorageSnapshot(t, f)
		for _, input := range []map[string]any{
			{"action": "retrieve", "workitem_type_id": created["id"]},
			{"action": "update", "workitem_type_id": created["id"], "description": "Foreign update"},
			{"action": "delete", "workitem_type_id": created["id"]},
			{"action": "import_to_project", "project_id": testsupport.LabelProject, "workitem_type_ids": created["id"]},
			{"action": "list", "project_id": foreignProject},
			{"action": "retrieve", "workitem_type_id": created["id"], "project_id": foreignProject},
			{"action": "resolve", "project_id": foreignProject, "name": created["name"]},
			{"action": "create", "name": uuid.NewString(), "is_active": true, "project_ids": foreignProject},
		} {
			rejected := own("workitem_type", input)
			require.True(t, rejected.IsError, input["action"])
			require.Empty(t, rejected.StructuredContent)
		}
		require.Equal(t, before, typeStorageSnapshot(t, f))
		resolved := milestoneObject(t, own("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": created["name"]}))
		require.NotEqual(t, created["id"], resolved["id"], "a same-named foreign type cannot satisfy resolution")
		require.Equal(t, testsupport.LabelWorkspace, resolved["workspace"])
	})
}

func TestMCPWorkitemTypeCurrentWorkspaceAndProjectPermissions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		available := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		unlinked := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": uuid.NewString(), "is_active": true}))
		for _, role := range []int{15, 5} {
			_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=$1;`, role)
			require.NoError(t, err)
			_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=$1`, role)
			require.NoError(t, err)
			before := typeStorageSnapshot(t, f)
			for _, input := range []map[string]any{
				{"action": "create", "name": uuid.NewString()},
				{"action": "update", "workitem_type_id": available["id"], "description": "Denied"},
				{"action": "delete", "workitem_type_id": available["id"]},
				{"action": "resolve", "project_id": testsupport.LabelProject, "name": unlinked["name"]},
				{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()},
				{"action": "import_to_project", "project_id": testsupport.LabelProject, "workitem_type_ids": unlinked["id"]},
			} {
				milestoneFailure(t, call("workitem_type", input), "forbidden")
			}
			require.Equal(t, available, milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": available["name"]})))
			require.Equal(t, before, typeStorageSnapshot(t, f))
		}
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
		require.NoError(t, err)
	})
}

func TestMCPWorkitemTypeProjectVisibility(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		created := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "name": uuid.NewString()}))
		for _, condition := range []struct{ name, hide, restore string }{
			{"archived project", `UPDATE projects SET archived_at=now() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`},
			{"deleted project", `UPDATE projects SET deleted_at=now() WHERE id=$1`, `UPDATE projects SET deleted_at=NULL WHERE id=$1`},
			{"inactive member", `UPDATE project_members SET is_active=false WHERE project_id=$1`, `UPDATE project_members SET is_active=true WHERE project_id=$1`},
			{"deleted member", `UPDATE project_members SET deleted_at=now() WHERE project_id=$1`, `UPDATE project_members SET deleted_at=NULL WHERE project_id=$1`},
		} {
			t.Run(condition.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), condition.hide, testsupport.LabelProject)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), condition.restore, testsupport.LabelProject)
					require.NoError(t, restoreErr)
				})
				before := typeStorageSnapshot(t, f)
				for _, input := range []map[string]any{
					{"action": "list"}, {"action": "retrieve", "workitem_type_id": created["id"]},
					{"action": "resolve", "name": created["name"]}, {"action": "import_to_project", "workitem_type_ids": created["id"]},
				} {
					milestoneFailure(t, call("workitem_type", input), "forbidden")
				}
				require.Equal(t, before, typeStorageSnapshot(t, f))
			})
		}
	})
}
