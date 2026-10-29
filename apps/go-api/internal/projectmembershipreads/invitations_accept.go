package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *InvitationsRepository) invitationJoin(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	project := c.Operation == "project-join"
	table, kind := "workspace_member_invites", "workspace"
	if project {
		table = "project_member_invites"
		kind = "project"
	}
	var token, email, workspaceID string
	var role int
	var responded *time.Time
	args := []any{c.Slug, c.ID}
	query := `SELECT i.token,i.email,i.workspace_id::text,i.role,i.responded_at FROM ` + table + ` i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.id=$2 AND i.deleted_at IS NULL`
	if project {
		query += ` AND i.project_id=$3`
		args = append(args, c.ProjectID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&token, &email, &workspaceID, &role, &responded)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, inviteMissing()
	}
	if err != nil {
		return nil, err
	}
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	submitted, _ := fields["token"].(string)
	if submitted == "" || submitted != token {
		return nil, inviteFailure(403, "You do not have permission to join the "+kind)
	}
	if c.UserID == "" {
		return nil, inviteFailure(401, "Authentication required to accept "+kind+" invitation")
	}
	var actorEmail string
	if err = r.writer.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, c.UserID).Scan(&actorEmail); err != nil {
		return nil, err
	}
	if !strings.EqualFold(email, actorEmail) {
		return nil, inviteFailure(403, "You do not have permission to accept this invitation")
	}
	if responded != nil {
		return nil, inviteFailure(400, "You have already responded to the invitation request")
	}
	accepted, storedAccepted := false, false
	if raw, present := fields["accepted"]; present {
		if project {
			var valid bool
			accepted, valid = raw.(bool)
			if !valid {
				return nil, inviteFailure(400, "`accepted` must be a boolean")
			}
			storedAccepted = accepted
		} else {
			accepted = invitationTruth(raw)
			var valid bool
			storedAccepted, valid = invitationModelBoolean(raw)
			if !valid {
				return nil, inviteFailure(400, "Please provide valid detail")
			}
		}
	}
	if _, err = r.writer.Exec(ctx, `UPDATE `+table+` SET accepted=$2,responded_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, c.ID, storedAccepted, c.UserID); err != nil {
		return nil, err
	}
	if !accepted {
		if project {
			return json.RawMessage(`{"message":"Project Invitation was not accepted"}`), nil
		}
		return json.RawMessage(`{"message":"Workspace Invitation was not accepted"}`), nil
	}
	userID := c.UserID
	if !project {
		err = r.writer.QueryRow(ctx, `SELECT id::text FROM users WHERE email=$1 ORDER BY created_at ASC LIMIT 1`, email).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return json.RawMessage(`{"message":"Workspace Invitation Accepted"}`), nil
		}
		if err != nil {
			return nil, err
		}
	}
	var membership string
	err = r.writer.QueryRow(ctx, `SELECT id::text FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`, workspaceID, userID).Scan(&membership)
	if errors.Is(err, pgx.ErrNoRows) {
		workspaceRole := role
		if project && role >= 15 {
			workspaceRole = 15
		}
		err = insertWorkspaceMembership(ctx, r.writer, workspaceID, userID, workspaceRole, c.UserID)
	} else if err == nil {
		if project {
			_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, membership, c.UserID)
		} else {
			_, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,role=$2,updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, membership, role, c.UserID)
		}
	}
	if err != nil {
		return nil, err
	}
	if project {
		var target string
		err = r.writer.QueryRow(ctx, `SELECT id::text FROM project_members WHERE workspace_id=$1 AND project_id=$2 AND member_id=$3 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`, workspaceID, c.ProjectID, c.UserID).Scan(&target)
		if errors.Is(err, pgx.ErrNoRows) {
			err = insertProjectMembership(ctx, r.writer, c.ProjectID, workspaceID, c.UserID, role, c.UserID, true)
		} else if err == nil {
			_, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=true,updated_at=clock_timestamp(),updated_by_id=$2,workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id) WHERE id=$1`, target, c.UserID)
		}
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"message":"Project Invitation Accepted"}`), nil
	}
	if _, err = r.writer.Exec(ctx, `UPDATE users SET email=lower(btrim(email)),display_name=CASE WHEN display_name='' THEN split_part(lower(btrim(email)),'@',1) ELSE display_name END,is_staff=(is_staff OR is_superuser),token=CASE WHEN token_updated_at IS NOT NULL THEN replace(gen_random_uuid()::text,'-','')||replace(gen_random_uuid()::text,'-','') ELSE token END,token_updated_at=CASE WHEN token_updated_at IS NOT NULL THEN clock_timestamp() ELSE NULL END,updated_at=clock_timestamp() WHERE id=$1`, userID); err != nil {
		return nil, err
	}
	if err = r.trackInvitationEvent(ctx, userID, c.Slug, workspaceID, "user_joined_workspace", map[string]any{"role": role, "joined_at": time.Now().UTC().Format("2006-01-02 15:04:05.999999+00:00")}); err != nil {
		return nil, err
	}
	return json.RawMessage(`{"message":"Workspace Invitation Accepted"}`), r.invitationDelete(ctx, InvitationCommand{Operation: "workspace-delete", Slug: c.Slug, ID: c.ID, UserID: c.UserID})
}

func invitationIDs(value any) ([]string, error) {
	raw, ok := value.([]any)
	if value == nil {
		return nil, errors.New("invitation ids must be iterable")
	}
	if !ok {
		return nil, errors.New("invalid id list")
	}
	ids := make([]string, 0, len(raw))
	for _, id := range raw {
		u, err := djangoinput.UUID(id)
		if err != nil {
			return nil, inviteFailure(400, "Invalid input.")
		}
		ids = append(ids, u.String())
	}
	return ids, nil
}

func (r *InvitationsRepository) acceptWorkspaceInvites(ctx context.Context, c InvitationCommand, fields map[string]any) error {
	rawIDs, exists := fields["invitations"]
	if !exists {
		rawIDs = []any{}
	}
	ids, err := invitationIDs(rawIDs)
	if err != nil {
		return err
	}
	rows, err := r.writer.Query(ctx, `SELECT i.id::text,i.workspace_id::text,w.slug,i.role FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id::text=ANY($1) AND i.email=(SELECT email FROM users WHERE id=$2) AND i.deleted_at IS NULL ORDER BY i.created_at DESC`, ids, c.UserID)
	if err != nil {
		return err
	}
	type invitation struct {
		id, workspace, slug string
		role                int
	}
	invites := []invitation{}
	for rows.Next() {
		var i invitation
		if err = rows.Scan(&i.id, &i.workspace, &i.slug, &i.role); err != nil {
			rows.Close()
			return err
		}
		invites = append(invites, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, i := range invites {
		if c.Cache != nil {
			if err = c.Cache.Invalidate(ctx, "/api/workspaces/"+i.slug+"/members/", "", true); err != nil {
				return err
			}
		}
		if _, err = r.writer.Exec(ctx, `UPDATE workspace_members SET is_active=true,role=$3 WHERE workspace_id=$1 AND member_id=$2 AND deleted_at IS NULL`, i.workspace, c.UserID, i.role); err != nil {
			return err
		}
		if err = r.trackInvitationEvent(ctx, c.UserID, i.slug, i.workspace, "user_joined_workspace", map[string]any{"role": i.role, "joined_at": time.Now().UTC().Format("2006-01-02 15:04:05.999999+00:00")}); err != nil {
			return err
		}
	}
	for _, i := range invites {
		if err = insertWorkspaceMembership(ctx, r.writer, i.workspace, c.UserID, i.role, c.UserID); err != nil {
			return err
		}
	}
	for _, i := range invites {
		if _, err = r.writer.Exec(ctx, `UPDATE workspace_member_invites SET deleted_at=clock_timestamp() WHERE id=$1`, i.id); err != nil {
			return err
		}
	}
	return nil
}

func (r *InvitationsRepository) joinUserProjects(ctx context.Context, c InvitationCommand, fields map[string]any) (json.RawMessage, error) {
	rawIDs, exists := fields["project_ids"]
	if !exists {
		rawIDs = []any{}
	}
	ids, err := invitationIDs(rawIDs)
	if err != nil {
		return nil, err
	}
	var role int
	var workspace string
	if err = r.writer.QueryRow(ctx, `SELECT m.role,m.workspace_id::text FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL`, c.Slug, c.UserID).Scan(&role, &workspace); err != nil {
		return nil, err
	}
	rows, err := r.writer.Query(ctx, `SELECT p.id::text,p.network FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id::text=ANY($1) AND w.slug=$2 AND p.deleted_at IS NULL ORDER BY p.created_at DESC`, ids, c.Slug)
	if err != nil {
		return nil, err
	}
	projects := []string{}
	for rows.Next() {
		var id string
		var network int
		if err = rows.Scan(&id, &network); err != nil {
			rows.Close()
			return nil, err
		}
		if network == 0 && role != 20 {
			rows.Close()
			return nil, inviteFailure(403, "Only workspace admins can join private project")
		}
		projects = append(projects, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, id := range projects {
		if _, err = r.writer.Exec(ctx, `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2 AND deleted_at IS NULL`, id, c.UserID); err != nil {
			return nil, err
		}
	}
	for _, id := range projects {
		if err = insertProjectMembership(ctx, r.writer, id, workspace, c.UserID, role, c.UserID, false); err != nil {
			return nil, err
		}
	}
	for _, id := range projects {
		_, err = r.writer.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,'{}',$8,65535,clock_timestamp(),clock_timestamp(),$4) ON CONFLICT DO NOTHING`, uuid.NewString(), id, workspace, c.UserID, memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences)
		if err != nil {
			return nil, err
		}
	}
	return json.RawMessage(`{"message":"Projects joined successfully"}`), nil
}
