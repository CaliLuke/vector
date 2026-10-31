//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func customerAssociationSnapshot(t *testing.T, f milestoneFixture) [][]byte {
	t.Helper()
	var items []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'items',COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM issues i),'[]'),
 'revisions',COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY issue_id) FROM issue_edit_revisions r),'[]'))`).Scan(&items))
	return [][]byte{customerStorageSnapshot(t, f), items}
}

func TestMCPCustomerAssociationFailureRollsBackLinksActivityAndRevisions(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		item, unlinked := f.item(t), f.item(t)
		input := map[string]any{"action": "manage_workitems", "customer_id": customer["id"], "link_ids": item, "unlink_ids": unlinked, "request_id": uuid.NewString()}
		var initialRevision int64
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, item).Scan(&initialRevision))
		before := customerAssociationSnapshot(t, f)
		denied := call("customer", input)
		milestoneFailure(t, denied, "not_found")
		require.Empty(t, denied.StructuredContent)
		require.Equal(t, before, customerAssociationSnapshot(t, f), "failure after the add must roll back links, activity, item/catalogue revisions and receipts")
		delete(input, "unlink_ids")
		accepted := milestoneObject(t, call("customer", input))
		require.Equal(t, false, accepted["replayed"], "rollback leaves the retry identity available")
		var revision int64
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, item).Scan(&revision))
		require.Greater(t, revision, initialRevision, "accepted association advances the native item revision")
		before = customerAssociationSnapshot(t, f)
		replay := milestoneObject(t, call("customer", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, accepted["result"], replay["result"])
		require.Equal(t, before, customerAssociationSnapshot(t, f))
		duplicate := maps.Clone(input)
		duplicate["request_id"] = uuid.NewString()
		require.Equal(t, false, milestoneObject(t, call("customer", duplicate))["replayed"])
		var current int64
		var effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, item).Scan(&current))
		require.Equal(t, revision, current, "an existing association does not advance the item revision")
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->>'issue_id'=$1`, item).Scan(&effects))
		require.Equal(t, 1, effects, "duplicate adds and receipt replay do not enqueue another activity")
	})
}

func TestMCPCustomerAssociationFiltersPreserveOtherRequestsAndCustomers(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		owner := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		other := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		item := f.item(t)
		direct := map[string]any{"action": "manage_workitems", "customer_id": owner["id"], "link_ids": item}
		catalogueNull(t, call("customer", direct))
		requests := []string{}
		for _, name := range []string{"First", "Second"} {
			request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": owner["id"], "name": name, "workitem_ids": item}))
			requests = append(requests, request["id"].(string))
		}
		catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": other["id"], "link_ids": item}))
		list := map[string]any{"action": "list_workitems", "customer_id": owner["id"]}
		require.Len(t, catalogueArray(t, call("customer", list)), 1, "direct and request-specific associations project one item")
		catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": owner["id"], "customer_request_id": requests[0], "unlink_ids": item}))
		require.Empty(t, catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": owner["id"], "customer_request_id": requests[0]})))
		require.Len(t, catalogueArray(t, call("customer", list)), 1, "filtered removal preserves direct and other-request links")
		catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": owner["id"], "request_id": requests[1]}))
		require.Len(t, catalogueArray(t, call("customer", list)), 1, "request deletion preserves the direct link")
		catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": owner["id"], "customer_request_id": requests[0], "link_ids": item}))
		catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": owner["id"], "link_ids": item, "unlink_ids": item}))
		require.Empty(t, catalogueArray(t, call("customer", list)), "unfiltered removal wins and removes every association for this customer")
		require.Len(t, catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": other["id"]})), 1)
		catalogueNull(t, call("customer", direct))
		catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": owner["id"]}))
		milestoneFailure(t, call("customer_request", map[string]any{"action": "retrieve", "customer_id": owner["id"], "request_id": requests[0]}), "not_found")
		require.Len(t, catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": other["id"]})), 1, "customer deletion keeps the other customer's link and its live item")
	})
}
