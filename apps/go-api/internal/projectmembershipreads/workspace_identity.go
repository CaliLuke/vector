package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type WorkspaceIdentity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// WorkspaceIdentity returns only a workspace visible in the current primary
// snapshot, with immutable ownership retained through workspace renames.
func (r *DirectoryReader) WorkspaceIdentity(ctx context.Context, actor, workspace uuid.UUID, slug string) (WorkspaceIdentity, error) {
	var value WorkspaceIdentity
	err := r.primary.QueryRow(ctx, `SELECT w.id::text,w.name,w.slug FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id JOIN users u ON u.id=wm.member_id WHERE w.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND wm.member_id=$3 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) AND u.is_active`, workspace, slug, actor).Scan(&value.ID, &value.Name, &value.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceIdentity{}, ErrForbidden
	}
	return value, err
}

// ListWorkspaces returns active memberships in live workspaces from the primary.
// Token workspace metadata and pending invitations do not grant access.
func (r *DirectoryReader) ListWorkspaces(ctx context.Context, actor uuid.UUID) ([]WorkspaceIdentity, error) {
	var raw []byte
	err := r.primary.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(row ORDER BY row.slug, row.id), '[]'::jsonb)
 FROM (SELECT w.id::text,w.name,w.slug FROM workspaces w
 WHERE w.deleted_at IS NULL AND EXISTS (
 SELECT 1 FROM workspace_members wm JOIN users u ON u.id=wm.member_id
 WHERE wm.workspace_id=w.id AND u.id=$1 AND u.is_active
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))) row`, actor).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var workspaces []WorkspaceIdentity
	if decodeErr := json.Unmarshal(raw, &workspaces); decodeErr != nil {
		return nil, decodeErr
	}
	return workspaces, nil
}
