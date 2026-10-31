//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCustomerConcurrentUpsertsPreserveOneIdentityAndDisjointFields(t *testing.T) {
	for _, external := range []bool{false, true} {
		identity := "exact name"
		if external {
			identity = "external pair"
		}
		for _, receipts := range []bool{false, true} {
			mode := "without receipts"
			if receipts {
				mode = "independent receipts"
			}
			t.Run(identity+"/"+mode, func(t *testing.T) {
				f := customerIdentityFixture(t)
				repository := customers.NewRepository(f.pool)
				first := customerIdentityCommand()
				second := first
				second.Fields = map[string]any{"name": first.Fields["name"], "employees": int64(7)}
				if external {
					for _, fields := range []map[string]any{first.Fields, second.Fields} {
						fields["external_source"], fields["external_id"] = "crm", "same-pair"
					}
					second.Fields["name"] = "Second name"
				}
				if receipts {
					one, two := uuid.New(), uuid.New()
					first.RetryID, second.RetryID = &one, &two
				}
				outcomes := raceCustomerCommands(t, f, repository, first, second)
				require.NoError(t, outcomes[0].err)
				require.NoError(t, outcomes[1].err)
				require.Equal(t, outcomes[0].value.ID, outcomes[1].value.ID)
				var count, employees, events, savedReceipts int
				var email string
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*),min(email),min(employees),(SELECT count(*) FROM customer_events),(SELECT count(*) FROM mutation_receipts WHERE tool='customer') FROM customers WHERE deleted_at IS NULL`).Scan(&count, &email, &employees, &events, &savedReceipts))
				require.Equal(t, 1, count)
				require.Equal(t, "retained@example.test", email)
				require.Equal(t, 7, employees)
				require.Equal(t, 2, events, "two independent writes retain both audit events")
				if receipts {
					require.Equal(t, 2, savedReceipts)
				} else {
					require.Zero(t, savedReceipts)
				}
			})
		}
	}
}

func raceCustomerCommands(t *testing.T, f milestoneFixture, repository *customers.Repository, first, second customers.Command) [2]concurrentMutationOutcome[projectmutations.Result] {
	t.Helper()
	var calls [2]func(context.Context) (projectmutations.Result, error)
	for i, command := range [2]customers.Command{first, second} {
		calls[i] = func(ctx context.Context) (projectmutations.Result, error) { return repository.Mutate(ctx, command) }
	}
	return competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, calls)
}

type customerMutationScenario struct{ tool, action string }

var customerMutationScenarios = []customerMutationScenario{
	{"customer", "create"}, {"customer", "upsert"}, {"customer", "update"}, {"customer", "delete"},
	{"customer_request", "create"}, {"customer_request", "update"}, {"customer_request", "delete"},
}

func customerMutationCase(t *testing.T, scenario customerMutationScenario) (milestoneFixture, *customers.Repository, customers.Command) {
	t.Helper()
	f := customerIdentityFixture(t)
	repository := customers.NewRepository(f.pool)
	command := customerIdentityCommand()
	command.Fields["external_source"], command.Fields["external_id"] = "crm", uuid.NewString()
	if scenario.tool == "customer_request" {
		parent, err := repository.Mutate(t.Context(), command)
		require.NoError(t, err)
		command.Tool, command.CustomerID = scenario.tool, uuid.MustParse(parent.ID)
		command.Fields = map[string]any{"name": "Request " + uuid.NewString()}
		command.Add = []uuid.UUID{uuid.MustParse(f.item(t))}
	}
	action := scenario.action
	if action != "create" {
		created, err := repository.Mutate(t.Context(), command)
		require.NoError(t, err)
		command.ExpectedRevision = &created.Revision
		if action != "upsert" {
			command.ID = uuid.MustParse(created.ID)
			command.Action = action
			command.Add = nil
			command.Fields = map[string]any{}
		}
		if action != "delete" {
			command.Fields["name"] = "Updated buyer"
		}
	} else {
		// Obtain the public revision representation without duplicating its format.
		query := command.Query
		query.Action, query.IncludeRevision = "list", true
		raw, err := repository.Read(t.Context(), query)
		require.NoError(t, err)
		var result struct{ Revision string }
		require.NoError(t, json.Unmarshal(raw, &result))
		require.NotEmpty(t, result.Revision)
		command.ExpectedRevision = &result.Revision
	}
	key := uuid.New()
	command.RetryID = &key
	return f, repository, command
}

func TestCustomerCompetingReceiptsAndRevisionsCommitOnce(t *testing.T) {
	for _, scenario := range customerMutationScenarios {
		for _, mode := range []string{"same receipt", "changed input", "competing revision"} {
			t.Run(scenario.tool+"/"+scenario.action+"/"+mode, func(t *testing.T) {
				f, repository, first := customerMutationCase(t, scenario)
				second := first
				expectedError := error(nil)
				switch mode {
				case "changed input":
					second.Fields = maps.Clone(first.Fields)
					if first.Tool == "customer" {
						second.Fields["external_source"] = "different-source"
					} else if first.Action == "delete" {
						second.ExpectedRevision = nil
					} else {
						second.Fields["name"] = "Different request"
					}
					expectedError = mutationreceipts.ErrConflict
				case "competing revision":
					key := uuid.New()
					second.RetryID = &key
					expectedError = projectmutations.ErrRevision
				}
				outcomes := raceCustomerCommands(t, f, repository, first, second)
				if outcomes[0].err != nil {
					outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
				}
				require.NoError(t, outcomes[0].err)
				require.ErrorIs(t, outcomes[1].err, expectedError)
				if expectedError == nil {
					assertConcurrentReceiptReplay(t, outcomes[0].value, outcomes[1].value)
				} else {
					require.Equal(t, projectmutations.Result{}, outcomes[1].value)
				}
				var savedReceipts int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool=$1`, first.Tool).Scan(&savedReceipts))
				require.Equal(t, 1, savedReceipts)
				beforeReplay := customerStorageSnapshot(t, f)
				successfulReplays := 0
				for _, command := range []customers.Command{first, second} {
					replay, err := repository.Mutate(t.Context(), command)
					if err != nil {
						require.ErrorIs(t, err, expectedError)
						continue
					}
					successfulReplays++
					require.NotNil(t, replay.Replayed)
					require.True(t, *replay.Replayed)
					require.Equal(t, outcomes[0].value.ID, replay.ID)
					require.Equal(t, outcomes[0].value.Record, replay.Record)
				}
				require.Positive(t, successfulReplays)
				require.Equal(t, beforeReplay, customerStorageSnapshot(t, f))
			})
		}
	}
}
