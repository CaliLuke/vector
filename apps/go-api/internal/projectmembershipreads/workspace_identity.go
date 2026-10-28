package projectmembershipreads

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type WorkspaceIdentity struct{ ID, Name, Slug string }

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
