package parity

import (
	"slices"
	"testing"
)

func TestPlaneTables(t *testing.T) {
	tables, err := PlaneTables()
	if err != nil {
		t.Fatalf("PlaneTables() error = %v", err)
	}
	if got, want := len(tables), 96; got != want {
		t.Fatalf("len(PlaneTables()) = %d, want %d", got, want)
	}
	for _, required := range []string{"api_tokens", "sessions", "stickies", "users", "workspaces"} {
		if !slices.Contains(tables, required) {
			t.Errorf("PlaneTables() does not contain %q", required)
		}
	}
	first := tables[0]
	tables[0] = "changed-by-caller"
	again, err := PlaneTables()
	if err != nil {
		t.Fatalf("second PlaneTables() error = %v", err)
	}
	if again[0] != first {
		t.Fatal("PlaneTables() returned mutable shared state")
	}
}
