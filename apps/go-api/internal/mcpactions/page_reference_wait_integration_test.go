//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPageReferencesRemainAuthorizedAfterWaiting(t *testing.T) {
	for _, tc := range []struct{ tool, action, key, sql, restore, code string }{
		{"page", "update", "page_id", `UPDATE pages SET access=1 WHERE id=$1`, `UPDATE pages SET access=0 WHERE id=$1`, "forbidden"},
		{"page", "attach_to_workitem", "page_id", `UPDATE pages SET access=1 WHERE id=$1`, `UPDATE pages SET access=0 WHERE id=$1`, "forbidden"},
		{"page", "set_collection", "collection_id", `UPDATE page_collection_members SET access=0 WHERE collection_id=$1`, `UPDATE page_collection_members SET access=2 WHERE collection_id=$1`, "forbidden"},
		{"collection", "add_member", "user_id", `UPDATE users SET is_active=false WHERE id=$1`, `UPDATE users SET is_active=true WHERE id=$1`, "not_found"},
	} {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			var changes []mutationWaitChange
			for _, mode := range []string{"personal token", "OAuth"} {
				changes = append(changes, mutationWaitChange{name: mode, mode: mode, code: tc.code,
					apply: func(t *testing.T, ctx context.Context, tx pgx.Tx, input map[string]any) []byte {
						_, err := tx.Exec(ctx, tc.sql, input[tc.key])
						require.NoError(t, err)
						return pageSnapshotQuery(t, ctx, tx.QueryRow)
					},
					restore: func(t *testing.T, ctx context.Context, f milestoneFixture, input map[string]any) {
						_, err := f.pool.Exec(ctx, tc.restore, input[tc.key])
						require.NoError(t, err)
					},
				})
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{changes: changes, tool: tc.tool, schemaLock: true, fixture: templateFixture, snapshot: pageSnapshot, prepare: func(t *testing.T, f milestoneFixture, _ catalogueCall) map[string]any {
				input := pageWriteInput(t, f, pageMutationCase{tc.tool, tc.action})
				if tc.tool == "page" {
					other, _ := customerReferenceTarget(t, f, "USER")
					table := "pages"
					if tc.action == "set_collection" {
						table = "page_collections"
					}
					_, err := f.pool.Exec(t.Context(), `UPDATE `+table+` SET owned_by_id=$2 WHERE id=$1 AND owned_by_id=$3`, input[tc.key], other, testsupport.LabelActor)
					require.NoError(t, err)
				}
				return input
			}})
		})
	}
}
