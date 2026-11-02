//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCatalogueRecordReceiptsRetainAcceptedTimezone(t *testing.T) {
	runCatalogue(t, newMilestoneFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET initiatives=true`)
		require.NoError(t, err)
		for _, scenario := range []struct{ tool, action string }{
			{"release", "create"}, {"release", "update"}, {"release", "update_changelog"},
			{"release_tag", "create"}, {"release_tag", "update"},
			{"initiative", "create"}, {"initiative", "update"},
			{"page", "create"}, {"page", "update"},
			{"collection", "create"}, {"collection", "update"},
		} {
			t.Run(scenario.tool+"/"+scenario.action, func(t *testing.T) {
				input := map[string]any{"action": scenario.action, "request_id": uuid.NewString()}
				nameField, selector := "name", scenario.tool+"_id"
				if scenario.tool == "release_tag" {
					nameField, selector = "version", "tag_id"
				}
				input[nameField] = uuid.NewString()
				seed := map[string]any{"action": "create", nameField: uuid.NewString()}
				if scenario.tool == "page" {
					seed["description_html"], seed["access"] = "<p>Receipt timezone</p>", 1
				}
				if scenario.tool == "collection" {
					seed["access"] = "private"
				}
				if scenario.action == "create" {
					seed["request_id"] = input["request_id"]
					input = seed
				} else {
					created := milestoneObject(t, call(scenario.tool, seed))
					input[selector] = created["id"]
				}
				if scenario.action == "update_changelog" {
					delete(input, "name")
					input["description_stripped"] = "Accepted release notes"
				}
				assertCatalogueReceiptTimezoneReplay(t, f, call, scenario.tool, input)
			})
		}
	})
}

func TestMCPReleaseResultsPreserveOpaqueDescriptionJSON(t *testing.T) {
	runCatalogue(t, newMilestoneFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata' WHERE id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
		created := milestoneObject(t, call("release", map[string]any{"action": "create", "name": uuid.NewString()}))
		id := created["id"]
		opaque := map[string]any{"type": "doc", "content": []any{}, "attrs": map[string]any{"created_at": "2024-01-02T03:04:05Z", "updated_at": "2024-02-03T04:05:06Z"}}
		body, err := json.Marshal(opaque)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE work_item_releases SET description_json=$2::jsonb WHERE id=$1`, id, body)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE release_changelogs SET description_json=$2::jsonb WHERE release_id=$1`, id, body)
		require.NoError(t, err)
		retrieved := milestoneObject(t, call("release", map[string]any{"action": "retrieve", "release_id": id}))
		assertCatalogueTimezone(t, retrieved)
		require.Equal(t, opaque, retrieved["description"].(map[string]any)["description_json"])
		page := milestoneObject(t, call("release", map[string]any{"action": "list"}))
		require.Contains(t, page["results"], retrieved)
		changelog := milestoneObject(t, call("release", map[string]any{"action": "get_changelog", "release_id": id}))
		assertCatalogueTimezone(t, changelog)
		require.Equal(t, opaque, changelog["changelog"].(map[string]any)["description_json"])
		updated := milestoneObject(t, call("release", map[string]any{"action": "update", "release_id": id, "name": uuid.NewString(), "request_id": uuid.NewString()}))
		require.Equal(t, opaque, updated["result"].(map[string]any)["description"].(map[string]any)["description_json"])
	})
}

func TestMCPRelationAndIntakeReceiptsRetainAcceptedTimezone(t *testing.T) {
	runCatalogue(t, newMilestoneFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		for _, action := range []string{"create_definition", "update_definition"} {
			t.Run("relation/"+action, func(t *testing.T) {
				input := map[string]any{"action": action, "name": uuid.NewString(), "request_id": uuid.NewString()}
				if action == "update_definition" {
					created := milestoneObject(t, call("workitem_relation", map[string]any{"action": "create_definition", "name": uuid.NewString()}))
					input["definition_id"] = created["id"]
				}
				assertCatalogueReceiptTimezoneReplay(t, f, call, "workitem_relation", input)
			})
		}
		for _, custom := range []bool{false, true} {
			name := "dependency"
			if custom {
				name = "custom"
			}
			t.Run("relation/create/"+name, func(t *testing.T) {
				first := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
				second := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
				input := map[string]any{"action": "create", "project_id": testsupport.LabelProject, "workitem_id": first["id"], "workitem_ids": []any{second["id"]}, "relation_type": "blocking", "request_id": uuid.NewString()}
				if custom {
					definition := milestoneObject(t, call("workitem_relation", map[string]any{"action": "create_definition", "name": uuid.NewString(), "outward": "supports", "inward": "supported by"}))
					delete(input, "relation_type")
					input["relation_definition_id"], input["relation_definition_label"] = definition["id"], "supports"
				}
				assertCatalogueReceiptTimezoneReplay(t, f, call, "workitem_relation", input)
			})
		}
		scoped := catalogueScoped(call, map[string]any{"project_id": testsupport.LabelProject})
		milestoneObject(t, scoped("state", map[string]any{"action": "create", "name": uuid.NewString(), "color": "#123456", "group": "unstarted", "default": true}))
		milestoneObject(t, scoped("project", map[string]any{"action": "update_features", "intakes": true}))
		for _, action := range []string{"create", "update"} {
			t.Run("intake/"+action, func(t *testing.T) {
				input := map[string]any{"action": action, "name": uuid.NewString(), "priority": "high", "request_id": uuid.NewString()}
				if action == "update" {
					created := milestoneObject(t, scoped("intake", map[string]any{"action": "create", "name": uuid.NewString()}))
					input = map[string]any{"action": "update", "workitem_id": created["issue"], "status": 1, "request_id": input["request_id"]}
				}
				assertCatalogueReceiptTimezoneReplay(t, f, scoped, "intake", input)
			})
		}
	})
}

func assertCatalogueReceiptTimezoneReplay(t *testing.T, f milestoneFixture, call catalogueCall, tool string, input map[string]any) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata' WHERE id=$1`, testsupport.LabelActor)
	require.NoError(t, err)
	accepted := milestoneObject(t, call(tool, input))
	require.Equal(t, false, accepted["replayed"])
	assertReceiptTimestampResult(t, accepted["result"])
	_, err = f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='UTC' WHERE id=$1`, testsupport.LabelActor)
	require.NoError(t, err)
	accepted["replayed"] = true
	require.Equal(t, accepted, milestoneObject(t, call(tool, input)), "timezone changes must not reinterpret the accepted receipt")
}

func assertReceiptTimestampResult(t *testing.T, value any) {
	t.Helper()
	if rows, ok := value.([]any); ok {
		require.NotEmpty(t, rows)
		for _, row := range rows {
			assertReceiptTimestampResult(t, row)
		}
		return
	}
	row, ok := value.(map[string]any)
	require.True(t, ok, "a record or record collection is required")
	assertCatalogueTimezone(t, row)
}
