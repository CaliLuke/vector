//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Option definitions and populated customer values make rollback exercise the
// complete property aggregate, including changes made before a rejected commit.
func customerPropertyMutationCase(t *testing.T, action string) (milestoneFixture, *customers.Repository, customers.Command) {
	t.Helper()
	f := customerIdentityFixture(t)
	repository := customers.NewRepository(f.pool)
	command := customerIdentityCommand()
	command.Tool = "customer_property"
	command.Fields = map[string]any{"display_name": "Property " + uuid.NewString(), "property_type": "OPTION"}
	command.Options = []map[string]any{{"name": "First", "is_default": true}, {"name": "Second"}}
	if action != "create" {
		property, err := repository.Mutate(t.Context(), command)
		require.NoError(t, err)
		command.ID = uuid.MustParse(property.ID)
		command.Action = action
		command.Fields, command.Options = map[string]any{}, nil
		if action == "update" {
			command.Fields["description"] = "Updated property"
			command.Options = []map[string]any{{"name": "Third"}}
		}
	}
	parent, err := repository.Mutate(t.Context(), customerIdentityCommand())
	require.NoError(t, err)
	if action == "set_values" {
		var optionID string
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id FROM customer_property_options WHERE property_id=$1 AND name='Second'`, command.ID).Scan(&optionID))
		command.Values = map[string][]string{command.ID.String(): {optionID}}
		command.ID, command.CustomerID = uuid.Nil, uuid.MustParse(parent.ID)
	}
	command.ExpectedRevision = &parent.Revision
	key := uuid.New()
	command.RetryID = &key
	return f, repository, command
}

func customerPropertyWriteInput(t *testing.T, call catalogueCall, action string) map[string]any {
	t.Helper()
	input := map[string]any{
		"action": "create", "workspace_slug": "workspace-one", "display_name": "Property " + uuid.NewString(),
		"property_type": "OPTION", "options": `[{"name":"First","is_default":true},{"name":"Second"}]`,
	}
	if action != "create" {
		property := milestoneObject(t, call("customer_property", input))
		parent := milestoneObject(t, call("customer", map[string]any{"action": "create", "workspace_slug": "workspace-one", "name": uuid.NewString()}))
		input = map[string]any{"action": action, "workspace_slug": "workspace-one", "property_id": property["id"]}
		switch action {
		case "update":
			input["description"], input["options"] = "Updated property", `[{"name":"Third"}]`
		case "set_values":
			options := property["options"].([]any)
			encoded, err := json.Marshal(map[string][]any{property["id"].(string): {options[1].(map[string]any)["id"]}})
			require.NoError(t, err)
			delete(input, "property_id")
			input["customer_id"], input["values"] = parent["id"], string(encoded)
		}
	}
	input["request_id"] = uuid.NewString()
	return input
}
