//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func customerIdentityFixture(t *testing.T) milestoneFixture {
	t.Helper()
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE workspace_features SET customers=true`)
	require.NoError(t, err)
	return f
}

func customerIdentityCommand() customers.Command {
	return customers.Command{
		Query:  customers.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Tool: "customer", Action: "create"},
		Fields: map[string]any{"name": "Buyer " + uuid.NewString(), "email": "retained@example.test"},
	}
}

func customerStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return customerStorageSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}

// Read through the changing transaction when a lock holder updates catalogue
// references, so the expected state includes its legitimate uncommitted changes.
func customerStorageSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	snapshot := map[string]string{}
	for _, table := range []string{"customers", "customer_requests", "customer_work_items", "customer_properties", "customer_property_options", "customer_property_values", "customer_events", "customer_revisions", "mutation_receipts", "work_item_effects"} {
		var rows string
		require.NoError(t, query(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM `+pgx.Identifier{table}.Sanitize()+` t`).Scan(&rows))
		snapshot[table] = rows
	}
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	return raw
}

func customerIdentityWriteInput(t *testing.T, call catalogueCall, action string) map[string]any {
	t.Helper()
	input := map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString(), "external_source": "crm", "external_id": uuid.NewString()}
	if action != "create" {
		created := milestoneObject(t, call("customer", input))
		if action == "upsert" {
			input["name"] = "Renamed buyer"
		} else {
			input = map[string]any{"action": action, "workspace_slug": "workspace-one", "customer_id": created["id"]}
			if action == "update" {
				input["name"] = "Renamed buyer"
			}
		}
	}
	input["request_id"] = uuid.NewString()
	return input
}

func customerMutationWriteInput(t *testing.T, call catalogueCall, scenario customerMutationScenario) map[string]any {
	t.Helper()
	if scenario.tool == "customer" {
		return customerIdentityWriteInput(t, call, scenario.action)
	}
	if scenario.tool == "customer_property" {
		return customerPropertyWriteInput(t, call, scenario.action)
	}
	parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}))
	input := map[string]any{"action": "create", "workspace_slug": "workspace-one", "customer_id": parent["id"], "name": uuid.NewString()}
	if scenario.action != "create" {
		created := milestoneObject(t, call("customer_request", input))
		input["request_id"], input["action"] = created["id"], scenario.action
		delete(input, "name")
		if scenario.action == "update" {
			input["name"] = "Updated request"
		}
	}
	input["mutation_id"] = uuid.NewString()
	return input
}
