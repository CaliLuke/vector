//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyReceiptsRetainAcceptedTimezone(t *testing.T) {
	for _, action := range []string{"create", "update", "create_option", "update_option"} {
		t.Run(action, func(t *testing.T) {
			runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
				enableTypeCatalogue(t, f, "workspace-one")
				input := propertyRecordMutationInput(t, call, action)
				assertPropertyTimezoneReplay(t, f, call, input)
			})
		})
	}
}

func TestMCPPropertyValueReceiptsRetainAcceptedTimezone(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		name := "scalar"
		if multiple {
			name = "multiple"
		}
		t.Run(name, func(t *testing.T) {
			runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
				enableTypeCatalogue(t, f, "workspace-one")
				typeRow := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
				property := milestoneObject(t, call("workitem_property", map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "OPTION", "workitem_type_id": typeRow["id"], "is_multi": multiple, "options": `[{"name":"First"},{"name":"Second"}]`}))
				item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Timezone receipt", "type_id": typeRow["id"]}))
				options := property["options"].([]any)
				value := options[0].(map[string]any)["id"]
				if multiple {
					value = []any{value, options[1].(map[string]any)["id"]}
				}
				input := map[string]any{"action": "set_value", "project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"], "value": value}
				assertPropertyTimezoneReplay(t, f, call, input)
			})
		})
	}
}

func assertPropertyTimezoneReplay(t *testing.T, f milestoneFixture, call catalogueCall, input map[string]any) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='Asia/Kolkata'`)
	require.NoError(t, err)
	input["request_id"] = uuid.NewString()
	accepted := milestoneObject(t, call("workitem_property", input))
	require.Equal(t, false, accepted["replayed"])
	assertPropertyResultTimezone(t, accepted["result"])
	_, err = f.pool.Exec(t.Context(), `UPDATE users SET user_timezone='UTC'`)
	require.NoError(t, err)
	accepted["replayed"] = true
	require.Equal(t, accepted, milestoneObject(t, call("workitem_property", input)), "replay keeps the accepted result and receipt metadata after the actor changes timezone")
}

func assertPropertyResultTimezone(t *testing.T, result any) {
	t.Helper()
	switch value := result.(type) {
	case map[string]any:
		assertCatalogueTimezone(t, value)
		if options, ok := value["options"].([]any); ok {
			for _, option := range options {
				assertCatalogueTimezone(t, option.(map[string]any))
			}
		}
	case []any:
		require.Len(t, value, 2)
		for _, record := range value {
			assertCatalogueTimezone(t, record.(map[string]any))
		}
	default:
		t.Fatalf("expected a property, option, or property value result, got %T", result)
	}
}
