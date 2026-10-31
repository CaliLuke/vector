//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerPropertyKindsPreserveValidatedValues(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		for _, tc := range []struct {
			kind, relation, initial, replacement, invalid string
		}{
			{"TEXT", "", "retail", "enterprise", "two\nlines"},
			{"DATETIME", "", "2026-01-01", "2026-02-03T12:34:56Z", "2026-02-30"},
			{"DECIMAL", "", "-1.25e3", "9.75", "NaN"},
			{"BOOLEAN", "", "True", "False", "true"},
			{"URL", "", "https://example.test/old", "https://example.test/new", "javascript:alert(1)"},
			{"EMAIL", "", "old@example.test", "new@example.test", "not-an-email"},
			{"FILE", "", `{"asset":"opaque"}`, "opaque/file-id", "invalid\x00metadata"},
			{"RELATION", "USER", testsupport.LabelActor, testsupport.LabelActor, uuid.NewString()},
			{"RELATION", "ISSUE", f.item(t), f.item(t), uuid.NewString()},
		} {
			t.Run(tc.kind+tc.relation, func(t *testing.T) {
				property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": tc.kind + " " + uuid.NewString(), "property_type": tc.kind, "relation_type": tc.relation, "default_value": tc.initial}))
				propertyID := property["id"].(string)
				customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
				read := map[string]any{"action": "get_values", "customer_id": customer["id"], "property_id": propertyID}
				require.Equal(t, map[string]any{propertyID: []any{tc.initial}}, milestoneObject(t, call("customer_property", read)))
				catalogueNull(t, setCustomerValues(t, call, customer["id"].(string), map[string][]string{propertyID: {tc.replacement}}))
				retained := map[string]any{propertyID: []any{tc.replacement}}
				require.Equal(t, retained, milestoneObject(t, call("customer_property", read)))
				milestoneFailure(t, setCustomerValues(t, call, customer["id"].(string), map[string][]string{propertyID: {tc.invalid}}), "invalid_params")
				require.Equal(t, retained, milestoneObject(t, call("customer_property", read)))
				catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
				catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
			})
		}
	})
}

func setCustomerValues(t *testing.T, call catalogueCall, customerID string, values map[string][]string) rpcResult {
	t.Helper()
	encoded, err := json.Marshal(values)
	require.NoError(t, err)
	return call("customer_property", map[string]any{"action": "set_values", "customer_id": customerID, "values": string(encoded)})
}

func TestMCPCustomerPropertyInactiveValuesAreRetained(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Active " + uuid.NewString(), "property_type": "TEXT", "default_value": "retained"}))
		propertyID := property["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		customerID := customer["id"].(string)
		read := map[string]any{"action": "get_values", "customer_id": customerID}
		before := milestoneObject(t, call("customer_property", read))
		require.Equal(t, map[string]any{propertyID: []any{"retained"}}, before)
		milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_active": false}))
		require.Empty(t, milestoneObject(t, call("customer_property", read)))
		milestoneFailure(t, setCustomerValues(t, call, customerID, map[string][]string{propertyID: {"hidden write"}}), "invalid_params")
		milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_active": true}))
		require.Equal(t, before, milestoneObject(t, call("customer_property", read)))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
	})
}

func TestMCPCustomerPropertyReadonlyDefaultsCannotBeOverwritten(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Readonly " + uuid.NewString(), "property_type": "TEXT", "display_format": "readonly", "default_value": "retained"}))
		propertyID := property["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		customerID := customer["id"].(string)
		milestoneFailure(t, setCustomerValues(t, call, customerID, map[string][]string{propertyID: {"replacement"}}), "invalid_params")
		milestoneFailure(t, setCustomerValues(t, call, customerID, map[string][]string{propertyID: {}}), "invalid_params")
		values := milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customerID}))
		require.Equal(t, map[string]any{propertyID: []any{"retained"}}, values)
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
	})
}

func TestMCPCustomerRequiredPropertyRejectsAtomicClear(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		required := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Required " + uuid.NewString(), "property_type": "TEXT", "is_required": true}))
		optional := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Optional " + uuid.NewString(), "property_type": "TEXT", "default_value": "retained"}))
		requiredID, optionalID := required["id"].(string), optional["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		customerID := customer["id"].(string)
		catalogueNull(t, setCustomerValues(t, call, customerID, map[string][]string{requiredID: {"required value"}}))
		read := map[string]any{"action": "get_values", "customer_id": customerID}
		before := milestoneObject(t, call("customer_property", read))
		require.Equal(t, map[string]any{requiredID: []any{"required value"}, optionalID: []any{"retained"}}, before)
		milestoneFailure(t, setCustomerValues(t, call, customerID, map[string][]string{requiredID: {}, optionalID: {"replacement"}}), "invalid_params")
		require.Equal(t, before, milestoneObject(t, call("customer_property", read)))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": requiredID}))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": optionalID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
	})
}
