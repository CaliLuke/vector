//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerRequestsRejectAnotherCustomersRequest(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		owner := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		other := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		item := f.item(t)
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": owner["id"], "name": "Private request", "workitem_ids": item}))
		before := customerStorageSnapshot(t, f)
		for _, action := range []string{"retrieve", "update", "delete"} {
			t.Run(action, func(t *testing.T) {
				input := map[string]any{"action": action, "customer_id": other["id"], "request_id": request["id"]}
				if action != "retrieve" {
					input["mutation_id"] = uuid.NewString()
				}
				if action == "update" {
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
				input := map[string]any{"action": action, "customer_id": other["id"], "customer_request_id": request["id"]}
				if action == "manage_workitems" {
					input["link_ids"] = item
				}
				denied := call("customer", input)
				milestoneFailure(t, denied, "not_found")
				require.Empty(t, denied.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		}
		page := milestoneObject(t, call("customer_request", map[string]any{"action": "list", "customer_id": other["id"]}))
		require.Empty(t, page["results"])
	})
}

func TestMCPCustomerRequestReceiptCannotBeRedirectedToAnotherCustomer(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		owner := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		other := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		for _, action := range []string{"create", "update", "delete"} {
			t.Run(action, func(t *testing.T) {
				request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": owner["id"], "name": uuid.NewString()}))
				input := map[string]any{"action": action, "customer_id": owner["id"], "mutation_id": uuid.NewString()}
				if action != "delete" {
					input["name"] = "Accepted request"
				}
				if action != "create" {
					input["request_id"] = request["id"]
				}
				accepted := milestoneObject(t, call("customer_request", input))
				before := customerStorageSnapshot(t, f)
				redirected := maps.Clone(input)
				redirected["customer_id"] = other["id"]
				denied := call("customer_request", redirected)
				milestoneFailure(t, denied, "request_conflict")
				require.Empty(t, denied.StructuredContent)
				require.Equal(t, before, customerStorageSnapshot(t, f))
				replay := milestoneObject(t, call("customer_request", input))
				require.Equal(t, true, replay["replayed"])
				require.Equal(t, accepted["result"], replay["result"])
				require.Equal(t, before, customerStorageSnapshot(t, f))
			})
		}
	})
}
