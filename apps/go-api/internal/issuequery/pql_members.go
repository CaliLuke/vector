package issuequery

import (
	"fmt"
	"strings"
	"uuid"
)

func pqlUserField(field string) bool {
	return field == "createdBy" || field == "assignee" || field == "mention" || field == "subscriber"
}

// All expressions and role lists passed to these helpers are authored SQL.
func pqlWorkspaceRosterGuard(actor, workspace, roles string) string {
	return `EXISTS(SELECT 1 FROM workspace_members pql_actor_member
 JOIN users pql_actor_user ON pql_actor_user.id=pql_actor_member.member_id
 JOIN workspaces pql_actor_workspace ON pql_actor_workspace.id=pql_actor_member.workspace_id
 WHERE pql_actor_workspace.id=` + workspace + ` AND pql_actor_workspace.deleted_at IS NULL
 AND pql_actor_member.member_id=` + actor + ` AND pql_actor_user.is_active
 AND pql_actor_member.is_active AND pql_actor_member.deleted_at IS NULL
 AND pql_actor_member.role IN (` + roles + `))`
}

func pqlProjectRosterGuard(actor, project, workspace string) string {
	return `(` + pqlWorkspaceRosterGuard(actor, workspace, "5,15,20") + ` AND EXISTS(
 SELECT 1 FROM projects pql_roster_project JOIN project_members pql_roster_actor
 ON pql_roster_actor.project_id=pql_roster_project.id AND pql_roster_actor.workspace_id=pql_roster_project.workspace_id
 WHERE pql_roster_project.id=` + project + ` AND pql_roster_project.workspace_id=` + workspace + `
 AND pql_roster_project.deleted_at IS NULL AND pql_roster_actor.member_id=` + actor + `
 AND pql_roster_actor.is_active AND pql_roster_actor.deleted_at IS NULL))`
}

func (c *pqlCompiler) userTargetScope(pos int) (string, error) {
	actor, err := c.actorParameter(pos)
	if err != nil {
		return "", err
	}
	project := pqlProjectRosterGuard(actor, "issue.project_id", "issue.workspace_id")
	workspace := pqlWorkspaceRosterGuard(actor, "issue.workspace_id", "15,20")
	return `((` + project + ` AND EXISTS(SELECT 1 FROM project_members pql_target_member
 WHERE pql_target_member.member_id=pql_target.id AND pql_target_member.project_id=issue.project_id
 AND pql_target_member.workspace_id=issue.workspace_id AND pql_target_member.deleted_at IS NULL))
 OR (` + workspace + ` AND EXISTS(SELECT 1 FROM workspace_members pql_target_member
 WHERE pql_target_member.member_id=pql_target.id AND pql_target_member.workspace_id=issue.workspace_id
 AND pql_target_member.deleted_at IS NULL)))`, nil
}

func (c *pqlCompiler) userList(field string, v pqlValue) (string, error) {
	invalid := pqlError(v.pos, "invalid list function for field")
	if !pqlUserField(field) {
		return "", invalid
	}
	actor, err := c.actorParameter(v.pos)
	if err != nil {
		return "", err
	}
	switch v.text {
	case "workspaceMembers":
		if len(v.arguments) != 0 {
			return "", invalid
		}
		return `SELECT pql_member_user.id FROM users pql_member_user
 JOIN workspace_members pql_list_member ON pql_list_member.member_id=pql_member_user.id
 WHERE pql_list_member.workspace_id=issue.workspace_id AND pql_list_member.deleted_at IS NULL AND ` +
			pqlWorkspaceRosterGuard(actor, "issue.workspace_id", "15,20"), nil
	case "membersOf":
		if len(v.arguments) != 1 || v.arguments[0].kind != "string" || !strings.HasPrefix(v.arguments[0].text, "project:") {
			return "", invalid
		}
		id, err := uuid.Parse(strings.TrimPrefix(v.arguments[0].text, "project:"))
		if err != nil {
			return "", invalid
		}
		c.args = append(c.args, id.String())
		project := fmt.Sprintf("$%d::uuid", len(c.args))
		return `SELECT pql_member_user.id FROM users pql_member_user
 JOIN project_members pql_list_member ON pql_list_member.member_id=pql_member_user.id
 WHERE pql_list_member.workspace_id=issue.workspace_id AND pql_list_member.deleted_at IS NULL
 AND pql_list_member.project_id=` + project + ` AND ` + pqlProjectRosterGuard(actor, project, "issue.workspace_id"), nil
	default:
		return "", invalid
	}
}

func (c *pqlCompiler) scalarUserList(e *pqlExpression, column string) (string, error) {
	set, err := c.userList(e.field, *e.listFunction)
	if err != nil {
		return "", err
	}
	// PostgreSQL treats NULL IN an empty subquery as false. Preserve the scalar
	// unknown-value contract even when a requested roster is empty or invisible.
	return `(CASE WHEN ` + column + ` IS NULL THEN NULL ELSE ` + column + ` ` + e.op + ` (` + set + `) END)`, nil
}
