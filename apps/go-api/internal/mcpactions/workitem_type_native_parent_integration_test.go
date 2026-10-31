//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/stretchr/testify/require"
)

func TestTypeNativeEditCompletesBeforeWorkspaceDeletion(t *testing.T) {
	f, repository, command := typeMutationFixture(t, "update")
	catalogue, err := repository.Catalog(t.Context(), testsupport.LabelActor, "workspace-one")
	require.NoError(t, err)
	cipher, err := instancecipher.New("type-workspace-deletion-fixture")
	require.NoError(t, err)
	workspaces, err := instancemanagement.NewRepository(f.pool, cipher)
	require.NoError(t, err)
	native, deletion := contendWithNativeCatalogueWriter(t, f, `CREATE FUNCTION pause_type_parent_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
 CREATE TRIGGER pause_type_parent_write BEFORE UPDATE ON issue_types FOR EACH ROW EXECUTE FUNCTION pause_type_parent_write()`,
		func(ctx context.Context) (workitemtypes.Catalog, error) {
			return repository.Mutate(ctx, testsupport.LabelActor, "workspace-one", "update", command.TypeID.String(), workitemtypes.Input{Revision: catalogue.Revision, Name: "Accepted native edit", Icon: "task", Color: "#8b5cf6", Active: true})
		}, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, workspaces.SoftDeleteWorkspace(ctx, stduuid.MustParse(command.ActorID.String()), instancemanagement.WorkspaceDeletion{ID: testsupport.LabelWorkspace, Slug: "workspace-one"}, time.Now())
		})
	require.NoError(t, native.err)
	require.NoError(t, deletion.err)
	var name string
	var deleted bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT t.name,w.deleted_at IS NOT NULL FROM issue_types t JOIN workspaces w ON w.id=t.workspace_id WHERE t.id=$1`, command.TypeID).Scan(&name, &deleted))
	require.Equal(t, "Accepted native edit", name)
	require.True(t, deleted)
}

func TestTypeNativeImportRetainsProjectScopeUntilCommit(t *testing.T) {
	for _, change := range []struct{ name, query string }{
		{"membership revoked", `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`},
		{"project deleted", `UPDATE projects SET deleted_at=now() WHERE id=$1`},
	} {
		t.Run(change.name, func(t *testing.T) {
			f, repository, command := typeMutationFixture(t, "import_to_project")
			catalogue, err := repository.ProjectCatalog(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject)
			require.NoError(t, err)
			native, revocation := contendWithNativeCatalogueWriter(t, f, `CREATE FUNCTION pause_project_type_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
 CREATE TRIGGER pause_project_type_write BEFORE INSERT ON project_issue_types FOR EACH ROW EXECUTE FUNCTION pause_project_type_write()`,
				func(ctx context.Context) (workitemtypes.ProjectCatalog, error) {
					return repository.MutateProject(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, workitemtypes.ProjectInput{Revision: catalogue.Revision, Action: "import", TypeIDs: []string{command.TypeIDs[0].String()}})
				}, func(ctx context.Context) (struct{}, error) {
					args := []any{command.ProjectID}
					if change.name == "membership revoked" {
						args = append(args, command.ActorID)
					}
					_, updateErr := f.pool.Exec(ctx, change.query, args...)
					return struct{}{}, updateErr
				})
			require.NoError(t, native.err)
			require.NoError(t, revocation.err)
			var imported, accessible bool
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT
 EXISTS(SELECT 1 FROM project_issue_types WHERE issue_type_id=$1 AND project_id=$2 AND is_active AND deleted_at IS NULL),
 EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id WHERE p.id=$2 AND p.deleted_at IS NULL AND pm.member_id=$3 AND pm.is_active)`, command.TypeIDs[0], command.ProjectID, command.ActorID).Scan(&imported, &accessible))
			require.True(t, imported)
			require.False(t, accessible)
		})
	}
}
