package issuequery

import (
	"slices"
	"time"
)

// PQLPropertyKind is a comparison family, not a database column or native
// property definition. Resolvers cannot supply SQL through this boundary.
type PQLPropertyKind string

const (
	PQLPropertyText      PQLPropertyKind = "text"
	PQLPropertyNumber    PQLPropertyKind = "numeric"
	PQLPropertyDate      PQLPropertyKind = "date"
	PQLPropertyBoolean   PQLPropertyKind = "boolean"
	PQLPropertySelection PQLPropertyKind = "selection"
)

type PQLPropertyDefinition struct {
	Kind PQLPropertyKind
	// ValidateSelections distinguishes unavailable identities from operational
	// errors. It runs during preparation, inside the caller's read snapshot.
	ValidateSelections func([]string) (bool, error)
}

type PQLPropertyResolver func(string) (PQLPropertyDefinition, bool, error)

// PreparedPQL owns a completely validated expression. It retains comparison
// families, never a resolver callback or an authorization decision for items.
type PreparedPQL struct {
	root       *pqlExpression
	request    PQLContext
	location   *time.Location
	properties map[string]PQLPropertyDefinition
}

// PQLUsesProperties inspects parsed syntax, not text inside literals. Ordinary
// callers keep BuildPQL; property queries require an authorized snapshot first.
func PQLUsesProperties(raw string) (bool, error) {
	root, err := parsePQL(raw)
	if err != nil {
		return false, err
	}
	var uses func(*pqlExpression) bool
	uses = func(e *pqlExpression) bool {
		return e != nil && (e.field == "cf" || uses(e.left) || uses(e.right))
	}
	return uses(root), nil
}

func PreparePQL(raw string, request PQLContext, resolve PQLPropertyResolver) (*PreparedPQL, error) {
	root, err := parsePQL(raw)
	if err != nil {
		return nil, err
	}
	plan := &PreparedPQL{root: root, request: request, properties: map[string]PQLPropertyDefinition{}}
	if root == nil {
		return plan, nil
	}
	c, err := newPQLCompiler(request, nil)
	if err != nil {
		return nil, err
	}
	c.properties, c.resolveProperty, c.preparingProperties = plan.properties, resolve, true
	if _, err = c.expression(root); err != nil {
		return nil, err
	}
	for id, definition := range plan.properties {
		definition.ValidateSelections = nil
		plan.properties[id] = definition
	}
	plan.location = c.location
	return plan, nil
}

func (plan *PreparedPQL) PropertyIDs() []string {
	ids := make([]string, 0, len(plan.properties))
	for id := range plan.properties {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// SQL binds effective values from every authorized candidate, before counting
// or pagination. Missing rows are unknown; present nil values are applicable
// empty values. Neither values nor argument slices are retained or mutated.
func (plan *PreparedPQL) SQL(values map[string][]PQLPropertyValue, args *[]any) (string, error) {
	bindings, err := plan.propertyBindings(values)
	if err != nil || plan.root == nil {
		return "", err
	}
	c := pqlCompiler{request: plan.request, location: plan.location, args: slices.Clone(*args), properties: plan.properties, propertyValues: bindings}
	result, err := c.expression(plan.root)
	if err != nil {
		return "", err
	}
	*args = c.args
	return result, nil
}
