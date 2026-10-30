//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkspaceFeaturesPreserveOmittedFlagsAndAuditChanges(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET project_grouping=true,initiatives=true,teams=true,customers=true,wiki=true,pi=true WHERE workspace_id=$1`, testsupport.LabelWorkspace)
		require.NoError(t, err)
		before := milestoneObject(t, call("workspace", map[string]any{"action": "get_features"}))
		require.Equal(t, map[string]any{"project_grouping": true, "initiatives": true, "teams": true, "customers": true, "wiki": true, "pi": true, "work_item_types": true, "releases": true, "states_owned_by_workspace": false}, before)
		otherWorkspace := uuid.NewString()
		_, err = f.pool.Exec(t.Context(), `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1::uuid,'Other',$2,$1::text,'UTC','',now(),now())`, otherWorkspace, testsupport.LabelActor)
		require.NoError(t, err)
		var otherBefore, otherAfter []byte
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(f) FROM workspace_features f WHERE workspace_id=$1`, otherWorkspace).Scan(&otherBefore))
		count := workspaceFeatureAuditCount(t, f)
		updated := milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "customers": false, "teams": false, "wiki": nil}))
		require.Equal(t, map[string]any{"project_grouping": true, "initiatives": true, "teams": false, "customers": false, "wiki": true, "pi": true, "work_item_types": true, "releases": true, "states_owned_by_workspace": false}, updated)
		require.Equal(t, updated, milestoneObject(t, call("workspace", map[string]any{"action": "get_features"})))
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
		var old, current []byte
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT before_value,after_value FROM feature_events WHERE workspace_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, testsupport.LabelWorkspace).Scan(&old, &current))
		var recordedBefore, recordedAfter map[string]any
		require.NoError(t, json.Unmarshal(old, &recordedBefore))
		require.NoError(t, json.Unmarshal(current, &recordedAfter))
		require.Equal(t, before, recordedBefore)
		require.Equal(t, updated, recordedAfter)
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(f) FROM workspace_features f WHERE workspace_id=$1`, otherWorkspace).Scan(&otherAfter))
		require.JSONEq(t, string(otherBefore), string(otherAfter))
	})
}

func TestMCPWorkspaceFeatureReceiptsPreserveNativeEditsAndAuditOnce(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET customers=true,wiki=true WHERE workspace_id=$1`, testsupport.LabelWorkspace)
		require.NoError(t, err)
		before := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
		count := workspaceFeatureAuditCount(t, f)
		input := map[string]any{"action": "update_features", "customers": false, "request_id": uuid.NewString(), "expected_revision": before["revision"]}
		created := milestoneObject(t, call("workspace", input))
		require.Equal(t, testsupport.LabelWorkspace, created["resource_id"])
		require.Equal(t, false, created["replayed"])
		require.Equal(t, false, created["result"].(map[string]any)["customers"])
		require.NotEqual(t, before["revision"], created["revision"])
		conflict := maps.Clone(input)
		conflict["customers"] = true
		milestoneFailure(t, call("workspace", conflict), "request_conflict")
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))

		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_features SET wiki=false WHERE workspace_id=$1`, testsupport.LabelWorkspace)
		require.NoError(t, err)
		native := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
		require.Equal(t, false, native["wiki"])
		require.NotEqual(t, created["revision"], native["revision"])
		replay := milestoneObject(t, call("workspace", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, created["revision"], replay["revision"])
		stale := map[string]any{"action": "update_features", "wiki": true, "expected_revision": created["revision"]}
		milestoneFailure(t, call("workspace", stale), "revision_conflict")
		require.Equal(t, native, milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true})))
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))

		_, err = f.pool.Exec(t.Context(), `UPDATE projects SET module_view=NOT module_view WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		stale["expected_revision"] = native["revision"]
		milestoneFailure(t, call("workspace", stale), "revision_conflict")
		afterProjectEdit := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
		require.NotEqual(t, native["revision"], afterProjectEdit["revision"])
		delete(native, "revision")
		delete(afterProjectEdit, "revision")
		require.Equal(t, native, afterProjectEdit)
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
	})
}

func TestMCPWorkspaceFeatureAuditFailureRollsBackFlagsRevisionAndReceipt(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET customers=false WHERE workspace_id=$1`, testsupport.LabelWorkspace)
		require.NoError(t, err)
		before := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
		count := workspaceFeatureAuditCount(t, f)
		_, err = f.pool.Exec(t.Context(), `ALTER TABLE feature_events ADD CONSTRAINT reject_feature_audit CHECK(false) NOT VALID`)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, cleanupErr := f.pool.Exec(context.WithoutCancel(t.Context()), `ALTER TABLE feature_events DROP CONSTRAINT IF EXISTS reject_feature_audit`)
			require.NoError(t, cleanupErr)
		})
		input := map[string]any{"action": "update_features", "customers": true, "request_id": uuid.NewString(), "expected_revision": before["revision"]}
		milestoneFailure(t, call("workspace", input), "internal_error")
		require.Equal(t, before, milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true})))
		require.Equal(t, count, workspaceFeatureAuditCount(t, f))
		var receipts int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE request_id=$1`, input["request_id"]).Scan(&receipts))
		require.Zero(t, receipts)
		_, err = f.pool.Exec(t.Context(), `ALTER TABLE feature_events DROP CONSTRAINT reject_feature_audit`)
		require.NoError(t, err)
		accepted := milestoneObject(t, call("workspace", input))
		require.Equal(t, false, accepted["replayed"])
		require.Equal(t, true, accepted["result"].(map[string]any)["customers"])
		require.NotEqual(t, before["revision"], accepted["revision"])
		require.Equal(t, true, milestoneObject(t, call("workspace", input))["replayed"])
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
	})
}

func workspaceFeatureAuditCount(t *testing.T, f milestoneFixture) int {
	t.Helper()
	var count int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM feature_events WHERE workspace_id=$1`, testsupport.LabelWorkspace).Scan(&count))
	return count
}

func TestMCPWorkspaceFeatureReadsDoNotGrantWritePermission(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, role := range []struct {
			name  string
			value int
		}{{"guest", 5}, {"member", 15}} {
			t.Run(role.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=$2 WHERE member_id=$1`, testsupport.LabelActor, role.value)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, restoreErr)
				})
				before := milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true}))
				count := workspaceFeatureAuditCount(t, f)
				milestoneFailure(t, call("workspace", map[string]any{"action": "update_features", "customers": true, "request_id": uuid.NewString()}), "forbidden")
				require.Equal(t, before, milestoneObject(t, call("workspace", map[string]any{"action": "get_features", "include_revision": true})))
				require.Equal(t, count, workspaceFeatureAuditCount(t, f))
			})
		}
	})
}

func TestMCPFeatureNullFlagsMatchOmittedOnReceiptReplay(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, target := range []struct {
			tool, seed string
			flags      []string
		}{
			{"workspace", `UPDATE workspace_features SET project_grouping=true,initiatives=true,teams=true,customers=true,wiki=true,pi=true WHERE workspace_id=$1`, []string{"project_grouping", "initiatives", "teams", "customers", "wiki", "pi"}},
			{"project", `UPDATE projects SET module_view=true,cycle_view=true,issue_views_view=true,page_view=true,intake_view=true,is_issue_type_enabled=true,epics_enabled=true,parallel_cycles_enabled=true,project_updates_enabled=true,workflows_enabled=true WHERE workspace_id=$1`, []string{"modules", "cycles", "views", "pages", "intakes", "workitem_types", "epics", "parallel_cycles", "project_updates", "workflows"}},
		} {
			t.Run(target.tool, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), target.seed, testsupport.LabelWorkspace)
				require.NoError(t, err)
				read := map[string]any{"action": "get_features"}
				omitted := map[string]any{"action": "update_features", "request_id": uuid.NewString()}
				if target.tool == "project" {
					read["project_id"], omitted["project_id"] = testsupport.LabelProject, testsupport.LabelProject
				}
				before := milestoneObject(t, call(target.tool, read))
				count := workspaceFeatureAuditCount(t, f)
				explicit := maps.Clone(omitted)
				for _, flag := range target.flags {
					explicit[flag] = nil
				}
				created := milestoneObject(t, call(target.tool, explicit))
				require.Equal(t, before, created["result"])
				replay := milestoneObject(t, call(target.tool, omitted))
				require.Equal(t, true, replay["replayed"])
				require.Equal(t, created["result"], replay["result"])
				require.Equal(t, before, milestoneObject(t, call(target.tool, read)))
				require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
				invalid := maps.Clone(omitted)
				invalid["request_id"], invalid[target.flags[0]] = uuid.NewString(), "false"
				milestoneFailure(t, call(target.tool, invalid), "invalid_params")
				require.Equal(t, before, milestoneObject(t, call(target.tool, read)))
				require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
			})
		}
	})
}

func TestMCPProjectFeatureFalseAndNullPreserveNativeMapping(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE projects SET module_view=true,cycle_view=true,issue_views_view=true,page_view=true,intake_view=true,is_issue_type_enabled=true,epics_enabled=true,parallel_cycles_enabled=true,project_updates_enabled=true,workflows_enabled=true WHERE id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		updated := milestoneObject(t, call("project", map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": false, "workitem_types": false, "modules": nil}))
		require.Equal(t, map[string]any{"modules": true, "cycles": false, "views": true, "pages": true, "intakes": true, "work_item_types": false, "epics": true, "parallel_cycles": true, "project_updates": true, "workflows": true}, updated)
		require.Equal(t, updated, milestoneObject(t, call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject})))
		var modules, cycles, types bool
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT module_view,cycle_view,is_issue_type_enabled FROM projects WHERE id=$1`, testsupport.LabelProject).Scan(&modules, &cycles, &types))
		require.True(t, modules)
		require.False(t, cycles)
		require.False(t, types)
	})
}

func TestMCPWorkspaceFeaturesRejectForeignAndRevokedWorkspaceAccess(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20`)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES('00000000-0000-4000-8000-000000000099','Other',$1,'other','UTC','',now(),now())`, testsupport.LabelActor)
	require.NoError(t, err)
	for _, mode := range []string{"personal token", "OAuth"} {
		for _, version := range []string{"2025-11-25", "2026-07-28"} {
			t.Run(mode+"/"+version, func(t *testing.T) {
				handler, token := f.handler(t, nil), "test-one"
				if mode == "OAuth" {
					handler, token = oauthHandler(t, f.fixture, f.oauth.Authenticate), "oauth-user"
				}
				call := func(input string) rpcResult {
					return oauthToolCall(t, handler, version, token, "", "workspace", input, 200)
				}
				milestoneObject(t, call(`{"action":"get_features","workspace_slug":"workspace-one"}`))
				var before, after []byte
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(to_jsonb(f) ORDER BY workspace_id) FROM workspace_features f`).Scan(&before))
				milestoneFailure(t, call(`{"action":"get_features","workspace_slug":"other"}`), "forbidden")
				milestoneFailure(t, call(`{"action":"update_features","workspace_slug":"other","customers":true}`), "forbidden")
				_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE member_id=$1`, testsupport.LabelActor)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE workspace_members SET is_active=true WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, restoreErr)
				})
				milestoneFailure(t, call(`{"action":"get_features","workspace_slug":"workspace-one"}`), "forbidden")
				milestoneFailure(t, call(`{"action":"update_features","workspace_slug":"workspace-one","customers":true}`), "forbidden")
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(to_jsonb(f) ORDER BY workspace_id) FROM workspace_features f`).Scan(&after))
				require.JSONEq(t, string(before), string(after))
				var events int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM feature_events`).Scan(&events))
				require.Zero(t, events)
			})
		}
	}
}
