//go:build integration

package mcpactions_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPPropertyRichTextRejectsUnavailableImageWithoutReservingReceipt(t *testing.T) {
	f := newMilestoneFixture(t)
	otherWorkspace, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
	otherOwner := uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u WHERE id=$2`, otherOwner, testsupport.LabelActor)
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		value := newRichPropertyValue(t, f, root)
		otherItem := milestoneObject(t, root("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Other image scope", "type_id": value.typeID}))["id"]
		otherProject := milestoneObject(t, root("project", map[string]any{"action": "create", "name": "Image scope " + strings.ReplaceAll(uuid.NewString(), "-", ""), "identifier": strings.ToUpper(uuid.NewString()[:8])}))["id"]
		for _, example := range []struct {
			name, patch string
			args        []any
		}{
			{"unfinished upload", "is_uploaded=false", nil},
			{"soft-deleted image", "deleted_at=now()", nil},
			{"deleted image", "is_deleted=true", nil},
			{"archived image", "is_archived=true", nil},
			{"another uploader", "created_by_id=$2", []any{otherOwner}},
			{"another work item", "issue_id=$2", []any{otherItem}},
			{"another project", "project_id=$2", []any{otherProject}},
			{"another workspace", "workspace_id=$2,project_id=$3", []any{otherWorkspace, foreignProject}},
			{"user-bound image", "user_id=$2", []any{testsupport.LabelActor}},
			{"unbound attachment", "entity_type='ISSUE_ATTACHMENT'", nil},
		} {
			t.Run(example.name, func(t *testing.T) {
				image := newPropertyImage(t, f)
				args := append([]any{image}, example.args...)
				_, err := f.pool.Exec(t.Context(), `UPDATE file_assets SET `+example.patch+` WHERE id=$1`, args...)
				require.NoError(t, err)
				input := propertyImageInput(image)
				before := propertyStorageSnapshot(t, f)
				milestoneFailure(t, value.call("workitem_property", input), "invalid_params")
				require.Equal(t, before, propertyStorageSnapshot(t, f), "rejected references must preserve values, history, assets, receipts and effects")
				_, err = f.pool.Exec(t.Context(), `UPDATE file_assets SET workspace_id=$2,project_id=$3,created_by_id=$4,issue_id=NULL,user_id=NULL,entity_type='ISSUE_DESCRIPTION',is_uploaded=true,is_deleted=false,is_archived=false,deleted_at=NULL WHERE id=$1`, image, testsupport.LabelWorkspace, testsupport.LabelProject, testsupport.LabelActor)
				require.NoError(t, err)
				accepted := milestoneObject(t, value.call("workitem_property", input))
				require.Equal(t, false, accepted["replayed"])
				assertPropertyImageBound(t, f, image, value.itemID)
				after := propertyStorageSnapshot(t, f)
				accepted["replayed"] = true
				require.Equal(t, accepted, milestoneObject(t, value.call("workitem_property", input)))
				require.Equal(t, after, propertyStorageSnapshot(t, f))
			})
		}
	})
}

func TestMCPPropertyRichTextStorageFailureRollsBackImageAndHistory(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		for _, table := range []string{"work_item_property_versions", "work_item_effects", "mutation_receipts"} {
			t.Run(table, func(t *testing.T) {
				value := newRichPropertyValue(t, f, root)
				image := newPropertyImage(t, f)
				assertCatalogueStorageFailure(t, f, value.call, "workitem_property", propertyImageInput(image), table, func() []byte { return propertyStorageSnapshot(t, f) })
				assertPropertyImageBound(t, f, image, value.itemID)
				assertMCPPropertyHistory(t, f, value.itemID, value.propertyID, 1)
			})
		}
	})
}
