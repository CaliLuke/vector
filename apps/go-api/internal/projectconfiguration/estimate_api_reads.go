package projectconfiguration

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EstimateRead struct {
	ActorID, WorkspaceID, ProjectID, EstimateID uuid.UUID
	WorkspaceSlug                               string
	Points, IncludeRevision                     bool
}

// ReadEstimate binds identity, workspace and resource ownership in the query
// returning the data. Missing selection differs from denied project scope.
func (r *Repository) ReadEstimate(ctx context.Context, in EstimateRead) (json.RawMessage, error) {
	if in.ActorID == uuid.Nil || in.WorkspaceID == uuid.Nil || in.ProjectID == uuid.Nil || in.WorkspaceSlug == "" || in.Points && in.EstimateID == uuid.Nil {
		return nil, projectmutations.ErrInvalid
	}
	query := estimateReadScopeSQL
	if in.Points {
		query += ` SELECT CASE WHEN e.id IS NULL THEN NULL ELSE (SELECT COALESCE(jsonb_agg(` + estimatePointAPIJSON + ` ORDER BY ep.key,ep.created_at,ep.id),'[]'::jsonb)
 FROM estimate_points ep WHERE ep.estimate_id=e.id AND ep.project_id=p.id AND ep.workspace_id=p.workspace_id AND ep.deleted_at IS NULL) END
 FROM scope p LEFT JOIN estimates e ON e.id=$5 AND e.project_id=p.id AND e.workspace_id=p.workspace_id AND e.deleted_at IS NULL`
	} else {
		query += ` SELECT CASE WHEN e.id IS NULL THEN NULL ELSE ` + estimateAPIJSON + ` || CASE WHEN $6 THEN jsonb_build_object('revision','project_estimate:'||p.id||':'||rev.revision) ELSE '{}'::jsonb END END
 FROM scope p LEFT JOIN estimates e ON e.id=p.estimate_id AND e.project_id=p.id AND e.workspace_id=p.workspace_id AND e.deleted_at IS NULL
 JOIN project_estimate_revisions rev ON rev.project_id=p.id WHERE $5::uuid IS NOT NULL`
	}
	// Both variants use the same positional parameters.
	if in.Points {
		query += ` WHERE $6::boolean IN (true,false)`
	}
	var body []byte
	err := r.db.QueryRow(ctx, query, in.ActorID, in.WorkspaceID, in.WorkspaceSlug, in.ProjectID, in.EstimateID, in.IncludeRevision).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, projectmutations.ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, projectmutations.ErrNotFound
	}
	return body, nil
}

const estimateReadScopeSQL = `WITH scope AS MATERIALIZED (
 SELECT p.* FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 JOIN users u ON u.id=$1 AND u.is_active
 JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=u.id
 JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=w.id AND pm.member_id=u.id
 WHERE w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND p.id=$4 AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)
)`
const estimateAPIJSON = `jsonb_build_object('id',e.id,'name',e.name,'description',e.description,'type',e.type,'last_used',e.last_used,
 'external_source',e.external_source,'external_id',e.external_id,'created_at',e.created_at,'updated_at',e.updated_at,
 'created_by',e.created_by_id,'updated_by',e.updated_by_id,'project',e.project_id,'workspace',e.workspace_id)`
const estimatePointAPIJSON = `jsonb_build_object('id',ep.id,'estimate',ep.estimate_id,'key',ep.key,'value',ep.value,'description',ep.description,
 'external_source',ep.external_source,'external_id',ep.external_id,'created_at',ep.created_at,'updated_at',ep.updated_at,
 'created_by',ep.created_by_id,'updated_by',ep.updated_by_id,'project',ep.project_id,'workspace',ep.workspace_id)`
