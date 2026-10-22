// Package projectlabelreads contains the read-only project issue-label boundary.
package projectlabelreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var (
	ErrForbidden = errors.New("active workspace membership required")
	ErrNotFound  = errors.New("project label not found")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("project label reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) AuthorizeWorkspace(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, projectLabelWorkspaceAuthorizationSQL, userID, slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize project label read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) List(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeWorkspace(ctx, userID, slug); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'parent', label.parent_id,
  'name', label.name,
  'color', label.color,
  'id', label.id,
  'project_id', label.project_id,
  'workspace_id', label.workspace_id,
  'sort_order', label.sort_order
) ORDER BY label.sort_order), '[]'::json)
FROM labels label
JOIN projects project ON project.id = label.project_id
JOIN workspaces workspace ON workspace.id = label.workspace_id
WHERE workspace.slug = $2
  AND project.id = $3
  AND label.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM project_members project_member
    WHERE project_member.project_id = project.id
      AND project_member.member_id = $1
  )
`, userID, slug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list project labels: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Detail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	labelID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeWorkspace(ctx, userID, slug); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT json_build_object(
  'parent', label.parent_id,
  'name', label.name,
  'color', label.color,
  'id', label.id,
  'project_id', label.project_id,
  'workspace_id', label.workspace_id,
  'sort_order', label.sort_order
)
FROM labels label
JOIN projects project ON project.id = label.project_id
JOIN workspaces workspace ON workspace.id = label.workspace_id
WHERE workspace.slug = $2
  AND project.id = $3
  AND label.id = $4
  AND label.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM project_members project_member
    WHERE project_member.project_id = project.id
      AND project_member.member_id = $1
  )
LIMIT 1
`, userID, slug, projectID, labelID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project label: %w", err)
	}
	return json.RawMessage(data), nil
}

const projectLabelWorkspaceAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace_member.member_id = $1
    AND workspace.slug = $2
    AND workspace_member.is_active = true
    AND workspace_member.deleted_at IS NULL
)
`
