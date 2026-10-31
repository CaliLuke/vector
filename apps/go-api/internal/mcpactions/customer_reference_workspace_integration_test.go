//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func customerForeignReferenceTargets(t *testing.T, f milestoneFixture) map[string]string {
	t.Helper()
	workspace, project := addPrivateProjectInOtherWorkspace(t, f)
	member, _ := customerReferenceTarget(t, f, "USER")
	changed, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET workspace_id=$2 WHERE member_id=$1`, member, workspace)
	require.NoError(t, err)
	require.EqualValues(t, 1, changed.RowsAffected())
	propertyID, optionID := uuid.NewString(), uuid.NewString()
	_, err = f.pool.Exec(t.Context(), `INSERT INTO customer_properties(id,workspace_id,name,display_name,property_type) VALUES($1,$2,'foreign_options','Foreign options','OPTION')`, propertyID, workspace)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO customer_property_options(id,workspace_id,property_id,name) VALUES($1,$2,$3,'Foreign option')`, optionID, workspace, propertyID)
	require.NoError(t, err)
	return map[string]string{"ISSUE": f.itemInScope(t, workspace, project), "USER": member, "OPTION": optionID}
}

func TestMCPCustomerReferencesKeepSelectedWorkspaceScope(t *testing.T) {
	f := customerIdentityFixture(t)
	foreign := customerForeignReferenceTargets(t, f)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, kind := range []string{"ISSUE", "USER", "OPTION"} {
			t.Run(kind, func(t *testing.T) {
				input := customerValueReferenceInput(t, f, call, kind)
				arguments := customerReferenceArguments(t, input)
				propertyID := arguments["property"].(string)
				foreignValues, err := json.Marshal(map[string][]string{propertyID: {foreign[kind]}})
				require.NoError(t, err)
				rejected := maps.Clone(input)
				rejected["values"] = string(foreignValues)
				before := customerStorageSnapshot(t, f)
				denied := call("customer_property", rejected)
				milestoneFailure(t, denied, "invalid_params")
				require.Empty(t, denied.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
				accepted := milestoneObject(t, call("customer_property", input))
				require.Equal(t, false, accepted["replayed"], "a foreign reference cannot consume the local mutation key")
				for _, action := range []string{"create", "update"} {
					if kind == "OPTION" && action == "create" {
						continue // New options have no existing IDs to select on creation.
					}
					defaults := map[string]any{"action": action, "default_value": foreign[kind], "request_id": uuid.NewString()}
					if action == "create" {
						defaults["display_name"], defaults["property_type"], defaults["relation_type"] = uuid.NewString(), "RELATION", kind
					} else {
						defaults["property_id"] = propertyID
					}
					before = customerStorageSnapshot(t, f)
					denied = call("customer_property", defaults)
					milestoneFailure(t, denied, "invalid_params")
					require.Empty(t, denied.StructuredContent)
					require.Equal(t, before, customerStorageSnapshot(t, f))
					defaults["default_value"] = arguments["target"]
					require.Equal(t, false, milestoneObject(t, call("customer_property", defaults))["replayed"])
				}
				// Imported or subsequently moved references still cannot be exposed by reads.
				imported, err := json.Marshal([]string{foreign[kind]})
				require.NoError(t, err)
				changed, err := f.pool.Exec(t.Context(), `UPDATE customer_property_values SET value_strings=$3::jsonb WHERE customer_id=$1 AND property_id=$2`, input["customer_id"], propertyID, imported)
				require.NoError(t, err)
				require.EqualValues(t, 1, changed.RowsAffected())
				if kind != "OPTION" {
					changed, err = f.pool.Exec(t.Context(), `UPDATE customer_properties SET default_value=$2::jsonb WHERE id=$1`, propertyID, imported)
					require.NoError(t, err)
					require.EqualValues(t, 1, changed.RowsAffected())
				}
				before = customerStorageSnapshot(t, f)
				read := map[string]any{"action": "get_values", "customer_id": input["customer_id"], "property_id": propertyID}
				require.Equal(t, map[string]any{propertyID: []any{}}, milestoneObject(t, call("customer_property", read)))
				if kind != "OPTION" {
					definition := milestoneObject(t, call("customer_property", map[string]any{"action": "retrieve", "property_id": propertyID}))
					require.Equal(t, []any{}, definition["default_value"])
				}
				replayed := milestoneObject(t, call("customer_property", input))
				require.Equal(t, true, replayed["replayed"])
				require.Equal(t, accepted["result"], replayed["result"])
				require.Equal(t, before, customerStorageSnapshot(t, f), "read masking and receipt replay cannot rewrite imported data")
				catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
			})
		}
	})
}
