package issuequery

import (
	"fmt"
	"strings"
	"unicode"
	"uuid"
)

func (c *pqlCompiler) actorParameter(pos int) (string, error) {
	id, err := uuid.Parse(c.request.ActorID)
	if err != nil {
		return "", pqlError(pos, "authenticated actor is required")
	}
	c.args = append(c.args, id.String())
	return fmt.Sprintf("$%d::uuid", len(c.args)), nil
}

func (c *pqlCompiler) hierarchy(e *pqlExpression) (string, error) {
	actor, err := c.actorParameter(e.pos)
	if err != nil {
		return "", err
	}
	switch e.field {
	case "isTopLevel":
		return "(" + VisibleParentSQL("issue", actor) + " IS NULL)", nil
	case "isSubWorkItem":
		return "(" + VisibleParentSQL("issue", actor) + " IS NOT NULL)", nil
	default: // hasChildren, selected by the predicate allowlist.
		return `EXISTS(SELECT 1 FROM issues pql_related WHERE pql_related.parent_id=issue.id AND ` + IssueVisibleSQL("pql_related", actor, "issue.workspace_id") + `)`, nil
	}
}

func (c *pqlCompiler) itemRelation(e *pqlExpression) (string, error) {
	if len(e.values) != 1 || e.values[0].kind != "string" {
		return "", pqlError(e.pos, "relation requires one quoted item identifier")
	}
	identity, err := c.itemIdentity(e.values[0])
	if err != nil {
		return "", err
	}
	actor, err := c.actorParameter(e.pos)
	if err != nil {
		return "", err
	}
	from := `issues pql_related JOIN projects pql_related_project ON pql_related_project.id=pql_related.project_id`
	var relation string
	switch e.field {
	case "childOf":
		relation = "issue.parent_id=pql_related.id"
	case "parentOf":
		relation = "pql_related.parent_id=issue.id"
	default:
		from += ` JOIN issue_relations pql_link ON ` + pqlRelationDirection(e.field)
		relation = `pql_link.deleted_at IS NULL AND pql_link.workspace_id=issue.workspace_id
 AND pql_link.project_id IN (issue.project_id,pql_related.project_id)`
	}
	return `EXISTS(SELECT 1 FROM ` + from + ` WHERE ` + relation + ` AND ` + identity + ` AND ` +
		IssueVisibleSQL("issue", actor, "issue.workspace_id") + ` AND ` +
		IssueVisibleSQL("pql_related", actor, "issue.workspace_id") + `)`, nil
}

func pqlRelationDirection(name string) string {
	switch name {
	case "blockedBy":
		return `pql_link.relation_type='blocked_by' AND pql_link.issue_id=issue.id AND pql_link.related_issue_id=pql_related.id`
	case "blocks":
		return `pql_link.relation_type='blocked_by' AND pql_link.related_issue_id=issue.id AND pql_link.issue_id=pql_related.id`
	default:
		kind := "relates_to"
		if name == "duplicateOf" {
			kind = "duplicate"
		}
		return `pql_link.relation_type='` + kind + `' AND ((pql_link.issue_id=issue.id AND pql_link.related_issue_id=pql_related.id) OR (pql_link.related_issue_id=issue.id AND pql_link.issue_id=pql_related.id))`
	}
}

func (c *pqlCompiler) itemIdentity(v pqlValue) (string, error) {
	if id, err := uuid.Parse(v.text); err == nil {
		c.args = append(c.args, id.String())
		return fmt.Sprintf("pql_related.id=$%d::uuid", len(c.args)), nil
	}
	separator := strings.LastIndexByte(v.text, '-')
	invalid := pqlError(v.pos, "invalid item identifier")
	if separator <= 0 || separator == len(v.text)-1 || strings.ContainsRune(v.text, 0) || strings.ContainsFunc(v.text, unicode.IsSpace) {
		return "", invalid
	}
	sequence := v.text[separator+1:]
	for _, ch := range sequence {
		if ch < '0' || ch > '9' {
			return "", invalid
		}
	}
	c.args = append(c.args, v.text[:separator], sequence)
	return fmt.Sprintf("(pql_related_project.identifier=$%d::text AND pql_related.sequence_id::numeric=$%d::numeric)", len(c.args)-1, len(c.args)), nil
}
