//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPProjectFeaturePermissionsUseCurrentProjectMembership(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, access := range []struct {
			name, change string
			read, write  bool
		}{
			{"project admin and workspace guest", `UPDATE workspace_members SET role=5`, true, true},
			{"project admin and workspace member", `UPDATE workspace_members SET role=15`, true, true},
			{"workspace admin and project guest", `UPDATE project_members SET role=5`, true, false},
			{"workspace admin and project member", `UPDATE project_members SET role=15`, true, false},
			{"workspace admin without active project membership", `UPDATE project_members SET is_active=false`, false, false},
			{"deleted project membership", `UPDATE project_members SET deleted_at=now()`, false, false},
			{"deleted project", `UPDATE projects SET deleted_at=now()`, false, false},
		} {
			t.Run(access.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20,is_active=true,deleted_at=NULL; UPDATE projects SET network=0,archived_at=NULL,deleted_at=NULL,cycle_view=false`)
				require.NoError(t, err)
				_, err = f.pool.Exec(t.Context(), access.change)
				require.NoError(t, err)
				t.Cleanup(func() {
					_, restoreErr := f.pool.Exec(context.WithoutCancel(t.Context()), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20,is_active=true,deleted_at=NULL; UPDATE projects SET archived_at=NULL,deleted_at=NULL`)
					require.NoError(t, restoreErr)
				})
				read := call("project", map[string]any{"action": "get_features", "project_id": testsupport.LabelProject})
				if access.read {
					require.Equal(t, false, milestoneObject(t, read)["cycles"])
				} else {
					milestoneFailure(t, read, "forbidden")
				}
				before := featureMutationSnapshot(t, f)
				count := workspaceFeatureAuditCount(t, f)
				updated := call("project", map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "cycles": true, "request_id": uuid.NewString()})
				if access.write {
					result := milestoneObject(t, updated)
					require.Equal(t, true, result["result"].(map[string]any)["cycles"])
					require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
				} else {
					milestoneFailure(t, updated, "forbidden")
					require.JSONEq(t, string(before), string(featureMutationSnapshot(t, f)))
				}
			})
		}
	})
}

func TestMCPProjectFeatureAuditFailureRollsBackDefaultIntake(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, err := f.pool.Exec(t.Context(), `DELETE FROM intakes; UPDATE projects SET intake_view=false`)
		require.NoError(t, err)
		before := featureMutationSnapshot(t, f)
		count := workspaceFeatureAuditCount(t, f)
		_, err = f.pool.Exec(t.Context(), `ALTER TABLE feature_events ADD CONSTRAINT reject_project_feature_audit CHECK(false) NOT VALID`)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, cleanupErr := f.pool.Exec(context.WithoutCancel(t.Context()), `ALTER TABLE feature_events DROP CONSTRAINT IF EXISTS reject_project_feature_audit`)
			require.NoError(t, cleanupErr)
		})
		input := map[string]any{"action": "update_features", "project_id": testsupport.LabelProject, "intakes": true, "request_id": uuid.NewString()}
		milestoneFailure(t, call("project", input), "internal_error")
		require.JSONEq(t, string(before), string(featureMutationSnapshot(t, f)))
		_, err = f.pool.Exec(t.Context(), `ALTER TABLE feature_events DROP CONSTRAINT reject_project_feature_audit`)
		require.NoError(t, err)
		accepted := milestoneObject(t, call("project", input))
		require.Equal(t, true, accepted["result"].(map[string]any)["intakes"])
		require.Equal(t, false, accepted["replayed"])
		require.Equal(t, true, milestoneObject(t, call("project", input))["replayed"])
		var intakes int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM intakes WHERE project_id=$1 AND is_default AND deleted_at IS NULL`, testsupport.LabelProject).Scan(&intakes))
		require.Equal(t, 1, intakes)
		require.Equal(t, count+1, workspaceFeatureAuditCount(t, f))
	})
}
