//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerUpdatesPreserveOmittedFields(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		name := uuid.NewString()
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "email": "retained@example.test", "employees": 5}))
		customerID := customer["id"].(string)
		updated := milestoneObject(t, call("customer", map[string]any{"action": "update", "customer_id": customerID, "name": "", "email": "", "employees": 0}))
		require.Equal(t, customerID, updated["id"])
		require.Equal(t, name, updated["name"])
		require.Equal(t, "retained@example.test", updated["email"])
		require.Equal(t, float64(5), updated["employees"])
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": customerID, "name": "Retained request", "link": "https://example.test/request"}))
		requestID := request["id"].(string)
		updatedRequest := milestoneObject(t, call("customer_request", map[string]any{"action": "update", "customer_id": customerID, "request_id": requestID, "name": "", "link": ""}))
		require.Equal(t, requestID, updatedRequest["id"])
		require.Equal(t, "Retained request", updatedRequest["name"])
		require.Equal(t, "https://example.test/request", updatedRequest["link"])
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Sentinel " + uuid.NewString(), "property_type": "TEXT", "default_value": "retained"}))
		updatedProperty := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": property["id"], "default_value": "   ", "description": ""}))
		require.Equal(t, property["id"], updatedProperty["id"])
		require.Equal(t, []any{"retained"}, updatedProperty["default_value"])
		catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": property["id"]}))
		catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": customerID, "request_id": requestID}))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
	})
}
