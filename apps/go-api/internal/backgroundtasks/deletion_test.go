package backgroundtasks

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/deletion"
)

func TestDeletionRetentionRejectsInvalidConfiguration(t *testing.T) {
	for _, value := range []string{"-1", "0", "invalid", "36501"} {
		t.Setenv("HARD_DELETE_AFTER_DAYS", value)
		if _, err := HardDeleteDaysFromEnv(); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	for value, want := range map[string]int{"": 60, "1": 1, "90": 90} {
		t.Setenv("HARD_DELETE_AFTER_DAYS", value)
		if got, err := HardDeleteDaysFromEnv(); err != nil || got != want {
			t.Fatalf("%q: %d %v", value, got, err)
		}
	}
}
func TestDeletionReversePolicySkipsHiddenAndUnmanagedRelations(t *testing.T) {
	p := deletion.Policy{Models: []deletion.Model{
		{Table: "parent"},
		{Table: "child", SoftDelete: true, Relations: []deletion.Relation{{TargetTable: "parent", Column: "parent_id", OnDelete: "CASCADE", SoftReverse: true}, {TargetTable: "parent", Column: "hidden_id", OnDelete: "CASCADE", SoftReverse: false}, {TargetTable: "parent", Column: "ignored_id", OnDelete: "DO_NOTHING", SoftReverse: true}}},
		{Table: "nonsoft", Relations: []deletion.Relation{{TargetTable: "parent", Column: "cascade_id", OnDelete: "CASCADE", SoftReverse: true}, {TargetTable: "parent", Column: "nullable_id", OnDelete: "SET_NULL", SoftReverse: true}}},
	}}
	rs := deletionRelations(p, p.Models[0])
	if len(rs) != 2 || rs[0].relation.Column != "parent_id" || rs[1].relation.Column != "nullable_id" {
		t.Fatalf("unexpected soft reverse policy: %#v", rs)
	}
}
