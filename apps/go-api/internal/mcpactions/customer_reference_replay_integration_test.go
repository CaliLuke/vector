//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerDefaultReferencesReauthorizeReplay(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		for _, action := range []string{"create", "update"} {
			for _, relation := range []string{"ISSUE", "USER"} {
				t.Run(action+"/"+relation, func(t *testing.T) {
					target, setActive := customerReferenceTarget(t, f, relation)
					input := customerDefaultReplayInput(t, call, action, relation, target)
					accepted := milestoneObject(t, call("customer_property", input))
					require.Equal(t, false, accepted["replayed"])
					require.Equal(t, true, milestoneObject(t, call("customer_property", input))["replayed"])
					t.Cleanup(func() { setActive(true) })
					setActive(false)
					fresh := maps.Clone(input)
					fresh["request_id"] = uuid.NewString()
					if action == "create" {
						fresh["display_name"] = "Fresh " + uuid.NewString()
					}
					milestoneFailure(t, call("customer_property", fresh), "invalid_params")
					milestoneFailure(t, call("customer_property", input), "invalid_params")
					setActive(true)
					replayed := milestoneObject(t, call("customer_property", input))
					require.Equal(t, true, replayed["replayed"])
					require.Equal(t, accepted["result"], replayed["result"])
					remove := map[string]any{"action": "delete", "property_id": accepted["resource_id"], "request_id": uuid.NewString()}
					deleted := milestoneObject(t, call("customer_property", remove))
					require.Equal(t, false, deleted["replayed"])
					setActive(false)
					require.Equal(t, true, milestoneObject(t, call("customer_property", remove))["replayed"])
					milestoneFailure(t, call("customer_property", input), "invalid_params")
					setActive(true)
					afterDelete := milestoneObject(t, call("customer_property", input))
					require.Equal(t, true, afterDelete["replayed"])
					require.Equal(t, accepted["result"], afterDelete["result"])
				})
			}
		}
	})
}

func customerDefaultReplayInput(t *testing.T, call catalogueCall, action, relation, target string) map[string]any {
	t.Helper()
	input := map[string]any{"action": "create", "display_name": "Reference " + uuid.NewString(), "property_type": "RELATION", "relation_type": relation}
	if action == "update" {
		property := milestoneObject(t, call("customer_property", input))
		input = map[string]any{"action": "update", "property_id": property["id"]}
	}
	input["default_value"], input["request_id"] = target, uuid.NewString()
	return input
}

func customerReferenceTarget(t *testing.T, f milestoneFixture, relation string) (string, func(bool)) {
	t.Helper()
	if relation == "ISSUE" {
		id := f.item(t)
		return id, func(active bool) {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			defer cancel()
			changed, err := f.pool.Exec(ctx, `UPDATE project_members SET is_active=$3 WHERE project_id=$1 AND member_id=$2`, testsupport.LabelProject, testsupport.LabelActor, active)
			require.NoError(t, err)
			require.EqualValues(t, 1, changed.RowsAffected())
		}
	}
	id := uuid.NewString()
	inserted, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u WHERE u.id=$2`, id, testsupport.LabelActor)
	require.NoError(t, err)
	require.EqualValues(t, 1, inserted.RowsAffected())
	inserted, err = f.pool.Exec(t.Context(), `INSERT INTO workspace_members SELECT (jsonb_populate_record(NULL::workspace_members,to_jsonb(wm)||jsonb_build_object('id',gen_random_uuid()::text,'member_id',$1::text))).* FROM workspace_members wm WHERE wm.workspace_id=$2 AND wm.member_id=$3`, id, testsupport.LabelWorkspace, testsupport.LabelActor)
	require.NoError(t, err)
	require.EqualValues(t, 1, inserted.RowsAffected())
	return id, func(active bool) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
		defer cancel()
		changed, updateErr := f.pool.Exec(ctx, `UPDATE workspace_members SET is_active=$3 WHERE workspace_id=$1 AND member_id=$2`, testsupport.LabelWorkspace, id, active)
		require.NoError(t, updateErr)
		require.EqualValues(t, 1, changed.RowsAffected())
	}
}

func TestMCPCustomerDefaultOptionsReauthorizeReplay(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		property := milestoneObject(t, call("customer_property", map[string]any{"action": "create", "display_name": "Option reference " + uuid.NewString(), "property_type": "OPTION", "options": `[{"name":"Choice"}]`}))
		optionID := property["options"].([]any)[0].(map[string]any)["id"].(string)
		input := map[string]any{"action": "update", "property_id": property["id"], "default_value": optionID, "request_id": uuid.NewString()}
		accepted := milestoneObject(t, call("customer_property", input))
		require.Equal(t, false, accepted["replayed"])
		setActive := func(active bool) {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
			defer cancel()
			changed, err := f.pool.Exec(ctx, `UPDATE customer_property_options SET is_active=$2 WHERE id=$1`, optionID, active)
			require.NoError(t, err)
			require.EqualValues(t, 1, changed.RowsAffected())
		}
		t.Cleanup(func() { setActive(true) })
		setActive(false)
		fresh := maps.Clone(input)
		fresh["request_id"] = uuid.NewString()
		milestoneFailure(t, call("customer_property", fresh), "invalid_params")
		milestoneFailure(t, call("customer_property", input), "invalid_params")
		setActive(true)
		replayed := milestoneObject(t, call("customer_property", input))
		require.Equal(t, true, replayed["replayed"])
		require.Equal(t, accepted["result"], replayed["result"])
		remove := map[string]any{"action": "delete", "property_id": property["id"], "request_id": uuid.NewString()}
		milestoneObject(t, call("customer_property", remove))
		require.Equal(t, true, milestoneObject(t, call("customer_property", remove))["replayed"])
		milestoneFailure(t, call("customer_property", input), "invalid_params")
	})
}

func TestMCPCustomerDefaultUpdateKeepsWorkspaceBinding(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET customers=true WHERE workspace_id=$1`, otherWorkspace)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": true}))
		propertyID := uuid.NewString()
		_, insertErr := f.pool.Exec(t.Context(), `INSERT INTO customer_properties(id,workspace_id,name,display_name,property_type,relation_type) VALUES($1,$2,$3,$3,'RELATION','USER')`, propertyID, otherWorkspace, "Foreign "+uuid.NewString())
		require.NoError(t, insertErr)
		var before, after []byte
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(p) FROM customer_properties p WHERE id=$1`, propertyID).Scan(&before))
		milestoneFailure(t, call("customer_property", map[string]any{"action": "update", "property_id": propertyID, "default_value": testsupport.LabelActor, "request_id": uuid.NewString()}), "not_found")
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(p) FROM customer_properties p WHERE id=$1`, propertyID).Scan(&after))
		require.JSONEq(t, string(before), string(after))
	})
}
