package workitems

import (
	"slices"
	"testing"
)

func TestUnassignedPlanningIssuesPreservesTargetOrderAndRepeatedIDs(t *testing.T) {
	targets := []map[string]any{{"id": "second"}, {"id": "assigned"}, {"id": "first"}, {"id": "second"}}
	existing := []map[string]any{{"issue_id": "assigned"}, {"issue_id": "unrequested"}, {"issue_id": "assigned"}}
	got := unassignedPlanningIssues(targets, existing)
	if !slices.Equal(got, []string{"second", "first", "second"}) {
		t.Fatalf("selected = %v, want unassigned targets in their original order", got)
	}
}

func TestUnassignedPlanningIssuesReturnsEmptyCollectionWhenAllAreAssigned(t *testing.T) {
	got := unassignedPlanningIssues([]map[string]any{{"id": "assigned"}}, []map[string]any{{"issue_id": "assigned"}})
	if got == nil || len(got) != 0 {
		t.Fatalf("selected = %#v, want an empty collection", got)
	}
}
