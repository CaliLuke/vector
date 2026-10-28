//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type catalogueSafetyCase struct {
	tool, table, selector string
	fields                map[string]any
}

// Exercise each domain's own revision triggers and authorization callback. The
// shared receipt engine's contention and commit-uncertainty tests are separate.
func TestMCPCatalogueReceiptsRespectNativeEditsAndCurrentPermissions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_features SET customers=true,initiatives=true`)
		require.NoError(t, err)
		for _, tc := range []catalogueSafetyCase{
			{tool: "customer", table: "customers", selector: "customer_id"},
			{tool: "initiative", table: "initiatives", selector: "initiative_id"},
			{tool: "release", table: "work_item_releases", selector: "release_id"},
			{tool: "template", table: "resource_templates", selector: "template_id", fields: map[string]any{"kind": "project", "template_data": `{}`}},
			{tool: "page", table: "pages", selector: "page_id", fields: map[string]any{"description_html": "<p>Private page</p>", "access": 1}},
			{tool: "collection", table: "page_collections", selector: "collection_id", fields: map[string]any{"access": "private"}},
		} {
			t.Run(tc.tool, func(t *testing.T) { catalogueReceiptSafety(t, f, call, tc) })
		}
	})
}

func catalogueSafetyInput(tc catalogueSafetyCase, action, name string) map[string]any {
	input := map[string]any{"action": action, "name": name}
	if action == "create" {
		maps.Copy(input, tc.fields)
	}
	if tc.tool == "template" {
		input["kind"] = "project"
	}
	return input
}

func catalogueReceiptSafety(t *testing.T, f milestoneFixture, call catalogueCall, tc catalogueSafetyCase) {
	t.Helper()
	create := catalogueSafetyInput(tc, "create", uuid.NewString())
	create["request_id"] = uuid.NewString()
	created := milestoneObject(t, call(tc.tool, create))
	id := created["resource_id"].(string)
	conflict := maps.Clone(create)
	conflict["name"] = "Different input"
	milestoneFailure(t, call(tc.tool, conflict), "request_conflict")

	// A writer outside the MCP repository must invalidate its prior revision.
	_, err := f.pool.Exec(t.Context(), `UPDATE `+tc.table+` SET name=$2 WHERE id=$1`, id, "Native edit")
	require.NoError(t, err)
	update := catalogueSafetyInput(tc, "update", "Reviewed edit")
	update[tc.selector], update["expected_revision"] = id, created["revision"]
	milestoneFailure(t, call(tc.tool, update), "revision_conflict")
	var name string
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name FROM `+tc.table+` WHERE id=$1`, id).Scan(&name))
	require.Equal(t, "Native edit", name)
	replay := milestoneObject(t, call(tc.tool, create))
	require.Equal(t, true, replay["replayed"])
	require.Equal(t, created["result"], replay["result"])

	delete(update, "expected_revision")
	update["request_id"] = uuid.NewString()
	accepted := milestoneObject(t, call(tc.tool, update))
	require.Equal(t, false, accepted["replayed"])
	t.Run("revoked write access blocks replay", func(t *testing.T) {
		revokeCatalogueWrite(t, f, tc, id)
		milestoneFailure(t, call(tc.tool, update), "forbidden")
	})
	require.Equal(t, true, milestoneObject(t, call(tc.tool, update))["replayed"])

	if tc.tool == "page" {
		milestoneObject(t, call("page", map[string]any{"action": "archive", "page_id": id}))
	}
	remove := map[string]any{"action": "delete", tc.selector: id, "request_id": uuid.NewString()}
	if tc.tool == "template" {
		remove["kind"] = "project"
	}
	deleted := milestoneObject(t, call(tc.tool, remove))
	require.Equal(t, false, deleted["replayed"])
	require.Nil(t, deleted["result"])
	require.Equal(t, true, milestoneObject(t, call(tc.tool, remove))["replayed"])
}

func revokeCatalogueWrite(t *testing.T, f milestoneFixture, tc catalogueSafetyCase, id string) {
	t.Helper()
	if tc.tool != "page" && tc.tool != "collection" {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=5 WHERE member_id=$1`, testsupport.LabelActor)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
			require.NoError(t, restoreErr)
		})
		return
	}
	owner := uuid.NewString()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u LIMIT 1`, owner)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `UPDATE `+tc.table+` SET owned_by_id=$2 WHERE id=$1`, id, owner)
	require.NoError(t, err)
	if tc.tool == "collection" {
		_, err = f.pool.Exec(t.Context(), `UPDATE page_collection_members SET access=0 WHERE collection_id=$1 AND member_id=$2`, id, testsupport.LabelActor)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE `+tc.table+` SET owned_by_id=$2 WHERE id=$1`, id, testsupport.LabelActor)
		require.NoError(t, restoreErr)
	})
}
