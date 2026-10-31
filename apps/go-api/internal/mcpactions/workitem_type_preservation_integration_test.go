//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkitemTypePreservesDefaultAndHistoricalUsage(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		repository := enableTypeCatalogue(t, f, "workspace-one")
		catalogue, err := repository.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
		require.NoError(t, err)
		require.NotEmpty(t, catalogue.Types)
		require.True(t, catalogue.Types[0].Default)
		before := typeStorageSnapshot(t, f)
		for _, input := range []map[string]any{
			{"action": "delete", "workitem_type_id": catalogue.Types[0].ID},
			{"action": "update", "workitem_type_id": catalogue.Types[0].ID, "is_active": false},
			{"action": "create", "project_id": testsupport.LabelProject, "name": uuid.NewString()},
			{"action": "update", "project_id": testsupport.LabelProject, "workitem_type_id": catalogue.Types[0].ID, "name": "Project-owned"},
			{"action": "delete", "project_id": testsupport.LabelProject, "workitem_type_id": catalogue.Types[0].ID},
		} {
			require.True(t, call("workitem_type", input).IsError)
			require.Equal(t, before, typeStorageSnapshot(t, f))
		}
		created := milestoneObject(t, call("workitem_type", map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": uuid.NewString()}))
		item := milestoneObject(t, call("workitem", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": "Retained type history", "type_id": created["id"]}))["id"].(string)
		for _, state := range []string{
			`UPDATE issues SET archived_at=NULL,deleted_at=NULL WHERE id=$1`,
			`UPDATE issues SET archived_at=now(),deleted_at=NULL WHERE id=$1`,
			`UPDATE issues SET archived_at=NULL,deleted_at=now() WHERE id=$1`,
		} {
			_, err = f.pool.Exec(t.Context(), state, item)
			require.NoError(t, err)
			before = typeStorageSnapshot(t, f)
			milestoneFailure(t, call("workitem_type", map[string]any{"action": "delete", "workitem_type_id": created["id"], "request_id": uuid.NewString()}), "conflict")
			require.Equal(t, before, typeStorageSnapshot(t, f))
			var retained string
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT type_id FROM issues WHERE id=$1`, item).Scan(&retained))
			require.Equal(t, created["id"], retained)
		}
	})
}
