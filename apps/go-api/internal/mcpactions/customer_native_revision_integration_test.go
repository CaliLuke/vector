//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// Direct SQL writers must invalidate API revisions even when they do not use
// the customer repository. Each case starts with its own populated catalogue.
func TestCustomerRevisionsIncludeAllCatalogueRowWriters(t *testing.T) {
	for _, write := range []struct{ name, statement string }{
		{"customer/insert", `INSERT INTO customers(workspace_id,name) VALUES(@workspace,'Native customer')`},
		{"customer/update", `UPDATE customers SET description_html='<p>Native change</p>' WHERE id=@customer`},
		{"customer/delete", `DELETE FROM customers WHERE id=@customer`},
		{"request/insert", `INSERT INTO customer_requests(workspace_id,customer_id,name) VALUES(@workspace,@customer,'Native request')`},
		{"request/update", `UPDATE customer_requests SET name='Native request' WHERE id=@request`},
		{"request/delete", `DELETE FROM customer_requests WHERE id=@request`},
		{"association/insert", `INSERT INTO customer_work_items(workspace_id,customer_id,issue_id,project_id) VALUES(@workspace,@customer,@item,@project)`},
		{"association/update", `UPDATE customer_work_items SET request_id=NULL WHERE request_id=@request`},
		{"association/delete", `DELETE FROM customer_work_items WHERE request_id=@request`},
		{"property/insert", `INSERT INTO customer_properties(workspace_id,name,display_name,property_type) VALUES(@workspace,'native','Native','TEXT')`},
		{"property/update", `UPDATE customer_properties SET description='Native property' WHERE id=@property`},
		{"property/delete", `DELETE FROM customer_properties WHERE id=@property`},
		{"option/insert", `INSERT INTO customer_property_options(workspace_id,property_id,name) VALUES(@workspace,@property,'Native option')`},
		{"option/update", `UPDATE customer_property_options SET description='Native option' WHERE property_id=@property AND name='Second'`},
		{"option/delete", `DELETE FROM customer_property_options WHERE property_id=@property AND name='Second'`},
		{"value/insert", `INSERT INTO customer_property_values(workspace_id,property_id,customer_id,value_strings) VALUES(@workspace,@property,@empty_customer,'[]')`},
		{"value/update", `UPDATE customer_property_values SET value_strings='[]' WHERE property_id=@property AND customer_id=@customer`},
		{"value/delete", `DELETE FROM customer_property_values WHERE property_id=@property AND customer_id=@customer`},
	} {
		t.Run(write.name, func(t *testing.T) {
			f, repository, command, arguments := customerNativeRevisionFixture(t)
			oldRevision := customerCatalogueRevision(t, repository, command.Query)
			command.ExpectedRevision = &oldRevision
			changed, err := f.pool.Exec(t.Context(), write.statement, arguments)
			require.NoError(t, err)
			require.EqualValues(t, 1, changed.RowsAffected())
			afterNativeWrite := customerStorageSnapshot(t, f)
			rejected, err := repository.Mutate(t.Context(), command)
			require.ErrorIs(t, err, projectmutations.ErrRevision)
			require.Equal(t, projectmutations.Result{}, rejected)
			require.Equal(t, afterNativeWrite, customerStorageSnapshot(t, f), "stale API writes must preserve the native result and leave no receipt or audit event")
			currentRevision := customerCatalogueRevision(t, repository, command.Query)
			require.NotEqual(t, oldRevision, currentRevision)
			command.ExpectedRevision = &currentRevision
			accepted, err := repository.Mutate(t.Context(), command)
			require.NoError(t, err)
			require.NotNil(t, accepted.Replayed)
			require.False(t, *accepted.Replayed, "rejected revision guards do not consume retry keys")
			stored := customerStorageSnapshot(t, f)
			replayed, err := repository.Mutate(t.Context(), command)
			require.NoError(t, err)
			require.NotNil(t, replayed.Replayed)
			require.True(t, *replayed.Replayed)
			accepted.Replayed = replayed.Replayed
			require.Equal(t, accepted, replayed)
			require.Equal(t, stored, customerStorageSnapshot(t, f))
		})
	}
}

func customerCatalogueRevision(t *testing.T, repository *customers.Repository, query customers.Query) string {
	t.Helper()
	query.Action, query.IncludeRevision = "list", true
	query.ID, query.CustomerID, query.RequestID = uuid.Nil, uuid.Nil, uuid.Nil
	raw, err := repository.Read(t.Context(), query)
	require.NoError(t, err)
	var page struct{ Revision string }
	require.NoError(t, json.Unmarshal(raw, &page))
	require.NotEmpty(t, page.Revision)
	return page.Revision
}

func customerNativeRevisionFixture(t *testing.T) (milestoneFixture, *customers.Repository, customers.Command, pgx.NamedArgs) {
	t.Helper()
	f, repository, values := customerPropertyMutationCase(t, "set_values")
	arguments := pgx.NamedArgs{
		"workspace": values.WorkspaceID, "customer": values.CustomerID,
		"item": uuid.MustParse(f.item(t)), "project": uuid.MustParse(testsupport.LabelProject),
		"empty_customer": uuid.New(),
	}
	for propertyID := range values.Values {
		arguments["property"] = uuid.MustParse(propertyID)
	}
	request := customers.Command{
		Query: customers.Query{Tool: "customer_request", Action: "create", WorkspaceID: values.WorkspaceID,
			WorkspaceSlug: values.WorkspaceSlug, ActorID: values.ActorID, CustomerID: values.CustomerID},
		Fields: map[string]any{"name": "Existing request"}, Add: []uuid.UUID{arguments["item"].(uuid.UUID)},
	}
	created, err := repository.Mutate(t.Context(), request)
	require.NoError(t, err)
	arguments["request"] = uuid.MustParse(created.ID)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO customers(id,workspace_id,name) VALUES(@empty_customer,@workspace,'No default values')`, arguments)
	require.NoError(t, err)
	command := customerIdentityCommand()
	command.Tool = "customer_property"
	command.Fields = map[string]any{"display_name": "Guarded " + uuid.NewString(), "property_type": "TEXT"}
	key := uuid.New()
	command.RetryID = &key
	return f, repository, command, arguments
}
