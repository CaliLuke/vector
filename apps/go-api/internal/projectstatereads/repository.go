// Package projectstatereads contains the project workflow state application boundary.
package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("active project membership with an allowed role required")
	ErrNotFound  = errors.New("project state not found")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("project state reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) AuthorizeProject(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, projectStateAuthorizationSQL, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize project state read: %w", err)
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
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, `
WITH state_rows AS (
  SELECT state.id, state.project_id, state.workspace_id, state.name, state.color,
    state."group", state."default", state.description, state.sequence,
    row_number() OVER (PARTITION BY state."group" ORDER BY state.sequence)::float8 /
      count(*) OVER (PARTITION BY state."group")::float8 AS state_order
  FROM states state
  JOIN projects project ON project.id = state.project_id
  JOIN workspaces workspace ON workspace.id = state.workspace_id
  JOIN project_members project_member
    ON project_member.project_id = project.id
    AND project_member.member_id = $1
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
  WHERE workspace.slug = $2
    AND project.id = $3
    AND project.archived_at IS NULL
    AND state.deleted_at IS NULL
    AND state."group" <> 'triage'
    AND state.is_triage = false
)
SELECT COALESCE(json_agg(json_build_object(
  'id', id,
  'project_id', project_id,
  'workspace_id', workspace_id,
  'name', name,
  'color', color,
  'group', "group",
  'default', "default",
  'description', description,
  'sequence', sequence,
  'order', state_order
) ORDER BY sequence), '[]'::json)
FROM state_rows
`, userID, slug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list project states: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Detail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	stateID uuid.UUID,
) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT json_build_object(
  'id', state.id,
  'project_id', state.project_id,
  'workspace_id', state.workspace_id,
  'name', state.name,
  'color', state.color,
  'group', state."group",
  'default', state."default",
  'description', state.description,
  'sequence', state.sequence
)
FROM states state
JOIN projects project ON project.id = state.project_id
JOIN workspaces workspace ON workspace.id = state.workspace_id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.member_id = $1
  AND project_member.is_active = true
WHERE workspace.slug = $2
  AND project.id = $3
  AND state.id = $4
  AND project.archived_at IS NULL
  AND state.deleted_at IS NULL
  AND state."group" <> 'triage'
  AND state.is_triage = false
ORDER BY state.sequence
LIMIT 1
`, userID, slug, projectID, stateID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project state: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Intake(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT json_build_object(
  'id', state.id,
  'project_id', state.project_id,
  'workspace_id', state.workspace_id,
  'name', state.name,
  'color', state.color,
  'group', state."group",
  'default', state."default",
  'description', state.description,
  'sequence', state.sequence
)
FROM states state
JOIN projects project ON project.id = state.project_id
JOIN workspaces workspace ON workspace.id = state.workspace_id
WHERE workspace.slug = $1
  AND project.id = $2
  AND state.deleted_at IS NULL
  AND state."group" = 'triage'
ORDER BY state.sequence
LIMIT 1
`, slug, projectID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project intake state: %w", err)
	}
	return json.RawMessage(data), nil
}

const projectStateAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN projects project ON project.id = project_member.project_id
  JOIN workspaces workspace ON workspace.id = project_member.workspace_id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND project.id = $3
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    AND (
      project_member.role IN (5, 15, 20)
      OR EXISTS (
        SELECT 1
        FROM workspace_members workspace_member
        WHERE workspace_member.workspace_id = workspace.id
          AND workspace_member.member_id = $1
          AND workspace_member.role = 20
          AND workspace_member.is_active = true
          AND workspace_member.deleted_at IS NULL
      )
    )
)
`
