//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPIntakeTriageAndDeletionPreserveWorkitemOwnership(t *testing.T) {
	runCatalogue(t, coreFixture(t), func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		milestoneObject(t, call("state", map[string]any{"action": "create", "name": "Accepted default", "color": "#123456", "group": "unstarted", "default": true}))
		submit := func(name string) string {
			t.Helper()
			return milestoneObject(t, call("intake", map[string]any{"action": "create", "name": name, "description_html": "<p>Submission</p>", "priority": "high"}))["issue"].(string)
		}
		item := submit("Triage lifecycle")
		for _, input := range []map[string]any{
			{"status": 0}, {"status": 2}, {"status": 0, "snoozed_till": "not-a-date"}, {"status": 2, "duplicate_to": uuid.NewString()},
		} {
			input["action"] = "update"
			input["workitem_id"] = item
			before := coreSnapshot(t, f)
			result := call("intake", input)
			require.True(t, result.IsError)
			require.Equal(t, before, coreSnapshot(t, f))
		}
		declined := milestoneObject(t, call("intake", map[string]any{"action": "update", "workitem_id": item, "status": -1, "source": "email", "source_email": "sender@example.test"}))
		require.EqualValues(t, -1, declined["status"])
		snoozed := milestoneObject(t, call("intake", map[string]any{"action": "update", "workitem_id": item, "status": 0, "snoozed_till": "2099-01-01T00:00:00Z"}))
		require.EqualValues(t, 0, snoozed["status"])
		require.Equal(t, "email", snoozed["source"])
		accepted := milestoneObject(t, call("intake", map[string]any{"action": "update", "workitem_id": item, "status": 1, "request_id": uuid.NewString()}))
		require.EqualValues(t, 1, accepted["result"].(map[string]any)["status"])
		deletion := map[string]any{"action": "delete", "workitem_id": item, "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call("intake", deletion))["replayed"])
		require.Equal(t, true, milestoneObject(t, call("intake", deletion))["replayed"])
		require.Equal(t, item, milestoneObject(t, call("workitem", map[string]any{"action": "retrieve", "workitem_id": item}))["id"])
		pending := submit("Pending deletion")
		catalogueNull(t, call("intake", map[string]any{"action": "delete", "workitem_id": pending}))
		var deleted bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM issues WHERE id=$1`, pending).Scan(&deleted))
		require.True(t, deleted)
		duplicate := submit("Duplicate")
		target := f.item(t)
		marked := milestoneObject(t, call("intake", map[string]any{"action": "update", "workitem_id": duplicate, "status": 2, "duplicate_to": target}))
		require.EqualValues(t, 2, marked["status"])
		require.Equal(t, target, marked["duplicate_to"])
	})
}
func TestMCPIntakeReadsAndReplayUseCurrentQueuePermissions(t *testing.T) {
	runCatalogueCredentials(t, coreFixture(t), func(t *testing.T, f milestoneFixture, root catalogueCall) {
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject})
		own := milestoneObject(t, call("intake", map[string]any{"action": "create", "name": "Own submission"}))["issue"]
		hidden := milestoneObject(t, call("intake", map[string]any{"action": "create", "name": "Other submission"}))["issue"]
		other, _ := customerReferenceTarget(t, f, "USER")
		_, err := f.pool.Exec(t.Context(), `UPDATE issues SET created_by_id=$2 WHERE id=$1`, hidden, other)
		require.NoError(t, err)
		edit := map[string]any{"action": "update", "workitem_id": own, "source": "api", "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call("intake", edit))["replayed"])
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE member_id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
		require.Equal(t, own, milestoneObject(t, call("intake", map[string]any{"action": "retrieve", "workitem_id": own}))["issue"])
		milestoneFailure(t, call("intake", map[string]any{"action": "retrieve", "workitem_id": hidden}), "not_found")
		listed := catalogueRows(t, call("intake", map[string]any{"action": "list", "per_page": 1000}))
		for _, row := range listed {
			require.NotEqual(t, hidden, row.(map[string]any)["issue"])
		}
		before := coreSnapshot(t, f)
		milestoneFailure(t, call("intake", edit), "forbidden")
		milestoneFailure(t, call("intake", map[string]any{"action": "delete", "workitem_id": hidden}), "forbidden")
		require.Equal(t, before, coreSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
		require.Equal(t, true, milestoneObject(t, call("intake", edit))["replayed"])
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET intake_view=false WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		before = coreSnapshot(t, f)
		milestoneFailure(t, call("intake", edit), "not_found")
		require.Equal(t, before, coreSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET intake_view=true WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
	})
}
