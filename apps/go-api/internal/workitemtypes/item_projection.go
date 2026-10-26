package workitemtypes

import (
	"slices"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// ProjectedValue describes the comparison domain as well as the value. Inactive
// or unattached properties must never match negative or empty predicates.
type ProjectedValue struct {
	Format     string         `json:"format,omitempty"`
	ResultKind formula.Kind   `json:"result_kind,omitempty"`
	Kind       property.Kind  `json:"kind"`
	ContextID  string         `json:"context_id"`
	CatalogID  string         `json:"catalog_id"`
	Status     string         `json:"status"`
	Value      any            `json:"value"`
	Error      string         `json:"error"`
	Levels     map[string]any `json:"levels,omitempty"`
}

type ItemProjection struct {
	Values      map[string]ProjectedValue
	Computed    map[string]FormulaResult
	EvaluatedAt time.Time
}

type projectedProperty struct {
	resolved  ResolvedProperty
	catalogID string
}

// ItemProjectionPlan is scoped to one project/type in an authorized snapshot.
// Only attached definitions may be supplied. No item lookup, authorization,
// defaults, reference-label lookup or persistence takes place in this pure plan.
type ItemProjectionPlan struct {
	properties []projectedProperty
	formulas   *ItemFormulaPlan
}

func CompileItemProjection(definitions []property.Definition, projectID, typeID string) (*ItemProjectionPlan, error) {
	plan := &ItemProjectionPlan{formulas: CompileItemFormulas(definitions, projectID, typeID)}
	for _, def := range definitions {
		config, contextID, err := property.Effective(def, projectID, typeID)
		if err != nil {
			return nil, err
		}
		_, catalogID, err := property.OptionCatalog(def, property.Scope{ProjectID: projectID, TypeID: typeID})
		if err != nil {
			return nil, err
		}
		config.Options = slices.Clone(config.Options)
		config.Levels = slices.Clone(config.Levels)
		plan.properties = append(plan.properties, projectedProperty{resolved: ResolvedProperty{ID: def.ID, Name: def.Name, Title: def.Title, Description: def.Description, Icon: def.Icon, Kind: def.Kind, Configuration: config, ContextID: contextID}, catalogID: catalogID})
	}
	return plan, nil
}

// Project shares one supplied instant across every formula. Missing definitions
// are absent from Values (not applicable); attached inactive fields are explicit.
// Draft callers omit cascading fields, matching the editor contract.
func (plan *ItemProjectionPlan) Project(standard ItemFormulaInputs, saved map[string]any, instant time.Time, draft bool) ItemProjection {
	out := ItemProjection{Values: map[string]ProjectedValue{}, Computed: plan.formulas.Evaluate(standard, saved, instant), EvaluatedAt: instant}
	for _, field := range plan.properties {
		resolved := field.resolved
		value := ProjectedValue{Kind: resolved.Kind, ContextID: resolved.ContextID, CatalogID: field.catalogID, Status: "not_applicable"}
		if resolved.Configuration.Active && (!draft || resolved.Kind != property.Cascading) {
			value = projectPropertyValue(field, saved[resolved.ID], out.Computed[resolved.ID])
		}
		out.Values[resolved.ID] = value
	}
	return out
}

func projectPropertyValue(field projectedProperty, saved any, computed FormulaResult) ProjectedValue {
	resolved := field.resolved
	value := ProjectedValue{Kind: resolved.Kind, ContextID: resolved.ContextID, CatalogID: field.catalogID, Status: "empty"}
	if resolved.Kind == property.Date {
		value.Format = resolved.Configuration.Format
	}
	if resolved.Kind == property.Formula {
		value.ResultKind = computed.Kind
		value.Status, value.Value, value.Error = computed.Status, computed.Value, computed.Error
		return value
	}
	normalized, err := property.ReadValue(resolved.Kind, resolved.Configuration, saved)
	if err != nil {
		value.Status, value.Error = "error", err.Error()
		return value
	}
	value.Value = normalized
	if normalized != nil {
		value.Status = "value"
	}
	if resolved.Kind == property.Cascading {
		value.Levels = map[string]any{}
		path, _ := normalized.([]string)
		for index, level := range resolved.Configuration.Levels {
			value.Levels[level.ID] = nil
			if index < len(path) {
				value.Levels[level.ID] = path[index]
			}
		}
	}
	return value
}
