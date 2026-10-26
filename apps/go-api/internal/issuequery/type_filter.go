package issuequery

import (
	"fmt"
	"strings"
)

func isTypeFilterField(field string) bool {
	return field == "type_id" || field == "type_id__exact" || field == "type_id__in"
}

// BuildTypeFilterField lets a scope-specific legacy compiler share the type
// contract without changing its relation joins or legacy field semantics.
// A nil catalog never enables type fields.
func BuildTypeFilterField(field string, value any, args *[]any, typeIDs map[string]bool) (sql string, handled bool, err error) {
	if typeIDs == nil || !isTypeFilterField(field) {
		return "", false, nil
	}
	if err = validateTypeFilterField(field, value, typeIDs); err != nil {
		return "", true, err
	}
	sql, err = renderTypeFilter(field, value, args)
	return sql, true, err
}

func typeFilterIDs(field string, value any) ([]string, error) {
	if value == nil || value == "" {
		return nil, nil
	}
	var values []string
	switch typed := value.(type) {
	case string:
		values = []string{typed}
		if field == "type_id__in" {
			values = strings.Split(typed, ",")
		}
	case []any:
		if field != "type_id__in" {
			return nil, invalidIssueFilterset(field, "Choose one work item type.")
		}
		for _, entry := range typed {
			text, ok := entry.(string)
			if !ok || text == "" {
				return nil, invalidIssueFilterset(field, "Enter valid type IDs.")
			}
			values = append(values, text)
		}
	default:
		return nil, invalidIssueFilterset(field, "Enter valid type IDs.")
	}
	return issueFilterUUIDs(values, field)
}

func validateTypeFilterField(field string, value any, typeIDs map[string]bool) error {
	ids, err := typeFilterIDs(field, value)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if !typeIDs[id] {
			return newFilterError("A selected work item type is unavailable. Update this filter.", "invalid_filter_reference")
		}
	}
	return nil
}

func renderTypeFilter(field string, value any, args *[]any) (string, error) {
	ids, err := typeFilterIDs(field, value)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		if field == "type_id__in" {
			return "FALSE", nil
		}
		return "issue.type_id IS NULL", nil
	}
	*args = append(*args, ids)
	// Untyped items do not match a positive selection and therefore do match its
	// negation. Avoid SQL UNKNOWN silently dropping them from exclusion filters.
	return fmt.Sprintf("COALESCE(issue.type_id = ANY($%d::uuid[]),FALSE)", len(*args)), nil
}

// UsesTypeFilters selects the authenticated snapshot path, not filter validity.
// Malformed input retains its established compiler error after authorization.
func UsesTypeFilters(raw string) bool {
	decoded, err := parseComplexFilter(raw)
	return err == nil && usesDiscoveryField(decoded, 1, isTypeFilterField)
}

func UsesPropertyFilters(raw string) bool {
	decoded, err := parseComplexFilter(raw)
	return err == nil && usesDiscoveryField(decoded, 1, func(field string) bool { return strings.HasPrefix(field, "customproperty_") })
}

func usesDiscoveryField(node any, depth int, matches func(string) bool) bool {
	if depth > 5 {
		return false
	}
	object, ok := node.(orderedIssueFilterObject)
	if !ok {
		return false
	}
	for _, field := range object {
		if matches(field.name) {
			return true
		}
		switch strings.ToLower(field.name) {
		case "not":
			if usesDiscoveryField(field.value, depth+1, matches) {
				return true
			}
		case "and", "or":
			if children, valid := field.value.([]any); valid {
				for _, child := range children {
					if usesDiscoveryField(child, depth+1, matches) {
						return true
					}
				}
			}
		}
	}
	return false
}
