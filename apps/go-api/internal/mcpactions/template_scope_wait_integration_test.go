//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPTemplateProjectWritesObserveGovernanceAndScopeChangesWhileWaiting(t *testing.T) {
	for _, action := range []string{"create", "update", "delete"} {
		t.Run(action, func(t *testing.T) {
			changes := []mutationWaitChange{}
			for _, mode := range []string{"personal token", "OAuth"} {
				for _, change := range []struct{ name, sql, code string }{
					{"workspace governance enabled", `UPDATE workspace_template_settings SET owned_by_workspace=true WHERE workspace_id='` + testsupport.LabelWorkspace + `'`, "workspace_owned"},
					{"project archived", `UPDATE projects SET archived_at=now() WHERE id='` + testsupport.LabelProject + `'`, "forbidden"},
					{"project administrator demoted", `UPDATE project_members SET role=15 WHERE project_id='` + testsupport.LabelProject + `'`, "forbidden"},
					{"workspace membership revoked", `UPDATE workspace_members SET is_active=false WHERE workspace_id='` + testsupport.LabelWorkspace + `'`, "forbidden"},
				} {
					changes = append(changes, mutationWaitChange{name: change.name + "/" + mode, mode: mode, code: change.code,
						apply: func(t *testing.T, ctx context.Context, tx pgx.Tx, _ map[string]any) []byte {
							_, err := tx.Exec(ctx, change.sql)
							require.NoError(t, err)
							return templateSnapshotQuery(t, ctx, tx.QueryRow)
						},
					})
				}
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{
				changes: changes, tool: "template", schemaLock: true, fixture: templateFixture, snapshot: templateStorageSnapshot,
				prepare: func(t *testing.T, _ milestoneFixture, call catalogueCall) map[string]any {
					return templateWriteInput(t, call, templateScope{"workitem", testsupport.LabelProject}, action)
				},
			})
		})
	}
}
