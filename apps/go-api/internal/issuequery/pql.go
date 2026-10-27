package issuequery

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"uuid"
)

// PQLContext binds relative values to one authenticated request. SQL predicates
// do not authorize candidates: repositories must AND them with their scope.
type PQLContext struct {
	ActorID, Timezone string
	Now               time.Time
}

// BuildPQL compiles the executable subset documented in docs/pql.md against
// the trusted issue/project SQL aliases. On failure args is unchanged, including
// its backing array. A blank query returns no predicate.
func BuildPQL(raw string, request PQLContext, args *[]any) (string, error) {
	root, err := parsePQL(raw)
	if err != nil || root == nil {
		return "", err
	}
	zone := request.Timezone
	if zone == "" {
		zone = "UTC"
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return "", pqlError(0, "invalid timezone")
	}
	compiler := pqlCompiler{request: request, location: location, args: slices.Clone(*args)}
	result, err := compiler.expression(root)
	if err != nil {
		return "", err
	}
	*args = compiler.args
	return result, nil
}

type pqlCompiler struct {
	request  PQLContext
	location *time.Location
	args     []any
}

type pqlField struct{ sql, kind string }

var pqlFields = map[string]pqlField{
	"priority":   {"issue.priority", "priority"},
	"title":      {"issue.name", "text"},
	"id":         {"(project.identifier || '-' || issue.sequence_id::text)", "text"},
	"isDraft":    {"issue.is_draft", "boolean"},
	"isArchived": {"(issue.archived_at IS NOT NULL)", "boolean"},
	"dueDate":    {"issue.target_date", "date"},
	"startDate":  {"issue.start_date", "date"},
	"createdAt":  {"issue.created_at", "timestamptz"},
	"updatedAt":  {"issue.updated_at", "timestamptz"},
	"project":    {"issue.project_id", "uuid"},
	"createdBy":  {"issue.created_by_id", "uuid"},
}

func (c *pqlCompiler) expression(e *pqlExpression) (string, error) {
	if e.op == "AND" || e.op == "OR" || e.op == "NOT" {
		left, err := c.expression(e.left)
		if err != nil {
			return "", err
		}
		if e.op == "NOT" {
			return "(NOT " + left + ")", nil
		}
		right, err := c.expression(e.right)
		if err != nil {
			return "", err
		}
		return "(" + left + " " + e.op + " " + right + ")", nil
	}
	field, ok := pqlFields[e.field]
	if !ok {
		return "", pqlError(e.pos, "field or predicate is not available")
	}
	if e.op == "predicate" {
		if (e.field != "isDraft" && e.field != "isArchived") || len(e.values) != 0 {
			return "", pqlError(e.pos, "invalid predicate or arguments")
		}
		return "(" + field.sql + " = TRUE)", nil
	}
	if !pqlOperatorAllowed(field.kind, e.op) {
		return "", pqlError(e.pos, "operator is not valid for field type")
	}
	switch e.op {
	case "IS NULL", "IS NOT NULL":
		return "(" + field.sql + " " + e.op + ")", nil
	case "IS EMPTY":
		return "(" + field.sql + " = '')", nil
	case "IS NOT EMPTY":
		return "(" + field.sql + " <> '')", nil
	}
	operands := make([]string, len(e.values))
	for i, v := range e.values {
		value, err := c.operand(field.kind, e.field, v)
		if err != nil {
			return "", err
		}
		if e.op == "~" {
			value = EscapeLikePattern(value.(string)) + "%"
			if e.field != "id" {
				value = "%" + value.(string)
			}
		}
		c.args = append(c.args, value)
		cast := field.kind
		if cast == "priority" {
			cast = "text"
		}
		operands[i] = fmt.Sprintf("$%d::%s", len(c.args), cast)
	}
	return pqlComparison(field.sql, e.op, operands), nil
}

func pqlComparison(column, op string, operands []string) string {
	switch op {
	case "BETWEEN":
		return "(" + column + " BETWEEN " + operands[0] + " AND " + operands[1] + ")"
	case "IN", "NOT IN":
		return "(" + column + " " + op + " (" + strings.Join(operands, ",") + "))"
	case "~":
		return "(" + column + " ILIKE " + operands[0] + ` ESCAPE '\')`
	default:
		return "(" + column + " " + op + " " + operands[0] + ")"
	}
}

func pqlOperatorAllowed(kind, op string) bool {
	switch op {
	case "=", "!=", "IS NULL", "IS NOT NULL":
		return true
	case "IN", "NOT IN":
		return kind == "text" || kind == "priority" || kind == "uuid"
	case "IS EMPTY", "IS NOT EMPTY", "~":
		return kind == "text"
	case ">", ">=", "<", "<=", "BETWEEN":
		return kind == "date" || kind == "timestamptz"
	}
	return false
}

func (c *pqlCompiler) operand(kind, field string, v pqlValue) (any, error) {
	invalid := pqlError(v.pos, "invalid value or function for field type")
	switch kind {
	case "text", "priority":
		if v.kind != "string" || strings.ContainsRune(v.text, 0) {
			return nil, invalid
		}
		if kind == "priority" && !slices.Contains([]string{"urgent", "high", "medium", "low", "none"}, v.text) {
			return nil, invalid
		}
		return v.text, nil
	case "boolean":
		if v.kind != "boolean" {
			return nil, invalid
		}
		return v.text == "true", nil
	case "uuid":
		text := v.text
		if v.kind == "call" && field == "createdBy" && v.text == "currentUser" && len(v.arguments) == 0 {
			text = c.request.ActorID
		} else if v.kind != "string" {
			return nil, invalid
		}
		id, err := uuid.Parse(text)
		if err != nil {
			return nil, invalid
		}
		return id.String(), nil
	case "date", "timestamptz":
		value, err := c.dateValue(v, kind == "date")
		if err != nil {
			return nil, err
		}
		if kind == "date" {
			return value.In(c.location).Format(time.DateOnly), nil
		}
		return value.Truncate(time.Microsecond), nil
	}
	return nil, invalid
}
