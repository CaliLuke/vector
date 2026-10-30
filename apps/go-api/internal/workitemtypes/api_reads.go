package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type APIRead struct {
	APIScope
	TypeID          uuid.UUID
	Limit, Offset   int
	IncludeRevision bool
}

func (r *Repository) ReadAPI(ctx context.Context, in APIRead) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return nil, err
	}
	if _, err = authorizeAPI(ctx, tx, in.APIScope, false); err != nil {
		return nil, err
	}
	raw, err := readAPITypeRows(ctx, tx, in)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func readAPITypeRows(ctx context.Context, tx pgx.Tx, in APIRead) (json.RawMessage, error) {
	if in.Limit < 0 || in.Offset < 0 {
		return nil, projectmutations.ErrInvalid
	}
	var limit any
	if in.ProjectID != uuid.Nil && in.TypeID == uuid.Nil {
		if in.Limit == 0 {
			in.Limit = 1000
		}
		if in.Limit > 1000 {
			return nil, projectmutations.ErrInvalid
		}
		limit = in.Limit
	} else {
		in.Offset = 0
	}
	projection := apiTypeJSON
	if in.IncludeRevision {
		projection += `||jsonb_build_object('revision','workitem_type:'||t.workspace_id||':'||rev.revision)`
	}
	query := `SELECT ` + projection + ` AS data FROM issue_types t JOIN work_item_type_api_revisions rev ON rev.workspace_id=t.workspace_id
 WHERE t.workspace_id=$1 AND t.deleted_at IS NULL AND ($4::uuid IS NULL OR t.id=$4)
 AND ($3::uuid IS NULL OR t.is_default OR t.mandatory OR EXISTS(SELECT 1 FROM project_issue_types link
 WHERE link.issue_type_id=t.id AND link.workspace_id=t.workspace_id AND link.project_id=$3 AND link.deleted_at IS NULL))
 ORDER BY t.created_at,t.id LIMIT $5 OFFSET $6`
	var project, target any
	if in.ProjectID != uuid.Nil {
		project = in.ProjectID
	}
	if in.TypeID != uuid.Nil {
		target = in.TypeID
	}
	if in.TypeID == uuid.Nil {
		query = `SELECT COALESCE(jsonb_agg(data),'[]'::jsonb) FROM (` + query + `) rows`
	}
	var raw []byte
	err := tx.QueryRow(ctx, query, in.WorkspaceID, in.ActorID, project, target, limit, in.Offset).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, projectmutations.ErrNotFound
	}
	return raw, err
}

const apiTypeJSON = `jsonb_build_object('id',t.id,'deleted_at',t.deleted_at,'name',t.name,'description',t.description,'logo_props',t.logo_props,
 'is_epic',t.is_epic,'is_default',t.is_default,'is_active',t.is_active,'level',t.level::integer,
 'external_source',t.external_source,'external_id',t.external_id,'created_at',t.created_at,'updated_at',t.updated_at,
 'created_by',t.created_by_id,'updated_by',t.updated_by_id,'workspace',t.workspace_id,
 'project_ids',COALESCE((SELECT jsonb_agg(p.id ORDER BY p.id) FROM projects p
 WHERE p.workspace_id=t.workspace_id AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=p.workspace_id AND pm.member_id=$2 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20))
 AND (t.is_default OR t.mandatory OR EXISTS(SELECT 1 FROM project_issue_types link WHERE link.project_id=p.id AND link.workspace_id=p.workspace_id AND link.issue_type_id=t.id AND link.deleted_at IS NULL))),'[]'::jsonb))`
