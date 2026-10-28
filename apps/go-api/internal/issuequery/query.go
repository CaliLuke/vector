// Package issuequery builds SQL fragments for issue filters and ordering.
// It owns only pure query helpers; repositories retain their query/domain behavior.
package issuequery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"uuid"
)

var ErrInvalidComplexRange = errors.New("invalid complex date range")

type FilterError struct {
	Message string
	Code    string
	Errors  any
}

func (err *FilterError) Error() string { return err.Message }

type Options struct {
	IncludeProperties bool
	PropertyFilter    *BoundPropertyFilter
	Limit             int
	Page              int
	OrderBy           string
	CursorValue       float64
	CursorSet         bool
	OrderBySet        bool
	GroupBy           string
	SubGroupBy        string
	State             string
	StateGroup        string
	EstimatePoint     string
	Priority          string
	Assignees         string
	AssigneesSet      bool
	Labels            string
	Project           string
	LabelsSet         bool
	LoggedBy          string
	CreatedBy         string
	Parent            string
	Mentions          string
	Name              string
	CreatedAt         string
	UpdatedAt         string
	StartDate         string
	TargetDate        string
	CompletedAt       string
	CycleSet          bool
	Cycle             string
	Module            string
	ModuleSet         bool
	IntakeStatus      string
	InboxStatus       string
	SubIssue          string
	Subscriber        string
	StartTargetDate   string
	SubscriberSet     bool
	Type              string
	TypeSet           bool
	Filters           string
}

func RequesterDateSQL(column string) string {
	return `((
      ` + column + ` AT TIME ZONE COALESCE(
        (SELECT NULLIF(requester_user.user_timezone, '') FROM users requester_user WHERE requester_user.id = $1),
        'UTC'
      )
    )::date)`
}

func AddDateFilters(where []string, args []any, value, column string) ([]string, []any) {
	filters := strings.Split(value, ",")
	for _, filter := range filters {
		if filter == "" {
			return where, args
		}
	}
	type dateFilter struct {
		kind       string
		value      any
		comparison string
		sign       string
		relative   bool
	}
	positions := make(map[string]int, 3)
	selected := make([]dateFilter, 0, 3)
	put := func(filter dateFilter) {
		if position, exists := positions[filter.kind]; exists {
			selected[position] = filter
			return
		}
		positions[filter.kind] = len(selected)
		selected = append(selected, filter)
	}
	for _, filter := range filters {
		parts := strings.Split(filter, ";")
		if len(parts) == 1 {
			put(dateFilter{kind: "contains", value: "%" + EscapeLikePattern(parts[0]) + "%"})
			continue
		}
		duration := strings.SplitN(parts[0], "_", 2)
		if len(duration) == 2 && asciiDigits(duration[0]) && (duration[1] == "weeks" || duration[1] == "months") {
			if len(parts) == 3 {
				count, _ := strconv.Atoi(duration[0])
				days := count * 7
				if duration[1] == "months" {
					days = count * 30
				}
				comparison := "<="
				kind := "lte"
				if parts[1] == "after" {
					comparison, kind = ">=", "gte"
				}
				sign := "-"
				if parts[2] == "fromnow" {
					sign = "+"
				}
				put(dateFilter{kind: kind, value: days, comparison: comparison, sign: sign, relative: true})
			}
			continue
		}
		comparison := "<="
		kind := "lte"
		for _, part := range parts {
			if part == "after" {
				comparison, kind = ">=", "gte"
				break
			}
		}
		put(dateFilter{kind: kind, value: parts[0], comparison: comparison})
	}
	for _, filter := range selected {
		args = append(args, filter.value)
		switch {
		case filter.kind == "contains":
			where = append(where, fmt.Sprintf("(%s)::text LIKE $%d ESCAPE '\\'", column, len(args)))
		case filter.relative:
			where = append(where, fmt.Sprintf("%s %s (CURRENT_DATE %s ($%d * INTERVAL '1 day'))::date", column, filter.comparison, filter.sign, len(args)))
		default:
			where = append(where, fmt.Sprintf("%s %s $%d::date", column, filter.comparison, len(args)))
		}
	}
	return where, args
}

var issueComplexFilterFields = map[string]struct{}{
	"assignee_id": {}, "assignee_id__exact": {}, "assignee_id__in": {},
	"cycle_id": {}, "cycle_id__exact": {}, "cycle_id__in": {},
	"module_id": {}, "module_id__exact": {}, "module_id__in": {},
	"mention_id": {}, "mention_id__exact": {}, "mention_id__in": {},
	"label_id": {}, "label_id__exact": {}, "label_id__in": {},
	"created_by_id": {}, "created_by_id__exact": {}, "created_by_id__in": {},
	"is_archived": {}, "is_archived__exact": {},
	"state_group": {}, "state_group__exact": {}, "state_group__in": {},
	"state_id": {}, "state_id__exact": {}, "state_id__in": {},
	"project_id": {}, "project_id__exact": {}, "project_id__in": {},
	"subscriber_id": {}, "subscriber_id__exact": {}, "subscriber_id__in": {},
	"created_at": {}, "created_at__exact": {}, "created_at__range": {},
	"updated_at": {}, "updated_at__exact": {}, "updated_at__range": {},
	"start_date": {}, "start_date__exact": {}, "start_date__range": {},
	"target_date": {}, "target_date__exact": {}, "target_date__range": {},
	"is_draft": {}, "is_draft__exact": {},
	"priority": {}, "priority__exact": {}, "priority__in": {},
}

type orderedIssueFilterObject []orderedIssueFilterField

type orderedIssueFilterField struct {
	name         string
	value        any
	propertyLeaf int // positive only in an authorized prepared property filter
}

func decodeOrderedIssueFilter(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return token, nil
	}
	switch delimiter {
	case '{':
		object := make(orderedIssueFilterObject, 0)
		for decoder.More() {
			nameToken, filterErr := decoder.Token()
			if filterErr != nil {
				return nil, filterErr
			}
			value, filterErr := decodeOrderedIssueFilter(decoder)
			if filterErr != nil {
				return nil, filterErr
			}
			name := nameToken.(string)
			replaced := false
			for index := range object {
				if object[index].name == name {
					object[index].value = value
					replaced = true
					break
				}
			}
			if !replaced {
				object = append(object, orderedIssueFilterField{name: name, value: value})
			}
		}
		_, err = decoder.Token()
		return object, err
	case '[':
		values := make([]any, 0)
		for decoder.More() {
			value, decodeOrderedIssueFilterErr := decodeOrderedIssueFilter(decoder)
			if decodeOrderedIssueFilterErr != nil {
				return nil, decodeOrderedIssueFilterErr
			}
			values = append(values, value)
		}
		_, err = decoder.Token()
		return values, err
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter")
	}
}

func normalizePythonJSONConstants(raw string) string {
	var output strings.Builder
	output.Grow(len(raw))
	inString, escaped := false, false
	for index := 0; index < len(raw); {
		character := raw[index]
		if inString {
			output.WriteByte(character)
			index++
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		if character == '"' {
			inString = true
			output.WriteByte(character)
			index++
			continue
		}
		const sentinel = `"\u0000python-json:`
		switch {
		case strings.HasPrefix(raw[index:], "-Infinity"):
			output.WriteString(sentinel + `-Infinity"`)
			index += len("-Infinity")
		case strings.HasPrefix(raw[index:], "Infinity"):
			output.WriteString(sentinel + `Infinity"`)
			index += len("Infinity")
		case strings.HasPrefix(raw[index:], "NaN"):
			output.WriteString(sentinel + `NaN"`)
			index += len("NaN")
		default:
			output.WriteByte(character)
			index++
		}
	}
	return output.String()
}

func BuildComplexFilter(raw string, args *[]any) (string, error) {
	return buildComplexFilter(raw, args, nil)
}

// BuildComplexFilterWithTypes is only for authenticated callers that resolved
// this catalogue in the same authorized read snapshot. A nil catalogue retains
// the legacy/public field allowlist. Inactive retained types may be included.
func BuildComplexFilterWithTypes(raw string, args *[]any, typeIDs map[string]bool) (string, error) {
	return buildComplexFilter(raw, args, typeIDs)
}

func buildComplexFilter(raw string, args *[]any, typeIDs map[string]bool) (string, error) {
	if raw == "" {
		return "", nil
	}
	decoded, err := parseComplexFilter(raw)
	if err != nil {
		return "", err
	}
	if emptyComplexFilter(decoded) {
		return "", nil
	}
	if err := validateIssueComplexFilter(decoded, 1, typeIDs); err != nil {
		return "", err
	}
	return renderIssueComplexFilter(decoded, args)
}

func emptyComplexFilter(decoded any) bool {
	switch value := decoded.(type) {
	case nil:
		return true
	case bool:
		if !value {
			return true
		}
	case string:
		if value == "" {
			return true
		}
	case json.Number:
		number, _ := strconv.ParseFloat(value.String(), 64)
		if number == 0 {
			return true
		}
	case []any:
		if len(value) == 0 {
			return true
		}
	case orderedIssueFilterObject:
		if len(value) == 0 {
			return true
		}
	}
	return false
}

func validateIssueComplexFilter(node any, depth int, typeIDs map[string]bool) error {
	return validateIssueFilterWithProperties(node, depth, typeIDs, nil)
}

func validateIssueFilterWithProperties(node any, depth int, typeIDs map[string]bool, accept func(*orderedIssueFilterField) (bool, error)) error {
	if depth > 5 {
		return newFilterError("Filter nesting is too deep (max 5); found depth "+strconv.Itoa(depth), "max_depth_exceeded")
	}
	object, ok := node.(orderedIssueFilterObject)
	if !ok {
		return newFilterError("Each filter node must be a JSON object", "invalid_filter_node")
	}
	if len(object) == 0 {
		return newFilterError("Filter objects must not be empty", "empty_filter_object")
	}
	logicalFields := make([]orderedIssueFilterField, 0, 1)
	for _, field := range object {
		switch strings.ToLower(field.name) {
		case "or", "and", "not":
			logicalFields = append(logicalFields, field)
		}
	}
	if len(logicalFields) > 1 {
		return newFilterError("A filter object cannot contain multiple logical operators at the same level", "multiple_logical_operators")
	}
	if len(logicalFields) == 1 {
		logical := logicalFields[0]
		if len(object) != 1 {
			return newFilterError("Cannot mix logical operator '"+logical.name+"' with field keys at the same level", "mixed_operator_and_fields")
		}
		operator := strings.ToLower(logical.name)
		if operator == "not" {
			child, ok := logical.value.(orderedIssueFilterObject)
			if !ok {
				return newFilterError("'not' must be a single JSON object", "invalid_not_child")
			}
			return validateIssueFilterWithProperties(child, depth+1, typeIDs, accept)
		}
		children, ok := logical.value.([]any)
		if !ok || len(children) == 0 {
			return newFilterError("'"+operator+"' must be a non-empty list of filter objects", "invalid_operator_children")
		}
		for _, child := range children {
			if _, ok := child.(orderedIssueFilterObject); !ok {
				return newFilterError("All children of '"+operator+"' must be JSON objects", "invalid_operator_child_type")
			}
			if err := validateIssueFilterWithProperties(child, depth+1, typeIDs, accept); err != nil {
				return err
			}
		}
		return nil
	}
	for index := range object {
		field := &object[index]
		if list, ok := field.value.([]any); ok {
			for _, item := range list {
				if !isIssueFilterScalar(item) {
					return newFilterError("List value for '"+field.name+"' must contain only scalar items", "non_scalar_list_item")
				}
			}
		} else if !isIssueFilterScalar(field.value) {
			return newFilterError("Value for '"+field.name+"' must be a scalar, null, or list/tuple of scalars", "invalid_value_type")
		}
		if isTypeFilterField(field.name) && typeIDs != nil {
			if err := validateTypeFilterField(field.name, field.value, typeIDs); err != nil {
				return err
			}
			continue
		}
		if _, ok := issueComplexFilterFields[field.name]; !ok {
			if accept != nil {
				handled, err := accept(field)
				if err != nil {
					return err
				}
				if handled {
					continue
				}
			}
			return newFilterError("Filtering on field '"+field.name+"' is not allowed", "invalid_filter_field")
		}
	}
	return nil
}

func renderIssueComplexFilter(node any, args *[]any) (string, error) {
	order := make([]string, 0)
	seen := make(map[string]bool)
	collectPositiveIssueFilterRelations(node.(orderedIssueFilterObject), &order, seen)
	aliases := make(map[string]string, len(order))
	for _, field := range order {
		aliases[field] = "complex_" + strings.TrimSuffix(field, "_id")
	}
	predicate, err := renderIssueComplexFilterTree(node.(orderedIssueFilterObject), args, aliases)
	if err != nil || len(order) == 0 {
		return predicate, err
	}
	return `EXISTS (
    SELECT 1
    FROM (VALUES (1)) complex_seed(dummy)
    ` + issueFilterRelationJoins(order, aliases) + `
    WHERE ` + predicate + `
  )`, nil
}

func collectPositiveIssueFilterRelations(node orderedIssueFilterObject, order *[]string, seen map[string]bool) {
	if _, negated := orderedIssueFilterValue(node, "not"); negated {
		return
	}
	for _, operator := range []string{"or", "and"} {
		if childrenValue, exists := orderedIssueFilterValue(node, operator); exists {
			for _, child := range childrenValue.([]any) {
				collectPositiveIssueFilterRelations(child.(orderedIssueFilterObject), order, seen)
			}
			return
		}
	}
	for _, field := range node {
		baseField := strings.TrimSuffix(strings.TrimSuffix(field.name, "__exact"), "__in")
		if _, related := issueComplexRelation(baseField); related && !seen[baseField] {
			seen[baseField] = true
			*order = append(*order, baseField)
		}
	}
}

func renderIssueComplexFilterTree(node orderedIssueFilterObject, args *[]any, aliases map[string]string) (string, error) {
	for _, operator := range []string{"or", "and", "not"} {
		value, exists := orderedIssueFilterValue(node, operator)
		if !exists {
			continue
		}
		if operator == "not" {
			child, err := renderIssueComplexFilter(value, args)
			if err != nil {
				return "", err
			}
			return "(" + child + ") IS NOT TRUE", nil
		}
		children := value.([]any)
		clauses := make([]string, 0, len(children))
		for _, child := range children {
			clause, err := renderIssueComplexFilterTree(child.(orderedIssueFilterObject), args, aliases)
			if err != nil {
				return "", err
			}
			clauses = append(clauses, "("+clause+")")
		}
		joiner := " AND "
		if operator == "or" {
			joiner = " OR "
		}
		return strings.Join(clauses, joiner), nil
	}
	clauses := make([]string, 0, len(node))
	for _, field := range node {
		baseField := strings.TrimSuffix(strings.TrimSuffix(field.name, "__exact"), "__in")
		if relation, related := issueComplexRelation(baseField); related {
			values := issueFilterValues(field.value)
			empty := len(values) == 0 || (len(values) == 1 && values[0] == "")
			if empty {
				if strings.HasSuffix(field.name, "__in") {
					clauses = append(clauses, "FALSE")
				} else {
					predicate := "empty_relation.issue_id = issue.id"
					if baseField == "label_id" {
						predicate = LiveIssueLabelPredicate("issue", "empty_relation")
					}
					clauses = append(clauses, "NOT EXISTS (SELECT 1 FROM "+relation[0]+" empty_relation WHERE "+predicate+")")
				}
				continue
			}
			ids, err := issueFilterUUIDs(values, field.name)
			if err != nil {
				return "", err
			}
			if len(ids) == 0 {
				clauses = append(clauses, "FALSE")
				continue
			}
			*args = append(*args, strings.Join(ids, ","))
			predicate := aliases[baseField] + ".deleted_at IS NULL"
			if baseField == "label_id" {
				predicate = LiveIssueLabelPredicate("issue", aliases[baseField])
			}
			clauses = append(clauses, fmt.Sprintf(
				"(%s.%s::text = ANY(string_to_array($%d, ',')) AND %s)",
				aliases[baseField], relation[1], len(*args), predicate,
			))
			continue
		}
		clause, err := renderIssueComplexFilterField(field.name, field.value, args)
		if err != nil {
			return "", err
		}
		if clause != "" {
			clauses = append(clauses, clause)
		}
	}
	if len(clauses) == 0 {
		return "TRUE", nil
	}
	return strings.Join(clauses, " AND "), nil
}
func orderedIssueFilterValue(object orderedIssueFilterObject, name string) (any, bool) {
	for _, field := range object {
		if strings.EqualFold(field.name, name) {
			return field.value, true
		}
	}
	return nil, false
}

func issueComplexRelation(field string) ([2]string, bool) {
	relation, exists := map[string][2]string{
		"assignee_id":   {"issue_assignees", "assignee_id"},
		"cycle_id":      {"cycle_issues", "cycle_id"},
		"module_id":     {"module_issues", "module_id"},
		"mention_id":    {"issue_mentions", "mention_id"},
		"label_id":      {"issue_labels", "label_id"},
		"subscriber_id": {"issue_subscribers", "subscriber_id"},
	}[field]
	return relation, exists
}

func renderIssueComplexFilterField(field string, value any, args *[]any) (string, error) {
	if isTypeFilterField(field) {
		return renderTypeFilter(field, value, args)
	}
	values := issueFilterValues(value)
	inLookup := strings.HasSuffix(field, "__in")
	rangeLookup := strings.HasSuffix(field, "__range")
	baseField := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(field, "__exact"), "__in"), "__range")
	empty := len(values) == 0 || (len(values) == 1 && values[0] == "")
	if empty && inLookup {
		return "FALSE", nil
	}
	if empty && rangeLookup {
		return "", ErrInvalidComplexRange
	}
	if len(values) == 0 || (len(values) == 1 && values[0] == "") {
		switch baseField {
		case "created_by_id":
			return "issue.created_by_id IS NULL", nil
		case "state_id":
			return "issue.state_id IS NULL", nil
		case "project_id":
			return "issue.project_id IS NULL", nil
		case "state_group":
			return `EXISTS (SELECT 1 FROM states complex_state WHERE complex_state.id = issue.state_id AND complex_state."group" = '')`, nil
		case "priority":
			return "issue.priority = ''", nil
		case "is_draft":
			return "issue.is_draft IS NULL", nil
		case "created_at":
			return RequesterDateSQL("issue.created_at") + " IS NULL", nil
		case "updated_at":
			return RequesterDateSQL("issue.updated_at") + " IS NULL", nil
		case "start_date":
			return "issue.start_date IS NULL", nil
		case "target_date":
			return "issue.target_date IS NULL", nil
		default:
			return "", nil
		}
	}
	switch baseField {
	case "assignee_id", "cycle_id", "module_id", "mention_id", "label_id", "subscriber_id":
		ids, err := issueFilterUUIDs(values, field)
		if err != nil {
			return "", err
		}
		if len(ids) == 0 {
			return "FALSE", nil
		}
		*args = append(*args, strings.Join(ids, ","))
		relation := map[string][2]string{
			"assignee_id":   {"issue_assignees", "assignee_id"},
			"cycle_id":      {"cycle_issues", "cycle_id"},
			"module_id":     {"module_issues", "module_id"},
			"mention_id":    {"issue_mentions", "mention_id"},
			"label_id":      {"issue_labels", "label_id"},
			"subscriber_id": {"issue_subscribers", "subscriber_id"},
		}[baseField]
		predicate := "complex_relation.issue_id = issue.id AND complex_relation.deleted_at IS NULL"
		if baseField == "label_id" {
			predicate = LiveIssueLabelPredicate("issue", "complex_relation")
		}
		return fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s complex_relation WHERE %s AND complex_relation.%s::text = ANY(string_to_array($%d, ',')))",
			relation[0], predicate, relation[1], len(*args),
		), nil
	case "created_by_id", "state_id", "project_id":
		ids, err := issueFilterUUIDs(values, field)
		if err != nil {
			return "", err
		}
		if len(ids) == 0 {
			return "FALSE", nil
		}
		*args = append(*args, strings.Join(ids, ","))
		column := map[string]string{
			"created_by_id": "issue.created_by_id",
			"state_id":      "issue.state_id",
			"project_id":    "issue.project_id",
		}[baseField]
		return fmt.Sprintf("%s::text = ANY(string_to_array($%d, ','))", column, len(*args)), nil
	case "state_group", "priority":
		column := "issue.priority"
		if baseField == "state_group" {
			column = `(SELECT complex_state."group" FROM states complex_state WHERE complex_state.id = issue.state_id)`
		}
		normalized := strings.TrimSpace(values[0])
		if inLookup {
			components := strings.Split(values[0], ",")
			for index := range components {
				components[index] = strings.TrimSpace(components[index])
			}
			normalized = strings.Join(components, ",")
		}
		*args = append(*args, normalized)
		if inLookup {
			return fmt.Sprintf("%s = ANY(string_to_array($%d, ','))", column, len(*args)), nil
		}
		return fmt.Sprintf("%s = $%d", column, len(*args)), nil
	case "is_archived", "is_draft":
		boolean, ok := issueFilterBoolean(values[0])
		if !ok {
			if baseField == "is_draft" {
				return "issue.is_draft IS NULL", nil
			}
			return "", nil
		}
		if baseField == "is_archived" {
			if boolean {
				return "issue.archived_at IS NOT NULL", nil
			}
			return "issue.archived_at IS NULL", nil
		}
		*args = append(*args, boolean)
		return fmt.Sprintf("issue.is_draft = $%d", len(*args)), nil
	case "created_at", "updated_at", "start_date", "target_date":
		column := map[string]string{
			"created_at":  RequesterDateSQL("issue.created_at"),
			"updated_at":  RequesterDateSQL("issue.updated_at"),
			"start_date":  "issue.start_date",
			"target_date": "issue.target_date",
		}[baseField]
		if rangeLookup {
			if len(values) == 1 {
				values = strings.Split(values[0], ",")
			}
			for _, candidate := range values {
				if candidate != "" && !issueFilterDate(candidate) {
					return "", invalidIssueFilterset(field, "Enter a valid date.")
				}
			}
			if len(values) != 2 {
				return "", ErrInvalidComplexRange
			}
			if values[0] == "" || values[1] == "" {
				return "FALSE", nil
			}
			*args = append(*args, values[0], values[1])
			return fmt.Sprintf("%s BETWEEN $%d::date AND $%d::date", column, len(*args)-1, len(*args)), nil
		}
		if !issueFilterDate(values[0]) {
			return "", invalidIssueFilterset(field, "Enter a valid date.")
		}
		*args = append(*args, values[0])
		return fmt.Sprintf("%s = $%d::date", column, len(*args)), nil
	}
	_ = inLookup
	return "", nil
}

func issueFilterValues(value any) []string {
	if list, ok := value.([]any); ok {
		if len(list) == 0 {
			return nil
		}
		if list[len(list)-1] == nil {
			return []string{"None"}
		}
		return []string{issueFilterScalarValue(list[len(list)-1])}
	}
	if value == nil {
		return []string{""}
	}
	return []string{issueFilterScalarValue(value)}
}

func issueFilterScalarValue(value any) string {
	if number, ok := value.(json.Number); ok && number.String() == "-0" {
		return "0"
	}
	result := fmt.Sprint(value)
	switch result {
	case "\x00python-json:NaN":
		return "nan"
	case "\x00python-json:Infinity":
		return "inf"
	case "\x00python-json:-Infinity":
		return "-inf"
	default:
		return result
	}
}

func issueFilterUUIDs(values []string, field string) ([]string, error) {
	if strings.HasSuffix(field, "__in") && len(values) == 1 && strings.Contains(values[0], ",") {
		values = strings.Split(values[0], ",")
		filtered := values[:0]
		for _, value := range values {
			if value != "" {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		identifier, err := uuid.Parse(value)
		if err != nil {
			return nil, invalidIssueFilterset(field, "Enter a valid UUID.")
		}
		normalized[index] = identifier.String()
	}
	return normalized, nil
}

// ParseUUIDFilterValues normalizes UUID filter values while preserving the
// Django-compatible comma-separated `__in` form.
func ParseUUIDFilterValues(values []string, field string) ([]string, error) {
	return issueFilterUUIDs(values, field)
}

func issueFilterBoolean(value string) (bool, bool) {
	switch strings.ToLower(value) {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}

func issueFilterDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func isIssueFilterScalar(value any) bool {
	switch value.(type) {
	case nil, string, json.Number, bool:
		return true
	default:
		return false
	}
}

func invalidIssueFilterset(field, message string) error {
	return &FilterError{
		Message: "Invalid filter parameters",
		Code:    "invalid_filterset",
		Errors:  map[string]any{field: []string{message}},
	}
}

func newFilterError(message, code string) error {
	return &FilterError{Message: message, Code: code}
}

func asciiDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func ValidUUIDFilter(value string, allowNone bool) (string, bool) {
	values := strings.Split(value, ",")
	valid := make([]string, 0, len(values))
	foundNone := false
	for _, candidate := range values {
		if allowNone && candidate == "None" {
			foundNone = true
			continue
		}
		if candidate == "" || candidate == "null" {
			continue
		}
		identifier, err := uuid.Parse(candidate)
		if err == nil {
			valid = append(valid, identifier.String())
		}
	}
	return strings.Join(valid, ","), foundNone
}

func FilterNullTokens(value string) string {
	values := strings.Split(value, ",")
	filtered := values[:0]
	for _, candidate := range values {
		if candidate == "" {
			return ""
		}
		if candidate != "null" {
			filtered = append(filtered, candidate)
		}
	}
	return strings.Join(filtered, ",")
}

func EscapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func NormalizeLegacyUUIDList(value string) (string, bool) {
	value = FilterNullTokens(value)
	if value == "" {
		return "", true
	}
	values := strings.Split(value, ",")
	for index, candidate := range values {
		identifier, err := uuid.Parse(candidate)
		if err != nil {
			return "", false
		}
		values[index] = identifier.String()
	}
	return strings.Join(values, ","), true
}

func NormalizeLegacyIntegerList(value string) (string, bool) {
	value = FilterNullTokens(value)
	if value == "" {
		return "", true
	}
	values := strings.Split(value, ",")
	for index, candidate := range values {
		parsed, err := strconv.Atoi(strings.TrimSpace(candidate))
		if err != nil {
			return "", false
		}
		values[index] = strconv.Itoa(parsed)
	}
	return strings.Join(values, ","), true
}

func ValidLegacyDateBounds(value string) bool {
	filters := strings.Split(value, ",")
	for _, filter := range filters {
		if filter == "" {
			return true
		}
	}
	for _, filter := range filters {
		parts := strings.Split(filter, ";")
		if len(parts) <= 1 {
			continue
		}
		duration := strings.SplitN(parts[0], "_", 2)
		if len(duration) == 2 && asciiDigits(duration[0]) && (duration[1] == "weeks" || duration[1] == "months") {
			continue
		}
		if !issueFilterDate(parts[0]) {
			return false
		}
	}
	return true
}

func order(value string, present bool) string {
	if !present {
		return "issue.created_at ASC"
	}
	switch value {
	case "created_at":
		return "issue.created_at ASC"
	case "-created_at":
		return "issue.created_at DESC"
	case "updated_at":
		return "issue.updated_at ASC NULLS LAST"
	case "-updated_at":

		return "issue.updated_at DESC NULLS LAST"
	case "sequence_id":
		return "issue.sequence_id ASC NULLS LAST"
	case "-sequence_id":
		return "issue.sequence_id DESC NULLS LAST"
	case "sort_order":
		return "issue.sort_order ASC NULLS LAST"
	case "-sort_order":
		return "issue.sort_order DESC NULLS LAST"
	case "target_date":
		return "issue.target_date ASC NULLS LAST"
	case "-target_date":
		return "issue.target_date DESC NULLS LAST"
	case "start_date":
		return "issue.start_date ASC NULLS LAST"
	case "-start_date":
		return "issue.start_date DESC NULLS LAST"
	case "completed_at":
		return "issue.completed_at ASC NULLS LAST"
	case "-completed_at":
		return "issue.completed_at DESC NULLS LAST"
	case "archived_at":
		return "issue.archived_at ASC NULLS LAST"
	case "-archived_at":
		return "issue.archived_at DESC NULLS LAST"
	case "priority":
		return "CASE issue.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 WHEN 'none' THEN 4 ELSE 5 END DESC"
	case "-priority":
		return "CASE issue.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 WHEN 'none' THEN 4 ELSE 5 END ASC"
	case "state__group":
		return "CASE (SELECT state.\"group\" FROM states state WHERE state.id = issue.state_id) WHEN 'backlog' THEN 0 WHEN 'unstarted' THEN 1 WHEN 'started' THEN 2 WHEN 'completed' THEN 3 WHEN 'cancelled' THEN 4 ELSE 5 END ASC"
	case "-state__group":
		return "CASE (SELECT state.\"group\" FROM states state WHERE state.id = issue.state_id) WHEN 'cancelled' THEN 0 WHEN 'completed' THEN 1 WHEN 'started' THEN 2 WHEN 'unstarted' THEN 3 WHEN 'backlog' THEN 4 ELSE 5 END DESC"
	case "state__name":
		return "(SELECT state.name FROM states state WHERE state.id = issue.state_id) ASC NULLS LAST"
	case "-state__name":
		return "(SELECT state.name FROM states state WHERE state.id = issue.state_id) DESC NULLS LAST"
	case "assignees__first_name":
		return "(SELECT min(users.first_name) FROM issue_assignees assignment JOIN users ON users.id = assignment.assignee_id WHERE assignment.issue_id = issue.id) ASC NULLS LAST"
	case "-assignees__first_name":
		return "(SELECT min(users.first_name) FROM issue_assignees assignment JOIN users ON users.id = assignment.assignee_id WHERE assignment.issue_id = issue.id) DESC NULLS LAST"
	case "labels__name":
		return "(SELECT min(labels.name) FROM issue_labels assignment JOIN labels ON labels.id = assignment.label_id WHERE " + LiveIssueLabelPredicate("issue", "assignment") + ") ASC NULLS LAST"
	case "-labels__name":
		return "(SELECT min(labels.name) FROM issue_labels assignment JOIN labels ON labels.id = assignment.label_id WHERE " + LiveIssueLabelPredicate("issue", "assignment") + ") DESC NULLS LAST"
	case "issue_module__module__name":
		return "(SELECT modules.name FROM modules WHERE modules.id = $4) ASC NULLS LAST"
	case "-issue_module__module__name":
		return "(SELECT modules.name FROM modules WHERE modules.id = $4) DESC NULLS LAST"
	default:
		return "issue.created_at DESC"
	}
}

// LegacyOrder returns the simple allowlisted issue ordering expression used
// when no relation filters need to constrain the ordering query.
func LegacyOrder(value string, present bool) string {
	return order(value, present)
}

func Order(options Options, assignees, labels, modules string, args *[]any) (string, error) {
	return relationOrder(options, assignees, labels, modules, args, false)
}

// ModuleOrder retains the existing module-list query's $4 module scope when
// no additional relationship filter changes the ordering join.
func ModuleOrder(options Options, assignees, labels, modules string, args *[]any) (string, error) {
	return relationOrder(options, assignees, labels, modules, args, true)
}

func relationOrder(options Options, assignees, labels, modules string, args *[]any, moduleScoped bool) (string, error) {
	present := options.OrderBySet || options.OrderBy != ""
	value := options.OrderBy
	direction := "ASC"
	if strings.HasPrefix(value, "-") {
		direction = "DESC"
	}

	var (
		table, relationColumn, nameJoin, nameExpression string
		values, targetField                             string
		relationPresent                                 bool
	)
	switch value {
	case "assignees__first_name", "-assignees__first_name":
		table, relationColumn = "issue_assignees", "assignee_id"
		nameJoin = "JOIN users order_name ON order_name.id = order_relation.assignee_id"
		nameExpression = "order_name.first_name"
		values, targetField, relationPresent = assignees, "assignee_id", options.AssigneesSet
	case "labels__name", "-labels__name":
		table, relationColumn = "issue_labels", "label_id"
		nameJoin = "JOIN labels order_name ON order_name.id = order_relation.label_id"
		nameExpression = "order_name.name"
		values, targetField, relationPresent = labels, "label_id", options.LabelsSet
	case "issue_module__module__name", "-issue_module__module__name":
		table, relationColumn = "module_issues", "module_id"
		nameJoin = "JOIN modules order_name ON order_name.id = order_relation.module_id"
		nameExpression = "order_name.name"
		values, targetField, relationPresent = modules, "module_id", options.ModuleSet
	default:
		return order(value, present), nil
	}

	joins, predicate, complexConstrained := "", "", false
	var err error
	if !relationPresent && values == "" {
		if options.PropertyFilter != nil {
			joins, predicate, complexConstrained, err = options.PropertyFilter.orderConstraint(targetField, "order_relation", args)
		} else {
			joins, predicate, complexConstrained, err = issueComplexOrderConstraint(options.Filters, targetField, "order_relation", args)
		}
		if err != nil {
			return "", err
		}
	}
	if moduleScoped && targetField == "module_id" && values == "" && !relationPresent && !complexConstrained {
		return order(value, present), nil
	}

	where := []string{"order_relation.issue_id = issue.id"}
	if targetField == "label_id" {
		where = append(where, LiveIssueLabelPredicate("issue", "order_relation"))
	}
	if values != "" || relationPresent {
		where = append(where, "order_relation.deleted_at IS NULL")
	}
	if values != "" {
		*args = append(*args, values)
		where = append(where, fmt.Sprintf(
			"order_relation.%s::text = ANY(string_to_array($%d, ','))",
			relationColumn, len(*args),
		))
	}
	if complexConstrained {
		where = append(where, "("+predicate+")")
	}
	return fmt.Sprintf(
		"(SELECT min(%s) FROM %s order_relation %s %s WHERE %s) %s NULLS LAST",
		nameExpression, table, nameJoin, joins, strings.Join(where, " AND "), direction,
	), nil
}

func issueComplexOrderConstraint(raw, targetField, targetAlias string, args *[]any) (string, string, bool, error) {
	if raw == "" {
		return "", "", false, nil
	}
	decoder := json.NewDecoder(strings.NewReader(normalizePythonJSONConstants(raw)))
	decoder.UseNumber()
	decoded, err := decodeOrderedIssueFilter(decoder)
	if err != nil {
		return "", "", false, err
	}
	object, ok := decoded.(orderedIssueFilterObject)
	if !ok || len(object) == 0 {
		return "", "", false, nil
	}
	joins, aliases, constrained := issueFilterOrderBindings(object, targetField, targetAlias)
	if !constrained {
		return "", "", false, nil
	}
	predicate, err := renderIssueComplexFilterTree(object, args, aliases)
	if err != nil {
		return "", "", false, err
	}
	return joins, predicate, true, nil
}

func issueFilterOrderBindings(object orderedIssueFilterObject, targetField, targetAlias string) (string, map[string]string, bool) {
	order := make([]string, 0)
	seen := make(map[string]bool)
	collectPositiveIssueFilterRelations(object, &order, seen)
	if !seen[targetField] {
		return "", nil, false
	}
	aliases := make(map[string]string, len(order))
	joins := make([]string, 0, len(order)-1)
	for _, field := range order {
		if field == targetField {
			aliases[field] = targetAlias
			continue
		}
		relation, _ := issueComplexRelation(field)
		alias := "order_complex_" + strings.TrimSuffix(field, "_id")
		aliases[field] = alias
		joins = append(joins, "LEFT JOIN "+relation[0]+" "+alias+
			" ON "+alias+".issue_id = issue.id AND "+alias+".deleted_at IS NULL")
	}
	return strings.Join(joins, " "), aliases, true
}

func parseComplexFilter(raw string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(normalizePythonJSONConstants(raw)))
	decoder.UseNumber()
	decoded, err := decodeOrderedIssueFilter(decoder)
	if err != nil {
		return nil, newFilterError(
			"Invalid JSON for 'filter'. Expected a valid JSON object.",
			"invalid_json",
		)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, newFilterError(
			"Invalid JSON for 'filter'. Expected a valid JSON object.",
			"invalid_json",
		)
	}
	return decoded, nil
}
