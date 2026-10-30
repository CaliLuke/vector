package worklogs

import (
	"context"
	"encoding/json"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/jackc/pgx/v5"
)

// Summary reports the pinned project contract's seconds, while individual logs
// store minutes. Visibility is applied before grouping, never after aggregation.
func (r *Repository) Summary(ctx context.Context, q Query) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id JOIN users u ON u.id=wm.member_id JOIN projects p ON p.workspace_id=w.id JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=w.id AND pm.member_id=u.id
 WHERE w.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND u.id=$3 AND u.is_active AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) AND p.id=$4 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20))`, q.WorkspaceID, q.WorkspaceSlug, q.ActorID, q.ProjectID).Scan(&allowed)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, projectmutations.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT jsonb_build_object('issue_id',l.issue_id,'duration',sum(l.duration::bigint)*60) FROM work_logs l JOIN issues i ON i.id=l.issue_id AND i.project_id=l.project_id AND i.workspace_id=l.workspace_id WHERE l.workspace_id=$1 AND l.project_id=$2 AND l.deleted_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$1::uuid")+` GROUP BY l.issue_id ORDER BY l.issue_id`, q.WorkspaceID, q.ProjectID, q.ActorID)
	if err != nil {
		return nil, err
	}
	out := []json.RawMessage{}
	for rows.Next() {
		var row json.RawMessage
		if err = rows.Scan(&row); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, row)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}
