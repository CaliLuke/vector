//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/stretchr/testify/require"
)

func TestCoreMutationsPreserveAtomicRetry(t *testing.T) {
	for _, tc := range coreMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f := coreFixture(t)
			cmd, mutate := coreNativeCommand(t, f, tc)
			assertCatalogueAtomicRetry(t, f, "mutation_receipts", coreSnapshot, func(ctx context.Context) (projectmutations.Result, error) { return mutate(ctx, cmd) })
		})
	}
}
func TestMCPCoreMutationsReauthorizeAfterWaiting(t *testing.T) {
	for _, tc := range coreMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			changes := mutationCredentialChanges()
			if tc.tool == "workitem" && tc.action == "create" {
				changes = nil
			}
			runMutationAuthorizationWait(t, mutationWaitScenario{reuseCredentials: true, changes: changes, tool: tc.tool, schemaLock: true, fixture: coreFixture, snapshot: coreSnapshot, prepare: func(t *testing.T, f milestoneFixture, _ catalogueCall) map[string]any {
				return coreWriteInput(t, f, tc)
			}})
		})
	}
}
func TestMCPCoreReceiptsBindEveryActionInput(t *testing.T) {
	runCatalogue(t, coreFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, tc := range coreMutationCases {
			t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
				input := coreWriteInput(t, f, tc)
				accepted := milestoneObject(t, call(tc.tool, input))
				require.Equal(t, false, accepted["replayed"])
				before := coreSnapshot(t, f)
				require.Equal(t, true, milestoneObject(t, call(tc.tool, input))["replayed"])
				conflict := maps.Clone(input)
				conflict["expected_revision"] = "changed-review"
				if tc.action == "create" {
					delete(conflict, "expected_revision")
					key := "name"
					if tc.tool == "workitem_comment" {
						key = "comment_html"
					}
					if tc.tool == "workitem_link" {
						key = "title"
					}
					conflict[key] = "Changed input"
				}
				milestoneFailure(t, call(tc.tool, conflict), "request_conflict")
				require.Equal(t, before, coreSnapshot(t, f))
			})
		}
	})
}
