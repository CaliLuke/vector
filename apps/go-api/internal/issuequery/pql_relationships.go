package issuequery

import "fmt"

// Columns and relationship definitions below are authored SQL, never input.
func pqlStateValue(column string) string {
	return `(SELECT pql_state.` + column + ` FROM states pql_state
 WHERE pql_state.id=issue.state_id AND pql_state.project_id=issue.project_id
 AND pql_state.workspace_id=issue.workspace_id AND pql_state.deleted_at IS NULL)`
}

func pqlTypeValue(column string) string {
	return `(SELECT pql_type.` + column + ` FROM issue_types pql_type
 WHERE pql_type.id=issue.type_id AND pql_type.workspace_id=issue.workspace_id
 AND pql_type.deleted_at IS NULL)`
}

type pqlCollection struct{ bridge, target, foreignKey string }

var pqlCollections = map[string]pqlCollection{
	"label":      {"issue_labels", "labels", "label_id"},
	"cycle":      {"cycle_issues", "cycles", "cycle_id"},
	"module":     {"module_issues", "modules", "module_id"},
	"milestone":  {"milestone_issues", "milestones", "milestone_id"},
	"assignee":   {"issue_assignees", "users", "assignee_id"},
	"mention":    {"issue_mentions", "users", "mention_id"},
	"subscriber": {"issue_subscribers", "users", "subscriber_id"},
}

const pqlProjectTargetScope = `pql_target.project_id=issue.project_id AND pql_target.workspace_id=issue.workspace_id AND pql_target.deleted_at IS NULL`

func (r pqlCollection) exists(condition, targetScope string) string {
	return `EXISTS(SELECT 1 FROM ` + r.bridge + ` pql_bridge
 JOIN ` + r.target + ` pql_target ON pql_target.id=pql_bridge.` + r.foreignKey + `
 WHERE pql_bridge.issue_id=issue.id AND pql_bridge.deleted_at IS NULL
 AND pql_bridge.project_id=issue.project_id AND pql_bridge.workspace_id=issue.workspace_id
 AND (` + targetScope + `) AND (` + condition + `))`
}

func (c *pqlCompiler) collection(e *pqlExpression, relation pqlCollection) (string, error) {
	targetScope := pqlProjectTargetScope
	if relation.target == "users" {
		var err error
		targetScope, err = c.userTargetScope(e.pos)
		if err != nil {
			return "", err
		}
	}
	if relation.target == "milestones" {
		actor, err := c.actorParameter(e.pos)
		if err != nil {
			return "", err
		}
		targetScope += ` AND issue.deleted_at IS NULL AND NOT issue.is_draft AND ` +
			MilestoneReadScopeSQL("issue.project_id", "issue.workspace_id", actor)
	}
	op, negate := e.op, false
	switch op {
	case "IS NULL", "IS EMPTY":
		return "(NOT " + relation.exists("TRUE", targetScope) + ")", nil
	case "IS NOT NULL", "IS NOT EMPTY":
		return relation.exists("TRUE", targetScope), nil
	case "!=":
		op, negate = "=", true
	case "NOT IN":
		op, negate = "IN", true
	case "=", "IN":
	default:
		return "", pqlError(e.pos, "operator is not valid for collection")
	}
	var condition string
	if e.listFunction != nil {
		var err error
		if relation.target == "users" {
			var set string
			set, err = c.userList(e.field, *e.listFunction)
			condition = "pql_target.id IN (" + set + ")"
		} else {
			condition, err = c.cycleList(e.field, *e.listFunction)
		}
		if err != nil {
			return "", err
		}
	} else {
		operands, err := c.operands("uuid", e)
		if err != nil {
			return "", err
		}
		condition = pqlComparison("pql_target.id", op, operands)
	}
	result := relation.exists(condition, targetScope)
	if negate {
		result = "(NOT " + result + ")"
	}
	return result, nil
}

func pqlStateList(field string, v pqlValue) ([]pqlValue, error) {
	groups, ok := map[string][]string{
		"openStates":   {"backlog", "unstarted", "started"},
		"closedStates": {"completed", "cancelled"},
		"activeStates": {"unstarted", "started"},
	}[v.text]
	if field != "stateGroup" || !ok || len(v.arguments) != 0 {
		return nil, pqlError(v.pos, "invalid list function for field")
	}
	values := make([]pqlValue, len(groups))
	for i, group := range groups {
		values[i] = pqlValue{pqlToken: pqlToken{kind: "string", text: group, pos: v.pos}}
	}
	return values, nil
}

func (c *pqlCompiler) cycleList(field string, v pqlValue) (string, error) {
	if field != "cycle" || len(v.arguments) != 0 {
		return "", pqlError(v.pos, "invalid list function for field")
	}
	var comparison string
	var boundaries []string
	switch v.text {
	case "activeCycle":
		comparison = `pql_target.start_date <= $%[1]d::timestamptz AND pql_target.end_date >= $%[2]d::timestamptz`
		boundaries = []string{"endOfDay", "today"}
	case "completedCycles":
		comparison = `pql_target.end_date < $%[1]d::timestamptz`
		boundaries = []string{"today"}
	case "upcomingCycles":
		comparison = `pql_target.start_date > $%[1]d::timestamptz`
		boundaries = []string{"endOfDay"}
	default:
		return "", pqlError(v.pos, "invalid list function for field")
	}
	first := len(c.args) + 1
	for _, name := range boundaries {
		boundary, err := c.operand("timestamptz", "", pqlValue{pqlToken: pqlToken{kind: "call", text: name, pos: v.pos}})
		if err != nil {
			return "", err
		}
		c.args = append(c.args, boundary)
	}
	// The compiler owns timezone resolution, including gaps/folds. Bind instants
	// instead of asking PostgreSQL to reinterpret a Go timezone name or database.
	return fmt.Sprintf(comparison, first, first+1), nil
}
