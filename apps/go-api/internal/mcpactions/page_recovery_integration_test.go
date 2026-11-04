//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPageMutationsPreserveAtomicRetry(t *testing.T) {
	for _, tc := range pageMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f := templateFixture(t)
			cmd := pageNativeCommand(t, f, tc)
			repo := pagereads.NewPageAPIRepository(f.pool)
			assertCatalogueAtomicRetry(t, f, "page_api_events", pageSnapshot, func(ctx context.Context) (projectmutations.Result, error) { return repo.MutatePageAPI(ctx, cmd) })
		})
	}
}
func TestMCPPageMutationsReauthorizeAfterWaiting(t *testing.T) {
	for _, tc := range pageMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{reuseCredentials: true, changes: mutationCredentialChanges(), tool: tc.tool, schemaLock: true, fixture: templateFixture, snapshot: pageSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, _ catalogueCall) map[string]any {
					return pageWriteInput(t, f, tc)
				},
			})
		})
	}
}
func TestMCPPageReceiptsBindEveryActionInput(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, tc := range pageMutationCases {
			t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
				input := pageWriteInput(t, f, tc)
				accepted := milestoneObject(t, call(tc.tool, input))
				require.Equal(t, false, accepted["replayed"])
				before := pageSnapshot(t, f)
				require.Equal(t, true, milestoneObject(t, call(tc.tool, input))["replayed"])
				conflict := maps.Clone(input)
				conflict["expected_revision"] = "changed-review"
				if tc.action == "create" {
					delete(conflict, "expected_revision")
					conflict["name"] = "Different create input"
				}
				milestoneFailure(t, call(tc.tool, conflict), "request_conflict")
				require.Equal(t, before, pageSnapshot(t, f))
			})
		}
	})
}
func TestPageCompetingWritesPreserveRevisionBoundaries(t *testing.T) {
	for _, tc := range []pageMutationCase{{"page", "update"}, {"page", "set_collection"}, {"page", "attach_to_workitem"}, {"collection", "update"}, {"collection", "add_member"}} {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f := templateFixture(t)
			first := pageNativeCommand(t, f, tc)
			repo := pagereads.NewPageAPIRepository(f.pool)
			second := first
			request := uuid.New()
			second.RequestID = &request
			outcomes := competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, [2]func(context.Context) (projectmutations.Result, error){func(ctx context.Context) (projectmutations.Result, error) { return repo.MutatePageAPI(ctx, first) }, func(ctx context.Context) (projectmutations.Result, error) { return repo.MutatePageAPI(ctx, second) }})
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
		})
	}
}
