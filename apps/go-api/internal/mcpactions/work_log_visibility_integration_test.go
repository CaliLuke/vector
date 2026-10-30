//go:build integration

package mcpactions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkLogVisibilityAndSummaryUnits(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		root = catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one"})
		item, otherItem := f.item(t), f.itemInScope(t, otherWorkspace, otherProject)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		foreign := milestoneObject(t, root("work_log", map[string]any{"action": "create", "workspace_slug": "other", "project_id": otherProject, "workitem_id": otherItem, "duration": 17}))
		first := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 30}))
		second := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 2147483647}))
		rows := catalogueArray(t, call("work_log", map[string]any{"action": "list", "per_page": 1}))
		require.Len(t, rows, 1)
		require.Equal(t, second["id"], rows[0]["id"])
		rows = catalogueArray(t, call("work_log", map[string]any{"action": "list", "per_page": 1, "cursor": "1:1:0"}))
		require.Len(t, rows, 1)
		require.Equal(t, first["id"], rows[0]["id"])
		summary := func() []map[string]any {
			return catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}))
		}
		require.Contains(t, summary(), map[string]any{"issue_id": item, "duration": float64((int64(2147483647) + 30) * 60)})
		for _, row := range summary() {
			require.NotEqual(t, otherItem, row["issue_id"])
		}
		milestoneFailure(t, call("work_log", map[string]any{"action": "update", "work_log_id": foreign["id"], "duration": 60}), "not_found")
		milestoneFailure(t, call("work_log", map[string]any{"action": "list", "workitem_id": otherItem}), "forbidden")
		milestoneFailure(t, root("project", map[string]any{"action": "worklog_summary", "project_id": otherProject}), "forbidden")
		for _, condition := range []struct{ name, set, restore string }{
			{"draft", "is_draft=true", "is_draft=false"}, {"deleted", "deleted_at=now()", "deleted_at=NULL"},
		} {
			t.Run(condition.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), "UPDATE issues SET "+condition.set+" WHERE id=$1", item)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, err := f.pool.Exec(context.WithoutCancel(t.Context()), "UPDATE issues SET "+condition.restore+" WHERE id=$1", item)
					require.NoError(t, err)
				})
				milestoneFailure(t, call("work_log", map[string]any{"action": "list"}), "forbidden")
				for _, row := range summary() {
					require.NotEqual(t, item, row["issue_id"])
				}
			})
		}
		require.Contains(t, summary(), map[string]any{"issue_id": item, "duration": float64((int64(2147483647) + 30) * 60)})
		t.Run("archived project", func(t *testing.T) {
			_, err := f.pool.Exec(t.Context(), `UPDATE projects SET archived_at=now() WHERE id=$1`, testsupport.LabelProject)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE projects SET archived_at=NULL WHERE id=$1`, testsupport.LabelProject)
				require.NoError(t, err)
			})
			milestoneFailure(t, call("work_log", map[string]any{"action": "list"}), "forbidden")
			milestoneFailure(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}), "forbidden")
		})
	})
}

func TestMCPWorkLogOwnershipFollowsNativeItemMove(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		item := f.item(t)
		destination := milestoneObject(t, root("project", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": "Move destination " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8])}))["id"].(string)
		original := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": item})
		moved := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": destination, "workitem_id": item})
		created := milestoneObject(t, original("work_log", map[string]any{"action": "create", "duration": 25, "request_id": uuid.NewString()}))
		id := created["work_log_id"].(string)
		otherItem := f.item(t)
		before := workLogStorageSnapshot(t, f)
		milestoneFailure(t, original("work_log", map[string]any{"action": "update", "work_log_id": id, "workitem_id": otherItem, "duration": 35}), "not_found")
		require.Equal(t, before, workLogStorageSnapshot(t, f))
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET project_id=$2,workspace_id=$3 WHERE id=$1`, item, otherProject, otherWorkspace)
		require.ErrorContains(t, err, "cannot move between workspaces")
		require.Equal(t, before, workLogStorageSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET project_id=$2 WHERE id=$1`, item, destination)
		require.NoError(t, err)
		milestoneFailure(t, original("work_log", map[string]any{"action": "list"}), "forbidden")
		rows := catalogueArray(t, moved("work_log", map[string]any{"action": "list", "include_revision": true}))
		require.Len(t, rows, 1)
		require.Equal(t, id, rows[0]["id"])
		require.Equal(t, testsupport.LabelWorkspace, rows[0]["workspace_id"])
		require.Equal(t, destination, rows[0]["project_id"])
		require.NotEqual(t, created["revision"], rows[0]["revision"])
		milestoneFailure(t, moved("work_log", map[string]any{"action": "update", "work_log_id": id, "duration": 35, "expected_revision": created["revision"]}), "revision_conflict")
		totals := catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "workspace_slug": "workspace-one", "project_id": destination}))
		require.Contains(t, totals, map[string]any{"issue_id": item, "duration": float64(1500)})
		totals = catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject}))
		for _, row := range totals {
			require.NotEqual(t, item, row["issue_id"])
		}
	})
}
