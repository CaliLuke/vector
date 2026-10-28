//go:build integration

package mcpactions_test

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func catalogueScoped(call catalogueCall, scope map[string]any) catalogueCall {
	return func(tool string, input map[string]any) rpcResult {
		fields := maps.Clone(scope)
		maps.Copy(fields, input)
		return call(tool, fields)
	}
}

func catalogueRows(t *testing.T, result rpcResult) []any {
	t.Helper()
	page := milestoneObject(t, result)
	rows, ok := page["results"].([]any)
	require.True(t, ok, page)
	return rows
}

func TestMCPCoreCatalogueBatch(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		t.Run("project and workspace", func(t *testing.T) { catalogueProjectWrites(t, call) })
		t.Run("work items and queries", func(t *testing.T) { catalogueWorkitems(t, call) })
		t.Run("planning", func(t *testing.T) { cataloguePlanning(t, f, call) })
		t.Run("comments links and work logs", func(t *testing.T) { catalogueCollaborationWrites(t, f, call) })
	})
}

func catalogueProjectWrites(t *testing.T, call catalogueCall) {
	workspace := milestoneObject(t, call("workspace", map[string]any{"action": "retrieve"}))
	require.Equal(t, "workspace-one", workspace["slug"])
	if workspace["connected_via"] == "oauth" {
		require.Equal(t, testsupport.LabelWorkspace, workspace["id"])
	} else {
		require.Nil(t, workspace["id"])
	}
	input := map[string]any{"action": "create", "name": "Core batch", "identifier": strings.ToUpper(uuid.NewString()[:8]), "description": "Before", "request_id": uuid.NewString()}
	created := milestoneObject(t, call("project", input))
	id := created["project_id"].(string)
	require.Equal(t, true, milestoneObject(t, call("project", input))["replayed"])
	updated := milestoneObject(t, call("project", map[string]any{"action": "update", "project_id": id, "description": "After", "is_time_tracking_enabled": true}))
	require.Equal(t, "After", updated["description"])
	require.Equal(t, true, updated["is_time_tracking_enabled"])
	catalogueNull(t, call("project", map[string]any{"action": "delete", "project_id": id}))
	milestoneFailure(t, call("project", map[string]any{"action": "retrieve", "project_id": id}), "not_found")
}

func catalogueWorkitems(t *testing.T, root catalogueCall) {
	project := milestoneObject(t, root("project", map[string]any{"action": "create", "name": "Items", "identifier": strings.ToUpper(uuid.NewString()[:8])}))
	projectID := project["id"].(string)
	call := catalogueScoped(root, map[string]any{"project_id": projectID})
	label := milestoneObject(t, call("label", map[string]any{"action": "create", "name": "Delivery", "color": "#123456"}))
	labelID := label["id"].(string)
	require.Equal(t, "Ready", milestoneObject(t, call("label", map[string]any{"action": "update", "label_id": labelID, "name": "Ready"}))["name"])
	input := map[string]any{"action": "create", "name": "Core lifecycle", "priority": "high", "start_date": "2026-10-20", "target_date": "2026-10-28", "request_id": uuid.NewString()}
	receipt := milestoneObject(t, call("workitem", input))
	id := receipt["workitem_id"].(string)
	require.Equal(t, true, milestoneObject(t, call("workitem", input))["replayed"])
	itemCall := catalogueScoped(call, map[string]any{"workitem_id": id})
	item := milestoneObject(t, itemCall("workitem", map[string]any{"action": "update", "name": "Core renamed", "start_date": nil}))
	require.Nil(t, item["start_date"])
	require.Equal(t, "2026-10-28", item["target_date"])
	assigned := milestoneObject(t, itemCall("workitem", map[string]any{"action": "manage_assignee", "add_user_id": testsupport.LabelActor}))
	require.Equal(t, []any{testsupport.LabelActor}, assigned["assignees"])
	labeled := milestoneObject(t, itemCall("workitem", map[string]any{"action": "manage_label", "add_label_id": labelID}))
	require.Equal(t, []any{labelID}, labeled["labels"])
	detail := milestoneObject(t, itemCall("workitem", map[string]any{"action": "retrieve"}))
	require.Equal(t, id, detail["id"])
	identifier := fmt.Sprintf("%s-%.0f", project["identifier"], detail["sequence_id"])
	require.Equal(t, id, milestoneObject(t, root("workitem", map[string]any{"action": "retrieve_by_identifier", "workitem_identifier": identifier}))["id"])
	query := `priority = "high" AND assignee = currentUser()`
	rows := catalogueRows(t, call("workitem", map[string]any{"action": "list", "pql": query, "per_page": 1}))
	require.Len(t, rows, 1)
	require.Equal(t, id, rows[0].(map[string]any)["id"])
	count := milestoneObject(t, call("workitem", map[string]any{"action": "count", "pql": query, "group_by": "priority", "sub_group_by": "labels__id"}))
	require.Equal(t, float64(1), count["total_count"])
	search := milestoneObject(t, root("workitem", map[string]any{"action": "search", "query": "Core renamed"}))
	require.Contains(t, fmt.Sprint(search), id)
	for _, detail := range []string{"brief", "full"} {
		reference := milestoneObject(t, root("get_pql_reference", map[string]any{"detail": detail}))
		require.Equal(t, detail, reference["detail"])
		require.Contains(t, reference["reference"], "currentUser()")
	}
	completed := milestoneObject(t, call("state", map[string]any{"action": "create", "name": "Batch done", "color": "#123456", "group": "completed"}))
	milestoneObject(t, itemCall("workitem", map[string]any{"action": "update", "state": completed["id"]}))
	require.Equal(t, true, milestoneObject(t, itemCall("workitem", map[string]any{"action": "archive"}))["archived"])
	archived := catalogueRows(t, call("workitem", map[string]any{"action": "list_archived"}))
	require.Len(t, archived, 1)
	require.Equal(t, id, archived[0].(map[string]any)["id"])
	require.Equal(t, false, milestoneObject(t, itemCall("workitem", map[string]any{"action": "archive", "archive": false}))["archived"])
	catalogueNull(t, itemCall("workitem", map[string]any{"action": "delete"}))
	catalogueNull(t, call("label", map[string]any{"action": "delete", "label_id": labelID}))
	catalogueNull(t, root("project", map[string]any{"action": "delete", "project_id": projectID}))
}

func cataloguePlanning(t *testing.T, f milestoneFixture, root catalogueCall) {
	call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
	item := f.item(t)
	milestoneObject(t, call("project", map[string]any{"action": "update_features", "cycles": true, "modules": true}))
	state := milestoneObject(t, call("state", map[string]any{"action": "create", "name": "Planning " + uuid.NewString(), "color": "#123456", "group": "unstarted"}))
	milestoneObject(t, call("workitem", map[string]any{"action": "update", "workitem_id": item, "state": state["id"]}))
	cycle := milestoneObject(t, call("cycle", map[string]any{"action": "create", "name": "First cycle", "owned_by": testsupport.LabelActor}))
	id := cycle["id"].(string)
	require.Equal(t, "First revised", milestoneObject(t, call("cycle", map[string]any{"action": "update", "cycle_id": id, "name": "First revised"}))["name"])
	catalogueNull(t, call("cycle", map[string]any{"action": "manage_workitems", "cycle_id": id, "add_ids": item}))
	require.Len(t, catalogueRows(t, call("cycle", map[string]any{"action": "list_workitems", "cycle_id": id})), 1)
	second := milestoneObject(t, call("cycle", map[string]any{"action": "create", "name": "Second cycle", "owned_by": testsupport.LabelActor}))
	catalogueNull(t, call("cycle", map[string]any{"action": "transfer_workitems", "cycle_id": id, "new_cycle_id": second["id"]}))
	require.Empty(t, catalogueRows(t, call("cycle", map[string]any{"action": "list_workitems", "cycle_id": id})))
	require.Len(t, catalogueRows(t, call("cycle", map[string]any{"action": "list_workitems", "cycle_id": second["id"]})), 1)
	milestoneObject(t, call("cycle", map[string]any{"action": "complete", "cycle_id": id}))
	for _, action := range []string{"archive", "unarchive", "delete"} {
		catalogueNull(t, call("cycle", map[string]any{"action": action, "cycle_id": id}))
	}
	module := milestoneObject(t, call("module", map[string]any{"action": "create", "name": "Module", "members": testsupport.LabelActor}))
	moduleID := module["id"].(string)
	require.Equal(t, "completed", milestoneObject(t, call("module", map[string]any{"action": "update", "module_id": moduleID, "status": "completed"}))["status"])
	catalogueNull(t, call("module", map[string]any{"action": "manage_workitems", "module_id": moduleID, "add_ids": item}))
	require.Len(t, catalogueRows(t, call("module", map[string]any{"action": "list_workitems", "module_id": moduleID})), 1)
	for _, action := range []string{"archive", "unarchive", "delete"} {
		catalogueNull(t, call("module", map[string]any{"action": action, "module_id": moduleID}))
	}
	var live bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&live))
	require.True(t, live, "planning deletion must retain its work items")
}

func catalogueCollaborationWrites(t *testing.T, f milestoneFixture, root catalogueCall) {
	item := f.item(t)
	call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
	comment := milestoneObject(t, call("workitem_comment", map[string]any{"action": "create", "comment_html": "<p>Initial</p>"}))
	id := comment["id"].(string)
	require.Contains(t, milestoneObject(t, call("workitem_comment", map[string]any{"action": "update", "comment_id": id, "comment_html": "<p>Edited</p>"}))["comment_html"], "Edited")
	catalogueNull(t, call("workitem_comment", map[string]any{"action": "delete", "comment_id": id}))
	link := milestoneObject(t, call("workitem_link", map[string]any{"action": "create", "url": "https://example.test/design", "title": "Design"}))
	id = link["id"].(string)
	require.Equal(t, "Specification", milestoneObject(t, call("workitem_link", map[string]any{"action": "update", "link_id": id, "title": "Specification"}))["title"])
	catalogueNull(t, call("workitem_link", map[string]any{"action": "delete", "link_id": id}))
	log := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 30, "description": "Implementation"}))
	id = log["id"].(string)
	require.Equal(t, float64(45), milestoneObject(t, call("work_log", map[string]any{"action": "update", "work_log_id": id, "duration": 45}))["duration"])
	logs := catalogueArray(t, call("work_log", map[string]any{"action": "list"}))
	require.Len(t, logs, 1)
	require.Equal(t, id, logs[0]["id"])
	totals := catalogueArray(t, root("project", map[string]any{"action": "worklog_summary", "project_id": testsupport.LabelProject}))
	require.Contains(t, totals, map[string]any{"issue_id": item, "duration": float64(2700)})
	catalogueNull(t, call("work_log", map[string]any{"action": "delete", "work_log_id": id}))
	require.Empty(t, catalogueArray(t, call("work_log", map[string]any{"action": "list"})))
}
