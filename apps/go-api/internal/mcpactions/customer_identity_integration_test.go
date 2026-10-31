//go:build integration

package mcpactions_test

import (
	"maps"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerUpsertsUseCompleteExternalIdentityBeforeName(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		name, external := "ACME "+uuid.NewString(), uuid.NewString()
		first := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "email": "retained@example.test", "employees": 12, "external_source": "crm", "external_id": external}))
		second := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "external_source": "crm", "external_id": uuid.NewString()}))
		require.NotEqual(t, first["id"], second["id"], "different external pairs may share a display name")
		upsert := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": "Renamed " + name, "external_source": "crm", "external_id": external, "stage": "negotiating"}))
		require.Equal(t, first["id"], upsert["id"])
		require.Equal(t, "Renamed "+name, upsert["name"])
		require.Equal(t, "retained@example.test", upsert["email"])
		require.Equal(t, float64(12), upsert["employees"])
		require.Equal(t, first["created_at"], upsert["created_at"])
		require.Equal(t, "negotiating", upsert["stage"])
		require.Equal(t, second, milestoneObject(t, call("customer", map[string]any{"action": "retrieve", "customer_id": second["id"]})))
	})
}

func TestMCPCustomerUpsertsMatchExactNameWithoutCompleteExternalPair(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		for _, partial := range []string{"none", "external_source", "external_id"} {
			t.Run(partial, func(t *testing.T) {
				name := "EXACT " + uuid.NewString()
				first := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "email": "retained@example.test"}))
				input := map[string]any{"action": "create", "name": name, "employees": 9}
				if partial != "none" {
					input[partial] = "partial"
				}
				upsert := milestoneObject(t, call("customer", input))
				require.Equal(t, first["id"], upsert["id"])
				require.Equal(t, "retained@example.test", upsert["email"])
				require.Equal(t, float64(9), upsert["employees"])
				input["name"] = strings.ToLower(name)
				separate := milestoneObject(t, call("customer", input))
				require.NotEqual(t, first["id"], separate["id"], "name matching is exact, not the list's case-insensitive search")
			})
		}
	})
}

func TestMCPCustomerAmbiguousNamesAndIdentityCollisionsAreAtomic(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		name := uuid.NewString()
		first := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "external_source": "crm", "external_id": uuid.NewString()}))
		second := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "external_source": "crm", "external_id": uuid.NewString()}))
		before := customerStorageSnapshot(t, f)
		milestoneFailure(t, call("customer", map[string]any{"action": "create", "name": name, "email": "rejected@example.test", "request_id": uuid.NewString()}), "invalid_params")
		require.Equal(t, before, customerStorageSnapshot(t, f), "ambiguous upsert must not choose a customer or consume a receipt")
		collision := map[string]any{"action": "update", "customer_id": second["id"], "name": "Rejected rename", "external_source": first["external_source"], "external_id": first["external_id"], "request_id": uuid.NewString()}
		milestoneFailure(t, call("customer", collision), "invalid_params")
		require.Equal(t, before, customerStorageSnapshot(t, f), "uniqueness failure must roll back every field, audit, revision and receipt")
		collision["external_id"] = uuid.NewString()
		accepted := milestoneObject(t, call("customer", collision))
		require.Equal(t, false, accepted["replayed"], "a rejected identity change leaves its retry key reusable")
		require.Equal(t, second["id"], accepted["resource_id"])
	})
}

func TestMCPCustomerIdentityReuseAfterDeletionPreservesOldReceipt(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		input := map[string]any{"action": "create", "name": uuid.NewString(), "external_source": "crm", "external_id": uuid.NewString(), "request_id": uuid.NewString()}
		first := milestoneObject(t, call("customer", input))
		remove := map[string]any{"action": "delete", "external_source": input["external_source"], "external_id": input["external_id"], "request_id": uuid.NewString()}
		deleted := milestoneObject(t, call("customer", remove))
		require.Equal(t, first["resource_id"], deleted["resource_id"])
		fresh := maps.Clone(input)
		fresh["request_id"] = uuid.NewString()
		replacement := milestoneObject(t, call("customer", fresh))
		require.NotEqual(t, first["resource_id"], replacement["resource_id"])
		replayed := milestoneObject(t, call("customer", input))
		require.Equal(t, true, replayed["replayed"])
		require.Equal(t, first["result"], replayed["result"])
		require.Equal(t, true, milestoneObject(t, call("customer", remove))["replayed"], "old deletion replay must not delete the replacement identity")
		live := milestoneObject(t, call("customer", map[string]any{"action": "retrieve", "customer_id": replacement["resource_id"]}))
		require.Equal(t, replacement["result"], live)
		milestoneFailure(t, call("customer", map[string]any{"action": "retrieve", "customer_id": first["resource_id"]}), "not_found")
	})
}

func TestMCPCustomerIdentityMatchingIsWorkspaceBound(t *testing.T) {
	f := customerIdentityFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		name, external, foreignID := uuid.NewString(), uuid.NewString(), uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO customers(id,workspace_id,name,external_source,external_id,email) VALUES($1,$2,$3,'crm',$4,'foreign@example.test')`, foreignID, otherWorkspace, name, external)
		require.NoError(t, err)
		local := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name, "external_source": "crm", "external_id": external, "email": "local@example.test"}))
		require.NotEqual(t, foreignID, local["id"])
		require.Equal(t, testsupport.LabelWorkspace, local["workspace"])
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "external_source": "crm", "external_id": external}))
		var email string
		var active bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT email,deleted_at IS NULL FROM customers WHERE id=$1`, foreignID).Scan(&email, &active))
		require.Equal(t, "foreign@example.test", email)
		require.True(t, active)
		milestoneFailure(t, call("customer", map[string]any{"action": "retrieve", "customer_id": foreignID}), "not_found")
	})
}
