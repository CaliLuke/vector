//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPWorkitemTypeReceiptsAndNativeRevisions(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		repository := enableTypeCatalogue(t, f, "workspace-one")
		create := map[string]any{"action": "create", "name": uuid.NewString(), "is_active": true, "request_id": uuid.NewString()}
		created := milestoneObject(t, call("workitem_type", create))
		id := created["resource_id"].(string)
		require.Equal(t, create["request_id"], created["request_id"])
		require.NotEmpty(t, created["replay_expires_at"])
		assertTypeReceiptReplay(t, f, call, create, created)
		catalogue, err := repository.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
		require.NoError(t, err)
		_, err = repository.Mutate(t.Context(), testsupport.LabelActor, "workspace-one", "update", id, workitemtypes.Input{Revision: catalogue.Revision, Name: create["name"].(string), Description: "Native edit", Icon: "task", Color: "#8b5cf6", Active: true})
		require.NoError(t, err)
		update := map[string]any{"action": "update", "workitem_type_id": id, "description": "MCP edit", "request_id": uuid.NewString(), "expected_revision": created["revision"]}
		before := typeStorageSnapshot(t, f)
		milestoneFailure(t, call("workitem_type", update), "revision_conflict")
		require.Equal(t, before, typeStorageSnapshot(t, f))
		current := milestoneObject(t, call("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": id, "include_revision": true}))
		update["expected_revision"] = current["revision"]
		updated := milestoneObject(t, call("workitem_type", update))
		assertTypeReceiptReplay(t, f, call, update, updated)
		changed := maps.Clone(update)
		changed["description"] = "Conflicting input"
		milestoneFailure(t, call("workitem_type", changed), "request_conflict")
		importedInput := map[string]any{"action": "import_to_project", "project_id": testsupport.LabelProject, "workitem_type_ids": id, "request_id": uuid.NewString(), "expected_revision": updated["revision"]}
		imported := milestoneObject(t, call("workitem_type", importedInput))
		require.Nil(t, imported["result"])
		require.Equal(t, testsupport.LabelProject, imported["resource_id"])
		importedInput["workitem_type_ids"] = id + "," + id
		assertTypeReceiptReplay(t, f, call, importedInput, imported)
		project, err := repository.ProjectCatalog(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject)
		require.NoError(t, err)
		_, err = repository.MutateProject(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, workitemtypes.ProjectInput{Revision: project.Revision, Action: "pause", TypeIDs: []string{id}})
		require.NoError(t, err)
		resolve := map[string]any{"action": "resolve", "project_id": testsupport.LabelProject, "name": create["name"], "request_id": uuid.NewString(), "expected_revision": imported["revision"]}
		before = typeStorageSnapshot(t, f)
		milestoneFailure(t, call("workitem_type", resolve), "revision_conflict")
		require.Equal(t, before, typeStorageSnapshot(t, f))
		current = milestoneObject(t, call("workitem_type", map[string]any{"action": "retrieve", "workitem_type_id": id, "include_revision": true}))
		resolve["expected_revision"] = current["revision"]
		resolved := milestoneObject(t, call("workitem_type", resolve))
		resolve["name"] = "  " + create["name"].(string) + "  "
		assertTypeReceiptReplay(t, f, call, resolve, resolved)
		deletion := map[string]any{"action": "delete", "workitem_type_id": id, "request_id": uuid.NewString(), "expected_revision": resolved["revision"]}
		deleted := milestoneObject(t, call("workitem_type", deletion))
		require.Nil(t, deleted["result"])
		assertTypeReceiptReplay(t, f, call, deletion, deleted)
		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=15`)
		require.NoError(t, err)
		for _, input := range []map[string]any{create, update, deletion} {
			milestoneFailure(t, call("workitem_type", input), "forbidden")
		}
		_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20`)
		require.NoError(t, err)
	})
}

func assertTypeReceiptReplay(t *testing.T, f milestoneFixture, call catalogueCall, input, accepted map[string]any) {
	t.Helper()
	before := typeStorageSnapshot(t, f)
	replay := milestoneObject(t, call("workitem_type", input))
	require.Equal(t, false, accepted["replayed"])
	require.Equal(t, true, replay["replayed"])
	for _, key := range []string{"resource_id", "result", "request_id", "revision", "replay_expires_at"} {
		require.Equal(t, accepted[key], replay[key], key)
	}
	require.Equal(t, before, typeStorageSnapshot(t, f))
}

func TestMCPWorkitemTypeProjectLinkReplayRechecksProjectAuthority(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		enableTypeCatalogue(t, f, "workspace-one")
		for _, action := range []string{"create", "update"} {
			input := map[string]any{"action": "create", "name": uuid.NewString(), "is_active": true, "project_ids": testsupport.LabelProject, "request_id": uuid.NewString()}
			if action == "update" {
				row := milestoneObject(t, call("workitem_type", map[string]any{"action": "create", "name": uuid.NewString(), "is_active": true}))
				input["action"], input["workitem_type_id"] = "update", row["id"]
			}
			created := milestoneObject(t, call("workitem_type", input))
			require.Equal(t, false, created["replayed"])
			before := typeStorageSnapshot(t, f)
			_, err := f.pool.Exec(t.Context(), `UPDATE project_members SET role=15 WHERE project_id=$1`, testsupport.LabelProject)
			require.NoError(t, err)
			milestoneFailure(t, call("workitem_type", input), "forbidden")
			require.Equal(t, before, typeStorageSnapshot(t, f))
			_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE project_id=$1`, testsupport.LabelProject)
			require.NoError(t, err)
			require.Equal(t, true, milestoneObject(t, call("workitem_type", input))["replayed"])
		}
	})
}

func TestMCPWorkitemTypeResolveReplayRechecksCurrentAvailability(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		repository := enableTypeCatalogue(t, f, "workspace-one")
		input := map[string]any{"action": "resolve", "name": uuid.NewString(), "project_id": testsupport.LabelProject, "request_id": uuid.NewString()}
		created := milestoneObject(t, call("workitem_type", input))
		id := created["resource_id"].(string)
		catalogue, err := repository.ProjectCatalog(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject)
		require.NoError(t, err)
		_, err = repository.MutateProject(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, workitemtypes.ProjectInput{Revision: catalogue.Revision, Action: "pause", TypeIDs: []string{id}})
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=15 WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		before := typeStorageSnapshot(t, f)
		milestoneFailure(t, call("workitem_type", input), "forbidden")
		require.Equal(t, before, typeStorageSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE project_id=$1`, testsupport.LabelProject)
		require.NoError(t, err)
		replay := milestoneObject(t, call("workitem_type", input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, created["result"], replay["result"])
		require.Equal(t, before, typeStorageSnapshot(t, f), "an authorized replay must not resume the paused link")
	})
}
