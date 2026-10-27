package issuequery

func (c *pqlCompiler) predicate(e *pqlExpression) (string, error) {
	switch e.field {
	case "childOf", "parentOf", "linkedTo", "blockedBy", "blocks", "duplicateOf":
		return c.itemRelation(e)
	}
	if len(e.values) != 0 {
		return "", pqlError(e.pos, "invalid predicate arguments")
	}
	switch e.field {
	case "isDraft", "isArchived":
		return "(" + pqlFields[e.field].sql + " = TRUE)", nil
	case "hasNoLabel":
		return c.expression(&pqlExpression{field: "label", op: "IS EMPTY", pos: e.pos})
	case "hasNoAssignee":
		return c.expression(&pqlExpression{field: "assignee", op: "IS EMPTY", pos: e.pos})
	case "isTopLevel", "isSubWorkItem", "hasChildren":
		return c.hierarchy(e)
	case "hasStartAndDueDates":
		return "(issue.start_date IS NOT NULL AND issue.target_date IS NOT NULL)", nil
	case "isOverdue":
		return c.expression(&pqlExpression{op: "AND",
			left: &pqlExpression{field: "dueDate", op: "<", pos: e.pos,
				values: []pqlValue{{pqlToken: pqlToken{kind: "call", text: "today", pos: e.pos}}}},
			right: &pqlExpression{field: "stateGroup", op: "IN", pos: e.pos,
				listFunction: &pqlValue{pqlToken: pqlToken{kind: "call", text: "openStates", pos: e.pos}}},
		})
	case "isEpic":
		return "(" + pqlTypeValue("is_epic") + " = TRUE)", nil
	case "isIntake":
		return (pqlCollection{bridge: "intake_issues", target: "intakes", foreignKey: "intake_id"}).exists("TRUE", pqlProjectTargetScope), nil
	default:
		return "", pqlError(e.pos, "predicate is not available")
	}
}
