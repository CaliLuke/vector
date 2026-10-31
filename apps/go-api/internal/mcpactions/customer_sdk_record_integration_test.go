//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerSDKRecordPreservesFieldsAndLiveRequestCount(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		input := map[string]any{"action": "create", "name": uuid.NewString(), "description_html": "<p>Original buyer</p>", "email": "buyer@example.test", "website_url": "https://example.test", "domain": "Retail", "employees": 23, "stage": "Negotiation", "contract_status": "Pending", "revenue": "USD 123.45", "external_source": "crm", "external_id": uuid.NewString()}
		created := milestoneObject(t, call("customer", input))
		assertCustomerSDKMetadata(t, "Customer", created)
		for _, field := range []string{"name", "description_html", "email", "website_url", "domain", "stage", "contract_status", "revenue", "external_source", "external_id"} {
			require.Equal(t, input[field], created[field], field)
		}
		require.Equal(t, float64(23), created["employees"])
		require.Equal(t, "Original buyer", created["description_stripped"])
		require.Equal(t, map[string]any{}, created["description"])
		for _, field := range []string{"description_binary", "logo_url", "logo_asset", "archived_at"} {
			require.Nil(t, created[field], field)
		}
		require.Equal(t, float64(0), created["customer_request_count"])
		read := map[string]any{"action": "retrieve", "customer_id": created["id"]}
		require.Equal(t, created, milestoneObject(t, call("customer", read)))
		updated := milestoneObject(t, call("customer", map[string]any{"action": "update", "customer_id": created["id"], "name": "Revised buyer", "description_html": "<p>Revised buyer</p>"}))
		expected := maps.Clone(created)
		expected["name"], expected["description_html"], expected["description_stripped"] = "Revised buyer", "<p>Revised buyer</p>", "Revised buyer"
		expected["updated_at"] = updated["updated_at"]
		require.Equal(t, expected, updated, "partial updates preserve all omitted SDK fields and creation metadata")
		assertCustomerSDKMetadata(t, "Customer", updated)
		require.Equal(t, updated, milestoneObject(t, call("customer", read)))
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": created["id"], "name": "Request"}))
		require.Equal(t, float64(1), milestoneObject(t, call("customer", read))["customer_request_count"])
		catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": created["id"], "request_id": request["id"]}))
		require.Equal(t, float64(0), milestoneObject(t, call("customer", read))["customer_request_count"], "soft-deleted requests do not inflate the SDK count")
	})
}

func TestMCPCustomerPropertySDKRecordIncludesOptionMetadata(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		input := map[string]any{"action": "create", "display_name": "Market " + uuid.NewString(), "description": "Market segment", "property_type": "OPTION", "is_multi": true, "options": `[{"name":"Retail","description":"Retail buyer","is_default":true},{"name":"Enterprise"}]`}
		created := milestoneObject(t, call("customer_property", input))
		assertCustomerSDKMetadata(t, "CustomerProperty", created)
		for _, field := range []string{"display_name", "description", "property_type", "is_multi"} {
			require.Equal(t, input[field], created[field], field)
		}
		require.NotEmpty(t, created["name"])
		require.IsType(t, float64(0), created["sort_order"])
		require.Equal(t, false, created["is_required"])
		require.Equal(t, true, created["is_active"])
		require.Equal(t, []any{}, created["default_value"])
		require.Equal(t, map[string]any{}, created["settings"])
		for _, field := range []string{"relation_type", "validation_rules", "external_source", "external_id"} {
			require.Nil(t, created[field], field)
		}
		options := created["options"].([]any)
		require.Len(t, options, 2)
		for index, raw := range options {
			option := raw.(map[string]any)
			assertCustomerSDKMetadata(t, "CustomerPropertyOption", option)
			require.Equal(t, created["id"], option["property"])
			require.Equal(t, true, option["is_active"])
			require.Equal(t, index == 0, option["is_default"])
			require.Equal(t, []string{"Retail", "Enterprise"}[index], option["name"])
			require.IsType(t, float64(0), option["sort_order"])
			for _, field := range []string{"external_source", "external_id", "parent"} {
				require.Nil(t, option[field], field)
			}
		}
		first := options[0].(map[string]any)
		require.Equal(t, "Retail buyer", first["description"])
		require.Nil(t, options[1].(map[string]any)["description"])
		read := map[string]any{"action": "retrieve", "property_id": created["id"]}
		require.Equal(t, created, milestoneObject(t, call("customer_property", read)))
		patch, err := json.Marshal([]map[string]any{{"id": first["id"], "name": "Retail updated", "description": "Updated segment", "is_default": false}})
		require.NoError(t, err)
		updated := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": created["id"], "options": string(patch)}))
		assertCustomerSDKMetadata(t, "CustomerProperty", updated)
		patched := updated["options"].([]any)[0].(map[string]any)
		assertCustomerSDKMetadata(t, "CustomerPropertyOption", patched)
		expected := maps.Clone(first)
		expected["name"], expected["description"], expected["is_default"], expected["updated_at"] = "Retail updated", "Updated segment", false, patched["updated_at"]
		require.Equal(t, expected, patched)
		expectedProperty := maps.Clone(created)
		expectedProperty["options"], expectedProperty["updated_at"] = []any{patched, options[1]}, updated["updated_at"]
		require.Equal(t, expectedProperty, updated, "option patches preserve definition and omitted option projections")
		require.Equal(t, updated, milestoneObject(t, call("customer_property", read)))
	})
}
