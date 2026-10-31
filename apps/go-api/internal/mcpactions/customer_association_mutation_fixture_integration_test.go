//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// One mutation adds a new association and removes a populated association, so
// rollback and contention exercise both transitions and their durable activities.
func customerAssociationMutationCase(t *testing.T) (milestoneFixture, *customers.Repository, customers.Command) {
	t.Helper()
	f := customerIdentityFixture(t)
	repository := customers.NewRepository(f.pool)
	command := customerIdentityCommand()
	parent, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	previous, next := uuid.MustParse(f.item(t)), uuid.MustParse(f.item(t))
	command.Action, command.CustomerID = "manage_workitems", uuid.MustParse(parent.ID)
	command.Fields = map[string]any{}
	command.Add = []uuid.UUID{previous}
	seeded, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	command.Add, command.Remove = []uuid.UUID{next}, []uuid.UUID{previous}
	command.ExpectedRevision = &seeded.Revision
	key := uuid.New()
	command.RetryID = &key
	return f, repository, command
}

func customerAssociationWriteInput(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
	t.Helper()
	parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}))
	previous, next := f.item(t), f.item(t)
	input := map[string]any{"action": "manage_workitems", "workspace_slug": "workspace-one", "customer_id": parent["id"], "link_ids": previous}
	catalogueNull(t, call("customer", input))
	input["link_ids"], input["unlink_ids"], input["request_id"] = next, previous, uuid.NewString()
	return input
}
