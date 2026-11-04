//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000036_pages_collections.down.sql
var pageStorageDown string

//go:embed schema_migrations/000036_pages_collections.up.sql
var pageStorageUp string

func TestPageRollbackPreservesIndependentLinksAndRetainedAudit(t *testing.T) {
	for _, state := range []string{"collection", "workspace page", "project page link", "retained audit"} {
		t.Run(state, func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			id := uuid.NewString()
			table := "pages"
			switch state {
			case "collection":
				table = "page_collections"
				f.exec(t, `INSERT INTO page_collections(id,workspace_id,owned_by_id,name) SELECT $1,id,owner_id,'Collection' FROM workspaces WHERE id=$2`, id, f.workspace)
			case "retained audit":
				table = "page_api_events"
				f.exec(t, `INSERT INTO page_api_events(id,workspace_id,resource_id,resource_kind,action) VALUES($1,$2,$1,'page','delete')`, id, f.workspace)
			default:
				f.exec(t, `INSERT INTO pages(id,workspace_id,owned_by_id,name,description_json,description_html,access,color,is_locked,view_props,logo_props,is_global,sort_order,created_at,updated_at) SELECT $1,id,owner_id,'Page','{}','<p>Page</p>',0,'',false,'{}','{}',$3,0,now(),now() FROM workspaces WHERE id=$2`, id, f.workspace, state == "workspace page")
				if state == "project page link" {
					table = "work_item_pages"
					f.exec(t, `INSERT INTO work_item_pages(id,workspace_id,project_id,issue_id,page_id) VALUES($1,$2,$3,$4,$1)`, id, f.workspace, f.project, f.item)
				}
			}
			_, err := f.db.Exec(f.ctx, pageStorageDown)
			require.ErrorContains(t, err, "workspace page data must be migrated before rollback")
			f.exec(t, `ROLLBACK`)
			var count int
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM `+table+` WHERE id=$1`, id).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}
