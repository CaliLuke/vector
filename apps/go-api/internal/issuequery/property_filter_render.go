package issuequery

import (
	"fmt"
	"strings"
)

type propertyTruthPair struct{ yes, no string }

func legacyPropertyTruth(sql string) propertyTruthPair {
	if sql == "" {
		sql = "TRUE"
	}
	return propertyTruthPair{yes: "(" + sql + ") IS TRUE", no: "(" + sql + ") IS NOT TRUE"}
}

func hasPropertyFilter(node orderedIssueFilterObject) bool {
	for _, field := range node {
		if field.propertyLeaf != 0 {
			return true
		}
		switch child := field.value.(type) {
		case orderedIssueFilterObject:
			if hasPropertyFilter(child) {
				return true
			}
		case []any:
			for _, entry := range child {
				if object, ok := entry.(orderedIssueFilterObject); ok && hasPropertyFilter(object) {
					return true
				}
			}
		}
	}
	return false
}

// SQL returns only definite matches. Unknown values survive NOT and relation
// quantification as unknown, rather than becoming exclusion matches.
func (bound *BoundPropertyFilter) SQL(args *[]any) (string, error) {
	if len(bound.plan.root) == 0 {
		return "", nil
	}
	if !hasPropertyFilter(bound.plan.root) {
		return renderIssueComplexFilter(bound.plan.root, args)
	}
	pair, err := bound.scope(bound.plan.root, args)
	if err != nil {
		return "", err
	}
	// Callers combine this expression with authorization and route predicates.
	// Keep a top-level OR inside that conjunction.
	return "(" + pair.yes + ")", nil
}

func (bound *BoundPropertyFilter) scope(node orderedIssueFilterObject, args *[]any) (propertyTruthPair, error) {
	if !hasPropertyFilter(node) {
		sql, err := renderIssueComplexFilter(node, args)
		return legacyPropertyTruth(sql), err
	}
	order := []string{}
	collectPositiveIssueFilterRelations(node, &order, map[string]bool{})
	aliases := map[string]string{}
	for _, field := range order {
		aliases[field] = "complex_" + strings.TrimSuffix(field, "_id")
	}
	pair, err := bound.tree(node, args, aliases)
	if err != nil || len(order) == 0 {
		return pair, err
	}
	bindings := ` FROM (VALUES (1)) complex_seed(dummy) ` + issueFilterRelationJoins(order, aliases)
	return propertyTruthPair{
		yes: "EXISTS (SELECT 1" + bindings + " WHERE " + pair.yes + ")",
		no:  "NOT EXISTS (SELECT 1" + bindings + " WHERE (" + pair.no + ") IS NOT TRUE)",
	}, nil
}

func (bound *BoundPropertyFilter) tree(node orderedIssueFilterObject, args *[]any, aliases map[string]string) (propertyTruthPair, error) {
	if !hasPropertyFilter(node) {
		sql, err := renderIssueComplexFilterTree(node, args, aliases)
		return legacyPropertyTruth(sql), err
	}
	for _, operator := range []string{"or", "and", "not"} {
		value, exists := orderedIssueFilterValue(node, operator)
		if !exists {
			continue
		}
		if operator == "not" {
			child, err := bound.scope(value.(orderedIssueFilterObject), args)
			return propertyTruthPair{yes: child.no, no: child.yes}, err
		}
		pairs := []propertyTruthPair{}
		for _, child := range value.([]any) {
			pair, err := bound.tree(child.(orderedIssueFilterObject), args, aliases)
			if err != nil {
				return propertyTruthPair{}, err
			}
			pairs = append(pairs, pair)
		}
		return joinPropertyTruth(pairs, operator), nil
	}
	pairs := []propertyTruthPair{}
	legacy := orderedIssueFilterObject{}
	for _, field := range node {
		if field.propertyLeaf == 0 {
			legacy = append(legacy, field)
			continue
		}
		match := bound.matches[field.propertyLeaf]
		values := map[string]bool{}
		for _, id := range match.TrueIDs {
			values[id] = true
		}
		for _, id := range match.FalseIDs {
			values[id] = false
		}
		// One typed parameter serves both truth branches. Binding two arrays
		// eagerly leaves unused positional parameters when only one is emitted.
		*args = append(*args, values)
		lookup := fmt.Sprintf("($%d::jsonb ->> issue.id::text)::boolean", len(*args))
		pairs = append(pairs, propertyTruthPair{yes: "(" + lookup + ") IS TRUE", no: "(" + lookup + ") IS FALSE"})
	}
	if len(legacy) > 0 {
		sql, err := renderIssueComplexFilterTree(legacy, args, aliases)
		if err != nil {
			return propertyTruthPair{}, err
		}
		pairs = append(pairs, legacyPropertyTruth(sql))
	}
	return joinPropertyTruth(pairs, "and"), nil
}

func joinPropertyTruth(pairs []propertyTruthPair, operator string) propertyTruthPair {
	yes, no := []string{}, []string{}
	for _, pair := range pairs {
		yes = append(yes, "("+pair.yes+")")
		no = append(no, "("+pair.no+")")
	}
	positive, negative := " AND ", " OR "
	if operator == "or" {
		positive, negative = " OR ", " AND "
	}
	return propertyTruthPair{yes: strings.Join(yes, positive), no: strings.Join(no, negative)}
}

func issueFilterRelationJoins(order []string, aliases map[string]string) string {
	joins := []string{}
	for _, field := range order {
		relation, _ := issueComplexRelation(field)
		alias := aliases[field]
		joins = append(joins, "LEFT JOIN "+relation[0]+" "+alias+" ON "+alias+".issue_id = issue.id AND "+alias+".deleted_at IS NULL")
	}
	return strings.Join(joins, "\n    ")
}

func (bound *BoundPropertyFilter) orderConstraint(targetField, targetAlias string, args *[]any) (string, string, bool, error) {
	joins, aliases, constrained := issueFilterOrderBindings(bound.plan.root, targetField, targetAlias)
	if !constrained {
		return "", "", false, nil
	}
	pair, err := bound.tree(bound.plan.root, args, aliases)
	return joins, pair.yes, true, err
}
