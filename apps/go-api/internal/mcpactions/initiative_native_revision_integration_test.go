//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestInitiativeRevisionsObserveEveryNativeWriter(t *testing.T) {
	runCatalogue(t, initiativeFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		parent := milestoneObject(t, call("initiative", map[string]any{"action": "create", "name": "Native writer target"}))["id"].(string)
		item := f.item(t)
		other := uuid.NewString()
		args := pgx.NamedArgs{"record": other, "parent": parent, "workspace": testsupport.LabelWorkspace, "project": testsupport.LabelProject, "item": item}
		for _, query := range []string{
			`INSERT INTO initiatives(id,workspace_id,name) VALUES(@record,@workspace,'Native record')`,
			`UPDATE initiatives SET name='Native edit' WHERE id=@record`,
			`DELETE FROM initiatives WHERE id=@record`,
			`INSERT INTO initiative_projects(initiative_id,project_id,workspace_id) VALUES(@parent,@project,@workspace)`,
			`UPDATE initiative_projects SET created_at=created_at+interval '1 second' WHERE initiative_id=@parent`,
			`DELETE FROM initiative_projects WHERE initiative_id=@parent`,
			`INSERT INTO initiative_work_items(initiative_id,issue_id,project_id,workspace_id) VALUES(@parent,@item,@project,@workspace)`,
			`UPDATE initiative_work_items SET created_at=created_at+interval '1 second' WHERE initiative_id=@parent`,
			`DELETE FROM initiative_work_items WHERE initiative_id=@parent`,
		} {
			// Named arguments avoid unused positional parameters in the individual writes.
			t.Run(query, func(t *testing.T) {
				assertCatalogueNativeRevision(t, f, call, "initiative", "initiative_id", parent, query, args, initiativeSnapshot)
			})
		}
	})
}
