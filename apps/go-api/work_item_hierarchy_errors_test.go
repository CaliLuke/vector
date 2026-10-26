package systemapi

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestHierarchyWriteErrorsAreActionableWithoutDatabaseDetails(t *testing.T) {
	for _, test := range []struct{ message, field string }{
		{"The parent must belong to the same workspace", "parent_id"},
		{"A work item cannot be its own parent", "parent_id"},
		{"Parent relationships cannot contain a cycle", "parent_id"},
		{"The parent must be from the immediately higher hierarchy level", "parent_id"},
		{"The type does not match the existing child hierarchy", "type_id"},
		{"private item name from another constraint", "error"},
	} {
		t.Run(test.message, func(t *testing.T) {
			err := workItemError(&pgconn.PgError{Code: "23514", Message: test.message, Detail: "private database detail", Where: "private SQL"})
			var result *api.WorkItemError
			if !errors.As(err, &result) || result.Kind != "status_400" {
				t.Fatalf("unexpected error: %v", err)
			}
			raw, marshalErr := json.Marshal(result.Body)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			var body map[string]any
			if err = json.Unmarshal(raw, &body); err != nil {
				t.Fatal(err)
			}
			if body[test.field] == nil || strings.Contains(string(raw), "private") {
				t.Fatalf("unsafe or unusable error: %s", raw)
			}
		})
	}
}
