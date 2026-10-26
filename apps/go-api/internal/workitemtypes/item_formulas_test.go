package workitemtypes_test

import (
	"reflect"
	"testing"
	"time"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestItemFormulasResolveEffectiveInputsAndBuiltinValues(t *testing.T) {
	definitions := []property.Definition{
		{ID: "number", Name: "effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		{ID: "flag", Name: "verified", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		{ID: "instructions", Name: "instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Configured guidance"}},
		{ID: "inactive", Name: "inactive", Kind: property.Number, Configuration: property.Configuration{Active: true}, Contexts: []property.Context{{ID: "context", Projects: []string{"project"}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}},
	}
	expressions := map[string]struct {
		expression, kind string
		value            any
		status           string
	}{
		"zero":            {"{{effort}} * 2", "Number", float64(0), "value"},
		"false":           {"{{verified}}", "Boolean", false, "value"},
		"configured":      {"{{instructions}}", "Text", "Configured guidance", "value"},
		"inactive_result": {"{{inactive}} + 1", "Number", nil, "empty"},
		"priority_result": {"{{priority}}", "Text", "HIGH", "value"},
		"date_result":     {"{{due_date}} - {{start_date}}", "Number", float64(2), "value"},
		"estimate_result": {"{{estimate_point}} + 1", "Number", nil, "empty"},
		"today_result":    {"TODAY()", "Date", "2026-10-24", "value"},
		"now_result":      {"NOW()", "Date", "2026-10-24T23:59:30Z", "value"},
	}
	for id, test := range expressions {
		definitions = append(definitions, property.Definition{ID: id, Name: id, Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: test.expression, Result: formula.Kind(test.kind)}})
	}
	instant := time.Date(2026, 10, 24, 23, 59, 30, 0, time.UTC)
	got := types.EvaluateItemFormulas(definitions, "project", "type", types.ItemFormulaInputs{Priority: "high", DueDate: "2026-10-26", StartDate: "2026-10-24"}, map[string]any{"number": float64(0), "flag": false, "instructions": "Wrong stored text", "inactive": float64(8)}, instant)
	for id, test := range expressions {
		result := got[id]
		if result.Status != test.status || !reflect.DeepEqual(result.Value, test.value) || result.Error != "" || string(result.Kind) != test.kind {
			t.Errorf("%s: %+v", id, result)
		}
	}
}

func TestItemFormulaRuntimeErrorsRemainLazyAndNeverUseOtherTypeFields(t *testing.T) {
	definitions := []property.Definition{
		{ID: "number", Name: "effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		formulaProjectionDefinition("lazy", "IF(false, {{effort}}, 3)", "Number"),
		formulaProjectionDefinition("bad", "{{effort}} + 1", "Number"),
		formulaProjectionDefinition("missing", "{{foreign}} + 1", "Number"),
		formulaProjectionDefinition("divide", "1 / 0", "Number"),
	}
	got := types.EvaluateItemFormulas(definitions, "project", "type", types.ItemFormulaInputs{}, map[string]any{"number": "wrong kind", "foreign": float64(7)}, time.Now())
	if result := got["lazy"]; result.Status != "value" || result.Value != float64(3) {
		t.Fatalf("invalid unchosen branch executed: %+v", result)
	}
	for _, id := range []string{"bad", "missing", "divide"} {
		if result := got[id]; result.Status != "error" || result.Value != nil || result.Error == "" {
			t.Errorf("%s: %+v", id, result)
		}
	}
}

func TestItemFormulaEstimatesUseCatalogValuesAndRejectNonNumericLabels(t *testing.T) {
	definitions := []property.Definition{{ID: "calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{estimate_point}} * 2", Result: "Number"}}}
	for _, test := range []struct {
		estimate *string
		status   string
		value    any
	}{
		{nil, "empty", nil}, {new("2.5"), "value", float64(5)}, {new("XL"), "error", nil},
	} {
		result := types.EvaluateItemFormulas(definitions, "project", "type", types.ItemFormulaInputs{Estimate: test.estimate}, nil, time.Now())["calculation"]
		if result.Status != test.status || !reflect.DeepEqual(result.Value, test.value) {
			t.Fatalf("estimate: %+v", result)
		}
	}
}

func formulaProjectionDefinition(id, expression string, result formula.Kind) property.Definition {
	return property.Definition{ID: id, Name: id, Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: expression, Result: result}}
}
