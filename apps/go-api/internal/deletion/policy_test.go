package deletion

import "testing"

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
