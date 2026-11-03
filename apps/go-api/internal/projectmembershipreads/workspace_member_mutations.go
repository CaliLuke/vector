package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

func workspaceMemberProjection() string {
	return `(to_jsonb(m)-'workspace_id'-'member_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('workspace',m.workspace_id,'created_by',m.created_by_id,'updated_by',m.updated_by_id,'member',jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + apicontract.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `)`
}

func (r *InvitationsRepository) mutateWorkspaceMember(ctx context.Context, c InvitationCommand) (_ json.RawMessage, resultErr error) {
	var newRole *int
	if c.Operation == "member-patch" {
		fields, err := memberFields(c.Body, c.ContentType, "role")
		if err != nil {
			return nil, err
		}
		if value, present := fields["role"]; present {
			role, err := memberRole(value)
			if err != nil {
				return nil, err
			}
			newRole = &role
		}
	}
	m, err := beginMemberMutation(ctx, r.writer, c.Slug)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	actorRole, err := m.actorRole(ctx, c.UserID)
	if err != nil {
		return nil, err
	}
	leave := c.Operation == "member-leave"
	if !leave && actorRole != 20 {
		return nil, inviteFailure(http.StatusForbidden, "Workspace administrator permission is required")
	}
	var id, user string
	var role int
	err = m.tx.QueryRow(ctx, `SELECT m.id::text,m.member_id::text,m.role FROM workspace_members m JOIN users u ON u.id=m.member_id
  WHERE m.workspace_id=$1 AND m.is_active AND m.deleted_at IS NULL AND NOT u.is_bot
  AND (($2 AND m.member_id=$3) OR (NOT $2 AND m.id::text=$4))`, m.workspace, leave, c.UserID, c.ID).Scan(&id, &user, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	if !leave && user == c.UserID {
		return nil, inviteFailure(http.StatusBadRequest, "Use leave workspace to remove your own membership")
	}
	var result json.RawMessage
	if c.Operation == "member-patch" {
		if newRole != nil {
			role = *newRole
			if role == 5 {
				if _, err = m.tx.Exec(ctx, `UPDATE project_members SET role=5,updated_at=clock_timestamp(),updated_by_id=$3 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM projects p WHERE p.id=project_members.project_id AND p.workspace_id=$1)`, m.workspace, user, c.UserID); err != nil {
					return nil, err
				}
			}
			if _, err = m.tx.Exec(ctx, `UPDATE workspace_members SET role=$2,updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, id, role, c.UserID); err != nil {
				return nil, err
			}
		}
		result, err = inviteJSON(ctx, m.tx, `SELECT `+workspaceMemberProjection()+` FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.id=$1`, id)
	} else {
		for _, table := range []string{"workspace_member_invites", "project_member_invites"} {
			if _, err = m.tx.Exec(ctx, `UPDATE `+table+` SET deleted_at=clock_timestamp(),updated_at=clock_timestamp() WHERE workspace_id=$1 AND lower(email)=lower((SELECT email FROM users WHERE id=$2)) AND deleted_at IS NULL`, m.workspace, user); err != nil {
				return nil, err
			}
		}
		if _, err = m.tx.Exec(ctx, `UPDATE project_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$3 WHERE workspace_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM projects p WHERE p.id=project_members.project_id AND p.workspace_id=$1)`, m.workspace, user, c.UserID); err != nil {
			return nil, err
		}
		_, err = m.tx.Exec(ctx, `UPDATE workspace_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, c.UserID)
	}
	if err != nil {
		return nil, err
	}
	return result, m.commit(ctx)
}
