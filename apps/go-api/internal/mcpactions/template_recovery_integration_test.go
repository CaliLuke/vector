//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestTemplateFailedWritesRetainAuditRevisionAndRetryIdentity(t *testing.T) {
	for _, action := range []string{"create", "update", "delete"} {
		for _, failure := range []string{"cancelled lock wait", "commit rejected"} {
			t.Run(action+"/"+failure, func(t *testing.T) {
				f, repository, command := templateMutationCase(t, action)
				before := templateStorageSnapshot(t, f)
				var rejected projectmutations.Result
				if failure == "cancelled lock wait" {
					cancelScopeWriteWhileLocked(t, f, schemaMutationLock, command.WorkspaceID, func(ctx context.Context) error {
						var err error
						rejected, err = repository.Mutate(ctx, command)
						return err
					})
				} else {
					_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_template_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected template commit rejection'; END $$;
CREATE CONSTRAINT TRIGGER reject_template_commit AFTER INSERT OR UPDATE OR DELETE ON resource_templates DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_template_commit()`)
					require.NoError(t, err)
					rejected, err = repository.Mutate(t.Context(), command)
					require.ErrorIs(t, err, mutationreceipts.ErrIndeterminate)
					_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_template_commit ON resource_templates; DROP FUNCTION reject_template_commit()`)
					require.NoError(t, err)
				}
				require.Equal(t, projectmutations.Result{}, rejected)
				require.Equal(t, before, templateStorageSnapshot(t, f))
				accepted, err := repository.Mutate(t.Context(), command)
				require.NoError(t, err)
				require.NotNil(t, accepted.Replayed)
				require.False(t, *accepted.Replayed)
				after := templateStorageSnapshot(t, f)
				require.NotEqual(t, before, after)
				replay, err := repository.Mutate(t.Context(), command)
				require.NoError(t, err)
				require.NotNil(t, replay.Replayed)
				require.True(t, *replay.Replayed)
				accepted.Replayed = replay.Replayed
				require.Equal(t, accepted, replay)
				require.Equal(t, after, templateStorageSnapshot(t, f))
			})
		}
	}
}

// Template kinds share templates.scope and the same pre-dispatch revalidator.
// Exercise both SQL scope branches here. Every kind/action is still covered by
// TestMCPTemplateWritesAndReplayRecheckScopeRoleAndGovernance, and every projection
// by TestMCPTemplateSDKRecordsPreserveBodyAndMergeFields.
func TestMCPTemplateWritesReauthorizeAfterWaiting(t *testing.T) {
	for _, scope := range []templateScope{{"workitem", ""}, {"workitem", testsupport.LabelProject}} {
		for _, action := range []string{"create", "update", "delete"} {
			t.Run(scope.kind+"/"+scope.project+"/"+action, func(t *testing.T) {
				changes := mutationCredentialChanges()
				if action == "create" {
					changes = nil // Full membership matrix once per workspace/project scope.
				}
				runMutationAuthorizationWait(t, mutationWaitScenario{
					changes: changes,
					tool:    "template", schemaLock: true, fixture: templateFixture, snapshot: templateStorageSnapshot,
					prepare: func(t *testing.T, _ milestoneFixture, call catalogueCall) map[string]any {
						return templateWriteInput(t, call, scope, action)
					},
				})
			})
		}
	}
}
