//go:build integration

package mcpactions_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func propertyRecordMutationInput(t *testing.T, call catalogueCall, action string) map[string]any {
	t.Helper()
	create := map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "OPTION", "options": `[{"name":"First"},{"name":"Second"}]`}
	if action == "create" {
		return create
	}
	property := milestoneObject(t, call("workitem_property", create))
	switch action {
	case "update":
		return map[string]any{"action": action, "workitem_property_id": property["id"], "description": "Updated description"}
	case "delete":
		return map[string]any{"action": action, "workitem_property_id": property["id"]}
	case "manage_type_properties":
		typeRow := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": uuid.NewString()}))
		return map[string]any{"action": action, "workitem_type_id": typeRow["id"], "attach_ids": property["id"]}
	case "create_option":
		return map[string]any{"action": action, "property_id": property["id"], "name": "New option"}
	case "update_option", "delete_option":
		input := map[string]any{"action": action, "property_id": property["id"], "option_id": property["options"].([]any)[0].(map[string]any)["id"]}
		if action == "update_option" {
			input["name"] = "Changed option"
		}
		return input
	default:
		t.Fatal("unknown property mutation", action)
		return nil
	}
}

func preparePropertyMutationInput(t *testing.T, f milestoneFixture, root catalogueCall, action string) map[string]any {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	enableTypeCatalogue(t, f, "workspace-one")
	call := catalogueScoped(root, map[string]any{"workspace_slug": "workspace-one"})
	var input map[string]any
	if strings.HasSuffix(action, "_value") {
		input = propertyValueMutationInput(t, call, action)
	} else {
		input = propertyRecordMutationInput(t, call, action)
	}
	input["workspace_slug"], input["request_id"] = "workspace-one", uuid.NewString()
	return input
}

func propertyValueMutationInput(t *testing.T, call catalogueCall, action string) map[string]any {
	t.Helper()
	typeRow := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
	property := milestoneObject(t, call("workitem_property", map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "TEXT", "workitem_type_id": typeRow["id"]}))
	item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Property mutation", "type_id": typeRow["id"]}))
	input := map[string]any{"action": "set_value", "project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"], "value": "Original value"}
	milestoneObject(t, call("workitem_property", input))
	input["action"], input["value"] = action, "Changed value"
	if action == "delete_value" {
		delete(input, "value")
	}
	return input
}

func propertyStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'properties',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM work_item_properties p),
 'links',(SELECT jsonb_agg(to_jsonb(l) ORDER BY type_id,property_id) FROM work_item_type_properties l),
 'settings',(SELECT jsonb_agg(to_jsonb(s) ORDER BY workspace_id) FROM work_item_type_settings s),
 'schema_revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id) FROM work_item_type_api_revisions r),
 'issues',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM issues i),
 'values',(SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM work_item_property_value_identities v),
 'item_revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY issue_id) FROM issue_edit_revisions r),
 'subscribers',(SELECT jsonb_agg(to_jsonb(s) ORDER BY id) FROM issue_subscribers s),
 'activities',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM issue_activities a),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,tool,action,request_id) FROM mutation_receipts r WHERE tool IN ('workitem_property','workitem_property_value')),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`).Scan(&snapshot))
	return snapshot
}
