package workitemtypes

import (
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func propertyPredicateOperators(kind property.Kind, result formula.Kind) []string {
	if kind == property.Formula {
		kind = property.Kind(result)
	}
	switch kind {
	case property.Text, property.Email, property.File:
		return []string{"exact", "contains", "isnull"}
	case property.URL:
		return []string{"exact", "contains"}
	case property.Boolean:
		return []string{"exact"}
	case property.Number, property.Date:
		return []string{"exact", "lt", "lte", "gt", "gte", "range", "isnull"}
	case property.Dropdown, property.Member, property.Issue, property.Release, property.Cascading:
		return []string{"exact", "in", "isnull"}
	default:
		return []string{}
	}
}
