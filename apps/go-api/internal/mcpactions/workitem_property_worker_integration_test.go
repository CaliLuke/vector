//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"maps"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type propertyWorkerCase struct {
	itemID, propertyID string
	inputs             []map[string]any
	receipts           []map[string]any
}

func TestMCPPropertyValueWorkerDeliversCommittedEffectsOnce(t *testing.T) {
	f := newMilestoneFixture(t)
	configureCatalogueWorker(t)
	receiver := newPropertyWebhookReceiver(t, f)
	var cases []propertyWorkerCase
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		cases = append(cases, preparePropertyWorkerCase(t, f, call))
	})
	require.Len(t, cases, 4)
	events := propertyWorkerEvents(t, f, cases)
	runtime := startCatalogueWorker(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	before := propertyStorageSnapshot(t, f)
	for _, value := range cases {
		assertPropertyWorkerHistory(t, f, value)
		receiver.assertChanges(t, value)
	}
	index := 0
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		value := cases[index]
		index++
		for i, input := range value.inputs {
			require.Equal(t, value.receipts[i], milestoneObject(t, call("workitem_property", input)))
		}
	})
	// Reconcile the same outbox identities after completion, as recovery would.
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	require.Equal(t, before, propertyStorageSnapshot(t, f))
	for _, value := range cases {
		receiver.assertChanges(t, value)
	}
}

func preparePropertyWorkerCase(t *testing.T, f milestoneFixture, call catalogueCall) propertyWorkerCase {
	t.Helper()
	input := preparePropertyMutationInput(t, f, call, "set_value")
	clear := maps.Clone(input)
	clear["action"], clear["request_id"] = "delete_value", uuid.NewString()
	delete(clear, "value")
	value := propertyWorkerCase{itemID: input["workitem_id"].(string), propertyID: input["property_id"].(string), inputs: []map[string]any{input, clear}}
	for _, request := range value.inputs {
		accepted := milestoneObject(t, call("workitem_property", request))
		require.Equal(t, false, accepted["replayed"])
		accepted["replayed"] = true
		before := propertyStorageSnapshot(t, f)
		require.Equal(t, accepted, milestoneObject(t, call("workitem_property", request)))
		require.Equal(t, before, propertyStorageSnapshot(t, f))
		value.receipts = append(value.receipts, accepted)
	}
	return value
}

func assertPropertyWorkerHistory(t *testing.T, f milestoneFixture, value propertyWorkerCase) {
	t.Helper()
	var raw []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(jsonb_build_array(old_value,new_value) ORDER BY created_at,id) FROM issue_activities WHERE issue_id=$1 AND old_identifier=$2 AND new_identifier=$2 AND actor_id=$3 AND field='custom_property' AND verb='updated'`, value.itemID, value.propertyID, testsupport.LabelActor).Scan(&raw))
	require.JSONEq(t, `[["","Original value"],["Original value","Changed value"],["Changed value",""]]`, string(raw))
	var stored map[string]any
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT custom_properties FROM issues WHERE id=$1`, value.itemID).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &stored))
	require.NotContains(t, stored, value.propertyID)
}
