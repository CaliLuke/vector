//go:build integration

package mcpactions_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCoreReadsShareVisibleItemsAcrossQueriesCountsAndSearch(t *testing.T) {
	runCatalogueCredentials(t, templateFixture(t), func(t *testing.T, f milestoneFixture, root catalogueCall) {
		project := milestoneObject(t, root("project", map[string]any{"action": "create", "name": "Read acceptance " + uuid.NewString()[:8], "identifier": "R" + strings.ToUpper(uuid.NewString()[:6])}))
		projectID := project["id"].(string)
		call := catalogueScoped(root, map[string]any{"project_id": projectID})
		create := func(name, priority string) map[string]any {
			t.Helper()
			return milestoneObject(t, call("workitem", map[string]any{"action": "create", "name": name, "priority": priority}))
		}
		visible := create("Query proof visible", "high")
		second := create("Query proof second", "low")
		hidden := create("Query proof hidden", "high")
		draft := create("Query proof draft", "high")
		archived := create("Query proof archive", "high")
		other, _ := customerReferenceTarget(t, f, "USER")
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET created_by_id=$2 WHERE id=$1`, hidden["id"], other)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET is_draft=true WHERE id=$1`, draft["id"])
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET archived_at=CURRENT_DATE WHERE id=$1`, archived["id"])
		require.NoError(t, err)
		for range 2 {
			label := milestoneObject(t, call("label", map[string]any{"action": "create", "name": uuid.NewString(), "color": "#123456"}))["id"]
			milestoneObject(t, call("workitem", map[string]any{"action": "manage_label", "workitem_id": visible["id"], "add_label_id": label}))
			release := milestoneObject(t, root("release", map[string]any{"action": "create", "name": uuid.NewString()}))["id"]
			catalogueNull(t, root("release", map[string]any{"action": "manage_workitems", "release_id": release, "add_ids": visible["id"]}))
		}
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, projectID, testsupport.LabelActor)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET guest_view_all_features=false WHERE id=$1`, projectID)
		require.NoError(t, err)
		listed := milestoneObject(t, call("workitem", map[string]any{"action": "list", "per_page": 1, "fields": "id,name"}))
		require.EqualValues(t, 2, listed["total_count"])
		require.Len(t, listed["results"], 1)
		filtered := catalogueRows(t, call("workitem", map[string]any{"action": "list", "pql": `priority = "high"`}))
		require.Len(t, filtered, 1)
		require.Equal(t, visible["id"], filtered[0].(map[string]any)["id"])
		count := milestoneObject(t, call("workitem", map[string]any{"action": "count", "pql": `priority = "high"`, "group_by": "release_work_items__release_id", "sub_group_by": "labels__id"}))
		require.EqualValues(t, 1, count["total_count"])
		groups := count["grouped_counts"].(map[string]any)
		require.Len(t, groups, 2)
		for _, value := range groups {
			group := value.(map[string]any)
			require.EqualValues(t, 1, group["count"])
			require.Len(t, group["sub_grouped_counts"], 2)
		}
		search := fmt.Sprint(milestoneObject(t, root("workitem", map[string]any{"action": "search", "query": "Query proof"})))
		require.Contains(t, search, visible["id"])
		require.Contains(t, search, second["id"])
		require.NotContains(t, search, hidden["id"])
		require.NotContains(t, search, draft["id"])
		identifier := fmt.Sprintf("%s-%.0f", project["identifier"], visible["sequence_id"])
		require.Equal(t, visible["id"], milestoneObject(t, root("workitem", map[string]any{"action": "retrieve_by_identifier", "workitem_identifier": identifier}))["id"])
		hiddenIdentifier := fmt.Sprintf("%s-%.0f", project["identifier"], hidden["sequence_id"])
		milestoneFailure(t, root("workitem", map[string]any{"action": "retrieve_by_identifier", "workitem_identifier": hiddenIdentifier}), "not_found")
		milestoneFailure(t, call("workitem", map[string]any{"action": "retrieve", "workitem_id": hidden["id"]}), "not_found")
		require.Equal(t, visible["id"], milestoneObject(t, call("workitem", map[string]any{"action": "retrieve", "workitem_id": visible["id"]}))["id"])
		archivedRows := catalogueRows(t, call("workitem", map[string]any{"action": "list_archived"}))
		require.Len(t, archivedRows, 1)
		require.Equal(t, archived["id"], archivedRows[0].(map[string]any)["id"])
	})
}
