package issuequery

import (
	"slices"
	"uuid"
)

// PropertyFilterLeaf is a scalar/list predicate validated against the caller's
// authorized effective property catalog. ID is request-local, not a stored ID.
type PropertyFilterLeaf struct {
	ID    int
	Field string
	Value any
}

type PropertyFilterMatches struct{ TrueIDs, FalseIDs []string }

// PreparedPropertyFilter retains the original AST, including legacy relation
// scopes. It never evaluates formulas or loads rows. The caller must evaluate
// every authorized candidate before binding the disjoint true/false sets.
type PreparedPropertyFilter struct {
	root   orderedIssueFilterObject
	leaves []PropertyFilterLeaf
}

type BoundPropertyFilter struct {
	plan    *PreparedPropertyFilter
	matches map[int]PropertyFilterMatches
}

// PreparePropertyFilter permits additional fields only through validate. The
// callback must recognize the field and validate its identity, operator and value
// in the authorized snapshot. Legacy/public callers keep BuildComplexFilter.
func PreparePropertyFilter(raw string, typeIDs map[string]bool, validate func(string, any) (bool, error)) (*PreparedPropertyFilter, error) {
	plan := &PreparedPropertyFilter{}
	if raw == "" {
		return plan, nil
	}
	decoded, err := parseComplexFilter(raw)
	if err != nil {
		return nil, err
	}
	if emptyComplexFilter(decoded) {
		return plan, nil
	}
	accept := func(field *orderedIssueFilterField) (bool, error) {
		if validate == nil {
			return false, nil
		}
		handled, validationErr := validate(field.name, clonePropertyOperand(field.value))
		if validationErr != nil || !handled {
			return handled, validationErr
		}
		field.propertyLeaf = len(plan.leaves) + 1
		plan.leaves = append(plan.leaves, PropertyFilterLeaf{ID: field.propertyLeaf, Field: field.name, Value: clonePropertyOperand(field.value)})
		return true, nil
	}
	if err = validateIssueFilterWithProperties(decoded, 1, typeIDs, accept); err != nil {
		return nil, err
	}
	plan.root = decoded.(orderedIssueFilterObject)
	return plan, nil
}

func clonePropertyOperand(value any) any {
	if values, ok := value.([]any); ok {
		return slices.Clone(values)
	}
	return value
}

func (plan *PreparedPropertyFilter) Leaves() []PropertyFilterLeaf {
	leaves := slices.Clone(plan.leaves)
	for i := range leaves {
		leaves[i].Value = clonePropertyOperand(leaves[i].Value)
	}
	return leaves
}

// BindEvaluatedItems materializes each validated leaf from already projected,
// authorized candidates. A missing comparison domain is unknown, never false.
// The callback receives the leaf ID and candidate index, keeping this SQL layer
// independent of property schemas, formula evaluation and database reads.
func (plan *PreparedPropertyFilter) BindEvaluatedItems(ids []string, evaluate func(int, int) (bool, bool)) (*BoundPropertyFilter, error) {
	matches := make(map[int]PropertyFilterMatches, len(plan.leaves))
	for _, leaf := range plan.leaves {
		match := PropertyFilterMatches{}
		for index, id := range ids {
			value, known := evaluate(leaf.ID, index)
			if !known {
				continue
			}
			if value {
				match.TrueIDs = append(match.TrueIDs, id)
			} else {
				match.FalseIDs = append(match.FalseIDs, id)
			}
		}
		matches[leaf.ID] = match
	}
	return plan.Bind(matches)
}

// Bind rejects incomplete or contradictory materializations. Neither set means
// unknown, not false. Copies protect subsequent SQL from caller mutation.
func (plan *PreparedPropertyFilter) Bind(matches map[int]PropertyFilterMatches) (*BoundPropertyFilter, error) {
	if len(matches) != len(plan.leaves) {
		return nil, newFilterError("Property filter evaluation is incomplete.", "invalid_property_matches")
	}
	bound := &BoundPropertyFilter{plan: plan, matches: map[int]PropertyFilterMatches{}}
	for _, leaf := range plan.leaves {
		match, exists := matches[leaf.ID]
		if !exists {
			return nil, newFilterError("Property filter evaluation is incomplete.", "invalid_property_matches")
		}
		normalized, err := normalizePropertyMatches(match)
		if err != nil {
			return nil, err
		}
		bound.matches[leaf.ID] = normalized
	}
	return bound, nil
}

func normalizePropertyMatches(match PropertyFilterMatches) (PropertyFilterMatches, error) {
	seen := map[string]bool{}
	out := PropertyFilterMatches{TrueIDs: []string{}, FalseIDs: []string{}}
	for index, ids := range [][]string{match.TrueIDs, match.FalseIDs} {
		for _, id := range ids {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return out, newFilterError("Property filter evaluation contains an invalid item.", "invalid_property_matches")
			}
			canonical := parsed.String()
			if previous, exists := seen[canonical]; exists {
				if previous != (index == 0) {
					return out, newFilterError("Property filter evaluation is contradictory.", "invalid_property_matches")
				}
				continue
			}
			seen[canonical] = index == 0
			if index == 0 {
				out.TrueIDs = append(out.TrueIDs, canonical)
			} else {
				out.FalseIDs = append(out.FalseIDs, canonical)
			}
		}
	}
	return out, nil
}
