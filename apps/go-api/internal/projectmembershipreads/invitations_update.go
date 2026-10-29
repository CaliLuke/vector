package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *InvitationsRepository) invitationDelete(ctx context.Context, c InvitationCommand) error {
	table, model := "workspace_member_invites", "workspacememberinvite"
	if c.Operation == "project-delete" {
		table = "project_member_invites"
		model = "projectmemberinvite"
	}
	var id string
	query := `SELECT i.id::text FROM ` + table + ` i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND w.slug=$2 AND i.deleted_at IS NULL`
	args := []any{c.ID, c.Slug}
	if c.ProjectID != "" {
		query += ` AND i.project_id=$3`
		args = append(args, c.ProjectID)
	}
	err := r.writer.QueryRow(ctx, query, args...).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		if c.Operation == "project-delete" {
			return invitationViewsetMissing("ProjectMemberInvite")
		}
		return inviteMissing()
	}
	if err != nil {
		return err
	}
	if _, err = r.writer.Exec(ctx, `UPDATE `+table+` SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, nullableInvitationActor(c.UserID)); err != nil {
		return err
	}
	return r.publisher.Task(ctx, "plane.bgtasks.deletion_task.soft_delete_related_objects", []any{"db", model, map[string]any{"__type__": "uuid", "__value__": map[string]string{"hex": strings.ReplaceAll(id, "-", "")}}}, map[string]any{"using": nil})
}

func (r *InvitationsRepository) invitationPatch(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	if _, err := r.invitationRead(ctx, InvitationCommand{Operation: "workspace-detail", Slug: c.Slug, ID: c.ID}); err != nil {
		return nil, err
	}
	fields, err := decodeInvitationSerializerPayload(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	updates, args, fieldError := invitationFieldUpdates(fields, c.ID, c.UserID, c.Timezone, workspaceInvitationFields)
	if err := invitationValidationErrors(fieldError, r.invitationRelationErrors(ctx, fields, workspaceInvitationFields)); err != nil {
		return nil, err
	}
	if _, err := r.writer.Exec(ctx, `UPDATE workspace_member_invites SET `+strings.Join(updates, ",")+` WHERE id=$1`, args...); err != nil {
		return nil, err
	}
	return inviteJSON(ctx, r.writer, `SELECT `+workspaceInviteProjection(false)+` FROM workspace_member_invites i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1`, c.ID)
}

func (r *InvitationsRepository) putAPIWorkspaceInvite(ctx context.Context, c InvitationCommand) (json.RawMessage, error) {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return nil, inviteFailure(400, "Please provide valid detail")
	}
	current, err := r.WorkspaceInvite(ctx, c.Slug, id)
	if err != nil {
		return nil, err
	}
	fields, err := invitationCommandFields(&c)
	if err != nil {
		return nil, err
	}
	return r.updateAPIWorkspaceInvite(ctx, c.Slug, id, current, fields, false)
}
