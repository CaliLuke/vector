package workitemtypes_test

import (
	"reflect"
	"testing"
	"time"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestProjectionDistinguishesApplicabilityEmptyAndZeroWithoutReadDefaults(t *testing.T) {
	definitions := []property.Definition{
		{ID: "number", Kind: property.Number, Configuration: property.Configuration{Active: true, Required: true, Default: float64(12)}},
		{ID: "boolean", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		{ID: "fixed", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Current instructions"}},
		{ID: "inactive", Kind: property.Number, Configuration: property.Configuration{Active: false}},
		{ID: "member", Kind: property.Member, Configuration: property.Configuration{Active: true, Required: true}},
		{ID: "release", Kind: property.Release, Configuration: property.Configuration{Active: true, Multiple: true}},
	}
	plan, err := types.CompileItemProjection(definitions, "project", "type")
	if err != nil {
		t.Fatal(err)
	}
	retained := "12345678-1234-1234-1234-123456789012"
	out := plan.Project(types.ItemFormulaInputs{}, map[string]any{"fixed": "Obsolete text", "inactive": float64(7), "member": retained, "release": []any{retained}, "detached": "private unused value"}, time.Now(), false)
	for id, expected := range map[string]struct {
		status string
		value  any
	}{
		"number": {"empty", nil}, "boolean": {"value", false}, "fixed": {"value", "Current instructions"},
		"inactive": {"not_applicable", nil}, "member": {"value", retained}, "release": {"value", []string{retained}},
	} {
		got := out.Values[id]
		if got.Status != expected.status || !reflect.DeepEqual(got.Value, expected.value) || got.Error != "" {
			t.Errorf("%s: %+v", id, got)
		}
	}
	if _, exists := out.Values["detached"]; exists {
		t.Fatal("unattached saved value exposed")
	}
	zero := plan.Project(types.ItemFormulaInputs{}, map[string]any{"number": float64(0)}, time.Now(), false).Values["number"]
	if zero.Status != "value" || zero.Value != float64(0) {
		t.Fatalf("zero: %+v", zero)
	}
	malformed := plan.Project(types.ItemFormulaInputs{}, map[string]any{"number": "wrong"}, time.Now(), false).Values["number"]
	if malformed.Status != "error" || malformed.Error == "" || malformed.Value != nil {
		t.Fatalf("malformed: %+v", malformed)
	}
}

func TestProjectionKeepsEffectiveContextSeparateFromOptionOwnership(t *testing.T) {
	option := "12345678-1234-1234-1234-123456789012"
	def := property.Definition{ID: "cascade", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: "level", Name: "Stable level"}}, Options: []property.Option{{ID: option, Name: "Same label"}}}}
	def.Contexts = []property.Context{
		{ID: "behavior", Projects: []string{"project-a"}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: true}},
		{ID: "catalog", Projects: []string{"project-b"}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: def.Configuration.Options}},
	}
	for _, tc := range []struct{ project, context, catalog string }{{"project-a", "behavior", ""}, {"project-b", "catalog", "catalog"}} {
		plan, err := types.CompileItemProjection([]property.Definition{def}, tc.project, "type")
		if err != nil {
			t.Fatal(err)
		}
		input := []string{option}
		out := plan.Project(types.ItemFormulaInputs{}, map[string]any{"cascade": input}, time.Now(), false).Values["cascade"]
		if out.ContextID != tc.context || out.CatalogID != tc.catalog || out.Levels["level"] != option {
			t.Fatalf("%s: %+v", tc.project, out)
		}
		input[0] = "changed by caller"
		if out.Value.([]string)[0] != option {
			t.Fatal("projection aliases row input")
		}
		draft := plan.Project(types.ItemFormulaInputs{}, map[string]any{"cascade": []string{option}}, time.Now(), true).Values["cascade"]
		if draft.Status != "not_applicable" || draft.Value != nil {
			t.Fatalf("draft cascade: %+v", draft)
		}
	}
}

func TestCompiledProjectionSharesTimeAndPreservesFormulaErrorsAcrossRows(t *testing.T) {
	definitions := []property.Definition{
		{ID: "number", Name: "effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		formulaProjectionDefinition("double", "{{effort}} * 2", "Number"),
		formulaProjectionDefinition("lazy", "IF(false, {{effort}}, 3)", "Number"),
		formulaProjectionDefinition("now", "NOW()", "Date"),
		formulaProjectionDefinition("today", "TODAY()", "Date"),
	}
	plan, err := types.CompileItemProjection(definitions, "project", "type")
	if err != nil {
		t.Fatal(err)
	}
	instant := time.Date(2026, 10, 25, 23, 59, 59, 0, time.UTC)
	for _, row := range []map[string]any{{"number": float64(4)}, {"number": "bad"}} {
		got := plan.Project(types.ItemFormulaInputs{}, row, instant, false)
		if got.Computed["now"].Value != "2026-10-25T23:59:59Z" || got.Computed["today"].Value != "2026-10-25" || got.EvaluatedAt != instant {
			t.Fatalf("inconsistent instant: %+v", got)
		}
		if got.Values["lazy"].Value != float64(3) || got.Values["lazy"].ResultKind != "Number" {
			t.Fatalf("lazy input failed: %+v", got.Values["lazy"])
		}
		if row["number"] == "bad" && got.Values["double"].Status != "error" {
			t.Fatalf("error hidden: %+v", got.Values["double"])
		}
		if row["number"] != "bad" && got.Values["double"].Value != float64(8) {
			t.Fatalf("wrong value: %+v", got.Values["double"])
		}
	}
	definitions[1].Configuration.Expression = "999"
	if got := plan.Project(types.ItemFormulaInputs{}, map[string]any{"number": float64(2)}, instant, false).Values["double"].Value; got != float64(4) {
		t.Fatalf("compiled scope changed: %v", got)
	}
}

func TestProjectionNormalizesEveryStoredKindWithoutReferenceLookups(t *testing.T) {
	selected := "12345678-1234-1234-1234-123456789012"
	for _, tc := range []struct {
		kind          property.Kind
		config        property.Configuration
		raw, expected any
	}{
		{property.Text, property.Configuration{Format: "Single line"}, "Text", "Text"},
		{property.URL, property.Configuration{}, "https://example.com/path", "https://example.com/path"},
		{property.Date, property.Configuration{}, "2026-10-25", "2026-10-25"},
		{property.RichText, property.Configuration{}, "<p>Saved text</p>", "<p>Saved text</p>"},
		{property.Dropdown, property.Configuration{}, selected, selected},
		{property.Member, property.Configuration{}, selected, selected},
		{property.Release, property.Configuration{Multiple: true}, []any{selected}, []string{selected}},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			tc.config.Active, tc.config.Required = true, true
			plan, err := types.CompileItemProjection([]property.Definition{{ID: "field", Kind: tc.kind, Configuration: tc.config}}, "project", "type")
			if err != nil {
				t.Fatal(err)
			}
			got := plan.Project(types.ItemFormulaInputs{}, map[string]any{"field": tc.raw}, time.Now(), false).Values["field"]
			if got.Status != "value" || !reflect.DeepEqual(got.Value, tc.expected) {
				t.Fatalf("%+v", got)
			}
			empty := plan.Project(types.ItemFormulaInputs{}, nil, time.Now(), false).Values["field"]
			if empty.Status != "empty" || empty.Value != nil {
				t.Fatalf("missing required value: %+v", empty)
			}
		})
	}
}

func TestProjectionRejectsOverlongCascadeWithoutChangingSavedPath(t *testing.T) {
	path := []string{"12345678-1234-1234-1234-123456789012", "12345678-1234-1234-1234-123456789013"}
	definition := property.Definition{ID: "cascade", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: "level", Name: "Level"}}}}
	plan, err := types.CompileItemProjection([]property.Definition{definition}, "project", "type")
	if err != nil {
		t.Fatal(err)
	}
	saved := map[string]any{"cascade": path}
	value := plan.Project(types.ItemFormulaInputs{}, saved, time.Now(), false).Values["cascade"]
	if value.Status != "error" || value.Error == "" || value.Value != nil || len(value.Levels) != 0 {
		t.Fatalf("malformed path entered comparison domain: %+v", value)
	}
	if !reflect.DeepEqual(saved["cascade"], path) || len(path) != 2 {
		t.Fatal("saved data changed")
	}
}

func TestProjectionCarriesEffectiveDatePresentation(t *testing.T) {
	definition := property.Definition{ID: "date", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "Jan 15, 2025"}, Contexts: []property.Context{
		{ID: "format-context", Projects: []string{"regional"}, Overrides: []string{"format"}, Configuration: property.Configuration{Format: "15/01/2025"}},
	}}
	for _, scope := range []struct{ project, format, context string }{{"default", "Jan 15, 2025", ""}, {"regional", "15/01/2025", "format-context"}} {
		plan, err := types.CompileItemProjection([]property.Definition{definition}, scope.project, "type")
		if err != nil {
			t.Fatal(err)
		}
		value := plan.Project(types.ItemFormulaInputs{}, map[string]any{"date": "2026-10-25"}, time.Now(), false).Values["date"]
		if value.Value != "2026-10-25" || value.Format != scope.format || value.ContextID != scope.context {
			t.Fatalf("date presentation: %+v", value)
		}
	}
}
