//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type changingPropertyReader struct {
	*workitemtypes.Repository
	change func()
}

func (r changingPropertyReader) ReadPropertyAPI(ctx context.Context, q workitemtypes.PropertyAPIQuery) (json.RawMessage, error) {
	r.change()
	return r.Repository.ReadPropertyAPI(ctx, q)
}
func (r changingPropertyReader) ReadPropertyValueAPI(ctx context.Context, q workitemtypes.PropertyValueAPIQuery) (json.RawMessage, error) {
	r.change()
	return r.Repository.ReadPropertyValueAPI(ctx, q)
}

type propertyReadFixture struct {
	projectID, typeID, propertyID, optionID, itemID string
}

func newPropertyReadFixture(t *testing.T, f milestoneFixture, call catalogueCall, workspace, project string) propertyReadFixture {
	t.Helper()
	enableTypeCatalogue(t, f, workspace)
	kind := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": project, "name": uuid.NewString()}))
	property := milestoneObject(t, call("workitem_property", map[string]any{"action": "create", "display_name": uuid.NewString(), "property_type": "OPTION", "workitem_type_id": kind["id"], "options": `[{"name":"First"},{"name":"Second"}]`}))
	option := property["options"].([]any)[0].(map[string]any)["id"]
	item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": project, "name": "Read scope", "type_id": kind["id"]}))
	milestoneObject(t, call("workitem_property", map[string]any{"action": "set_value", "project_id": project, "workitem_id": item["id"], "property_id": property["id"], "value": option}))
	return propertyReadFixture{projectID: project, typeID: kind["id"].(string), propertyID: property["id"].(string), optionID: option.(string), itemID: item["id"].(string)}
}

func (f propertyReadFixture) requests() []map[string]any {
	return []map[string]any{
		{"action": "list", "project_id": f.projectID, "workitem_type_id": f.typeID},
		{"action": "retrieve", "project_id": f.projectID, "workitem_property_id": f.propertyID},
		{"action": "list_options", "project_id": f.projectID, "property_id": f.propertyID},
		{"action": "retrieve_option", "project_id": f.projectID, "property_id": f.propertyID, "option_id": f.optionID},
		f.valueRead(),
	}
}

func (f propertyReadFixture) valueRead() map[string]any {
	return map[string]any{"action": "get_value", "project_id": f.projectID, "workitem_id": f.itemID, "property_id": f.propertyID}
}

type propertyReadScopeChange struct {
	name, change, restore string
	args                  []any
}

func propertyReadScopeChanges(otherWorkspace string) []propertyReadScopeChange {
	return []propertyReadScopeChange{
		{"inactive user", `UPDATE users SET is_active=false WHERE id=$1`, `UPDATE users SET is_active=true WHERE id=$1`, []any{testsupport.LabelActor}},
		{"inactive workspace membership", `UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, `UPDATE workspace_members SET is_active=true WHERE workspace_id=$1 AND member_id=$2`, []any{testsupport.LabelWorkspace, testsupport.LabelActor}},
		{"invalid workspace role", `UPDATE workspace_members SET role=99 WHERE workspace_id=$1 AND member_id=$2`, `UPDATE workspace_members SET role=20 WHERE workspace_id=$1 AND member_id=$2`, []any{testsupport.LabelWorkspace, testsupport.LabelActor}},
		{"inactive project membership", `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, []any{testsupport.LabelProject, testsupport.LabelActor}},
		{"invalid project role", `UPDATE project_members SET role=99 WHERE project_id=$1 AND member_id=$2`, `UPDATE project_members SET role=20 WHERE project_id=$1 AND member_id=$2`, []any{testsupport.LabelProject, testsupport.LabelActor}},
		{"archived project", `UPDATE projects SET archived_at=now() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`, []any{testsupport.LabelProject}},
		{"deleted project", `UPDATE projects SET deleted_at=now() WHERE id=$1`, `UPDATE projects SET deleted_at=NULL WHERE id=$1`, []any{testsupport.LabelProject}},
		{"reassigned workspace slug", `WITH renamed AS (UPDATE workspaces SET slug='renamed' WHERE id=$1 RETURNING id) UPDATE workspaces SET slug='workspace-one' WHERE id=$2 AND EXISTS(SELECT 1 FROM renamed)`, `WITH restored AS (UPDATE workspaces SET slug='other' WHERE id=$2 RETURNING id) UPDATE workspaces SET slug='workspace-one' WHERE id=$1 AND EXISTS(SELECT 1 FROM restored)`, []any{testsupport.LabelWorkspace, otherWorkspace}},
	}
}

func propertyValueReadScopeChanges(f propertyReadFixture) []propertyReadScopeChange {
	return []propertyReadScopeChange{
		{"deleted item", `UPDATE issues SET deleted_at=now() WHERE id=$1`, `UPDATE issues SET deleted_at=NULL WHERE id=$1`, []any{f.itemID}},
		{"disabled catalogue", `UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, `UPDATE work_item_type_settings SET enabled=true WHERE workspace_id=$1`, []any{testsupport.LabelWorkspace}},
		{"inactive property", `UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','false') WHERE id=$1`, `UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','true') WHERE id=$1`, []any{f.propertyID}},
		{"detached property", `DELETE FROM work_item_type_properties WHERE type_id=$1 AND property_id=$2`, `INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, []any{f.typeID, f.propertyID}},
		{"restricted guest", `WITH guest AS (UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$3 RETURNING id) UPDATE issues SET created_by_id=NULL WHERE id=$2 AND EXISTS(SELECT 1 FROM guest)`, `WITH restored AS (UPDATE project_members SET role=20 WHERE project_id=$1 AND member_id=$3 RETURNING id) UPDATE issues SET created_by_id=$3 WHERE id=$2 AND EXISTS(SELECT 1 FROM restored)`, []any{f.projectID, f.itemID, testsupport.LabelActor}},
	}
}

func assertPropertyReadScopeChange(t *testing.T, f milestoneFixture, call catalogueCall, input map[string]any, change propertyReadScopeChange, pending *func(), code string) {
	t.Helper()
	invoked := false
	*pending = func() {
		invoked = true
		_, err := f.pool.Exec(t.Context(), change.change, change.args...)
		require.NoError(t, err)
	}
	result := call("workitem_property", input)
	*pending = nil
	_, err := f.pool.Exec(t.Context(), change.restore, change.args...)
	require.NoError(t, err)
	require.True(t, invoked, "the change must happen after action admission and before the actual repository read")
	milestoneFailure(t, result, code)
	require.False(t, call("workitem_property", input).IsError, "restoring the scope restores the same read")
}
