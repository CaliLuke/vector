//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/stretchr/testify/require"
)

func TestTypeNativeDefinitionEditInvalidatesQueuedMCPMutations(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "resolve", "import_to_project"} {
		t.Run(action, func(t *testing.T) {
			f, repository, command := typeMutationFixture(t, action)
			catalogue, err := repository.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
			require.NoError(t, err)
			var id string
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id::text FROM issue_types WHERE name='Type mutation fixture'`).Scan(&id))
			native, mcp := contendWithNativeCatalogueWriter(t, f, `CREATE FUNCTION pause_type_definition() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
 CREATE TRIGGER pause_type_definition BEFORE UPDATE ON issue_types FOR EACH ROW WHEN (NEW.description='native edit') EXECUTE FUNCTION pause_type_definition()`,
				func(ctx context.Context) (workitemtypes.Catalog, error) {
					return repository.Mutate(ctx, testsupport.LabelActor, "workspace-one", "update", id, workitemtypes.Input{Revision: catalogue.Revision, Name: "Type mutation fixture", Description: "native edit", Icon: "task", Color: "#8b5cf6", Active: true})
				}, func(ctx context.Context) (projectmutations.Result, error) { return repository.MutateAPI(ctx, command) })
			require.NoError(t, native.err)
			require.ErrorIs(t, mcp.err, projectmutations.ErrRevision)
			var description string
			var receipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT description,(SELECT count(*) FROM mutation_receipts WHERE tool='workitem_type') FROM issue_types WHERE id=$1`, id).Scan(&description, &receipts))
			require.Equal(t, "native edit", description)
			require.Zero(t, receipts)
		})
	}
}

func TestTypeNativeImportInvalidatesQueuedMCPResolve(t *testing.T) {
	f, repository, command := typeMutationFixture(t, "resolve")
	catalogue, err := repository.ProjectCatalog(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject)
	require.NoError(t, err)
	var id string
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id::text FROM issue_types WHERE name='Type mutation fixture'`).Scan(&id))
	native, mcp := contendWithNativeCatalogueWriter(t, f, `CREATE FUNCTION pause_type_import() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
 CREATE TRIGGER pause_type_import BEFORE INSERT ON project_issue_types FOR EACH ROW EXECUTE FUNCTION pause_type_import()`,
		func(ctx context.Context) (workitemtypes.ProjectCatalog, error) {
			return repository.MutateProject(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, workitemtypes.ProjectInput{Revision: catalogue.Revision, Action: "import", TypeIDs: []string{id}})
		}, func(ctx context.Context) (projectmutations.Result, error) { return repository.MutateAPI(ctx, command) })
	require.NoError(t, native.err)
	require.ErrorIs(t, mcp.err, projectmutations.ErrRevision)
	var imports, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM project_issue_types WHERE issue_type_id=$1 AND is_active AND deleted_at IS NULL),(SELECT count(*) FROM mutation_receipts WHERE tool='workitem_type')`, id).Scan(&imports, &receipts))
	require.Equal(t, 1, imports)
	require.Zero(t, receipts)
}
