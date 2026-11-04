//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

var initiativeMutationActions = []string{"create", "update", "delete", "add_projects", "remove_projects", "manage_workitems"}

func TestMCPInitiativeWritesAndReplayRequireCurrentAdministrator(t *testing.T) {
	runCatalogueCredentials(t, initiativeFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range initiativeMutationActions {
			t.Run(action, func(t *testing.T) {
				input := initiativeWriteInput(t, f, call, action)
				for _, replay := range []bool{false, true} {
					if replay {
						require.Equal(t, false, milestoneObject(t, call("initiative", input))["replayed"])
					}
					_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=15 WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
					before := initiativeSnapshot(t, f)
					milestoneFailure(t, call("initiative", input), "forbidden")
					require.Equal(t, before, initiativeSnapshot(t, f))
					_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
				}
				require.Equal(t, true, milestoneObject(t, call("initiative", input))["replayed"])
			})
		}
	})
}

func TestMCPInitiativeWritesReauthorizeAfterWaiting(t *testing.T) {
	for _, action := range initiativeMutationActions {
		t.Run(action, func(t *testing.T) {
			changes := mutationCredentialChanges()
			if action == "create" {
				changes = nil
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{
				reuseCredentials: true, changes: changes, tool: "initiative", schemaLock: true, fixture: initiativeFixture, snapshot: initiativeSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					return initiativeWriteInput(t, f, call, action)
				},
			})
		})
	}
}

func TestMCPInitiativeReferencesReauthorizeAfterWaiting(t *testing.T) {
	for _, tc := range []struct{ name, action, sql, restore, code string }{
		{"feature", "update", `UPDATE workspace_features SET initiatives=false`, `UPDATE workspace_features SET initiatives=true`, "feature_disabled"},
		{"lead", "create", `UPDATE users SET is_active=false WHERE id=$1`, `UPDATE users SET is_active=true WHERE id=$1`, "not_found"},
		{"project role", "manage_workitems", `UPDATE project_members SET role=5 WHERE member_id=$1`, `UPDATE project_members SET role=20 WHERE member_id=$1`, "forbidden"},
		{"deleted item", "manage_workitems", `UPDATE issues SET deleted_at=now() WHERE id=$1`, `UPDATE issues SET deleted_at=NULL WHERE id=$1`, "not_found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes := []mutationWaitChange{}
			for _, mode := range []string{"personal token", "OAuth"} {
				changes = append(changes, mutationWaitChange{name: mode, mode: mode, code: tc.code,
					apply: func(t *testing.T, ctx context.Context, tx pgx.Tx, input map[string]any) []byte {
						_, err := tx.Exec(ctx, tc.sql, initiativeReferenceArgs(tc.name, input)...)
						require.NoError(t, err)
						return initiativeSnapshotQuery(t, ctx, tx.QueryRow)
					},
					restore: func(t *testing.T, ctx context.Context, f milestoneFixture, input map[string]any) {
						_, err := f.pool.Exec(ctx, tc.restore, initiativeReferenceArgs(tc.name, input)...)
						require.NoError(t, err)
					},
				})
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{reuseCredentials: true, changes: changes, tool: "initiative", schemaLock: true, fixture: initiativeFixture, snapshot: initiativeSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					input := initiativeWriteInput(t, f, call, tc.action)
					if tc.name == "lead" {
						lead, _ := customerReferenceTarget(t, f, "USER")
						input["lead"] = lead
					}
					return input
				},
			})
		})
	}
}

func initiativeReferenceArgs(name string, input map[string]any) []any {
	switch name {
	case "feature":
		return nil
	case "lead":
		return []any{input["lead"]}
	case "deleted item":
		return []any{input["add_ids"]}
	default:
		return []any{testsupport.LabelActor}
	}
}
