//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The pinned SDK 0.3.1 declares these response fields. The MCP handler treats
// empty optional strings as omitted, preserves a zero point key and false
// last_used, and creates a catalogue separately from selecting it on a project.
func TestMCPEstimateSDKFieldsAndSelection(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		created := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "Sizing", "last_used": false, "description": "Description", "external_source": "fixture", "external_id": uuid.NewString()}))
		require.ElementsMatch(t, strings.Fields("id name description type last_used external_id external_source created_at updated_at created_by updated_by project workspace"), slices.Collect(maps.Keys(created)))
		require.Equal(t, "categories", created["type"])
		require.Equal(t, false, created["last_used"])
		require.Equal(t, testsupport.LabelProject, created["project"])
		require.Equal(t, testsupport.LabelWorkspace, created["workspace"])
		require.Equal(t, testsupport.LabelActor, created["created_by"])
		require.Nil(t, created["updated_by"])
		milestoneFailure(t, call("project_estimate", map[string]any{"action": "retrieve"}), "not_found")
		id := created["id"].(string)
		selected := milestoneObject(t, call("project_estimate", map[string]any{"action": "link", "estimate_id": id}))
		require.Equal(t, id, selected["estimate"])
		require.Equal(t, created, milestoneObject(t, call("project_estimate", map[string]any{"action": "retrieve"})))
		updated := milestoneObject(t, call("project_estimate", map[string]any{"action": "update", "name": "Renamed", "description": "", "external_source": ""}))
		require.Equal(t, "Renamed", updated["name"])
		require.Equal(t, "Description", updated["description"])
		require.Equal(t, "fixture", updated["external_source"])
		require.Equal(t, false, updated["last_used"])
		require.Equal(t, testsupport.LabelActor, updated["updated_by"])
		pointCall := catalogueScoped(call, map[string]any{"estimate_id": id})
		points := catalogueArray(t, pointCall("project_estimate", map[string]any{"action": "create_points", "points": `[{"key":2,"value":"XL","description":"Large","external_source":"fixture","external_id":"point"},{"key":null,"value":"S","description":null,"ignored":"extra"}]`}))
		require.Len(t, points, 2)
		require.ElementsMatch(t, strings.Fields("id estimate key value description external_id external_source created_at updated_at created_by updated_by project workspace"), slices.Collect(maps.Keys(points[0])))
		require.Equal(t, float64(2), points[0]["key"])
		require.Equal(t, float64(0), points[1]["key"])
		require.Equal(t, "", points[1]["description"])
		require.Nil(t, points[1]["external_id"])
		require.Equal(t, id, points[0]["estimate"])
		rows := catalogueArray(t, pointCall("project_estimate", map[string]any{"action": "list_points"}))
		require.Equal(t, []map[string]any{points[1], points[0]}, rows)
		point := milestoneObject(t, pointCall("project_estimate", map[string]any{"action": "update_point", "estimate_point_id": points[0]["id"], "key": 0, "value": "L", "description": "", "external_source": ""}))
		require.Equal(t, float64(0), point["key"])
		require.Equal(t, "L", point["value"])
		require.Equal(t, "Large", point["description"])
		require.Equal(t, "fixture", point["external_source"])
		require.Equal(t, testsupport.LabelActor, point["updated_by"])
		catalogueNull(t, pointCall("project_estimate", map[string]any{"action": "delete_point", "estimate_point_id": point["id"]}))
		rows = catalogueArray(t, pointCall("project_estimate", map[string]any{"action": "list_points"}))
		require.Len(t, rows, 1)
		require.Equal(t, points[1]["id"], rows[0]["id"])
		catalogueNull(t, call("project_estimate", map[string]any{"action": "delete"}))
		milestoneFailure(t, call("project_estimate", map[string]any{"action": "retrieve"}), "not_found")
		milestoneFailure(t, pointCall("project_estimate", map[string]any{"action": "list_points"}), "not_found")
	})
}

func TestMCPEstimatePointBatchValidationIsAtomic(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		created := milestoneObject(t, call("project_estimate", map[string]any{"action": "create", "name": "Validated points " + uuid.NewString(), "type": "points"}))
		call = catalogueScoped(call, map[string]any{"estimate_id": created["id"]})
		for _, points := range []string{
			`[{"value":"valid"},{"value":"123456789012345678901"}]`,
			`[{"value":"valid"},{"key":1}]`,
			`[{"value":"valid"},{"value":"invalid","key":-1}]`,
			`[{"value":"valid"},{"value":"invalid","key":2147483648}]`,
			`[{"value":"valid"},{"value":"invalid","key":0.5}]`,
			`[{"value":"valid"},null]`, `[]`, `{}`, `[{"value":"valid"}] {}`,
		} {
			request := map[string]any{"action": "create_points", "points": points, "request_id": uuid.NewString()}
			result := call("project_estimate", request)
			require.True(t, result.IsError, points)
			require.Empty(t, catalogueArray(t, call("project_estimate", map[string]any{"action": "list_points"})))
			var receipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE request_id=$1`, request["request_id"]).Scan(&receipts))
			require.Zero(t, receipts)
		}
		valid := catalogueArray(t, call("project_estimate", map[string]any{"action": "create_points", "points": `[{"value":"12345678901234567890","key":2147483647}]`}))
		require.Len(t, valid, 1)
		require.Equal(t, float64(2147483647), valid[0]["key"])
	})
}
