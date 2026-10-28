// Package worklogs owns time entries, their item scope and durable activity.
package worklogs

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ProjectID, IssueID uuid.UUID
	WorkspaceSlug                            string
	Params                                   map[string]string
	Present                                  map[string]bool
	IncludeRevision                          bool
}

const projection = `jsonb_build_object('id',l.id,'created_at',l.created_at,'updated_at',l.updated_at,'duration',l.duration,'description',l.description,'created_by',l.created_by_id,'updated_by',l.updated_by_id,'logged_by',l.logged_by_id,'project_id',l.project_id,'workspace_id',l.workspace_id)`

func (r *Repository) List(ctx context.Context, q Query) (json.RawMessage, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id JOIN users u ON u.id=wm.member_id
 WHERE w.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND u.id=$3 AND u.is_active AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND EXISTS(SELECT 1 FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE i.id=$4 AND i.project_id=$5 AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "w.id")+`))`, q.WorkspaceID, q.WorkspaceSlug, q.ActorID, q.IssueID, q.ProjectID).Scan(&allowed)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, projectmutations.ErrForbidden
	}
	limit, _, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return nil, err
	}
	fields := projection
	if q.IncludeRevision {
		fields += `||jsonb_build_object('revision',(SELECT 'work_log:'||l.id::text||':'||revision::text FROM work_log_edit_revisions WHERE work_log_id=l.id))`
	}
	rows, err := tx.Query(ctx, `SELECT `+fields+` FROM work_logs l WHERE l.workspace_id=$1 AND l.project_id=$2 AND l.issue_id=$3 AND l.deleted_at IS NULL ORDER BY l.created_at DESC,l.id LIMIT $4 OFFSET $5`, q.WorkspaceID, q.ProjectID, q.IssueID, limit, offset)
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
func authorizeMutation(ctx context.Context, tx pgx.Tx, cmd projectmutations.Command) error {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT i.id FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$4::uuid")+` FOR SHARE`, cmd.ParentID, cmd.ProjectID, cmd.ActorID, cmd.WorkspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrNotFound
	}
	if err != nil {
		return err
	}
	if cmd.Action == "create" {
		return nil
	}
	// Retained deleted rows allow current owner/admin authorization on replay.
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT COALESCE(l.logged_by_id=$1,false) OR EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=l.project_id AND pm.workspace_id=l.workspace_id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role=20)
 FROM work_logs l WHERE l.id=$2 AND l.project_id=$3 AND l.workspace_id=$4 AND l.issue_id=$5 FOR UPDATE OF l`, cmd.ActorID, cmd.ResourceID, cmd.ProjectID, cmd.WorkspaceID, cmd.ParentID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !allowed {
		return projectmutations.ErrForbidden
	}
	return nil
}
