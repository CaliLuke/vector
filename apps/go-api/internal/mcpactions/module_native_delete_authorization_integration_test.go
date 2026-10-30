//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNativeModuleDeleteAuthorizationPreservesAppAndAPIContracts(t *testing.T) {
	for _, access := range []struct {
		name                       string
		creator                    bool
		workspaceRole, projectRole int
		appAllowed, apiAllowed     bool
	}{
		{"creator member", true, 15, 15, true, true},
		{"other member", false, 15, 15, false, false},
		{"project administrator", false, 15, 20, true, true},
		{"workspace administrator", false, 20, 15, true, false},
	} {
		t.Run(access.name, func(t *testing.T) {
			f := planningAuthorizationWaitFixture(t)
			repository, err := modulereads.NewLifecycleRepository(f.pool, rosterPublisher{}, "https://vector.example")
			require.NoError(t, err)
			created, err := repository.MutateResource(t.Context(), projectmutations.Command{
				ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace),
				ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one",
				Action: "create", Fields: map[string]any{"name": uuid.NewString()},
			})
			require.NoError(t, err)
			if !access.creator {
				_, err = f.pool.Exec(t.Context(), `UPDATE modules SET created_by_id=NULL WHERE id=$1`, created.ID)
				require.NoError(t, err)
			}
			_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=$1 WHERE member_id=$2`, access.workspaceRole, testsupport.LabelActor)
			require.NoError(t, err)
			_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=$1 WHERE member_id=$2`, access.projectRole, testsupport.LabelActor)
			require.NoError(t, err)
			before := planningMutationSnapshot(t, f)
			for _, route := range []struct {
				name         string
				api, allowed bool
			}{
				{"app", false, access.appAllowed},
				{"API", true, access.apiAllowed},
			} {
				t.Run(route.name, func(t *testing.T) {
					err := repository.CheckDeleteModule(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, created.ID, route.api)
					if route.allowed {
						require.NoError(t, err)
					} else if route.api {
						var denied *djangoapi.HTTPError
						require.ErrorAs(t, err, &denied)
						require.Equal(t, 403, denied.Status)
					} else {
						require.ErrorIs(t, err, modulereads.ErrForbidden)
					}
					require.Equal(t, before, planningMutationSnapshot(t, f), "authorization alone must not delete records or emit effects")
				})
			}
		})
	}
}
