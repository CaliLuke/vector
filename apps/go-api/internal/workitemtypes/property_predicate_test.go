package workitemtypes_test

import (
	"encoding/json"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestProjectedPredicateUsesTypedValuesAndExplicitEmptyDomain(t *testing.T) {
	const option = "10000000-0000-4000-8000-000000000001"
	for _, test := range []struct {
		name             string
		kind             property.Kind
		result           formula.Kind
		operator         string
		operand, current any
		status           string
		match, known     bool
	}{
		{"zero", property.Number, "", "exact", json.Number("0"), float64(0), "value", true, true},
		{"inclusive range", property.Number, "", "range", "0,2", float64(2), "value", true, true},
		{"strict less", property.Number, "", "lt", float64(2), float64(2), "value", false, true},
		{"less equal", property.Number, "", "lte", float64(2), float64(2), "value", true, true},
		{"greater", property.Number, "", "gt", float64(2), float64(3), "value", true, true},
		{"greater equal", property.Number, "", "gte", float64(2), float64(2), "value", true, true},
		{"empty equality unknown", property.Number, "", "exact", float64(2), nil, "empty", false, false},
		{"explicit empty", property.Number, "", "isnull", true, nil, "empty", true, true},
		{"not empty", property.Number, "", "isnull", false, float64(0), "value", true, true},
		{"inactive empty unknown", property.Number, "", "isnull", true, nil, "not_applicable", false, false},
		{"formula error empty unknown", property.Formula, formula.Number, "isnull", true, nil, "error", false, false},
		{"false", property.Boolean, "", "exact", false, false, "value", true, true},
		{"literal text", property.Text, "", "exact", " a,b ", " a,b ", "value", true, true},
		{"case insensitive substring", property.Text, "", "contains", "ABC", "abc", "value", true, true},
		{"unicode text equality", property.Text, "", "exact", "STRASSE", "Straße", "value", true, true},
		{"formula text case", property.Formula, formula.Text, "contains", "σ", "ς", "value", true, true},
		{"URL path retains case", property.URL, "", "contains", "/Path", "https://example.test/path", "value", false, true},
		{"url substring", property.URL, "", "contains", "/path", "https://example.test/path", "value", true, true},
		{"date inclusive", property.Date, "", "range", []any{"2026-01-01", "2026-01-03"}, "2026-01-03", "value", true, true},
		{"formula timestamp calendar day", property.Formula, formula.Date, "exact", "2026-01-03", "2026-01-02T23:30:00-02:00", "value", true, true},
		{"formula empty text value", property.Formula, formula.Text, "exact", "", "", "value", true, true},
		{"selection", property.Dropdown, "", "in", []any{option}, option, "value", true, true},
		{"member many", property.Member, "", "exact", option, []string{option}, "value", true, true},
		{"release", property.Release, "", "exact", option, option, "value", true, true},
		{"empty selection", property.Dropdown, "", "in", []any{}, option, "value", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, err := types.CompileProjectedPredicate(test.kind, test.result, "", "", test.operator, test.operand)
			if err != nil {
				t.Fatal(err)
			}
			match, known := p.Match(types.ProjectedValue{Kind: test.kind, ResultKind: test.result, Status: test.status, Value: test.current})
			if match != test.match || known != test.known {
				t.Fatalf("got %v/%v want %v/%v", match, known, test.match, test.known)
			}
		})
	}
}

func TestProjectedPredicateKeepsCatalogAndCascadingLevelIdentity(t *testing.T) {
	const option = "10000000-0000-4000-8000-000000000001"
	p, err := types.CompileProjectedPredicate(property.Cascading, "", "level", "catalog-a", "exact", option)
	if err != nil {
		t.Fatal(err)
	}
	value := types.ProjectedValue{Kind: property.Cascading, Status: "value", CatalogID: "catalog-a", Levels: map[string]any{"level": option}}
	if match, known := p.Match(value); !match || !known {
		t.Fatal("own catalog did not match")
	}
	value.CatalogID = "catalog-b"
	if _, known := p.Match(value); known {
		t.Fatal("identical option ID matched a different catalog")
	}
	value.CatalogID = "catalog-a"
	value.Levels = map[string]any{"other": option}
	if _, known := p.Match(value); known {
		t.Fatal("missing level became false")
	}
	empty, err := types.CompileProjectedPredicate(property.Cascading, "", "level", "catalog-a", "isnull", true)
	if err != nil {
		t.Fatal(err)
	}
	value.Levels = map[string]any{"level": nil}
	if match, known := empty.Match(value); !match || !known {
		t.Fatal("explicit absent level did not match empty")
	}
}

func TestProjectedPredicateRejectsUnsupportedOrMalformedOperands(t *testing.T) {
	for _, test := range []struct {
		kind     property.Kind
		operator string
		value    any
	}{
		{property.RichText, "contains", "text"}, {property.Number, "contains", "2"},
		{property.Number, "range", []any{float64(3), float64(1)}}, {property.Number, "exact", "NaN"},
		{property.Number, "range", "1,,2"}, {property.Number, "exact", true},
		{property.Date, "exact", "2026-02-30"}, {property.Boolean, "exact", "false"},
		{property.Boolean, "isnull", true}, {property.URL, "isnull", true},
		{property.Member, "in", "invalid"}, {property.Dropdown, "exact", []any{}},
	} {
		if _, err := types.CompileProjectedPredicate(test.kind, "", "", "", test.operator, test.value); err == nil {
			t.Fatalf("invalid operand accepted: %+v", test)
		}
	}
}
