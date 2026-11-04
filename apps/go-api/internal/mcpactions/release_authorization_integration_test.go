//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestMCPReleaseMutationsAndReplayRequireCurrentAdministrator(t *testing.T) {
	runCatalogueCredentials(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, tc := range releaseMutationCases {
			t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
				input := releaseWriteInput(t, f, call, tc)
				for _, replay := range []bool{false, true} {
					if replay {
						require.Equal(t, false, milestoneObject(t, call(tc.tool, input))["replayed"])
					}
					_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=15 WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
					before := releaseSnapshot(t, f)
					milestoneFailure(t, call(tc.tool, input), "forbidden")
					require.Equal(t, before, releaseSnapshot(t, f))
					_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20 WHERE member_id=$1`, testsupport.LabelActor)
					require.NoError(t, err)
				}
				require.Equal(t, true, milestoneObject(t, call(tc.tool, input))["replayed"])
				conflict := maps.Clone(input)
				conflict["expected_revision"] = "different-reviewed-revision"
				before := releaseSnapshot(t, f)
				milestoneFailure(t, call(tc.tool, conflict), "request_conflict")
				require.Equal(t, before, releaseSnapshot(t, f))
			})
		}
	})
}
func TestMCPReleaseMutationsReauthorizeAfterWaiting(t *testing.T) {
	for _, tc := range releaseMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			changes := mutationCredentialChanges()
			if tc.action == "create" && tc.tool == "release" {
				changes = nil
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{changes: changes, tool: tc.tool, schemaLock: true, fixture: templateFixture, snapshot: releaseSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					return releaseWriteInput(t, f, call, tc)
				},
			})
		})
	}
}
