//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Formats follow plane-sdk 0.3.1's TextDisplayFormat and DateDisplayFormat.
func TestMCPCustomerPropertyDisplayFormatsPreserveValueRules(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, format := range []struct{ kind, input, expected, alternative, initial, replacement string }{
			{"TEXT", "", "single-line", "multi-line", "Initial", "Replacement"},
			{"TEXT", "single-line", "single-line", "multi-line", "Initial", "Replacement"},
			{"TEXT", "multi-line", "multi-line", "single-line", "First\nSecond", "Third\nFourth"},
			{"TEXT", "readonly", "readonly", "single-line", "Initial", "Replacement"},
			{"DATETIME", "", "MMM dd, yyyy", "dd/MM/yyyy", "2026-10-31", "2026-11-01"},
			{"DATETIME", "MMM dd, yyyy", "MMM dd, yyyy", "dd/MM/yyyy", "2026-10-31", "2026-11-01"},
			{"DATETIME", "dd/MM/yyyy", "dd/MM/yyyy", "MM/dd/yyyy", "2026-10-31", "2026-11-01"},
			{"DATETIME", "MM/dd/yyyy", "MM/dd/yyyy", "dd/MM/yyyy", "2026-10-31", "2026-11-01"},
			{"DATETIME", "yyyy/MM/dd", "yyyy/MM/dd", "dd/MM/yyyy", "2026-10-31", "2026-11-01"},
		} {
			t.Run(format.kind+"/"+format.input, func(t *testing.T) {
				property := milestoneObject(t, call("customer_property", map[string]any{
					"action": "create", "display_name": uuid.NewString(), "property_type": format.kind,
					"display_format": format.input, "default_value": format.initial,
				}))
				propertyID := property["id"].(string)
				require.Equal(t, map[string]any{"display_format": format.expected}, property["settings"])
				customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
				customerID := customer["id"].(string)
				read := map[string]any{"action": "get_values", "customer_id": customerID, "property_id": propertyID}
				require.Equal(t, map[string]any{propertyID: []any{format.initial}}, milestoneObject(t, call("customer_property", read)))
				before := customerStorageSnapshot(t, f)
				changed := setCustomerValues(t, call, customerID, map[string][]string{propertyID: {format.replacement}})
				if format.expected == "readonly" {
					milestoneFailure(t, changed, "invalid_params")
					require.Equal(t, before, customerStorageSnapshot(t, f))
				} else {
					catalogueNull(t, changed)
					require.Equal(t, map[string]any{propertyID: []any{format.replacement}}, milestoneObject(t, call("customer_property", read)))
				}
				before = customerStorageSnapshot(t, f)
				for field, value := range map[string]string{"property_type": "BOOLEAN", "display_format": format.alternative} {
					milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, field: value}), "invalid_params")
					require.Equal(t, before, customerStorageSnapshot(t, f), "type and display format are fixed after creation")
				}
				updated := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "description": "Changed metadata"}))
				require.Equal(t, property["settings"], updated["settings"])
				require.Equal(t, format.kind, updated["property_type"])
				catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
				catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customerID}))
			})
		}
	})
}

func TestMCPCustomerPropertyInvalidFormatsLeaveNoState(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, format := range []struct{ kind, display, relation string }{
			{"TEXT", "unknown", ""}, {"TEXT", "MMM dd, yyyy", ""},
			{"DATETIME", "unknown", ""}, {"DATETIME", "single-line", ""},
			{"DECIMAL", "single-line", ""}, {"BOOLEAN", "single-line", ""},
			{"OPTION", "single-line", ""}, {"RELATION", "single-line", "ISSUE"},
			{"URL", "single-line", ""}, {"EMAIL", "single-line", ""}, {"FILE", "single-line", ""},
		} {
			t.Run(format.kind+"/"+format.display, func(t *testing.T) {
				before := customerStorageSnapshot(t, f)
				result := call("customer_property", map[string]any{
					"action": "create", "display_name": uuid.NewString(), "property_type": format.kind,
					"display_format": format.display, "relation_type": format.relation,
				})
				milestoneFailure(t, result, "invalid_params")
				require.Empty(t, result.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		}
	})
}
