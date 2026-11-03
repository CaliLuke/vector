package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/requestinput"
	"github.com/jackc/pgx/v5"
)

func (r *InvitationsRepository) invitationJoin(ctx context.Context, c InvitationCommand) (_ json.RawMessage, resultErr error) {
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	accepted, ok := fields["accepted"].(bool)
	if !ok {
		return nil, inviteFailure(400, "`accepted` must be a boolean")
	}
	token, _ := fields["token"].(string)
	if token == "" {
		return nil, inviteFailure(403, "Invitation token is required")
	}
	m, err := beginInvitationMemberships(ctx, r.writer, c.UserID)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	table, kind, projectColumn := "workspace_member_invites", "Workspace", "''"
	project := c.Operation == "project-join"
	if project {
		table, kind, projectColumn = "project_member_invites", "Project", "i.project_id::text"
	}
	var i membershipInvitation
	query := `SELECT i.id::text,i.workspace_id::text,w.slug,` + projectColumn + `,i.email,i.token,i.role,i.responded_at FROM ` + table + ` i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND w.slug=$2 AND i.deleted_at IS NULL AND w.deleted_at IS NULL`
	args := []any{c.ID, c.Slug}
	if project {
		query += ` AND i.project_id=$3`
		args = append(args, c.ProjectID)
	}
	err = m.tx.QueryRow(ctx, query+` FOR UPDATE OF i`, args...).Scan(&i.id, &i.workspace, &i.slug, &i.project, &i.email, &i.token, &i.role, &i.responded)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	if token != i.token || !strings.EqualFold(i.email, m.email) {
		return nil, inviteFailure(403, "You do not have permission to accept this invitation")
	}
	if i.responded != nil {
		return nil, inviteFailure(400, "You have already responded to the invitation request")
	}
	message := kind + " Invitation was not accepted"
	if accepted {
		if err = m.join(ctx, i); err != nil {
			return nil, err
		}
		err = m.consume(ctx, i)
		message = kind + " Invitation Accepted"
	} else {
		_, err = m.tx.Exec(ctx, `UPDATE `+table+` SET accepted=false,responded_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, i.id, c.UserID)
	}
	if err != nil {
		return nil, err
	}
	result, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return nil, err
	}
	return result, m.tx.Commit(ctx)
}

func invitationIDs(value any) ([]string, error) {
	raw, ok := value.([]any)
	if !ok {
		return nil, inviteFailure(400, "IDs must be an array")
	}
	ids := make([]string, 0, len(raw))
	for _, id := range raw {
		u, err := requestinput.UUID(id)
		if err != nil {
			return nil, inviteFailure(400, "Invalid input.")
		}
		ids = append(ids, u.String())
	}
	return ids, nil
}

func (r *InvitationsRepository) acceptWorkspaceInvites(ctx context.Context, c InvitationCommand, fields map[string]any) (resultErr error) {
	raw, exists := fields["invitations"]
	if !exists {
		raw = []any{}
	}
	ids, err := invitationIDs(raw)
	if err != nil {
		return err
	}
	m, err := beginInvitationMemberships(ctx, r.writer, c.UserID)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	invites, err := m.invitations(ctx, false, false, ids)
	if err != nil {
		return err
	}
	for _, i := range invites {
		if err = m.join(ctx, i); err != nil {
			return err
		}
		if err = m.consume(ctx, i); err != nil {
			return err
		}
	}
	return m.tx.Commit(ctx)
}

func (r *InvitationsRepository) joinUserProjects(ctx context.Context, c InvitationCommand, fields map[string]any) (_ json.RawMessage, resultErr error) {
	raw, exists := fields["project_ids"]
	if !exists {
		raw = []any{}
	}
	ids, err := invitationIDs(raw)
	if err != nil {
		return nil, err
	}
	m, err := beginMemberMutation(ctx, r.writer, c.Slug)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	role, err := m.actorRole(ctx, c.UserID)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		var network int
		err = m.tx.QueryRow(ctx, `SELECT network FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE`, id, m.workspace).Scan(&network)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, inviteMissing()
		}
		if err != nil {
			return nil, err
		}
		if network == 0 && role != 20 {
			return nil, inviteFailure(403, "Only workspace admins can join private project")
		}
		join := invitationMemberships{tx: m.tx, actor: c.UserID}
		if err = join.join(ctx, membershipInvitation{workspace: m.workspace, slug: c.Slug, project: id, role: role}); err != nil {
			return nil, err
		}
	}
	return json.RawMessage(`{"message":"Projects joined successfully"}`), m.commit(ctx)
}
