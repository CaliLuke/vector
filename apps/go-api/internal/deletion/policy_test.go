package deletion

import (
	"testing"

	"github.com/google/uuid"
)

func TestCapturedPolicyIsCompleteAndUnambiguous(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	models := map[string]bool{}
	relations, soft := 0, 0
	for _, m := range p.Models {
		if models[m.Table] {
			t.Fatalf("duplicate model table %s", m.Table)
		}
		models[m.Table] = true
		if m.SoftDelete {
			soft++
		}
		seen := map[string]bool{}
		for _, r := range m.Relations {
			relations++
			if seen[r.Column] {
				t.Fatalf("duplicate FK %s.%s", m.Table, r.Column)
			}
			seen[r.Column] = true
			if r.OnDelete != "CASCADE" && r.OnDelete != "SET_NULL" && r.OnDelete != "DO_NOTHING" {
				t.Fatalf("unsupported policy %s", r.OnDelete)
			}
		}
	}
	if len(p.Models) != 109 || soft != 92 || relations != 474 {
		t.Fatalf("source registry counts models=%d soft=%d relations=%d", len(p.Models), soft, relations)
	}
	for _, m := range p.Models {
		for _, r := range m.Relations {
			if !models[r.TargetTable] {
				t.Fatalf("unknown target %s", r.TargetTable)
			}
		}
	}
	if _, err := p.Lookup("db", "workspace"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Lookup("db", "workspaces; DROP TABLE users"); err == nil {
		t.Fatal("accepted unknown model")
	}
}

func TestModelIdentifiersMatchTheirDatabaseKeyType(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct {
		name, keyType, input string
		want                 any
	}{
		{"default UUID", "", id, uuid.MustParse(id)},
		{"explicit UUID", "uuid", id, uuid.MustParse(id)},
		{"integer", "bigint", "42", int64(42)},
		{"largest integer", "bigint", "9223372036854775807", int64(9223372036854775807)},
		{"UUID rejects integer", "uuid", "42", nil},
		{"integer rejects UUID", "bigint", id, nil},
		{"integer overflow", "bigint", "9223372036854775808", nil},
		{"invalid integer", "bigint", "42;DELETE FROM projects", nil},
		{"unsupported model key", "text", "anything", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (Model{PKType: tc.keyType}).ParseID(tc.input)
			if tc.want == nil {
				if err == nil {
					t.Fatalf("accepted incompatible model identifier: %v", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("identifier=%v (%T), want %v (%T), error=%v", got, got, tc.want, tc.want, err)
			}
		})
	}
}
