//go:build integration

package mcpactions_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerReadsRequireCurrentMembershipAndFeature(t *testing.T) {
	runCatalogueStatusFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, callStatus catalogueStatusCall) {
		call := catalogueCall(func(tool string, input map[string]any) rpcResult { return callStatus(tool, input, http.StatusOK) })
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "TEXT", "default_value": "Seeded value"}))
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": customer["id"], "name": "Request", "workitem_ids": f.item(t)}))
		reads := []struct {
			tool  string
			input map[string]any
		}{
			{"customer", map[string]any{"action": "list", "query": customer["name"]}},
			{"customer", map[string]any{"action": "retrieve", "customer_id": customer["id"]}},
			{"customer", map[string]any{"action": "list_workitems", "customer_id": customer["id"]}},
			{"customer_request", map[string]any{"action": "list", "customer_id": customer["id"]}},
			{"customer_request", map[string]any{"action": "retrieve", "customer_id": customer["id"], "request_id": request["id"]}},
			{"customer_property", map[string]any{"action": "list"}},
			{"customer_property", map[string]any{"action": "retrieve", "property_id": property["id"]}},
			{"customer_property", map[string]any{"action": "get_values", "customer_id": customer["id"]}},
		}
		expected := make([]rpcResult, len(reads))
		for index, read := range reads {
			expected[index] = call(read.tool, read.input)
			require.False(t, expected[index].IsError, expected[index].Content)
		}
		before := customerStorageSnapshot(t, f)
		setScope := func(role int, active, enabled bool) {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			defer cancel()
			changed, err := f.pool.Exec(ctx, `WITH feature AS (UPDATE workspace_features SET customers=$4 WHERE workspace_id=$1) UPDATE workspace_members SET role=$3,is_active=$5 WHERE workspace_id=$1 AND member_id=$2`, testsupport.LabelWorkspace, testsupport.LabelActor, role, enabled, active)
			require.NoError(t, err)
			require.EqualValues(t, 1, changed.RowsAffected())
		}
		for _, change := range []struct {
			name, code      string
			role            int
			active, enabled bool
		}{
			{"guest reads", "", 5, true, true}, {"member reads", "", 15, true, true},
			{"disabled feature", "feature_disabled", 20, true, false}, {"inactive membership", "forbidden", 20, false, true},
		} {
			t.Run(change.name, func(t *testing.T) {
				t.Cleanup(func() { setScope(20, true, true) })
				setScope(change.role, change.active, change.enabled)
				for index, read := range reads {
					if !change.active {
						require.Equal(t, rpcResult{}, callStatus(read.tool, read.input, http.StatusForbidden), "revoked membership is rejected before tool dispatch")
						continue
					}
					result := call(read.tool, read.input)
					if change.code == "" {
						require.Equal(t, expected[index], result, read.tool, read.input["action"])
					} else {
						milestoneFailure(t, result, change.code)
						require.Empty(t, result.StructuredContent)
					}
				}
				require.Equal(t, before, customerStorageSnapshot(t, f))
				setScope(20, true, true)
				for index, read := range reads {
					require.Equal(t, expected[index], call(read.tool, read.input))
				}
			})
		}
	})
}
