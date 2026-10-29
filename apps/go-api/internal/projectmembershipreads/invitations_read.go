package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"github.com/jackc/pgx/v5"
)

const inviteWorkspaceSQL = `jsonb_build_object('id',w.id,'name',w.name,'slug',w.slug)||`

func workspaceInviteProjection(public bool) string {
	full := `(to_jsonb(i)-'workspace_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('created_by',i.created_by_id,'updated_by',i.updated_by_id,'workspace',` + inviteWorkspaceSQL + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + `,'invite_link','/workspace-invitations/?invitation_id='||i.id::text||'&slug='||w.slug||'&token='||i.token)`
	if public {
		return `(` + full + `)-'token'-'invite_link'-'deleted_at'-'updated_by'`
	}
	return full
}

func projectInviteProjection(public bool) string {
	project := `jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'description',p.description,'cover_image',p.cover_image,'logo_props',p.logo_props)||` + djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url")
	full := `(to_jsonb(i)-'workspace_id'-'project_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('created_by',i.created_by_id,'updated_by',i.updated_by_id,'workspace',` + inviteWorkspaceSQL + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + `,'project',` + project + `)`
	if public {
		return `(` + full + `)-'token'-'email'-'created_at'-'updated_at'-'deleted_at'-'created_by'-'updated_by'`
	}
	return full
}

func (r *InvitationsRepository) invitationRead(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	project := strings.HasPrefix(c.Operation, "project-") || c.Operation == "user-project-list"
	public := strings.HasSuffix(c.Operation, "-public")
	table, projection, join := "workspace_member_invites", workspaceInviteProjection(public), ""
	if project {
		table = "project_member_invites"
		projection = projectInviteProjection(public)
		join = " JOIN projects p ON p.id=i.project_id"
	}
	where := "i.deleted_at IS NULL"
	args := []any{}
	if strings.HasPrefix(c.Operation, "user-") {
		where += ` AND i.email=(SELECT email FROM users WHERE id=$1)`
		args = append(args, c.UserID)
	} else {
		where += ` AND w.slug=$1`
		args = append(args, c.Slug)
	}
	if project && !strings.HasPrefix(c.Operation, "user-") {
		args = append(args, c.ProjectID)
		where += fmt.Sprintf(" AND i.project_id=$%d", len(args))
	}
	detail := strings.HasSuffix(c.Operation, "-detail") || public
	if detail {
		args = append(args, c.ID)
		where += fmt.Sprintf(" AND i.id=$%d", len(args))
	} else {
		projection = `COALESCE(jsonb_agg(` + projection + ` ORDER BY i.created_at DESC),'[]'::jsonb)`
	}

	var data []byte
	err := r.writer.QueryRow(ctx, `SELECT `+projection+` FROM `+table+` i JOIN workspaces w ON w.id=i.workspace_id`+join+` WHERE `+where, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		if c.Operation == "workspace-detail" {
			return nil, invitationViewsetMissing("WorkspaceMemberInvite")
		}
		if c.Operation == "project-detail" {
			return nil, invitationViewsetMissing("ProjectMemberInvite")
		}
		return nil, inviteMissing()
	}
	return json.RawMessage(data), err
}
