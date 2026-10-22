// Package deployboardreads contains the authenticated project deploy-board read boundary.
package deployboardreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var (
	ErrForbidden = errors.New("active project membership required")
	ErrNotFound  = errors.New("project deploy board not found")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("deploy board reads database is required")
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
	err := repository.database.QueryRow(ctx, deployBoardAuthorizationSQL, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize deploy board read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// Collection preserves Django's non-standard list action: one object when the
// deployment exists and the unsaved serializer defaults when it does not.
func (repository *Repository) Collection(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	if err := repository.database.QueryRow(ctx, deployBoardCollectionSQL, userID, slug, projectID).Scan(&data); err != nil {
		return nil, fmt.Errorf("read project deploy board collection: %w", err)
	}
	return json.RawMessage(data), nil
}

// Detail intentionally fixes the inherited Django retrieve query by binding
// the board ID to both route tenants before returning it.
func (repository *Repository) Detail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	boardID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.AuthorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, deployBoardDetailSQL, userID, slug, projectID, boardID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read project deploy board detail: %w", err)
	}
	return json.RawMessage(data), nil
}

const deployBoardAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN workspaces workspace
    ON workspace.id = project_member.workspace_id
  JOIN projects project
    ON project.id = project_member.project_id
   AND project.workspace_id = workspace.id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND project.id = $3
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
)
`

const deployBoardTimezoneSQL = `
WITH requested_timezone AS MATERIALIZED (
  SELECT COALESCE(
    (SELECT NULLIF(user_timezone, '') FROM users WHERE id = $1),
    'UTC'
  ) AS name
), activated_timezone AS MATERIALIZED (
  SELECT set_config('TimeZone', requested_timezone.name, true) AS name
  FROM requested_timezone
)
`

const deployBoardJSONSQL = `json_build_object(
  'anchor', deploy_board.anchor,
  'created_at', to_char(deploy_board.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(deploy_board.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(deploy_board.created_at, 'US') END ||
    CASE WHEN to_char(deploy_board.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(deploy_board.created_at, 'TZH:TZM') END,
  'created_by', deploy_board.created_by_id,
  'deleted_at', deploy_board.deleted_at,
  'entity_identifier', deploy_board.entity_identifier,
  'entity_name', deploy_board.entity_name,
  'id', deploy_board.id,
  'intake', deploy_board.intake_id,
  'is_activity_enabled', deploy_board.is_activity_enabled,
  'is_comments_enabled', deploy_board.is_comments_enabled,
  'is_disabled', deploy_board.is_disabled,
  'is_reactions_enabled', deploy_board.is_reactions_enabled,
  'is_votes_enabled', deploy_board.is_votes_enabled,
  'project', deploy_board.project_id,
  'project_details', CASE WHEN project.id IS NULL THEN NULL ELSE json_build_object(
    'id', project.id,
    'identifier', project.identifier,
    'name', project.name,
    'cover_image', project.cover_image,
    'cover_image_url', CASE
      WHEN project.cover_image_asset_id IS NOT NULL
        AND project_cover_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
        THEN '/api/assets/v2/static/' || project.cover_image_asset_id::text || '/'
      WHEN project_cover_asset.entity_type = 'ISSUE_ATTACHMENT'
        THEN '/api/assets/v2/workspaces/' || project_cover_asset_workspace.slug ||
          '/projects/' || project_cover_asset.project_id::text || '/issues/' ||
          project_cover_asset.issue_id::text || '/attachments/' || project_cover_asset.id::text || '/'
      WHEN project_cover_asset.entity_type IN
        ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
        THEN '/api/assets/v2/workspaces/' || project_cover_asset_workspace.slug ||
          '/projects/' || project_cover_asset.project_id::text || '/' || project_cover_asset.id::text || '/'
      WHEN project.cover_image_asset_id IS NOT NULL THEN NULL
      ELSE NULLIF(project.cover_image, '')
    END,
    'logo_props', project.logo_props,
    'description', project.description
  ) END,
  'updated_at', to_char(deploy_board.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(deploy_board.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(deploy_board.updated_at, 'US') END ||
    CASE WHEN to_char(deploy_board.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(deploy_board.updated_at, 'TZH:TZM') END,
  'updated_by', deploy_board.updated_by_id,
  'view_props', deploy_board.view_props,
  'workspace', deploy_board.workspace_id,
  'workspace_detail', json_build_object(
    'id', workspace.id,
    'name', workspace.name,
    'slug', workspace.slug,
    'logo_url', CASE
      WHEN workspace.logo_asset_id IS NOT NULL
        AND workspace_logo_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
        THEN '/api/assets/v2/static/' || workspace.logo_asset_id::text || '/'
      WHEN workspace_logo_asset.entity_type = 'ISSUE_ATTACHMENT'
        THEN '/api/assets/v2/workspaces/' || workspace_logo_asset_workspace.slug ||
          '/projects/' || workspace_logo_asset.project_id::text || '/issues/' ||
          workspace_logo_asset.issue_id::text || '/attachments/' || workspace_logo_asset.id::text || '/'
      WHEN workspace_logo_asset.entity_type IN
        ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
        THEN '/api/assets/v2/workspaces/' || workspace_logo_asset_workspace.slug ||
          '/projects/' || workspace_logo_asset.project_id::text || '/' || workspace_logo_asset.id::text || '/'
      WHEN workspace.logo_asset_id IS NOT NULL THEN NULL
      ELSE NULLIF(workspace.logo, '')
    END
  )
)`

const deployBoardJoinsSQL = `
JOIN workspaces workspace ON workspace.id = deploy_board.workspace_id
LEFT JOIN projects project ON project.id = deploy_board.project_id
LEFT JOIN file_assets project_cover_asset ON project_cover_asset.id = project.cover_image_asset_id
LEFT JOIN file_assets workspace_logo_asset ON workspace_logo_asset.id = workspace.logo_asset_id
LEFT JOIN workspaces project_cover_asset_workspace ON project_cover_asset_workspace.id = project_cover_asset.workspace_id
LEFT JOIN workspaces workspace_logo_asset_workspace ON workspace_logo_asset_workspace.id = workspace_logo_asset.workspace_id
`

const deployBoardCollectionSQL = deployBoardTimezoneSQL + `
SELECT COALESCE((
  SELECT ` + deployBoardJSONSQL + `
  FROM deploy_boards deploy_board
  CROSS JOIN activated_timezone
  ` + deployBoardJoinsSQL + `
  WHERE workspace.slug = $2
    AND deploy_board.entity_name = 'project'
    AND deploy_board.entity_identifier = $3
    AND project.id = $3
    AND project.workspace_id = workspace.id
    AND deploy_board.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
  ORDER BY deploy_board.created_at DESC
  LIMIT 1
), json_build_object(
  'created_by', NULL,
  'deleted_at', NULL,
  'entity_identifier', NULL,
  'entity_name', '',
  'intake', NULL,
  'is_activity_enabled', false,
  'is_comments_enabled', false,
  'is_disabled', false,
  'is_reactions_enabled', false,
  'is_votes_enabled', false,
  'updated_by', NULL,
  'view_props', NULL
))
`

const deployBoardDetailSQL = deployBoardTimezoneSQL + `
SELECT ` + deployBoardJSONSQL + `
FROM deploy_boards deploy_board
CROSS JOIN activated_timezone
` + deployBoardJoinsSQL + `
WHERE workspace.slug = $2
  AND project.id = $3
  AND project.workspace_id = workspace.id
  AND deploy_board.id = $4
  AND deploy_board.entity_name = 'project'
  AND deploy_board.entity_identifier = $3
  AND deploy_board.deleted_at IS NULL
  AND workspace.deleted_at IS NULL
  AND project.deleted_at IS NULL
LIMIT 1
`
