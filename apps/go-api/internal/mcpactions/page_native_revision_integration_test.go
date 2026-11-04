//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestPageRevisionsObserveEveryNativeWriter(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		page := milestoneObject(t, call("page", map[string]any{"action": "create", "name": "Revision target", "description_html": "<p>Original</p>"}))["id"]
		collection := milestoneObject(t, call("collection", map[string]any{"action": "create", "name": "Revision folder"}))["id"]
		other, _ := customerReferenceTarget(t, f, "USER")
		args := pgx.NamedArgs{"page": page, "collection": collection, "workspace": testsupport.LabelWorkspace, "project": testsupport.LabelProject, "actor": testsupport.LabelActor, "other": other, "item": f.item(t), "record": uuid.NewString()}
		for _, sql := range []string{
			`INSERT INTO pages SELECT (jsonb_populate_record(NULL::pages,to_jsonb(p)||jsonb_build_object('id',@record::text))).* FROM pages p WHERE id=@page`,
			`UPDATE pages SET name='Native edit' WHERE id=@record`,
			`DELETE FROM pages WHERE id=@record`,
			`INSERT INTO project_pages(id,workspace_id,project_id,page_id,created_at,updated_at) VALUES(@record,@workspace,@project,@page,now(),now())`,
			`UPDATE project_pages SET deleted_at=now() WHERE id=@record`,
			`DELETE FROM project_pages WHERE id=@record`,
			`INSERT INTO page_collections(id,workspace_id,owned_by_id,name) VALUES(@record,@workspace,@actor,'Native folder')`,
			`UPDATE page_collections SET name='Native edit' WHERE id=@record`,
			`DELETE FROM page_collections WHERE id=@record`,
			`INSERT INTO page_collection_members(id,workspace_id,collection_id,member_id,access) VALUES(@record,@workspace,@collection,@other,2)`,
			`UPDATE page_collection_members SET access=0 WHERE id=@record`,
			`DELETE FROM page_collection_members WHERE id=@record`,
			`INSERT INTO page_collection_pages(id,workspace_id,collection_id,page_id) VALUES(@record,@workspace,@collection,@page)`,
			`UPDATE page_collection_pages SET sort_order=0 WHERE id=@record`,
			`DELETE FROM page_collection_pages WHERE id=@record`,
			`INSERT INTO work_item_pages(id,workspace_id,project_id,issue_id,page_id) VALUES(@record,@workspace,@project,@item,@page)`,
			`UPDATE work_item_pages SET deleted_at=now() WHERE id=@record`,
			`DELETE FROM work_item_pages WHERE id=@record`,
		} {
			t.Run(sql, func(t *testing.T) {
				assertCatalogueNativeRevision(t, f, call, "collection", "collection_id", collection, sql, args, pageSnapshot)
			})
		}
	})
}
