package customers

import (
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
)

func TestMutateRejectsInvalidCustomerCommandsBeforeDatabaseAccess(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	for _, test := range []struct {
		name  string
		input Command
	}{
		{"request without customer", Command{Query: Query{Tool: "customer_request", Action: "create"}, Fields: map[string]any{"name": "Export"}}},
		{"options on customer", Command{Query: Query{Tool: "customer", Action: "create"}, Fields: map[string]any{"name": "Buyer"}, Options: []map[string]any{{"name": "Retail"}}}},
		{"unknown field", Command{Query: Query{Tool: "customer", Action: "create"}, Fields: map[string]any{"name": "Buyer", "admin": true}}},
		{"negative employee count", Command{Query: Query{Tool: "customer", Action: "create"}, Fields: map[string]any{"name": "Buyer", "employees": int64(-1)}}},
		{"noninteger employee count", Command{Query: Query{Tool: "customer", Action: "create"}, Fields: map[string]any{"name": "Buyer", "employees": 1.5}}},
		{"blank name", Command{Query: Query{Tool: "customer", Action: "create"}, Fields: map[string]any{"name": " \t"}}},
		{"nil association", Command{Query: Query{Tool: "customer", Action: "manage_workitems", CustomerID: id}, Add: []uuid.UUID{uuid.Nil}}},
		{"duplicate canonical property", Command{Query: Query{Tool: "customer_property", Action: "set_values", CustomerID: id}, Values: map[string][]string{id.String(): {"retail"}, strings.ToUpper(id.String()): {"enterprise"}}}},
		{"non-JSON option", Command{Query: Query{Tool: "customer_property", Action: "create"}, Fields: map[string]any{"display_name": "Segment", "property_type": "TEXT"}, Options: []map[string]any{{"value": make(chan int)}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRepository(nil).Mutate(t.Context(), test.input)
			if !errors.Is(err, projectmutations.ErrInvalid) {
				t.Fatalf("Mutate error = %v, want invalid command before receipt/database access", err)
			}
		})
	}
}
