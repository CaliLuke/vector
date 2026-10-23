package djangoapi

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ProjectEntityAllowed mirrors ProjectEntityPermission. Related project and
// workspace deletion and workspace membership are intentionally not predicates.
func ProjectEntityAllowed(ctx context.Context, db Database, user any, slug string, project any, requireMemberRole bool) (bool, error) {
	var allowed bool
	err := db.QueryRow(ctx, `SELECT EXISTS(
SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
WHERE pm.member_id=$1 AND pm.project_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL AND w.slug=$3
AND (NOT $4::boolean OR pm.role IN (15, 20)))`, user, project, slug, requireMemberRole).Scan(&allowed)
	return allowed, err
}
