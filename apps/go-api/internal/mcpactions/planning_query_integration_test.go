//go:build integration

package mcpactions_test

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningQueriesFilterBeforePagingAndProjection(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, otherProject := addPrivateProjectInOtherWorkspace(t, f)
	runSelectedWorkspaceMatrix(t, f, func(t *testing.T, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject})
		milestoneObject(t, call("project", map[string]any{"action": "update_features", "cycles": true, "modules": true}))
		for _, kind := range []string{"cycle", "module"} {
			t.Run(kind, func(t *testing.T) {
				_, parent := createPlanningReceipt(t, call, kind)
				parentID := parent[kind+"_id"].(string)
				items := seedPlanningQueryItems(t, f, call, kind, parentID, otherWorkspace, otherProject)
				before := planningMutationSnapshot(t, f)
				query := `priority = "high" AND assignee = currentUser()`
				input := map[string]any{"action": "list_workitems", kind + "_id": parentID, "pql": query,
					"per_page": 1, "order_by": "sequence_id", "fields": "id,name"}
				assertPlanningQueryPages(t, call, kind, input,
					map[string]any{"id": items[1], "name": "Query item 2"},
					map[string]any{"id": items[3], "name": "Query item 4"})
				count := milestoneObject(t, call("workitem", map[string]any{"action": "count",
					"pql": query + fmt.Sprintf(` AND %s = "%s"`, kind, parentID), "group_by": "priority"}))
				require.Equal(t, float64(2), count["total_count"])
				require.Equal(t, before, planningMutationSnapshot(t, f))
			})
		}
	})
}

// The first and third linked items fail the filter; the fifth is unlinked.
// Archived, draft, deleted and incorrectly scoped rows must not affect totals.
func seedPlanningQueryItems(t *testing.T, f milestoneFixture, call catalogueCall, kind, parentID, otherWorkspace, otherProject string) []string {
	t.Helper()
	items := make([]string, 9)
	for i := range items {
		items[i] = f.item(t)
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET name=$2,sequence_id=$3,priority='high' WHERE id=$1`, items[i], fmt.Sprintf("Query item %d", i+1), i+1)
		require.NoError(t, err)
	}
	_, err := f.pool.Exec(t.Context(), `UPDATE issues SET priority='low' WHERE id=$1`, items[0])
	require.NoError(t, err)
	assigned := []string{items[0], items[1], items[3], items[4], items[5], items[6], items[7], items[8]}
	_, err = f.pool.Exec(t.Context(), `INSERT INTO issue_assignees(id,created_at,updated_at,workspace_id,project_id,issue_id,assignee_id)
 SELECT gen_random_uuid(),now(),now(),workspace_id,project_id,id,$2 FROM issues WHERE id=ANY($1::uuid[])`, assigned, testsupport.LabelActor)
	require.NoError(t, err)
	linked := []string{items[0], items[1], items[2], items[3], items[5], items[6], items[7], items[8]}
	catalogueNull(t, call(kind, map[string]any{"action": "manage_workitems", kind + "_id": parentID, "add_ids": strings.Join(linked, ",")}))
	_, err = f.pool.Exec(t.Context(), `UPDATE issues SET archived_at=now() WHERE id=$1`, items[5])
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `UPDATE issues SET is_draft=true WHERE id=$1`, items[6])
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `UPDATE issues SET deleted_at=now() WHERE id=$1`, items[7])
	require.NoError(t, err)
	table, column := pgx.Identifier{kind + "_issues"}.Sanitize(), pgx.Identifier{kind + "_id"}.Sanitize()
	_, err = f.pool.Exec(t.Context(), "UPDATE "+table+" SET workspace_id=$2,project_id=$3 WHERE issue_id=$1", items[8], otherWorkspace, otherProject)
	require.NoError(t, err)
	foreign := f.itemInScope(t, otherWorkspace, otherProject)
	_, err = f.pool.Exec(t.Context(), `UPDATE issues SET priority='high' WHERE id=$1`, foreign)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO issue_assignees(id,created_at,updated_at,workspace_id,project_id,issue_id,assignee_id)
 SELECT gen_random_uuid(),now(),now(),workspace_id,project_id,id,$2 FROM issues WHERE id=$1`, foreign, testsupport.LabelActor)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), "INSERT INTO "+table+"(id,created_at,updated_at,workspace_id,project_id,issue_id,"+column+") VALUES(gen_random_uuid(),now(),now(),$1,$2,$3,$4)", testsupport.LabelWorkspace, testsupport.LabelProject, foreign, parentID)
	require.NoError(t, err)
	return items
}

func TestMCPPlanningQueryFailuresReturnReferenceWithoutPartialResults(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		_, parent := createPlanningReceipt(t, call, kind)
		item := f.item(t)
		catalogueNull(t, call(kind, map[string]any{"action": "manage_workitems", kind + "_id": parent[kind+"_id"], "add_ids": item}))
		before := planningMutationSnapshot(t, f)
		input := map[string]any{"action": "list_workitems", kind + "_id": parent[kind+"_id"],
			"pql": `priority = "none" OR unknownField = "private"`}
		failed := milestoneObject(t, call(kind, input))
		require.Equal(t, input["pql"], failed["failed_pql"])
		require.NotEmpty(t, failed["error"])
		require.Contains(t, failed["hint"], kind+" list_workitems")
		require.NotContains(t, failed, "results")
		reference, err := os.ReadFile("pql_full.txt")
		require.NoError(t, err)
		require.Equal(t, string(reference), failed["pql_reference"])
		fixed := maps.Clone(input)
		fixed["pql"] = `priority = "none"`
		rows := catalogueRows(t, call(kind, fixed))
		require.Len(t, rows, 1)
		require.Equal(t, item, rows[0].(map[string]any)["id"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
	})
}

func assertPlanningQueryPages(t *testing.T, call catalogueCall, kind string, input, firstRow, secondRow map[string]any) {
	t.Helper()
	first := milestoneObject(t, call(kind, input))
	require.Len(t, first, 7)
	require.Equal(t, float64(1), first["count"])
	require.Equal(t, float64(2), first["total_count"])
	require.Equal(t, true, first["next_page_results"])
	require.Equal(t, false, first["prev_page_results"])
	require.Equal(t, []any{firstRow}, first["results"])
	next := maps.Clone(input)
	next["cursor"] = first["next_cursor"]
	second := milestoneObject(t, call(kind, next))
	require.Len(t, second, 7)
	require.Equal(t, float64(1), second["count"])
	require.Equal(t, float64(2), second["total_count"])
	require.Equal(t, false, second["next_page_results"])
	require.Equal(t, true, second["prev_page_results"])
	require.Equal(t, []any{secondRow}, second["results"])
}
