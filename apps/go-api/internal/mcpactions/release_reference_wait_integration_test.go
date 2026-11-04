//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPReleaseReferencesReauthorizeAfterWaiting(t *testing.T) {
	for _, tc := range []struct{ name, tool, action, key, sql, restore, code string }{
		{"lead", "release", "create", "lead_id", `UPDATE users SET is_active=false WHERE id=$1`, `UPDATE users SET is_active=true WHERE id=$1`, "not_found"},
		{"tag", "release", "create", "tag_id", `UPDATE release_tags SET deleted_at=now() WHERE id=$1`, `UPDATE release_tags SET deleted_at=NULL WHERE id=$1`, "not_found"},
		{"label", "release_label", "attach", "label_ids", `UPDATE release_labels SET deleted_at=now() WHERE id=$1`, `UPDATE release_labels SET deleted_at=NULL WHERE id=$1`, "not_found"},
		{"project role", "release", "manage_workitems", "", `UPDATE project_members SET role=5 WHERE member_id=$1`, `UPDATE project_members SET role=20 WHERE member_id=$1`, "forbidden"},
		{"deleted item", "release", "manage_workitems", "add_ids", `UPDATE issues SET deleted_at=now() WHERE id=$1`, `UPDATE issues SET deleted_at=NULL WHERE id=$1`, "not_found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := func(input map[string]any) any {
				if tc.key == "" {
					return testsupport.LabelActor
				}
				return input[tc.key]
			}
			changes := []mutationWaitChange{}
			for _, mode := range []string{"personal token", "OAuth"} {
				changes = append(changes, mutationWaitChange{name: mode, mode: mode, code: tc.code,
					apply: func(t *testing.T, ctx context.Context, tx pgx.Tx, input map[string]any) []byte {
						_, err := tx.Exec(ctx, tc.sql, target(input))
						require.NoError(t, err)
						return releaseSnapshotQuery(t, ctx, tx.QueryRow)
					},
					restore: func(t *testing.T, ctx context.Context, f milestoneFixture, input map[string]any) {
						_, err := f.pool.Exec(ctx, tc.restore, target(input))
						require.NoError(t, err)
					},
				})
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{changes: changes, tool: tc.tool, schemaLock: true, fixture: templateFixture, snapshot: releaseSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					input := releaseWriteInput(t, f, call, releaseMutationCase{tc.tool, tc.action})
					switch tc.name {
					case "lead":
						id, _ := customerReferenceTarget(t, f, "USER")
						input[tc.key] = id
					case "tag":
						input[tc.key] = milestoneObject(t, call("release_tag", map[string]any{"action": "create", "workspace_slug": "workspace-one", "version": uuid.NewString()}))["id"]
					}
					return input
				},
			})
		})
	}
}
