//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type catalogueCall func(string, map[string]any) rpcResult

// Share one migrated database across the protocol/credential matrix. Each case
// owns its records and exercises the actual generated transport and repositories.
func TestMCPWorkspaceCatalogueBatch(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		t.Run("feature receipts and current authorization", func(t *testing.T) { catalogueFeatures(t, f, call) })
		t.Run("customer identities and request binding", func(t *testing.T) { catalogueCustomers(t, f, call) })
		t.Run("property cardinality and atomic values", func(t *testing.T) { catalogueCustomerProperties(t, call) })
		t.Run("template merge and summary", func(t *testing.T) { catalogueTemplates(t, call) })
		t.Run("initiative associations", func(t *testing.T) { catalogueInitiatives(t, f, call) })
	})
}

func runCatalogueMatrix(t *testing.T, exercise func(*testing.T, milestoneFixture, catalogueCall)) {
	t.Helper()
	runCatalogueFixtureMatrix(t, newMilestoneFixture(t), exercise)
}

func runCatalogueFixtureMatrix(t *testing.T, f milestoneFixture, exercise func(*testing.T, milestoneFixture, catalogueCall)) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			passed := t.Run(version+"/"+mode, func(t *testing.T) {
				pat, oauth := f.handler(t, nil), oauthHandler(t, f.fixture, f.oauth.Authenticate)
				call := catalogueCall(func(tool string, input map[string]any) rpcResult {
					body, marshalErr := json.Marshal(input)
					require.NoError(t, marshalErr)
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", tool, string(body), http.StatusOK)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", resourceCall(tool, string(body)), version)
				})
				exercise(t, f, call)
			})
			if !passed {
				return
			}
		}
	}
}

func catalogueNull(t *testing.T, result rpcResult) {
	t.Helper()
	require.False(t, result.IsError, result.Content)
	require.Empty(t, result.StructuredContent)
	require.Empty(t, result.Content)
}

func catalogueArray(t *testing.T, result rpcResult) []map[string]any {
	t.Helper()
	require.False(t, result.IsError, result.Content)
	require.Empty(t, result.StructuredContent)
	if len(result.Content) == 0 {
		return []map[string]any{}
	}
	require.Len(t, result.Content, 1)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &rows))
	return rows
}

func catalogueFeatures(t *testing.T, f milestoneFixture, call catalogueCall) {
	before := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
	input := map[string]any{"action": "update_features", "customers": true, "initiatives": true, "request_id": uuid.NewString(), "expected_revision": before["revision"]}
	created := milestoneObject(t, call("workspace", input))
	require.Equal(t, true, created["result"].(map[string]any)["customers"])
	require.Equal(t, false, created["replayed"])
	replay := milestoneObject(t, call("workspace", input))
	require.Equal(t, true, replay["replayed"])
	require.Equal(t, created["result"], replay["result"])
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=15`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE workspace_members SET role=20`)
		require.NoError(t, restoreErr)
	})
	milestoneFailure(t, call("workspace", input), "forbidden")
	project := milestoneObject(t, call("project", map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": true}))
	require.Equal(t, true, project["cycles"])
	read := milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject}))
	require.Equal(t, project, read)
}

func catalogueCustomers(t *testing.T, f milestoneFixture, call catalogueCall) {
	name, external := uuid.NewString(), uuid.NewString()
	input := map[string]any{"action": "create", "name": name, "email": "buyer@example.test", "external_source": "test", "external_id": external, "request_id": uuid.NewString()}
	receipt := milestoneObject(t, call("customer", input))
	id := receipt["resource_id"].(string)
	replay := milestoneObject(t, call("customer", input))
	require.Equal(t, true, replay["replayed"])
	require.Equal(t, receipt["result"], replay["result"])
	upsert := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": name + " renamed", "external_source": "test", "external_id": external}))
	require.Equal(t, id, upsert["id"])
	require.Equal(t, "buyer@example.test", upsert["email"])
	updated := milestoneObject(t, call("customer", map[string]any{"action": "update", "customer_id": id, "employees": 23}))
	require.Equal(t, float64(23), updated["employees"])
	detail := milestoneObject(t, call("customer", map[string]any{"action": "retrieve", "customer_id": id}))
	require.Equal(t, updated, detail)
	page := milestoneObject(t, call("customer", map[string]any{"action": "list", "query": name}))
	require.Equal(t, []any{detail}, page["results"])
	item := f.item(t)
	requestInput := map[string]any{"action": "create", "customer_id": id, "name": "Export", "workitem_ids": item, "mutation_id": uuid.NewString()}
	request := milestoneObject(t, call("customer_request", requestInput))
	requestID := request["resource_id"].(string)
	require.Equal(t, requestInput["mutation_id"], request["mutation_id"])
	require.NotContains(t, request, "request_id")
	require.Equal(t, true, milestoneObject(t, call("customer_request", requestInput))["replayed"])
	requestDetail := milestoneObject(t, call("customer_request", map[string]any{"action": "retrieve", "customer_id": id, "request_id": requestID}))
	require.Equal(t, request["result"], requestDetail)
	requestPage := milestoneObject(t, call("customer_request", map[string]any{"action": "list", "customer_id": id}))
	require.Equal(t, []any{requestDetail}, requestPage["results"])
	changed := milestoneObject(t, call("customer_request", map[string]any{"action": "update", "customer_id": id, "request_id": requestID, "name": "Export CSV"}))
	require.Equal(t, "Export CSV", changed["name"])
	other := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
	milestoneFailure(t, call("customer_request", map[string]any{"action": "retrieve", "customer_id": other["id"], "request_id": requestID}), "not_found")
	linked := catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": id, "customer_request_id": requestID}))
	require.Len(t, linked, 1)
	require.Equal(t, item, linked[0]["id"])
	catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": id, "customer_request_id": requestID, "link_ids": item, "unlink_ids": item}))
	require.Empty(t, catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": id})))
	catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": id, "request_id": requestID}))
	catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": other["id"]}))
	catalogueNull(t, call("customer", map[string]any{"action": "delete", "external_source": "test", "external_id": external}))
	catalogueNull(t, call("customer", map[string]any{"action": "delete", "external_source": "test", "external_id": external}))
	var activityCount int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE payload->>'issue_id'=$1 AND kind='activity'`, item).Scan(&activityCount))
	require.Equal(t, 2, activityCount)
}

func catalogueCustomerProperties(t *testing.T, call catalogueCall) {
	name := "Property " + uuid.NewString()
	property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": name, "property_type": "TEXT", "default_value": "retail, enterprise"}))
	propertyID := property["id"].(string)
	page := milestoneObject(t, call("customer_property", map[string]any{"action": "list"}))
	require.Equal(t, []any{property}, page["results"])
	detail := milestoneObject(t, call("customer_property", map[string]any{"action": "retrieve", "property_id": propertyID}))
	require.Equal(t, property, detail)
	customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
	values := milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"]}))
	require.Equal(t, []any{"retail, enterprise"}, values[propertyID])
	updated := milestoneObject(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_multi": true}))
	require.Equal(t, true, updated["is_multi"])
	encoded, err := json.Marshal(map[string][]string{propertyID: {"A", "B"}})
	require.NoError(t, err)
	catalogueNull(t, call("customer_property", map[string]any{"action": "set_values", "customer_id": customer["id"], "values": string(encoded)}))
	milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "is_multi": false}), "invalid_params")
	unchanged := milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"]}))
	require.Equal(t, []any{"A", "B"}, unchanged[propertyID])
	invalid, err := json.Marshal(map[string][]string{propertyID: {"replacement"}, uuid.NewString(): {"missing"}})
	require.NoError(t, err)
	milestoneFailure(t, call("customer_property", map[string]any{"action": "set_values", "customer_id": customer["id"], "values": string(invalid)}), "not_found")
	require.Equal(t, unchanged, milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"]})))
	catalogueNull(t, call("customer_property", map[string]any{"action": "delete", "property_id": propertyID}))
	require.Empty(t, milestoneObject(t, call("customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"]})))
	catalogueNull(t, call("customer", map[string]any{"action": "delete", "customer_id": customer["id"]}))
}

func catalogueTemplates(t *testing.T, call catalogueCall) {
	created := milestoneObject(t, call("template", map[string]any{"action": "create", "kind": "project", "name": "Launch", "description": "Project summary", "template_data": `{"priority":"high","settings":{"a":true},"serial":9007199254740993,"created_at":"2026-10-01T12:00:00+05:30"}`}))
	require.Equal(t, "Project summary", created["description"])
	updated := milestoneObject(t, call("template", map[string]any{"action": "update", "kind": "project", "template_id": created["id"], "template_data": `{"settings":{"b":true},"optional":null}`}))
	require.Equal(t, map[string]any{"priority": "high", "settings": map[string]any{"b": true}, "optional": nil, "serial": float64(9007199254740993), "created_at": "2026-10-01T12:00:00+05:30"}, updated["template_data"])
	listResult := call("template", map[string]any{"action": "list", "kind": "project"})
	listed := catalogueArray(t, listResult)
	require.Contains(t, listResult.Content[0].Text, "9007199254740993")
	require.Equal(t, []map[string]any{updated}, listed)
	catalogueNull(t, call("template", map[string]any{"action": "delete", "kind": "project", "template_id": created["id"]}))
}

func catalogueInitiatives(t *testing.T, f milestoneFixture, call catalogueCall) {
	created := milestoneObject(t, call("initiative", map[string]any{"action": "create", "name": "Delivery"}))
	id := created["id"].(string)
	updated := milestoneObject(t, call("initiative", map[string]any{"action": "update", "initiative_id": id, "state": "ACTIVE"}))
	require.Equal(t, "ACTIVE", updated["state"])
	detail := milestoneObject(t, call("initiative", map[string]any{"action": "retrieve", "initiative_id": id}))
	require.Equal(t, updated, detail)
	require.Equal(t, []map[string]any{detail}, catalogueArray(t, call("initiative", map[string]any{"action": "list"})))
	catalogueNull(t, call("initiative", map[string]any{"action": "add_projects", "initiative_id": id, "project_ids": testsupport.LabelProject}))
	projects := milestoneObject(t, call("initiative", map[string]any{"action": "list_projects", "initiative_id": id}))
	require.Len(t, projects["results"], 1)
	catalogueNull(t, call("initiative", map[string]any{"action": "remove_projects", "initiative_id": id, "project_ids": testsupport.LabelProject}))
	item := f.item(t)
	catalogueNull(t, call("initiative", map[string]any{"action": "manage_workitems", "initiative_id": id, "add_ids": item, "remove_ids": item}))
	items := milestoneObject(t, call("initiative", map[string]any{"action": "list_workitems", "initiative_id": id}))
	require.Len(t, items["results"], 1)
	catalogueNull(t, call("initiative", map[string]any{"action": "delete", "initiative_id": id}))
	var retained bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&retained))
	require.True(t, retained)
}

func assertCatalogueTimezone(t *testing.T, record map[string]any) {
	t.Helper()
	for _, field := range []string{"created_at", "updated_at"} {
		stamp, err := time.Parse(time.RFC3339Nano, record[field].(string))
		require.NoError(t, err)
		_, offset := stamp.Zone()
		require.Equal(t, 19800, offset, field)
	}
}
