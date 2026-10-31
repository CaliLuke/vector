//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerRequestResourcesStayInSelectedWorkspace(t *testing.T) {
	f := customerIdentityFixture(t)
	foreignWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	foreignCustomer, foreignRequest := uuid.NewString(), uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO customers(id,workspace_id,name) VALUES($1,$2,'Foreign buyer')`, foreignCustomer, foreignWorkspace)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO customer_requests(id,workspace_id,customer_id,name) VALUES($1,$2,$3,'Foreign request')`, foreignRequest, foreignWorkspace, foreignCustomer)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		local := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		item := f.item(t)
		before := customerStorageSnapshot(t, f)
		for _, attempt := range []struct {
			scope, action, customer, request string
		}{
			{"foreign parent", "list", foreignCustomer, ""},
			{"foreign parent", "create", foreignCustomer, ""},
			{"foreign parent", "retrieve", foreignCustomer, foreignRequest},
			{"foreign parent", "update", foreignCustomer, foreignRequest},
			{"foreign parent", "delete", foreignCustomer, foreignRequest},
			{"local parent", "retrieve", local["id"].(string), foreignRequest},
			{"local parent", "update", local["id"].(string), foreignRequest},
			{"local parent", "delete", local["id"].(string), foreignRequest},
		} {
			t.Run(attempt.scope+"/"+attempt.action, func(t *testing.T) {
				input := map[string]any{"action": attempt.action, "customer_id": attempt.customer}
				if attempt.request != "" {
					input["request_id"] = attempt.request
				}
				if attempt.action == "create" || attempt.action == "update" {
					input["name"] = "Rejected change"
				}
				denied := call("customer_request", input)
				milestoneFailure(t, denied, "not_found")
				require.Empty(t, denied.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		}
		for _, action := range []string{"list_workitems", "manage_workitems"} {
			t.Run(action, func(t *testing.T) {
				input := map[string]any{"action": action, "customer_id": local["id"], "customer_request_id": foreignRequest}
				if action == "manage_workitems" {
					input["link_ids"] = item
				}
				denied := call("customer", input)
				milestoneFailure(t, denied, "not_found")
				require.Empty(t, denied.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		}
	})
}
