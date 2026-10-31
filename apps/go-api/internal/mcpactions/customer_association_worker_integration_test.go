//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type customerWorkerCase struct {
	customerID, itemID, originalName, renamed string
	deletion                                  map[string]any
}

func TestMCPCustomerAssociationWorkerPreservesSnapshotsAndDeliversOnce(t *testing.T) {
	f := customerIdentityFixture(t)
	configureCatalogueWorker(t)
	var cases []customerWorkerCase
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		value := customerWorkerCase{itemID: f.item(t), originalName: uuid.NewString(), renamed: uuid.NewString()}
		customer := milestoneObject(t, call("customer", map[string]any{"action": "create", "name": value.originalName}))
		value.customerID = customer["id"].(string)
		add := map[string]any{"action": "manage_workitems", "customer_id": value.customerID, "link_ids": value.itemID, "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call("customer", add))["replayed"])
		require.Equal(t, true, milestoneObject(t, call("customer", add))["replayed"])
		milestoneObject(t, call("customer", map[string]any{"action": "update", "customer_id": value.customerID, "name": value.renamed}))
		request := milestoneObject(t, call("customer_request", map[string]any{"action": "create", "customer_id": value.customerID, "name": "Request", "workitem_ids": value.itemID}))
		catalogueNull(t, call("customer_request", map[string]any{"action": "delete", "customer_id": value.customerID, "request_id": request["id"]}))
		value.deletion = map[string]any{"action": "delete", "customer_id": value.customerID, "request_id": uuid.NewString()}
		require.Equal(t, false, milestoneObject(t, call("customer", value.deletion))["replayed"])
		cases = append(cases, value)
	})
	require.Len(t, cases, 4)
	events := catalogueWorkerEvents(t, f)
	var activities int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE field='customer'`).Scan(&activities))
	require.Zero(t, activities, "committed associations wait durably while the worker is stopped")
	runtime := startCatalogueWorker(t, f)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	for _, value := range cases {
		assertCustomerWorkerHistory(t, f, value)
	}
	before := customerStorageSnapshot(t, f)
	index := 0
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		require.Equal(t, true, milestoneObject(t, call("customer", cases[index].deletion))["replayed"])
		index++
	})
	waitCatalogueWorkerEvents(t, ctx, f, runtime, events)
	require.Equal(t, before, customerStorageSnapshot(t, f), "replaying receipts and completed outbox identities produces no additional effects")
	for _, value := range cases {
		assertCustomerWorkerHistory(t, f, value)
	}
}

func assertCustomerWorkerHistory(t *testing.T, f milestoneFixture, value customerWorkerCase) {
	t.Helper()
	var raw []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_agg(jsonb_build_array(verb,old_value,new_value,old_identifier,new_identifier,actor_id,workspace_id,project_id)) FROM issue_activities WHERE issue_id=$1 AND field='customer'`, value.itemID).Scan(&raw))
	var rows [][]any
	require.NoError(t, json.Unmarshal(raw, &rows))
	added := func(name string) []any {
		return []any{"created", nil, name, nil, value.customerID, testsupport.LabelActor, testsupport.LabelWorkspace, testsupport.LabelProject}
	}
	removed := []any{"deleted", value.renamed, nil, value.customerID, nil, testsupport.LabelActor, testsupport.LabelWorkspace, testsupport.LabelProject}
	require.ElementsMatch(t, [][]any{added(value.originalName), added(value.renamed), removed, removed}, rows, "activity retains the committed name even after rename and deletion")
	var live bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, value.itemID).Scan(&live))
	require.True(t, live)
}
