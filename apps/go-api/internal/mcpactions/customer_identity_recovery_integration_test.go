//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCustomerInterruptedMutationsPreserveStateAndRetryIdentity(t *testing.T) {
	for _, scenario := range customerMutationScenarios {
		for _, failure := range []string{"cancelled lock wait", "rejected commit"} {
			t.Run(scenario.tool+"/"+scenario.action+"/"+failure, func(t *testing.T) {
				f, repository, command := customerMutationCase(t, scenario)
				before := customerStorageSnapshot(t, f)
				var rejected projectmutations.Result
				if failure == "cancelled lock wait" {
					cancelScopeWriteWhileLocked(t, f, schemaMutationLock, command.WorkspaceID, func(ctx context.Context) error {
						var err error
						rejected, err = repository.Mutate(ctx, command)
						return err
					})
				} else {
					table := "customers"
					switch scenario.tool {
					case "customer_request":
						table = "customer_requests"
					case "customer_property":
						table = "customer_properties"
						if scenario.action == "set_values" {
							table = "customer_property_values"
						}
					}
					tableName := pgx.Identifier{table}.Sanitize()
					_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_customer_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected customer commit rejection'; END $$;
CREATE CONSTRAINT TRIGGER reject_customer_commit AFTER INSERT OR UPDATE OR DELETE ON `+tableName+` DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_customer_commit()`)
					require.NoError(t, err)
					rejected, err = repository.Mutate(t.Context(), command)
					require.ErrorIs(t, err, mutationreceipts.ErrIndeterminate)
					_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_customer_commit ON `+tableName+`; DROP FUNCTION reject_customer_commit()`)
					require.NoError(t, err)
				}
				require.Equal(t, projectmutations.Result{}, rejected, "failed writes return no accepted record or receipt")
				require.Equal(t, before, customerStorageSnapshot(t, f), "rollback includes customer data, audit, receipts, revisions and effects")
				accepted, err := repository.Mutate(t.Context(), command)
				require.NoError(t, err)
				require.NotNil(t, accepted.Replayed)
				require.False(t, *accepted.Replayed)
				saved := customerStorageSnapshot(t, f)
				require.NotEqual(t, before, saved)
				replay, err := repository.Mutate(t.Context(), command)
				require.NoError(t, err)
				require.NotNil(t, replay.Replayed)
				require.True(t, *replay.Replayed)
				accepted.Replayed = replay.Replayed
				require.Equal(t, accepted, replay)
				require.Equal(t, saved, customerStorageSnapshot(t, f))
			})
		}
	}
}

func TestMCPCustomerIdentityWritesReauthorizeAfterWaiting(t *testing.T) {
	for _, scenario := range customerMutationScenarios {
		t.Run(scenario.tool+"/"+scenario.action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{
				tool: scenario.tool, schemaLock: true, fixture: customerIdentityFixture, snapshot: customerStorageSnapshot,
				prepare: func(t *testing.T, _ milestoneFixture, call catalogueCall) map[string]any {
					return customerMutationWriteInput(t, call, scenario)
				},
			})
		})
	}
}
