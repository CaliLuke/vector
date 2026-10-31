//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type customerValueReferenceChange struct{ kind, name, disable, restore string }

var customerValueReferenceChanges = []customerValueReferenceChange{
	{"ISSUE", "private membership", `UPDATE project_members SET is_active=false WHERE project_id=@project AND member_id=@actor`, `UPDATE project_members SET is_active=true WHERE project_id=@project AND member_id=@actor`},
	{"ISSUE", "archived project", `UPDATE projects SET archived_at=now() WHERE id=@project`, `UPDATE projects SET archived_at=NULL WHERE id=@project`},
	{"ISSUE", "deleted project", `UPDATE projects SET deleted_at=now() WHERE id=@project`, `UPDATE projects SET deleted_at=NULL WHERE id=@project`},
	{"ISSUE", "draft item", `UPDATE issues SET is_draft=true WHERE id=@target`, `UPDATE issues SET is_draft=false WHERE id=@target`},
	{"ISSUE", "deleted item", `UPDATE issues SET deleted_at=now() WHERE id=@target`, `UPDATE issues SET deleted_at=NULL WHERE id=@target`},
	{"USER", "inactive user", `UPDATE users SET is_active=false WHERE id=@target`, `UPDATE users SET is_active=true WHERE id=@target`},
	{"USER", "inactive member", `UPDATE workspace_members SET is_active=false WHERE workspace_id=@workspace AND member_id=@target`, `UPDATE workspace_members SET is_active=true WHERE workspace_id=@workspace AND member_id=@target`},
	{"USER", "deleted member", `UPDATE workspace_members SET deleted_at=now() WHERE workspace_id=@workspace AND member_id=@target`, `UPDATE workspace_members SET deleted_at=NULL WHERE workspace_id=@workspace AND member_id=@target`},
	{"USER", "unsupported role", `UPDATE workspace_members SET role=99 WHERE workspace_id=@workspace AND member_id=@target`, `UPDATE workspace_members SET role=20 WHERE workspace_id=@workspace AND member_id=@target`},
	{"OPTION", "inactive option", `UPDATE customer_property_options SET is_active=false WHERE id=@target`, `UPDATE customer_property_options SET is_active=true WHERE id=@target`},
	{"OPTION", "deleted option", `UPDATE customer_property_options SET deleted_at=now() WHERE id=@target`, `UPDATE customer_property_options SET deleted_at=NULL WHERE id=@target`},
}

func customerValueReferenceInput(t *testing.T, f milestoneFixture, call catalogueCall, kind string) map[string]any {
	t.Helper()
	definition := map[string]any{"action": "create", "workspace_slug": "workspace-one", "display_name": uuid.NewString(), "property_type": "RELATION", "relation_type": kind}
	target := ""
	if kind == "OPTION" {
		definition["property_type"], definition["options"] = "OPTION", `[{"name":"Choice"}]`
		delete(definition, "relation_type")
	} else {
		target, _ = customerReferenceTarget(t, f, kind)
	}
	property := milestoneObject(t, call("customer_property", definition))
	if kind == "OPTION" {
		target = property["options"].([]any)[0].(map[string]any)["id"].(string)
	}
	parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}))
	encoded, err := json.Marshal(map[string][]string{property["id"].(string): {target}})
	require.NoError(t, err)
	return map[string]any{"action": "set_values", "workspace_slug": "workspace-one", "customer_id": parent["id"], "values": string(encoded), "request_id": uuid.NewString()}
}

func customerReferenceArguments(t *testing.T, input map[string]any) pgx.NamedArgs {
	t.Helper()
	arguments := pgx.NamedArgs{"workspace": testsupport.LabelWorkspace, "project": testsupport.LabelProject, "actor": testsupport.LabelActor}
	if target, ok := input["default_value"].(string); ok {
		arguments["target"], arguments["property"] = target, input["property_id"]
		return arguments
	}
	var values map[string][]string
	require.NoError(t, json.Unmarshal([]byte(input["values"].(string)), &values))
	require.Len(t, values, 1)
	for propertyID, targets := range values {
		require.Len(t, targets, 1)
		arguments["property"], arguments["target"] = propertyID, targets[0]
	}
	return arguments
}

func TestMCPCustomerValueReferencesRequireCurrentAccessOnReplay(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, change := range customerValueReferenceChanges {
			t.Run(change.kind+"/"+change.name, func(t *testing.T) {
				input := customerValueReferenceInput(t, f, call, change.kind)
				arguments := customerReferenceArguments(t, input)
				accepted := milestoneObject(t, call("customer_property", input))
				require.Equal(t, false, accepted["replayed"])
				set := func(statement string) {
					ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
					defer cancel()
					changed, err := f.pool.Exec(ctx, statement, arguments)
					require.NoError(t, err)
					require.EqualValues(t, 1, changed.RowsAffected())
				}
				t.Cleanup(func() { set(change.restore) })
				set(change.disable)
				before := customerStorageSnapshot(t, f)
				fresh := maps.Clone(input)
				fresh["request_id"] = uuid.NewString()
				for _, attempt := range []map[string]any{fresh, input} {
					denied := call("customer_property", attempt)
					milestoneFailure(t, denied, "invalid_params")
					require.Empty(t, denied.StructuredContent)
				}
				read := map[string]any{"action": "get_values", "customer_id": input["customer_id"], "property_id": arguments["property"]}
				require.Equal(t, map[string]any{arguments["property"].(string): []any{}}, milestoneObject(t, call("customer_property", read)))
				require.Equal(t, before, customerStorageSnapshot(t, f))
				set(change.restore)
				restored := customerStorageSnapshot(t, f)
				replayed := milestoneObject(t, call("customer_property", input))
				require.Equal(t, true, replayed["replayed"])
				require.Equal(t, accepted["result"], replayed["result"])
				require.Equal(t, map[string]any{arguments["property"].(string): []any{arguments["target"]}}, milestoneObject(t, call("customer_property", read)))
				require.Equal(t, restored, customerStorageSnapshot(t, f))
			})
		}
	})
}

func TestMCPCustomerValueReferencesReauthorizeAfterWaiting(t *testing.T) {
	for _, change := range customerValueReferenceChanges {
		t.Run(change.kind+"/"+change.name, func(t *testing.T) {
			runCustomerReferenceAuthorizationWait(t, change, func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
				return customerValueReferenceInput(t, f, call, change.kind)
			})
		})
	}
}

// Both default selection and stored values recheck references after the same
// catalogue lock. Capture legitimate catalogue changes in the lock holder's view.
func runCustomerReferenceAuthorizationWait(t *testing.T, change customerValueReferenceChange, prepare func(*testing.T, milestoneFixture, catalogueCall) map[string]any) {
	t.Helper()
	var changes []mutationWaitChange
	for _, mode := range []string{"personal token", "OAuth"} {
		changes = append(changes, mutationWaitChange{
			name: mode, mode: mode, code: "invalid_params",
			apply: func(t *testing.T, ctx context.Context, writer pgx.Tx, input map[string]any) []byte {
				changed, err := writer.Exec(ctx, change.disable, customerReferenceArguments(t, input))
				require.NoError(t, err)
				require.EqualValues(t, 1, changed.RowsAffected())
				return customerStorageSnapshotQuery(t, ctx, writer.QueryRow)
			},
		})
	}
	runMutationAuthorizationWait(t, mutationWaitScenario{
		tool: "customer_property", schemaLock: true, fixture: customerIdentityFixture,
		snapshot: customerStorageSnapshot, changes: changes, prepare: prepare,
	})
}
