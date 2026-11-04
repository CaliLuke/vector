//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestReleaseRevisionsObserveEveryNativeWriter(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		parent := milestoneObject(t, call("release", map[string]any{"action": "create", "name": "Native revision target"}))["id"]
		label := milestoneObject(t, call("release_label", map[string]any{"action": "create", "name": "Linked label"}))["id"]
		args := pgx.NamedArgs{"parent": parent, "label": label, "workspace": testsupport.LabelWorkspace, "project": testsupport.LabelProject, "item": f.item(t), "record": uuid.NewString()}
		for _, sql := range []string{
			`INSERT INTO work_item_releases(id,workspace_id,name) VALUES(@record,@workspace,'Native release')`,
			`UPDATE work_item_releases SET name='Native edit' WHERE id=@record`,
			`DELETE FROM work_item_releases WHERE id=@record`,
			`INSERT INTO release_tags(id,workspace_id,version) VALUES(@record,@workspace,'Native tag')`,
			`UPDATE release_tags SET version='Native edit' WHERE id=@record`,
			`DELETE FROM release_tags WHERE id=@record`,
			`INSERT INTO release_labels(id,workspace_id,name) VALUES(@record,@workspace,'Native label')`,
			`UPDATE release_labels SET name='Native edit' WHERE id=@record`,
			`DELETE FROM release_labels WHERE id=@record`,
			`UPDATE release_changelogs SET description_stripped='Native edit' WHERE release_id=@parent`,
			`DELETE FROM release_changelogs WHERE release_id=@parent`,
			`INSERT INTO release_changelogs(release_id,workspace_id) VALUES(@parent,@workspace)`,
			`INSERT INTO release_item_labels(release_id,label_id,workspace_id) VALUES(@parent,@label,@workspace)`,
			`UPDATE release_item_labels SET workspace_id=workspace_id WHERE release_id=@parent`,
			`DELETE FROM release_item_labels WHERE release_id=@parent`,
			`INSERT INTO release_work_items(release_id,issue_id,project_id,workspace_id) VALUES(@parent,@item,@project,@workspace)`,
			`UPDATE release_work_items SET created_at=created_at+interval '1 second' WHERE release_id=@parent`,
			`DELETE FROM release_work_items WHERE release_id=@parent`,
		} {
			t.Run(sql, func(t *testing.T) {
				assertCatalogueNativeRevision(t, f, call, "release", "release_id", parent, sql, args, releaseSnapshot)
			})
		}
	})
}
