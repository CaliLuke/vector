package workitemtypes

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type APIScope struct {
	ActorID, WorkspaceID, ProjectID uuid.UUID
	WorkspaceSlug                   string
}
type apiAccess struct{ workspaceAdmin, projectAdmin bool }

// Scope checks and data reads share the same snapshot. Writers lock identity and
// membership rows after the workspace schema lock and hold them through commit.
func authorizeAPI(ctx context.Context, tx pgx.Tx, in APIScope, write bool) (apiAccess, error) {
	var out apiAccess
	if in.ActorID == uuid.Nil || in.WorkspaceID == uuid.Nil || in.WorkspaceSlug == "" {
		return out, projectmutations.ErrInvalid
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE OF u,w,wm"
	}
	err := tx.QueryRow(ctx, `SELECT wm.role=20 FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id
 WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, in.ActorID, in.WorkspaceID, in.WorkspaceSlug).Scan(&out.workspaceAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, projectmutations.ErrForbidden
	}
	if err != nil {
		return out, err
	}
	if in.ProjectID != uuid.Nil {
		suffix = ""
		if write {
			suffix = " FOR SHARE OF p,pm"
		}
		err = tx.QueryRow(ctx, `SELECT pm.role=20 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id
 WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)`+suffix, in.ProjectID, in.WorkspaceID, in.ActorID).Scan(&out.projectAdmin)
		if errors.Is(err, pgx.ErrNoRows) {
			return out, projectmutations.ErrForbidden
		}
	}
	return out, err
}

type APICapabilities struct{ Read, WorkspaceWrite, ProjectWrite bool }

func (r *Repository) APIVisibility(ctx context.Context, in APIScope) (APICapabilities, error) {
	var out APICapabilities
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return out, err
	}
	access, err := authorizeAPI(ctx, tx, in, false)
	if errors.Is(err, projectmutations.ErrForbidden) || errors.Is(err, projectmutations.ErrInvalid) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Read, out.WorkspaceWrite = true, access.workspaceAdmin
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id
 WHERE p.workspace_id=$1 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.member_id=$2 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role=20)`, in.WorkspaceID, in.ActorID).Scan(&out.ProjectWrite)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
