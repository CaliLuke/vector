//go:build integration

package testsupport

import (
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestEstimateDeletionScenarioKeepsIssueReferencesAndDeletesInactiveSystemFirst(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, EstimateReadSchema)
	CheckEstimateDeletionReads(t, ctx, connection, []EstimateReadCase{{
		Name: "remaining system weights",
		SQL:  `SELECT COALESCE(sum(CASE WHEN id='00000000-0000-4000-8000-000000000101' THEN 3 ELSE 8 END),0)::float8 FROM estimates WHERE id IN ('00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000102') AND deleted_at IS NULL`,
		Want: [3]float64{11, 3, 0},
	}})
	var references int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM issues WHERE estimate_point_id IS NOT NULL`).Scan(&references); err != nil {
		t.Fatal(err)
	}
	if references != 8 {
		t.Fatalf("soft deletions changed issue references: got %d, want 8", references)
	}
}

func TestLabelReadAssertionsPreserveObjectsAndIgnoreSelectionOrder(t *testing.T) {
	object := LabelReadObject(t, []byte(`{"id":"issue-1","labels":["label-b","label-a"]}`))
	if object["id"] != "issue-1" {
		t.Fatalf("label projection lost its issue identity: %#v", object)
	}
	RequireLabelIDs(t, object["labels"], "label-a", "label-b")
	RequireLabelIDs(t, []string{})
}
