package activityreads

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) scopedAPIActivities(ctx context.Context, principal Principal, request Request, detail bool, projectID, issueID uuid.UUID) (Result, error) {
	if param(request, "fields") != "" || param(request, "expand") != "" {
		return Result{}, ErrForbidden
	}
	var limit, page, offset int
	var target any

	if detail {
		id, err := uuid.Parse(param(request, "pk"))
		if err != nil {
			return Result{}, ErrNotFound
		}
		target = id
		limit = 1
	} else {
		var err error
		limit, page, offset, err = pageWindow(request)
		if err != nil {
			return Result{}, err
		}
	}
	serializer := activityAPIJSON + `||jsonb_build_object('issue_comment',(SELECT c.id FROM issue_comments c WHERE c.id=a.issue_comment_id AND c.issue_id=a.issue_id AND c.project_id=a.project_id AND c.workspace_id=a.workspace_id))`
	query := `WITH ` + djangoapi.ProjectMemberReadScopeSQL(1, 2, 3, 4) + `visible_parent AS MATERIALIZED (
 SELECT i.id,i.project_id,i.workspace_id FROM issues i JOIN scope ON scope.id=i.project_id AND scope.workspace_id=i.workspace_id
 JOIN projects p ON p.id=i.project_id WHERE i.id=$5::uuid AND p.archived_at IS NULL AND ` + djangoapi.IssueVisibleSQL("i", "$3::uuid", "scope.workspace_id") + `),
 matching AS MATERIALIZED (SELECT a.* FROM issue_activities a JOIN visible_parent v ON v.id=a.issue_id AND v.project_id=a.project_id AND v.workspace_id=a.workspace_id
 WHERE a.deleted_at IS NULL AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft')) AND ($6::uuid IS NULL OR a.id=$6::uuid)),
 page AS (SELECT * FROM matching a ORDER BY a.created_at ASC LIMIT $7 OFFSET $8)
 SELECT jsonb_build_object('visible',EXISTS(SELECT 1 FROM visible_parent),'total_count',(SELECT count(*) FROM matching),
 'results',COALESCE((SELECT jsonb_agg(` + serializer + ` ORDER BY a.created_at ASC) FROM page a),'[]'::jsonb)) FROM scope`
	value, err := r.json(ctx, query, projectID, param(request, "slug"), principal.UserID, request.ExpectedWorkspaceID, issueID, target, limit, offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrForbidden
	}
	if err != nil {
		return Result{}, err
	}
	doc := value.(map[string]any)
	if doc["visible"] != true {
		return Result{}, ErrNotFound
	}
	if detail {
		rows := doc["results"].([]any)
		if len(rows) == 0 {
			return Result{}, ErrNotFound
		}
		return Result{Body: rows[0]}, nil
	}
	return paginatedResult(doc, limit, page)
}
