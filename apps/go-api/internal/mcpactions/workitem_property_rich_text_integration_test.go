//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyRichTextRecordsHistoryOncePerChange(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		typeRow := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		property := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": typeRow["id"], "display_name": uuid.NewString(), "property_type": "RELATION", "relation_type": "RICH_TEXT"}))
		item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Rich text history", "type_id": typeRow["id"]}))
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"]})
		input := map[string]any{"action": "set_value", "description_html": "<p>First note</p>", "request_id": uuid.NewString()}
		accepted := milestoneObject(t, call("workitem_property", input))
		result := accepted["result"].(map[string]any)
		require.Equal(t, result["id"], result["value"])
		require.Equal(t, map[string]any{"id": result["id"], "description_html": "<p>First note</p>", "description_stripped": "First note"}, result["value_detail"])
		require.Equal(t, result, milestoneObject(t, call("workitem_property", map[string]any{"action": "get_value"})))
		assertMCPPropertyHistory(t, f, item["id"], property["id"], 1)
		before := propertyStorageSnapshot(t, f)
		accepted["replayed"] = true
		require.Equal(t, accepted, milestoneObject(t, call("workitem_property", input)))
		require.Equal(t, before, propertyStorageSnapshot(t, f))
		input["request_id"] = uuid.NewString()
		milestoneObject(t, call("workitem_property", input))
		assertMCPPropertyHistory(t, f, item["id"], property["id"], 1)
		input["description_html"], input["request_id"] = "<p>Second note</p>", uuid.NewString()
		milestoneObject(t, call("workitem_property", input))
		assertMCPPropertyHistory(t, f, item["id"], property["id"], 2)
		catalogueNull(t, call("workitem_property", map[string]any{"action": "delete_value"}))
		assertMCPPropertyHistory(t, f, item["id"], property["id"], 3)
		var cleared bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT value='null'::jsonb AND source='change' AND actor_id=$3 AND property_title=$4 FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, item["id"], property["id"], testsupport.LabelActor, property["display_name"]).Scan(&cleared))
		require.True(t, cleared)
	})
}

func assertMCPPropertyHistory(t *testing.T, f milestoneFixture, item, property any, expected int) {
	t.Helper()
	var versions, activities int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$2),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_property')`, item, property).Scan(&versions, &activities))
	require.Equal(t, expected, versions)
	require.Equal(t, expected, activities)
}
