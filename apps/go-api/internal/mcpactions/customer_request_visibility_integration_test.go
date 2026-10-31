//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerRequestLinksRespectCurrentItemVisibility(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		item := f.item(t)
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": customer["id"], "name": "Linked request", "workitem_ids": item}))
		input := map[string]any{"action": "list_workitems", "customer_id": customer["id"], "customer_request_id": request["id"]}
		visible := catalogueArray(t, call("customer", input))
		require.Len(t, visible, 1)
		require.Equal(t, item, visible[0]["id"])
		require.Nil(t, request["work_item_ids"], "request projections deliberately omit linked IDs")
		for _, change := range []struct {
			name, hide, restore string
			id                  string
		}{
			{"draft item", `UPDATE issues SET is_draft=true WHERE id=$1`, `UPDATE issues SET is_draft=false WHERE id=$1`, item},
			{"deleted item", `UPDATE issues SET deleted_at=now() WHERE id=$1`, `UPDATE issues SET deleted_at=NULL WHERE id=$1`, item},
			{"archived project", `UPDATE projects SET archived_at=now() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`, testsupport.LabelProject},
			{"private project membership", `UPDATE project_members SET is_active=false WHERE project_id=$1`, `UPDATE project_members SET is_active=true WHERE project_id=$1`, testsupport.LabelProject},
		} {
			t.Run(change.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), change.hide, change.id)
				require.NoError(t, err)
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
					defer cancel()
					_, restoreErr := f.pool.Exec(ctx, change.restore, change.id)
					require.NoError(t, restoreErr)
				})
				require.Empty(t, catalogueArray(t, call("customer", input)))
				require.Empty(t, catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": customer["id"]})))
				detail := milestoneObject(t, call("customer_request", map[string]any{"action": "retrieve", "customer_id": customer["id"], "request_id": request["id"]}))
				require.Nil(t, detail["work_item_ids"])
			})
		}
		require.Equal(t, visible, catalogueArray(t, call("customer", input)))
	})
}

func TestMCPCustomerRequestsRejectMixedWorkspaceLinksAtomically(t *testing.T) {
	f := customerIdentityFixture(t)
	foreignWorkspace, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
	foreignItem := f.itemInScope(t, foreignWorkspace, foreignProject)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": uuid.NewString()}))
		localItem := f.item(t)
		input := map[string]any{"action": "create", "customer_id": customer["id"], "name": "Scoped request", "workitem_ids": localItem + "," + foreignItem, "mutation_id": uuid.NewString()}
		before := customerStorageSnapshot(t, f)
		denied := call("customer_request", input)
		milestoneFailure(t, denied, "not_found")
		require.Empty(t, denied.StructuredContent)
		require.Equal(t, before, customerStorageSnapshot(t, f), "cross-workspace links must not leave a partial request, link, effect or receipt")
		input["workitem_ids"] = localItem
		accepted := milestoneObject(t, call("customer_request", input))
		require.Equal(t, false, accepted["replayed"], "a rejected request leaves the mutation key reusable")
		linked := catalogueArray(t, call("customer", map[string]any{"action": "list_workitems", "customer_id": customer["id"], "customer_request_id": accepted["resource_id"]}))
		require.Len(t, linked, 1)
		require.Equal(t, localItem, linked[0]["id"])
	})
}
