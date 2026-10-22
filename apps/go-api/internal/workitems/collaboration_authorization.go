package workitems

import (
	"context"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
)

func collaborationPublic(q Request) bool {
	return strings.Contains(q.Operation, "_api_public_") || q.Params["anchor"] != ""
}
func collaborationComment(q Request) bool {
	return strings.Contains(q.Operation, "_comments") && !strings.Contains(q.Operation, "_reactions")
}
func collaborationLink(q Request) bool { return strings.Contains(q.Operation, "_links") }
func collaborationSafe(q Request) bool {
	return q.Method == "GET" || q.Method == "HEAD" || q.Method == "OPTIONS"
}

// AuthorizeCollaborationInitial preserves permission_classes before API
// throttling/timezone; app action decorators remain in their handlers.
func (r *Repository) AuthorizeCollaborationInitial(ctx context.Context, q Request) error {
	if collaborationPublic(q) {
		return nil
	}
	applies := q.API || collaborationLink(q) || strings.Contains(q.Operation, "_subscribers") || strings.Contains(q.Operation, "_subscribe")
	if !applies {
		return nil
	}
	unsafe := !collaborationSafe(q)
	if q.API && collaborationComment(q) || strings.Contains(q.Operation, "_subscribe") && !strings.Contains(q.Operation, "_subscribers") {
		unsafe = false
	}
	// Django converts model UUID filters before PostgreSQL receives them.
	projectID, err := djangoinput.UUID(q.ProjectID)
	if err != nil {
		return failure(400, "error", "Please provide valid detail")
	}
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, r.db, q.ActorID, q.Slug, projectID.String(), unsafe)
	if err != nil {
		return err
	}
	if !allowed {
		return failure(403, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

func (r *Repository) collaborationBoard(ctx context.Context, q Request) (map[string]any, error) {
	return r.row(ctx, `SELECT to_jsonb(b) FROM deploy_boards b WHERE b.anchor=$1 AND b.entity_name='project' AND b.deleted_at IS NULL`, q.Params["anchor"])
}
func collaborationMissing(q Request) error {
	if q.API {
		return failure(404, "error", "The requested resource does not exist.")
	}
	return requiredObject()
}

// Existing foreign references are rejected using actual database ownership.
// Missing references retain the source serializer/FK branch and its precedence.
func (r *Repository) collaborationReference(ctx context.Context, q Request, table string, id any) error {
	if id == nil || id == "" {
		return nil
	}
	var project, workspace string
	query := ""
	switch table {
	case "issues", "issue_comments", "descriptions":
		query = `SELECT COALESCE(t.project_id::text,''),w.slug FROM ` + table + ` t JOIN workspaces w ON w.id=t.workspace_id WHERE t.id=$1`
	default:
		panic("unsupported collaboration reference table")
	}
	err := r.db.QueryRow(ctx, query, id).Scan(&project, &workspace)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if project != q.ProjectID || workspace != q.Slug {
		return collaborationMissing(q)
	}
	return nil
}

func (r *Repository) collaborationMembership(ctx context.Context, q Request) (bool, error) {
	return djangoapi.ProjectEntityAllowed(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, false)
}
