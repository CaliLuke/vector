//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func templateScopedInput(scope templateScope, action string) map[string]any {
	input := map[string]any{"action": action, "kind": scope.kind, "workspace_slug": "workspace-one"}
	if scope.project != "" {
		input["project_id"] = scope.project
	}
	return input
}

func assertTemplateSDKRecord(t *testing.T, scope templateScope, record map[string]any) {
	t.Helper()
	keys := []string{"id", "name", "template_data", "created_at", "updated_at", "workspace", "project"}
	summary := "description"
	if scope.project != "" {
		keys = append(keys, "short_description", "template_type")
		summary = "short_description"
		require.Equal(t, scope.project, record["project"])
		require.Equal(t, scope.kind, record["template_type"])
	} else {
		keys = append(keys, "description", "logo_props")
		require.Nil(t, record["project"])
		require.Equal(t, map[string]any{}, record["logo_props"])
		if scope.kind != "project" {
			keys = append(keys, "description_html")
			summary = "description_html"
			require.Equal(t, map[string]any{}, record["description"])
		}
	}
	require.ElementsMatch(t, keys, slices.Collect(maps.Keys(record)))
	require.Equal(t, testsupport.LabelWorkspace, record["workspace"])
	require.Equal(t, "<p>Template summary</p>", record[summary])
	require.NotEmpty(t, record["created_at"])
	require.NotEmpty(t, record["updated_at"])
}

func TestMCPTemplateSDKRecordsPreserveBodyAndMergeFields(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, scope := range templateScopes() {
			t.Run(scope.kind+"/"+scope.project, func(t *testing.T) {
				input := templateScopedInput(scope, "create")
				input["name"], input["description"] = "Template "+uuid.NewString(), "<p>Template summary</p>"
				input["template_data"] = `{"name":"Templated resource","description_html":"<script>opaque</script>","nested":{"retained":true},"serial":9007199254740993}`
				response := call("template", input)
				record := milestoneObject(t, response)
				assertTemplateSDKRecord(t, scope, record)
				require.Equal(t, input["name"], record["name"])
				require.Contains(t, response.Content[0].Text, "9007199254740993")
				id := record["id"]
				update := templateScopedInput(scope, "update")
				update["template_id"], update["template_data"] = id, `{"nested":{"replacement":true},"optional":null}`
				response = call("template", update)
				updated := milestoneObject(t, response)
				assertTemplateSDKRecord(t, scope, updated)
				require.Equal(t, record["name"], updated["name"])
				require.Equal(t, record["created_at"], updated["created_at"])
				body := updated["template_data"].(map[string]any)
				require.Equal(t, "Templated resource", body["name"])
				require.Equal(t, "<script>opaque</script>", body["description_html"])
				require.Equal(t, map[string]any{"replacement": true}, body["nested"])
				require.Contains(t, body, "optional")
				require.Nil(t, body["optional"])
				require.Contains(t, response.Content[0].Text, "9007199254740993")
				var stored, original, changed string
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT template_data::text FROM resource_templates WHERE id=$1`, id).Scan(&stored))
				require.JSONEq(t, `{"name":"Templated resource","description_html":"<script>opaque</script>","nested":{"replacement":true},"optional":null,"serial":9007199254740993}`, stored)
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT before_value->'template_data',after_value->'template_data' FROM template_events WHERE template_id=$1 AND action='update'`, id).Scan(&original, &changed))
				require.JSONEq(t, input["template_data"].(string), original)
				require.JSONEq(t, stored, changed)
				rows := catalogueArray(t, call("template", templateScopedInput(scope, "list")))
				require.Contains(t, rows, updated)
				deletion := templateScopedInput(scope, "delete")
				deletion["template_id"] = id
				catalogueNull(t, call("template", deletion))
				require.NotContains(t, catalogueArray(t, call("template", templateScopedInput(scope, "list"))), updated)
				var deleted bool
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM resource_templates WHERE id=$1`, id).Scan(&deleted))
				require.True(t, deleted)
				var events int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM template_events WHERE template_id=$1`, id).Scan(&events))
				require.Equal(t, 3, events)
			})
		}
	})
}
