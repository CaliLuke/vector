//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerPropertyOptionDefaultsAndIdentity(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Options " + uuid.NewString(), "property_type": "OPTION", "options": `[{"name":"First","is_default":true},{"name":"Second"}]`}))
		propertyID := property["id"].(string)
		options := property["options"].([]any)
		require.Len(t, options, 2)
		firstID, secondID := options[0].(map[string]any)["id"].(string), options[1].(map[string]any)["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		customerID := customer["id"].(string)
		read := map[string]any{"action": "get_values", "customer_id": customerID, "property_id": propertyID}
		require.Equal(t, map[string]any{propertyID: []any{firstID}}, milestoneObject(t, call("customer_property", read)))

		patch, err := json.Marshal([]map[string]any{{"id": firstID, "name": "Renamed", "is_default": false}, {"id": secondID, "is_default": true}})
		require.NoError(t, err)
		updated := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "options": string(patch)}))
		updatedOptions := updated["options"].([]any)
		require.Len(t, updatedOptions, 2)
		require.Equal(t, firstID, updatedOptions[0].(map[string]any)["id"])
		require.Equal(t, "Renamed", updatedOptions[0].(map[string]any)["name"])
		require.Equal(t, secondID, updatedOptions[1].(map[string]any)["id"])
		require.Equal(t, "Second", updatedOptions[1].(map[string]any)["name"])
		require.Equal(t, true, updatedOptions[1].(map[string]any)["is_default"])
		require.Equal(t, map[string]any{propertyID: []any{firstID}}, milestoneObject(t, call("customer_property", read)))
		fresh := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		freshValues := milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": fresh["id"], "property_id": propertyID}))
		require.Equal(t, map[string]any{propertyID: []any{secondID}}, freshValues)

		appended := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "options": `[{"name":"Third"}]`}))
		appendedOptions := appended["options"].([]any)
		require.Len(t, appendedOptions, 3)
		require.Contains(t, appendedOptions, updatedOptions[0])
		require.Contains(t, appendedOptions, updatedOptions[1])
		catalogueNull(t, setCustomerValues(t, call, customerID, map[string][]string{propertyID: {secondID}}))
		require.Equal(t, map[string]any{propertyID: []any{secondID}}, milestoneObject(t, call("customer_property", read)))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": fresh["id"]}))
	})
}

func TestMCPCustomerPropertyRejectsForeignOptionsAtomically(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Own " + uuid.NewString(), "property_type": "OPTION", "options": `[{"name":"Own","is_default":true}]`}))
		other := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Other " + uuid.NewString(), "property_type": "OPTION", "options": `[{"name":"Other","is_default":true}]`}))
		propertyID, otherID := property["id"].(string), other["id"].(string)
		foreignOptionID := other["options"].([]any)[0].(map[string]any)["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		customerID := customer["id"].(string)
		read := map[string]any{"action": "get_values", "customer_id": customerID}
		before := milestoneObject(t, call("customer_property", read))
		milestoneFailure(t, setCustomerValues(t, call, customerID, map[string][]string{propertyID: {foreignOptionID}, otherID: {}}), "invalid_params")
		require.Equal(t, before, milestoneObject(t, call("customer_property", read)))
		patch, err := json.Marshal([]map[string]any{{"id": foreignOptionID, "name": "Foreign overwrite"}})
		require.NoError(t, err)
		milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "options": string(patch)}), "not_found")
		require.Equal(t, other, milestoneObject(t, call("customer_property", map[string]any{"action": "retrieve", "property_id": otherID})))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": otherID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
	})
}

func TestMCPCustomerPropertyOptionDefaultsRespectCardinality(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		options := `[{"name":"First","is_default":true},{"name":"Second","is_default":true}]`
		milestoneFailure(t, call("customer_property", map[string]any{"action": "create", "display_name": "Single " + uuid.NewString(), "property_type": "OPTION", "options": options}), "invalid_params")
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Multiple " + uuid.NewString(), "property_type": "OPTION", "is_multi": true, "options": options}))
		propertyID := property["id"].(string)
		createdOptions := property["options"].([]any)
		require.Len(t, createdOptions, 2)
		firstID, secondID := createdOptions[0].(map[string]any)["id"].(string), createdOptions[1].(map[string]any)["id"].(string)
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		read := map[string]any{"action": "get_values", "customer_id": customer["id"], "property_id": propertyID}
		require.Equal(t, map[string]any{propertyID: []any{firstID, secondID}}, milestoneObject(t, call("customer_property", read)))
		milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_multi": false}), "invalid_params")
		detail := milestoneObject(t, call("customer_property", map[string]any{"action": "retrieve", "property_id": propertyID}))
		require.Equal(t, true, detail["is_multi"])
		require.Equal(t, createdOptions, detail["options"])
		require.Equal(t, map[string]any{propertyID: []any{firstID, secondID}}, milestoneObject(t, call("customer_property", read)))
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
	})
}
