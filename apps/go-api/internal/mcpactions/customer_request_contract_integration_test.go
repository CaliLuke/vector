//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Field sets follow plane-sdk 0.3.1 CustomerRequest and PaginatedResponse.
// The source archive checksum is recorded in docs/mcp-parity.md.
func TestMCPCustomerRequestSDKRecordProjection(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		input := map[string]any{"action": "create", "customer_id": parent["id"], "name": "Export request", "description_html": "<p>Requested export</p>", "link": "https://example.test/requests/export", "workitem_ids": f.item(t)}
		created := milestoneObject(t, call("customer_request", input))
		assertCustomerRequestSDKRecord(t, created, input)
		read := map[string]any{"action": "retrieve", "customer_id": parent["id"], "request_id": created["id"]}
		require.Equal(t, created, milestoneObject(t, call("customer_request", read)))
		update := map[string]any{"action": "update", "customer_id": parent["id"], "request_id": created["id"], "name": "Revised request", "description_html": "<p>Revised export</p>", "link": "https://example.test/requests/revised"}
		updated := milestoneObject(t, call("customer_request", update))
		assertCustomerRequestSDKRecord(t, updated, update)
		require.Equal(t, created["id"], updated["id"])
		require.Equal(t, created["created_at"], updated["created_at"])
		require.Equal(t, updated, milestoneObject(t, call("customer_request", read)))
		linked := catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": parent["id"], "customer_request_id": created["id"]}))
		require.Len(t, linked, 1, "metadata updates preserve the creation-time association")
		require.Equal(t, input["workitem_ids"], linked[0]["id"])
	})
}

func assertCustomerRequestSDKRecord(t *testing.T, record, input map[string]any) {
	t.Helper()
	require.Equal(t, []string{"attachment_count", "created_at", "description", "description_html", "id", "link", "name", "work_item_ids"}, slices.Sorted(maps.Keys(record)))
	_, err := uuid.Parse(record["id"].(string))
	require.NoError(t, err)
	_, err = time.Parse(time.RFC3339Nano, record["created_at"].(string))
	require.NoError(t, err)
	require.Equal(t, input["name"], record["name"])
	require.Equal(t, input["description_html"], record["description_html"])
	require.Equal(t, input["link"], record["link"])
	require.Equal(t, map[string]any{}, record["description"], "HTML edits preserve the separate native JSON description")
	require.Nil(t, record["work_item_ids"], "the SDK's creation link input is write-only")
	require.Equal(t, float64(0), record["attachment_count"])
}

func TestMCPCustomerRequestSDKPagesKeepSearchAndParentScope(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		other := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		matching := []any{}
		for _, name := range []string{"Export one", "Export two"} {
			created := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": parent["id"], "name": name}))
			matching = append(matching, created)
		}
		milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": parent["id"], "name": "Unrelated request"}))
		milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": other["id"], "name": "Export other customer"}))
		removed := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": parent["id"], "name": "Export deleted"}))
		catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": parent["id"], "request_id": removed["id"]}))
		input := map[string]any{"action": "list", "customer_id": parent["id"], "query": "Export", "per_page": 1}
		first := milestoneObject(t, call("customer_request", input))
		assertCustomerSDKPage(t, first, false, true)
		input["cursor"] = first["next_cursor"]
		second := milestoneObject(t, call("customer_request", input))
		assertCustomerSDKPage(t, second, true, false)
		items := append(slices.Clone(first["results"].([]any)), second["results"].([]any)...)
		require.ElementsMatch(t, matching, items)
		input["cursor"] = second["prev_cursor"]
		require.Equal(t, first, milestoneObject(t, call("customer_request", input)))
		delete(input, "cursor")
		delete(input, "per_page")
		complete := milestoneObject(t, call("customer_request", input))
		require.ElementsMatch(t, matching, complete["results"])
		require.Equal(t, float64(2), complete["count"])
		require.Equal(t, float64(1), complete["total_pages"])
		require.Equal(t, false, complete["next_page_results"])
		input["query"] = "No matching request"
		empty := milestoneObject(t, call("customer_request", input))
		require.Equal(t, slices.Sorted(maps.Keys(first)), slices.Sorted(maps.Keys(empty)))
		assertCustomerSDKEmptyPage(t, empty)
	})
}

func assertCustomerSDKPage(t *testing.T, page map[string]any, previous, next bool) {
	t.Helper()
	require.Equal(t, []string{"count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results"}, slices.Sorted(maps.Keys(page)))
	require.Len(t, page["results"], 1)
	require.Equal(t, float64(1), page["count"])
	require.Equal(t, float64(2), page["total_count"])
	require.Equal(t, float64(2), page["total_results"])
	require.Equal(t, float64(2), page["total_pages"])
	require.Equal(t, previous, page["prev_page_results"])
	require.Equal(t, next, page["next_page_results"])
	require.IsType(t, "", page["next_cursor"])
	require.IsType(t, "", page["prev_cursor"])
	require.Nil(t, page["grouped_by"])
	require.Nil(t, page["sub_grouped_by"])
	require.Nil(t, page["extra_stats"])
}

func assertCustomerSDKEmptyPage(t *testing.T, empty map[string]any) {
	t.Helper()
	require.Len(t, empty, 12)
	require.Equal(t, []any{}, empty["results"])
	for _, field := range []string{"count", "total_count", "total_pages", "total_results"} {
		require.Equal(t, float64(0), empty[field], field)
	}
	for _, field := range []string{"extra_stats", "grouped_by", "sub_grouped_by"} {
		require.Nil(t, empty[field], field)
	}
	require.Equal(t, false, empty["next_page_results"])
	require.Equal(t, false, empty["prev_page_results"])
	require.IsType(t, "", empty["next_cursor"])
	require.IsType(t, "", empty["prev_cursor"])
}
