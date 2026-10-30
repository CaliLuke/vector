package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"

	"github.com/jackc/pgx/v5"
)

func invitationMemberPatchFields(c InvitationCommand) (map[string]any, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	// The source evaluates "role" in request.data before constructing the
	// serializer. Lists and strings support membership; the remaining scalar
	// types raise before validation, as does .get on a matching list/string.
	switch input.Kind {
	case "":
	case "str":
		var value string
		_ = json.Unmarshal(input.Raw, &value)
		if strings.Contains(value, "role") {
			return nil, errors.New("invitation data has no get method")
		}
	case "list":
		var values []any
		_ = json.Unmarshal(input.Raw, &values)
		for _, value := range values {
			if text, ok := value.(string); ok && text == "role" {
				return nil, errors.New("invitation data has no get method")
			}
		}
	default:
		return nil, errors.New("invitation data does not support membership")
	}
	return invitationSerializerData(input)
}

func workspaceMemberProjection() string {
	return `(to_jsonb(m)-'workspace_id'-'member_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('workspace',m.workspace_id,'created_by',m.created_by_id,'updated_by',m.updated_by_id,'member',jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `)`
}

func (r *InvitationsRepository) mutateWorkspaceMember(ctx context.Context, c InvitationCommand) (_ json.RawMessage, resultErr error) {
	leave := c.Operation == "member-leave"
	var id, userID, workspace string
	var role int
	query := `SELECT m.id::text,m.member_id::text,m.workspace_id::text,m.role FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id JOIN users u ON u.id=m.member_id WHERE w.slug=$1 AND m.deleted_at IS NULL AND m.is_active`
	args := []any{c.Slug}
	if leave {
		query += ` AND m.member_id=$2`
		args = append(args, c.UserID)
	} else {
		query += ` AND m.id=$2 AND NOT u.is_bot`
		args = append(args, c.ID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&id, &userID, &workspace, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	if c.Operation == "member-patch" {
		if userID == c.UserID {
			return nil, inviteFailure(400, "You cannot update your own role")
		}
		fields, memberUpdateErr := invitationMemberPatchFields(c)
		if memberUpdateErr != nil {
			return nil, memberUpdateErr
		}
		// Keep source int coercion before serializer validation, but make validated
		// role and every cascade one transaction (ACCESS-001).
		cascade := false
		if raw, ok := fields["role"]; ok {
			n, e := invitationInt(raw)
			if e != nil {
				return nil, e
			}
			cascade = n == 5
		}
		updates, args, fieldError := workspaceMemberUpdates(fields, id, c.UserID, c.Timezone)
		memberUpdateErr = invitationValidationErrors(fieldError, r.invitationRelationErrors(ctx, fields, workspaceMembershipFields))
		if memberUpdateErr != nil {
			return nil, memberUpdateErr
		}
		tx, memberUpdateErr := r.writer.Begin(ctx)
		if memberUpdateErr != nil {
			return nil, memberUpdateErr
		}
		// Cleanup keeps its own deadline and retains failures alongside the operation error.
		defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
		if cascade {
			if _, memberUpdateErr = tx.Exec(ctx, `UPDATE project_members SET role=5 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, workspace, userID); memberUpdateErr != nil {
				return nil, memberUpdateErr
			}
		}
		if _, memberUpdateErr = tx.Exec(ctx, `UPDATE workspace_members SET `+strings.Join(updates, ",")+` WHERE id=$1`, args...); memberUpdateErr != nil {
			return nil, memberUpdateErr
		}
		if memberUpdateErr = tx.Commit(ctx); memberUpdateErr != nil {
			return nil, memberUpdateErr
		}
		return inviteJSON(ctx, r.writer, `SELECT `+workspaceMemberProjection()+` FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.id=$1`, id)
	}
	if !leave {
		if userID == c.UserID {
			return nil, inviteFailure(400, "You cannot remove yourself from the workspace. Please use leave workspace")
		}
		var actorRole int
		if err = r.writer.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, workspace, c.UserID).Scan(&actorRole); err != nil {
			return nil, err
		}
		if actorRole < role {
			return nil, inviteFailure(400, "You cannot remove a user having role higher than you")
		}
	} else if role == 20 {
		var admins int
		if err = r.writer.QueryRow(ctx, `SELECT count(*) FROM workspace_members WHERE workspace_id=$1 AND role=20 AND is_active AND deleted_at IS NULL`, workspace).Scan(&admins); err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, inviteFailure(400, "You cannot leave the workspace as you are the only admin of the workspace you will have to either delete the workspace or promote another user to admin.")
		}
	}
	var sole bool
	// Reverse relation counts include inactive and deleted member rows; the
	// Project manager alone excludes deleted projects, as the source annotation.
	err = r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id WHERE p.workspace_id=$1 AND p.deleted_at IS NULL GROUP BY p.id HAVING count(*)=1 AND count(*) FILTER(WHERE pm.member_id=$2 AND pm.role=20)=1)`, workspace, userID).Scan(&sole)
	if err != nil {
		return nil, err
	}
	if sole {
		if leave {
			return nil, inviteFailure(400, "You are a part of some projects where you are the only admin, you should either leave the project or promote another user to admin.")
		}
		return nil, inviteFailure(400, "User is a part of some projects where they are the only admin, they should either leave that project or promote another user to admin.")
	}
	if _, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=false,updated_at=clock_timestamp() WHERE workspace_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, workspace, userID); err != nil {
		return nil, err
	}
	_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, c.UserID)
	return nil, err
}

var workspaceMembershipFields = []string{"role", "company_role", "is_active", "view_props", "default_props", "issue_props", "getting_started_checklist", "tips", "explored_features", "workspace", "created_by", "updated_by", "deleted_at"}

func workspaceMemberUpdates(fields map[string]any, id, actor, zone string) ([]string, []any, error) {
	return invitationFieldUpdates(fields, id, actor, zone, workspaceMembershipFields)
}
