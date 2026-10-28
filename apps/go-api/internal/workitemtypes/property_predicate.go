package workitemtypes

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"golang.org/x/text/cases"
)

// ProjectedPredicate compares already-authorized effective values. Its caller
// validates the definition, level, catalog and reference identities separately.
// Negation belongs to the Boolean compiler, never to a complement of matches.
type ProjectedPredicate struct {
	kind                     property.Kind
	result                   formula.Kind
	operator, level, catalog string
	operand                  any
}

func CompileProjectedPredicate(kind property.Kind, result formula.Kind, level, catalog, operator string, raw any) (*ProjectedPredicate, error) {
	p := &ProjectedPredicate{kind: kind, result: result, level: level, catalog: catalog, operator: operator}
	family := kind
	if family == property.Email || family == property.File {
		family = property.Text
	}
	if kind == property.Formula {
		family = property.Kind(result)
	}
	if family == property.RichText || !slices.Contains([]property.Kind{property.Text, property.Number, property.Date, property.Boolean, property.URL, property.Dropdown, property.Member, property.Issue, property.Release, property.Cascading}, family) {
		return nil, fmt.Errorf("this property does not support filtering")
	}
	if kind == property.Cascading && level == "" || kind != property.Cascading && level != "" {
		return nil, fmt.Errorf("choose a valid cascading level")
	}
	if raw == nil && operator == "exact" {
		p.operator = "isnull"
		raw = true
	}
	if !slices.Contains(propertyPredicateOperators(kind, result), p.operator) {
		return nil, fmt.Errorf("this property does not support that operator")
	}
	if p.operator == "isnull" {
		value, ok := raw.(bool)
		if !ok || family == property.Boolean || family == property.URL {
			return nil, fmt.Errorf("this property does not support this empty-value predicate")
		}
		p.operand = value
		return p, nil
	}
	var err error
	switch family {
	case property.Text, property.URL:
		value, ok := raw.(string)
		if !ok || len(value) > 65536 || operator != "exact" && operator != "contains" {
			return nil, fmt.Errorf("choose a text equality or contains predicate")
		}
		p.operand = value
		if family == property.Text {
			p.operand = cases.Fold().String(value)
		}
	case property.Boolean:
		value, ok := raw.(bool)
		if !ok || operator != "exact" {
			return nil, fmt.Errorf("choose true or false")
		}
		p.operand = value
	case property.Number, property.Date:
		if !slices.Contains([]string{"exact", "lt", "lte", "gt", "gte", "range"}, operator) {
			return nil, fmt.Errorf("choose a numeric or date comparison")
		}
		p.operand, err = orderedPredicateOperand(family, operator, raw)
	case property.Dropdown, property.Member, property.Issue, property.Release, property.Cascading:
		if operator != "exact" && operator != "in" {
			return nil, fmt.Errorf("choose a selection equality or inclusion predicate")
		}
		p.operand, err = selectionPredicateOperand(operator, raw)
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Match returns (matches, known). Inapplicable/error values and absent values
// outside an explicit empty test are unknown. False and zero remain ordinary
// values. Date controls compare UTC calendar days, including timestamp formulas.
func (p *ProjectedPredicate) Match(value ProjectedValue) (bool, bool) {
	if value.Kind != p.kind || (value.Status != "value" && value.Status != "empty") || p.kind == property.Formula && value.ResultKind != p.result {
		return false, false
	}
	if (p.kind == property.Cascading || p.kind == property.Dropdown) && value.CatalogID != p.catalog {
		return false, false
	}
	current := value.Value
	empty := value.Status == "empty"
	if p.kind == property.Cascading {
		var exists bool
		current, exists = value.Levels[p.level]
		if !exists {
			return false, false
		}
		empty = current == nil
	}
	if p.operator == "isnull" {
		return empty == p.operand.(bool), true
	}
	if empty {
		return false, false
	}
	family := p.kind
	if family == property.Email || family == property.File {
		family = property.Text
	}
	if family == property.Formula {
		family = property.Kind(p.result)
	}
	switch family {
	case property.Text, property.URL:
		text, ok := current.(string)
		if !ok {
			return false, false
		}
		if family == property.Text {
			text = cases.Fold().String(text)
		}
		if p.operator == "contains" {
			return strings.Contains(text, p.operand.(string)), true
		}
		return text == p.operand.(string), true
	case property.Boolean:
		boolean, ok := current.(bool)
		return boolean == p.operand.(bool), ok
	case property.Number, property.Date:
		number, err := orderedPredicateValue(family, current)
		if err != nil {
			return false, false
		}
		values := p.operand.([]float64)
		switch p.operator {
		case "exact":
			return number == values[0], true
		case "lt":
			return number < values[0], true
		case "lte":
			return number <= values[0], true
		case "gt":
			return number > values[0], true
		case "gte":
			return number >= values[0], true
		case "range":
			return number >= values[0] && number <= values[1], true
		}
	case property.Dropdown, property.Member, property.Issue, property.Release, property.Cascading:
		var values []string
		switch saved := current.(type) {
		case string:
			values = []string{saved}
		case []string:
			values = saved
		default:
			return false, false
		}
		for _, id := range p.operand.([]string) {
			if slices.Contains(values, id) {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}

func orderedPredicateOperand(kind property.Kind, operator string, raw any) ([]float64, error) {
	values := []any{raw}
	if operator == "range" {
		switch input := raw.(type) {
		case []any:
			values = input
		case string:
			parts := strings.Split(input, ",")
			values = []any{}
			for _, part := range parts {
				values = append(values, part)
			}
		default:
			return nil, fmt.Errorf("enter two range bounds")
		}
		if len(values) != 2 {
			return nil, fmt.Errorf("enter two range bounds")
		}
	}
	out := []float64{}
	for _, value := range values {
		number, err := orderedPredicateValue(kind, value)
		if err != nil {
			return nil, err
		}
		out = append(out, number)
	}
	if len(out) == 2 && out[0] > out[1] {
		return nil, fmt.Errorf("range bounds must be in ascending order")
	}
	return out, nil
}

func orderedPredicateValue(kind property.Kind, raw any) (float64, error) {
	if kind == property.Date {
		text, ok := raw.(string)
		if !ok {
			return 0, fmt.Errorf("enter a date")
		}
		date, err := property.ParseDate(text)
		if err != nil || date.UTC().Year() < 1 || date.UTC().Year() > 9999 {
			return 0, fmt.Errorf("enter a valid date")
		}
		day := date.UTC().Format(time.DateOnly)
		date, _ = time.Parse(time.DateOnly, day)
		return float64(date.Unix() / 86400), nil
	}
	var number float64
	var err error
	switch value := raw.(type) {
	case float64:
		number = value
	case json.Number:
		number, err = value.Float64()
	case string:
		number, err = strconv.ParseFloat(value, 64)
	default:
		return 0, fmt.Errorf("enter a number")
	}
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("enter a finite number")
	}
	return number, nil
}

func selectionPredicateOperand(operator string, raw any) ([]string, error) {
	values := []any{}
	switch input := raw.(type) {
	case string:
		if operator == "in" {
			for _, id := range strings.Split(input, ",") {
				values = append(values, id)
			}
		} else {
			values = append(values, input)
		}
	case []any:
		if operator != "in" {
			return nil, fmt.Errorf("choose one selection")
		}
		values = input
	default:
		return nil, fmt.Errorf("choose valid selections")
	}
	out := []string{}
	for _, value := range values {
		id, ok := value.(string)
		parsed, err := uuid.Parse(id)
		if !ok || err != nil || parsed.String() != id {
			return nil, fmt.Errorf("choose valid selection identities")
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}
