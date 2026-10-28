package projectmutations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WorkspaceRole verifies the immutable workspace binding in the caller's transaction.
// Writers retain identity and membership locks until their mutation commits.
func WorkspaceRole(ctx context.Context, tx pgx.Tx, actor, workspace uuid.UUID, slug string, lock bool) (int, error) {
	if actor == uuid.Nil || workspace == uuid.Nil || slug == "" {
		return 0, ErrInvalid
	}
	suffix := ""
	if lock {
		suffix = " FOR SHARE OF u,w,wm"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, actor, workspace, slug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrForbidden
	}
	return role, err
}
