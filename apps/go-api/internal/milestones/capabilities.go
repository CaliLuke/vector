package milestones

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Capabilities describes operations available in at least one currently visible
// project. Every invocation still authorizes its specific project and target.
type Capabilities struct{ Read, Write bool }

func (r *Repository) VisibleActions(ctx context.Context, actor, workspace uuid.UUID, slug string) (Capabilities, error) {
	if actor == uuid.Nil || workspace == uuid.Nil || slug == "" {
		return Capabilities{}, nil
	}
	return readTransaction(ctx, r, func(tx pgx.Tx) (Capabilities, error) {
		var result Capabilities
		err := tx.QueryRow(ctx, `SELECT count(*)>0,coalesce(bool_or(EXISTS(
 SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=p.workspace_id
 AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(15,20))),false)
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE w.id=$2 AND w.slug=$3 AND `+issuequery.MilestoneReadScopeSQL("p.id", "w.id", "$1::uuid"), actor, workspace, slug).Scan(&result.Read, &result.Write)
		return result, err
	})
}
