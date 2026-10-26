package workitemtypes

import (
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// ItemFormulaPlan is immutable after compilation. It contains only one selected
// type's attached fields and resolved scope, never values from a particular item.
// Keep it inside the authorized request snapshot, not in a cross-request cache.
type ItemFormulaPlan struct {
	fields   map[string]formula.Field
	inputs   []itemFormulaInput
	programs map[string]compiledItemFormula
}
type itemFormulaInput struct {
	field            formula.Field
	active, readOnly bool
	content          string
	err              error
}
type compiledItemFormula struct {
	expression *formula.Expression
	kind       formula.Kind
	err        error
}

// CompileItemFormulas resolves scope and compiles every active formula once.
// Compilation errors stay individual visible results, as with runtime errors.
func CompileItemFormulas(definitions []property.Definition, projectID, typeID string) *ItemFormulaPlan {
	plan := &ItemFormulaPlan{fields: formulaFields(formulaRecords(definitions)), programs: map[string]compiledItemFormula{}}
	for _, def := range definitions {
		config, _, err := property.Effective(def, projectID, typeID)
		if field, exists := plan.fields[def.Name]; exists {
			plan.inputs = append(plan.inputs, itemFormulaInput{field: field, active: config.Active, readOnly: def.Kind == property.Text && config.Format == "Read-only", content: config.Content, err: err})
		}
		if def.Kind != property.Formula || err == nil && !config.Active {
			continue
		}
		compiled := compiledItemFormula{kind: config.Result, err: err}
		if err == nil {
			compiled.expression, compiled.err = formula.Compile(config.Expression, config.Result, plan.fields)
		}
		plan.programs[def.ID] = compiled
	}
	return plan
}
