//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerPropertyDefaultStrings(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		for _, tc := range []struct {
			name, input string
			want        []any
		}{
			{"literal bracket", " [unfinished ", []any{"[unfinished"}},
			{"literal comma", " retail, enterprise ", []any{"retail, enterprise"}},
			{"encoded strings", `["retail", "enterprise"]`, []any{"retail", "enterprise"}},
			{"empty array", `[]`, []any{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Run("create", func(t *testing.T) {
					property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Default " + uuid.NewString(), "property_type": "TEXT", "is_multi": true, "default_value": tc.input}))
					require.Equal(t, tc.want, property["default_value"])
					assertCustomerPropertyDefault(t, call, property["id"].(string), tc.want)
				})
				t.Run("update", func(t *testing.T) {
					property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Default " + uuid.NewString(), "property_type": "TEXT", "is_multi": true, "default_value": "initial"}))
					updated := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": property["id"], "default_value": tc.input}))
					require.Equal(t, tc.want, updated["default_value"])
					assertCustomerPropertyDefault(t, call, property["id"].(string), tc.want)
				})
			})
		}
	})
}

func assertCustomerPropertyDefault(t *testing.T, call catalogueCall, propertyID string, want []any) {
	t.Helper()
	customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
	values := milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"], "property_id": propertyID}))
	require.Equal(t, map[string]any{propertyID: want}, values)
	catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
	catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
}

func TestMCPCustomerPropertyRejectsNonStringArrays(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Validation " + uuid.NewString(), "property_type": "TEXT", "is_multi": true, "default_value": "retained"}))
		propertyID := property["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		read := map[string]any{"action": "get_values", "customer_id": customer["id"], "property_id": propertyID}
		before := milestoneObject(t, call("customer_property", read))
		require.Equal(t, map[string]any{propertyID: []any{"retained"}}, before)
		for _, value := range []string{`null`, `[null]`, `[1]`, `["valid", null]`, `"scalar"`, `{}`} {
			t.Run(value, func(t *testing.T) {
				milestoneFailure(t, call("customer_property", map[string]any{"action": "set_values", "customer_id": customer["id"], "values": `{"` + propertyID + `":` + value + `}`}), "invalid_params")
				require.Equal(t, before, milestoneObject(t, call("customer_property", read)))
			})
		}
		for _, value := range []string{`[null]`, `[1]`, `["valid", null]`, `[null, "valid"]`} {
			t.Run("invalid default "+value, func(t *testing.T) {
				milestoneFailure(t, call("customer_property", map[string]any{"action": "create", "display_name": "Invalid " + uuid.NewString(), "property_type": "TEXT", "is_multi": true, "default_value": value}), "invalid_params")
				milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "default_value": value}), "invalid_params")
				detail := milestoneObject(t, call("customer_property", map[string]any{"action": "retrieve", "property_id": propertyID}))
				require.Equal(t, []any{"retained"}, detail["default_value"])
				require.Equal(t, before, milestoneObject(t, call("customer_property", read)))
			})
		}
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
	})
}

func TestMCPCustomerPropertyClearPreservesOtherValues(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		first := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "First " + uuid.NewString(), "property_type": "TEXT", "default_value": "first default"}))
		second := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Second " + uuid.NewString(), "property_type": "TEXT", "default_value": "second default"}))
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		firstID, secondID := first["id"].(string), second["id"].(string)
		read := map[string]any{"action": "get_values", "customer_id": customer["id"]}
		require.Equal(t, map[string]any{firstID: []any{"first default"}, secondID: []any{"second default"}}, milestoneObject(t, call("customer_property", read)))
		encoded, err := json.Marshal(map[string][]string{firstID: {}})
		require.NoError(t, err)
		catalogueNull(t, call("customer_property", map[string]any{"action": "set_values", "customer_id": customer["id"], "values": string(encoded)}))
		require.Equal(t, map[string]any{firstID: []any{}, secondID: []any{"second default"}}, milestoneObject(t, call("customer_property", read)))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": firstID}))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": secondID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
	})
}
