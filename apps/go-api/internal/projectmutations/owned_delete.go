package projectmutations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func authorizeOwnedDelete(ctx context.Context, tx pgx.Tx, resource Resource, cmd Command) error {
	// lockScope holds current membership through this transaction. Administrators
	// retain delete authority even if a native cascade already removed the target.
	var admin bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members
 WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active
 AND deleted_at IS NULL AND role=20)`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&admin)
	if err != nil {
		return err
	}
	if admin {
		return nil
	}
	// A member's authority depends on ownership. The purge worker retains this
	// deleted authorization scope while its delete receipt can still be replayed.
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT COALESCE(target.`+pgx.Identifier{resource.DeleteOwnerColumn}.Sanitize()+`=$4,false)
 FROM `+pgx.Identifier{resource.Table}.Sanitize()+` target
 WHERE target.id=$1 AND target.workspace_id=$2 AND target.project_id=$3 FOR UPDATE OF target`,
		cmd.ResourceID, cmd.WorkspaceID, cmd.ProjectID, cmd.ActorID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}
