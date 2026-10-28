package workitemtypes

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// FormulaResult keeps absence and evaluation failure distinct from valid zero,
// false, and empty text. The enclosing read carries the common evaluation time.
type FormulaResult struct {
	Kind   formula.Kind `json:"kind"`
	Status string       `json:"status"`
	Value  any          `json:"value"`
	Error  string       `json:"error"`
}

// ItemFormulaInputs contains normalized built-in fields, not a raw issue row.
// Estimate is the selected estimate point's catalog value, never its UUID.
// A nil estimate is absent; a nonnumeric catalog label produces a visible error
// if referenced. The caller resolves it under the same authorized snapshot.
type ItemFormulaInputs struct {
	Name, Priority, StartDate, DueDate string
	Estimate                           *string
}

// EvaluateItemFormulas uses only the selected type's attached definitions and
// one effective scope/instant. It never loads fields or follows references.
// Custom inputs must be writer-canonical values; creation previews apply defaults
// before this call. Existing reads must never fill missing values with defaults.
func EvaluateItemFormulas(definitions []property.Definition, projectID, typeID string, standard ItemFormulaInputs, custom map[string]any, instant time.Time) map[string]FormulaResult {
	return CompileItemFormulas(definitions, projectID, typeID).Evaluate(standard, custom, instant)
}

// Evaluate reuses a scope's compiled expressions across rows without sharing row
// values. The caller supplies the snapshot's instant for all rows and formulas.
func (plan *ItemFormulaPlan) Evaluate(standard ItemFormulaInputs, custom map[string]any, instant time.Time) map[string]FormulaResult {
	fields := plan.fields
	values := map[string]formula.Value{}
	invalid := map[string]error{}
	builtins := map[string]any{"name": standard.Name, "priority": strings.ToUpper(standard.Priority), "start_date": standard.StartDate, "due_date": standard.DueDate}
	if standard.Estimate != nil {
		// Estimate catalog values are stored as text. This is explicit catalog
		// decoding, not expression-language string-to-number coercion.
		number, err := strconv.ParseFloat(*standard.Estimate, 64)
		if err == nil {
			builtins["estimate_point"] = number
		} else {
			builtins["estimate_point"] = *standard.Estimate
		}
	}
	for name, field := range fields {
		raw := builtins[name]
		if field.ID != name {
			raw = custom[field.ID]
		}
		value, err := itemFormulaValue(field.Kind, raw)
		values[field.ID], invalid[field.ID] = value, err
	}
	for _, input := range plan.inputs {
		field := input.field
		if input.err != nil {
			invalid[field.ID] = input.err
			continue
		}
		if !input.active {
			values[field.ID] = formula.Value{Kind: field.Kind, Empty: true}
			invalid[field.ID] = nil
		}
		if input.active && input.readOnly {
			values[field.ID], invalid[field.ID] = itemFormulaValue(formula.Text, input.content)
		}
	}
	out := map[string]FormulaResult{}
	for id, compiled := range plan.programs {
		result := FormulaResult{Kind: compiled.kind, Status: "error"}
		err := compiled.err
		if err == nil {
			result, err = evaluateItemFormula(compiled, values, invalid, instant)
		}
		if err != nil {
			result.Error = err.Error()
		}
		out[id] = result
	}
	return out
}

func evaluateItemFormula(compiled compiledItemFormula, values map[string]formula.Value, invalid map[string]error, instant time.Time) (FormulaResult, error) {
	out := FormulaResult{Kind: compiled.kind, Status: "error"}
	// Invalid typed inputs must remain errors when actually referenced. Leave a
	// mismatched kind for the engine, which preserves lazy IF branch semantics.
	inputs := make(map[string]formula.Value, len(values))
	for id, value := range values {
		if invalid[id] != nil {
			value = formula.Value{Kind: "Invalid"}
		}
		inputs[id] = value
	}
	value, err := compiled.expression.Evaluate(inputs, instant)
	if err != nil {
		return out, err
	}
	if value.Empty {
		out.Status = "empty"
		return out, nil
	}
	out.Status = "value"
	switch value.Kind {
	case formula.Number:
		out.Value = value.Number
	case formula.Boolean:
		out.Value = value.Boolean
	case formula.Text:
		out.Value = value.Text
	case formula.Date:
		format := time.RFC3339Nano
		if value.DateOnly {
			format = time.DateOnly
		}
		out.Value = value.Date.UTC().Format(format)
	}
	return out, nil
}

func itemFormulaValue(kind formula.Kind, raw any) (formula.Value, error) {
	// Boolean has no third state in the value contract; an unset field is false.
	if kind == formula.Boolean && raw == nil {
		raw = false
	}
	out := formula.Value{Kind: kind, Empty: raw == nil}
	if out.Empty {
		return out, nil
	}
	switch kind {
	case formula.Text:
		value, ok := raw.(string)
		if ok {
			out.Text = value
			out.Empty = strings.TrimSpace(value) == ""
			return out, nil
		}
	case formula.Boolean:
		value, ok := raw.(bool)
		if ok {
			out.Boolean = value
			return out, nil
		}
	case formula.Number:
		switch value := raw.(type) {
		case float64:
			out.Number = value
			return out, nil
		case json.Number:
			number, err := value.Float64()
			out.Number = number
			return out, err
		}
	case formula.Date:
		value, ok := raw.(string)
		if ok {
			if value == "" {
				out.Empty = true
				return out, nil
			}
			date, err := property.ParseDate(value)
			if err == nil {
				out.Date = date
				out.DateOnly = len(value) == len(time.DateOnly)
				return out, nil
			}
		}
	}
	return out, fmt.Errorf("invalid %s input", kind)
}
