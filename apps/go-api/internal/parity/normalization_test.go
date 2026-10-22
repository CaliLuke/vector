package parity

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestCanonicalizeOnlyDeclaredFieldsAndValues(t *testing.T) {
	rules := Normalizations{
		VolatileFields: []VolatileFieldRule{{Table: "stickies", Column: "updated_at"}},
		VolatileValues: []VolatileValueRule{{
			Table:       "stickies",
			Column:      "id",
			Values:      []json.RawMessage{json.RawMessage(`"source-id"`), json.RawMessage(`"target-id"`)},
			Replacement: json.RawMessage(`"<sticky-id>"`),
		}},
	}
	if err := rules.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	got, err := rules.canonicalize(
		"stickies",
		json.RawMessage(`{"id":"source-id","name":"keep-me","updated_at":"2026-08-13T00:00:00Z"}`),
	)
	if err != nil {
		t.Fatalf("canonicalize() error = %v", err)
	}
	want := `{"id":"\u003csticky-id\u003e","name":"keep-me","updated_at":"\u003cvolatile\u003e"}`
	if string(got) != want {
		t.Fatalf("canonicalize() = %s, want %s", got, want)
	}
}

func TestCanonicalizeDoesNotWalkNestedValues(t *testing.T) {
	rules := Normalizations{VolatileFields: []VolatileFieldRule{{Table: "stickies", Column: "updated_at"}}}
	if err := rules.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	got, err := rules.canonicalize(
		"stickies",
		json.RawMessage(`{"description":{"updated_at":"meaningful"},"updated_at":"volatile"}`),
	)
	if err != nil {
		t.Fatalf("canonicalize() error = %v", err)
	}
	want := `{"description":{"updated_at":"meaningful"},"updated_at":"\u003cvolatile\u003e"}`
	if string(got) != want {
		t.Fatalf("canonicalize() = %s, want %s", got, want)
	}
}

func TestValidateColumnsRejectsRuleTypo(t *testing.T) {
	rules := Normalizations{VolatileFields: []VolatileFieldRule{{Table: "stickies", Column: "missing"}}}
	err := rules.validateColumns([]TableDump{{Name: "stickies", Columns: []Column{{Name: "id"}}}})
	if err == nil {
		t.Fatal("validateColumns() error = nil, want unknown-column error")
	}
}

func TestDeclaredNormalizationsOptionNameIdentifiesReviewedRules(t *testing.T) {
	rules := Normalizations{
		VolatileFields: []VolatileFieldRule{{Table: "users", Column: "updated_at"}},
		VolatileValues: []VolatileValueRule{{
			Table: "stickies", Column: "id", Values: []json.RawMessage{json.RawMessage(`"one"`)}, Replacement: json.RawMessage(`"<id>"`),
		}},
	}
	if err := rules.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	name := normalizationOptionName(rules)
	for _, identity := range []string{"Field_users_updated_at", "Values_stickies_id"} {
		if !strings.Contains(name, identity) {
			t.Fatalf("normalizationOptionName() = %q, want rule identity %q", name, identity)
		}
	}
	changed := rules
	changed.VolatileValues = slices.Clone(rules.VolatileValues)
	changed.VolatileValues[0].Replacement = json.RawMessage(`"<different-id>"`)
	if normalizationOptionName(changed) == name {
		t.Fatal("normalizationOptionName() did not identify exact reviewed rule-set change")
	}
}
