//go:build integration

package mcpactions_test

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// plane-sdk 0.3.1 CustomerWorkItem accepts project_identifier by field name,
// in addition to its project__identifier alias. The endpoint returns an array.
func TestMCPCustomerLinkedItemSDKRecordsAndSearch(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		export, importItem := f.item(t), f.item(t)
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET name=CASE WHEN id=$1 THEN 'Export report' ELSE 'Import data' END,sequence_id=CASE WHEN id=$1 THEN 1001 ELSE 2002 END WHERE id IN($1,$2)`, export, importItem)
		require.NoError(t, err)
		catalogueNull(t, call("customer", map[string]any{"action": "manage_workitems", "customer_id": customer["id"], "link_ids": export + "," + importItem}))
		input := map[string]any{"action": "list_workitems", "customer_id": customer["id"]}
		records := catalogueArray(t, call("customer", input))
		require.Len(t, records, 2)
		var selected map[string]any
		for _, record := range records {
			require.Equal(t, []string{"created_at", "id", "name", "priority", "project_id", "project_identifier", "sequence_id", "state_id", "updated_at"}, slices.Sorted(maps.Keys(record)))
			require.Contains(t, []string{export, importItem}, record["id"])
			require.Equal(t, testsupport.LabelProject, record["project_id"])
			for _, field := range []string{"created_at", "updated_at"} {
				_, parseErr := time.Parse(time.RFC3339Nano, record[field].(string))
				require.NoError(t, parseErr)
			}
			var name, priority, project string
			var state *string
			var sequence int64
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT i.name,i.priority,i.state_id::text,i.sequence_id,p.identifier FROM issues i JOIN projects p ON p.id=i.project_id WHERE i.id=$1`, record["id"]).Scan(&name, &priority, &state, &sequence, &project))
			require.Equal(t, name, record["name"])
			require.Equal(t, priority, record["priority"])
			if state == nil {
				require.Nil(t, record["state_id"])
			} else {
				require.Equal(t, *state, record["state_id"])
			}
			require.Equal(t, float64(sequence), record["sequence_id"])
			require.Equal(t, project, record["project_identifier"])
			if record["id"] == export {
				selected = record
			}
		}
		require.NotNil(t, selected)
		input["search"] = "eXpOrT"
		require.Equal(t, []map[string]any{selected}, catalogueArray(t, call("customer", input)))
		input["search"] = strings.ToLower(selected["project_identifier"].(string))
		require.ElementsMatch(t, records, catalogueArray(t, call("customer", input)))
		input["search"] = strconv.FormatInt(int64(selected["sequence_id"].(float64)), 10)
		require.Equal(t, []map[string]any{selected}, catalogueArray(t, call("customer", input)))
		input["search"] = "No matching linked work item"
		require.Empty(t, catalogueArray(t, call("customer", input)))
	})
}
