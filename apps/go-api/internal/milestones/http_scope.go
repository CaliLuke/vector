package milestones

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResolveScope binds a native HTTP request to the current workspace identity.
// Subsequent operations retain this ID and recheck membership in their own
// transaction, including when the slug is reassigned after this lookup.
func (r *Repository) ResolveScope(ctx context.Context, actor, project uuid.UUID, slug, timezone string) (Scope, error) {
	if actor == uuid.Nil || project == uuid.Nil || slug == "" {
		return Scope{}, ErrInvalidInput
	}
	return readTransaction(ctx, r, func(tx pgx.Tx) (Scope, error) {
		scope := Scope{ActorID: actor, ProjectID: project, WorkspaceSlug: slug, Timezone: timezone}
		err := tx.QueryRow(ctx, `SELECT p.workspace_id FROM projects p JOIN workspaces w ON w.id=p.workspace_id
  WHERE p.id=$2 AND w.slug=$3 AND `+issuequery.MilestoneReadScopeSQL("p.id", "w.id", "$1::uuid"), actor, project, slug).Scan(&scope.WorkspaceID)
		return scope, scopeError(err)
	})
}
