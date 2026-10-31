//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type richPropertyValue struct {
	call                              catalogueCall
	itemID, propertyID, typeID, title string
}

func newRichPropertyValue(t *testing.T, f milestoneFixture, root catalogueCall) richPropertyValue {
	t.Helper()
	enableTypeCatalogue(t, f, "workspace-one")
	kind := milestoneObject(t, root("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
	property := milestoneObject(t, root("workitem_property", map[string]any{"action": "create", "workitem_type_id": kind["id"], "display_name": uuid.NewString(), "property_type": "RELATION", "relation_type": "RICH_TEXT"}))
	item := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Rich text value", "type_id": kind["id"]}))
	return richPropertyValue{call: catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item["id"], "property_id": property["id"]}), itemID: item["id"].(string), propertyID: property["id"].(string), typeID: kind["id"].(string), title: property["display_name"].(string)}
}

func newPropertyImage(t *testing.T, f milestoneFixture) string {
	t.Helper()
	id := uuid.NewString()
	// An existing, completed upload: these tests exercise reference admission and
	// transactional binding, which never read or write the object's bytes.
	_, err := f.pool.Exec(t.Context(), `INSERT INTO file_assets(id,created_at,updated_at,attributes,asset,workspace_id,project_id,entity_type,size,is_uploaded,is_deleted,is_archived,created_by_id)
 VALUES($1,now(),now(),'{}',$1::uuid::text,$2,$3,'ISSUE_DESCRIPTION',1,true,false,false,$4)`, id, testsupport.LabelWorkspace, testsupport.LabelProject, testsupport.LabelActor)
	require.NoError(t, err)
	return id
}

func assertPropertyImageBound(t *testing.T, f milestoneFixture, image, item string) {
	t.Helper()
	var bound bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT issue_id=$2 AND project_id=$3 AND workspace_id=$4 AND entity_type='ISSUE_DESCRIPTION' AND draft_issue_id IS NULL FROM file_assets WHERE id=$1`, image, item, testsupport.LabelProject, testsupport.LabelWorkspace).Scan(&bound))
	require.True(t, bound, "the accepted rich-text image must belong to the target work item")
}

func propertyImageInput(image string) map[string]any {
	return map[string]any{"action": "set_value", "description_html": "<p>Image</p><img src=\"" + image + "\">", "request_id": uuid.NewString()}
}
