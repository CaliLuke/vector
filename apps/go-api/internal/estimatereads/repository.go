// Package estimatereads contains the read-only estimate boundary.
package estimatereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var (
	ErrForbidden = errors.New("active workspace or project membership required")
	ErrNotFound  = errors.New("estimate not found")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("estimate reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) AuthorizeWorkspace(ctx context.Context, userID uuid.UUID, slug string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, estimateWorkspaceAuthorizationSQL, userID, slug).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize workspace estimate read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) AuthorizeProject(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, estimateProjectAuthorizationSQL, userID, slug, projectID).Scan(&allowed); err != nil {
		return fmt.Errorf("authorize project estimate read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// WorkspaceSelected returns the distinct estimates selected by projects in the
// workspace. It deliberately does not return unselected estimates.
func (repository *Repository) WorkspaceSelected(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
) (json.RawMessage, error) {
	if err := repository.AuthorizeWorkspace(ctx, userID, slug); err != nil {
		return nil, err
	}
	var data []byte
	if err := repository.database.QueryRow(ctx, estimateWorkspaceSelectedSQL, userID, slug).Scan(&data); err != nil {
		return nil, fmt.Errorf("list selected workspace estimates: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) ProjectList(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	if err := repository.database.QueryRow(ctx, estimateProjectListSQL, userID, slug, projectID).Scan(&data); err != nil {
		return nil, fmt.Errorf("list project estimates: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) ProjectDetail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	estimateID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	if err := repository.database.QueryRow(ctx, estimateProjectDetailSQL, userID, slug, projectID, estimateID).Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project estimate: %w", err)
	}
	return json.RawMessage(data), nil
}

const estimateWorkspaceAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace_member.member_id = $1
    AND workspace.slug = $2
    AND workspace_member.is_active = true
    AND workspace_member.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
)
`

const estimateProjectAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN projects project
    ON project.id = project_member.project_id
    AND project.workspace_id = project_member.workspace_id
  JOIN workspaces workspace ON workspace.id = project_member.workspace_id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND project.id = $3
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
)
`

const estimateReadJSON = `jsonb_build_object(
  'id', estimate.id,
  'created_at', to_char(estimate.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(estimate.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(estimate.created_at, 'US') END ||
    CASE WHEN to_char(estimate.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(estimate.created_at, 'TZH:TZM') END,
  'updated_at', to_char(estimate.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(estimate.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(estimate.updated_at, 'US') END ||
    CASE WHEN to_char(estimate.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(estimate.updated_at, 'TZH:TZM') END,
  'created_by', estimate.created_by_id,
  'updated_by', estimate.updated_by_id,
  'deleted_at', estimate.deleted_at,
  'workspace', estimate.workspace_id,
  'project', estimate.project_id,
  'name', estimate.name,
  'description', estimate.description,
  'type', estimate.type,
  'last_used', estimate.last_used,
  'points', COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
      'id', point.id,
      'created_at', to_char(point.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(point.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(point.created_at, 'US') END ||
        CASE WHEN to_char(point.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(point.created_at, 'TZH:TZM') END,
      'updated_at', to_char(point.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(point.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(point.updated_at, 'US') END ||
        CASE WHEN to_char(point.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(point.updated_at, 'TZH:TZM') END,
      'created_by', point.created_by_id,
      'updated_by', point.updated_by_id,
      'deleted_at', point.deleted_at,
      'estimate', point.estimate_id,
      'workspace', point.workspace_id,
      'project', point.project_id,
      'key', point.key,
      'value', point.value,
      'description', point.description
    ) ORDER BY point.value)
    FROM estimate_points point
    WHERE point.estimate_id = estimate.id
      AND point.workspace_id = estimate.workspace_id
      AND point.project_id = estimate.project_id
      AND point.deleted_at IS NULL
  ), '[]'::jsonb)
)`

const estimateWorkspaceSelectedSQL = `
WITH requester_timezone AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester.user_timezone, ''), 'UTC'), true) AS configured
  FROM users requester
  WHERE requester.id = $1
)
SELECT COALESCE(jsonb_agg(` + estimateReadJSON + ` ORDER BY estimate.name), '[]'::jsonb)
FROM estimates estimate
JOIN workspaces workspace ON workspace.id = estimate.workspace_id
CROSS JOIN requester_timezone
WHERE workspace.slug = $2
  AND workspace.deleted_at IS NULL
  AND estimate.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM projects selected_project
    WHERE selected_project.workspace_id = workspace.id
      AND selected_project.estimate_id = estimate.id
      AND selected_project.deleted_at IS NULL
  )
`

const estimateProjectListSQL = `
WITH requester_timezone AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester.user_timezone, ''), 'UTC'), true) AS configured
  FROM users requester
  WHERE requester.id = $1
)
SELECT COALESCE(jsonb_agg(` + estimateReadJSON + ` ORDER BY estimate.name), '[]'::jsonb)
FROM estimates estimate
JOIN projects project
  ON project.id = estimate.project_id
  AND project.workspace_id = estimate.workspace_id
JOIN workspaces workspace ON workspace.id = estimate.workspace_id
CROSS JOIN requester_timezone
WHERE workspace.slug = $2
  AND project.id = $3
  AND workspace.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND estimate.deleted_at IS NULL
`

const estimateProjectDetailSQL = `
WITH requester_timezone AS MATERIALIZED (
  SELECT set_config('TimeZone', COALESCE(NULLIF(requester.user_timezone, ''), 'UTC'), true) AS configured
  FROM users requester
  WHERE requester.id = $1
)
SELECT ` + estimateReadJSON + `
FROM estimates estimate
JOIN projects project
  ON project.id = estimate.project_id
  AND project.workspace_id = estimate.workspace_id
JOIN workspaces workspace ON workspace.id = estimate.workspace_id
CROSS JOIN requester_timezone
WHERE workspace.slug = $2
  AND project.id = $3
  AND estimate.id = $4
  AND workspace.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND estimate.deleted_at IS NULL
LIMIT 1
`
