//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerValueReceiptsRespectCurrentPropertyDefinition(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, remove := range []bool{false, true} {
			name, code := "inactive definition", "invalid_params"
			if remove {
				name, code = "deleted definition", "not_found"
			}
			t.Run(name, func(t *testing.T) {
				input := customerPropertyWriteInput(t, call, "set_values")
				arguments := customerReferenceArguments(t, input)
				propertyID := arguments["property"].(string)
				accepted := milestoneObject(t, call("customer_property", input))
				require.Equal(t, false, accepted["replayed"])
				if remove {
					catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
				} else {
					milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_active": false}))
				}
				before := customerStorageSnapshot(t, f)
				fresh := maps.Clone(input)
				fresh["request_id"] = uuid.NewString()
				for _, attempt := range []map[string]any{fresh, input} {
					denied := call("customer_property", attempt)
					milestoneFailure(t, denied, code)
					require.Empty(t, denied.StructuredContent)
				}
				read := map[string]any{"action": "get_values", "customer_id": input["customer_id"], "property_id": propertyID}
				if remove {
					milestoneFailure(t, call("customer_property", read), "not_found")
				} else {
					require.Equal(t, map[string]any{propertyID: []any{}}, milestoneObject(t, call("customer_property", read)))
				}
				require.Equal(t, before, customerStorageSnapshot(t, f), "denied value retries preserve the definition mutation and consume no new receipts")
				if !remove {
					milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_active": true}))
					restored := customerStorageSnapshot(t, f)
					replayed := milestoneObject(t, call("customer_property", input))
					require.Equal(t, true, replayed["replayed"])
					require.Equal(t, accepted["result"], replayed["result"])
					require.Equal(t, map[string]any{propertyID: []any{arguments["target"]}}, milestoneObject(t, call("customer_property", read)))
					require.Equal(t, restored, customerStorageSnapshot(t, f))
					catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
				}
				catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": input["customer_id"]}))
			})
		}
	})
}
