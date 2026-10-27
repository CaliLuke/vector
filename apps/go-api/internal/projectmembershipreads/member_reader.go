package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

// MemberReader exposes the API project roster to authenticated transports.
// Project authorization belongs here, independently of credential format.
type MemberReader struct{ repository memberReadRepository }

type memberReadRepository interface {
	AuthorizeLifecycle(context.Context, LifecycleCommand) error
	ExecuteLifecycle(context.Context, LifecycleCommand) (any, error)
}

func NewMemberReader(repository memberReadRepository) (*MemberReader, error) {
	if repository == nil {
		return nil, errors.New("project member repository is required")
	}
	return &MemberReader{repository: repository}, nil
}

func (r *MemberReader) ListProjectMembers(ctx context.Context, actor uuid.UUID, slug string, project uuid.UUID) (json.RawMessage, error) {
	return r.listProjectMembers(ctx, actor, slug, project, nil)
}

// ListProjectMembersInWorkspace binds MCP's selected workspace and rechecks its
// membership with the project roster, without changing HTTP roster permissions.
func (r *MemberReader) ListProjectMembersInWorkspace(ctx context.Context, actor uuid.UUID, slug string, project, workspace uuid.UUID) (json.RawMessage, error) {
	return r.listProjectMembers(ctx, actor, slug, project, &workspace)
}

func (r *MemberReader) listProjectMembers(ctx context.Context, actor uuid.UUID, slug string, project uuid.UUID, expectedWorkspace *uuid.UUID) (json.RawMessage, error) {
	if actor == (uuid.UUID{}) || project == (uuid.UUID{}) || slug == "" {
		return nil, lifecycleFailure(http.StatusForbidden, "detail", "You do not have permission to perform this action.")
	}
	command := LifecycleCommand{
		Operation: "v1-list-project-members", Method: http.MethodGet,
		UserID: actor.String(), Slug: slug, ProjectID: project.String(), ExpectedWorkspaceID: expectedWorkspace,
	}
	if err := r.repository.AuthorizeLifecycle(ctx, command); err != nil {
		return nil, err
	}
	result, err := r.repository.ExecuteLifecycle(ctx, command)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// projectMembers uses the primary database and checks the actor in the same
// statement as the roster read. An earlier permission check cannot authorize a
// read after project membership is revoked. Preserve the HTTP roster's UserLite
// shape and inclusion of inactive (but not deleted) target memberships.
func (r *LifecycleRepository) projectMembers(ctx context.Context, c LifecycleCommand) (any, error) {
	value, err := djangoapi.JSON(ctx, r.database, `
WITH scope AS (
 SELECT p.id, p.workspace_id
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE p.id=$3 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
 AND ($4::uuid IS NULL OR (w.id=$4 AND EXISTS (
  SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1
   AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN (5,15,20)
 )))
 AND EXISTS (
  SELECT 1 FROM project_members actor JOIN users actor_user ON actor_user.id=actor.member_id
  WHERE actor.project_id=p.id AND actor.workspace_id=p.workspace_id AND actor.member_id=$1
    AND actor.is_active AND actor.deleted_at IS NULL AND actor_user.is_active
 )
)
SELECT COALESCE((
 SELECT jsonb_agg(`+lifecycleUserSQL+` ORDER BY u.created_at DESC, u.id)
 FROM users u WHERE u.id IN (
  SELECT pm.member_id FROM project_members pm
  WHERE pm.project_id=scope.id AND pm.workspace_id=scope.workspace_id AND pm.deleted_at IS NULL
 )
),'[]'::jsonb) FROM scope`, c.UserID, c.Slug, c.ProjectID, c.ExpectedWorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, lifecycleFailure(http.StatusForbidden, "detail", "You do not have permission to perform this action.")
	}
	return value, err
}
