// Package projectmembershipreads contains the project membership domain boundary.
package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

var (
	ErrForbidden = errors.New("active workspace membership required")
	ErrNotFound  = errors.New("project membership not found")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("project membership reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) AuthorizeWorkspace(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM workspace_members workspace_member
  JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
  WHERE workspace.slug = $2
    AND workspace_member.member_id = $1
    AND workspace_member.is_active = true
    AND workspace_member.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
)
`, userID, slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize workspace member: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) Membership(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT json_build_object(
  'id', pm.id,
  'created_at', pm.created_at,
  'updated_at', pm.updated_at,
  'created_by', pm.created_by_id,
  'updated_by', pm.updated_by_id,
  'deleted_at', pm.deleted_at,
  'comment', pm.comment,
  'role', pm.role,
  'view_props', pm.view_props,
  'default_props', pm.default_props,
  'preferences', pm.preferences,
  'sort_order', pm.sort_order,
  'is_active', pm.is_active,
  'member', json_build_object(
    'id', member.id,
    'first_name', member.first_name,
    'last_name', member.last_name,
    'avatar', member.avatar,
    'avatar_url', CASE
      WHEN member.avatar_asset_id IS NOT NULL
        THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
      ELSE NULLIF(member.avatar, '')
    END,
    'is_bot', member.is_bot,
    'display_name', member.display_name
  ),
  'project', json_build_object(
    'id', project.id,
    'identifier', project.identifier,
    'name', project.name,
    'cover_image', project.cover_image,
    'cover_image_url', CASE
      WHEN project.cover_image_asset_id IS NOT NULL
        THEN '/api/assets/v2/static/' || project.cover_image_asset_id::text || '/'
      ELSE NULLIF(project.cover_image, '')
    END,
    'logo_props', project.logo_props,
    'description', project.description
  ),
  'workspace', json_build_object(
    'id', workspace.id,
    'name', workspace.name,
    'slug', workspace.slug,
    'logo_url', CASE
      WHEN workspace.logo_asset_id IS NOT NULL
        THEN '/api/assets/v2/static/' || workspace.logo_asset_id::text || '/'
      ELSE NULLIF(workspace.logo, '')
    END
  )
)
FROM project_members pm
JOIN users member ON member.id = pm.member_id
JOIN projects project ON project.id = pm.project_id
JOIN workspaces workspace ON workspace.id = pm.workspace_id
WHERE pm.project_id = $1
  AND workspace.slug = $2
  AND pm.member_id = $3
  AND pm.is_active = true
  AND pm.deleted_at IS NULL
`, projectID, slug, userID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project membership: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Roles(ctx context.Context, userID uuid.UUID, slug string) (json.RawMessage, error) {
	var allowed bool
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT
  EXISTS (
    SELECT 1
    FROM workspace_members workspace_member
    JOIN workspaces workspace ON workspace.id = workspace_member.workspace_id
    WHERE workspace.slug = $2
      AND workspace_member.member_id = $1
      AND workspace_member.is_active = true
      AND workspace_member.deleted_at IS NULL
  ),
  COALESCE((
    SELECT json_object_agg(project_member.project_id::text, project_member.role)
    FROM project_members project_member
    JOIN workspaces workspace ON workspace.id = project_member.workspace_id
    JOIN workspace_members active_workspace_member
      ON active_workspace_member.workspace_id = workspace.id
      AND active_workspace_member.member_id = project_member.member_id
      AND active_workspace_member.is_active = true
      AND active_workspace_member.deleted_at IS NULL
    WHERE workspace.slug = $2
      AND project_member.member_id = $1
      AND project_member.is_active = true
      AND project_member.deleted_at IS NULL
  ), '{}'::json)
`, userID, slug).Scan(&allowed, &data)
	if err != nil {
		return nil, fmt.Errorf("list project roles: %w", err)
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) WorkspaceMembers(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	searchTerms []string,
) (json.RawMessage, error) {
	return repository.authorizedJSON(ctx, `
WITH requester AS MATERIALIZED (
  SELECT wm.role,
    set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM workspace_members wm
  JOIN workspaces w ON w.id = wm.workspace_id
  JOIN users requester_user ON requester_user.id = wm.member_id
  WHERE w.slug = $2 AND wm.member_id = $1 AND wm.is_active = true
    AND wm.role IN (5, 15, 20) AND wm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT jsonb_agg(
    (to_jsonb(wm) - ARRAY['created_by_id','updated_by_id','member_id','workspace_id']) ||
    jsonb_build_object(
      'created_by', wm.created_by_id,
      'updated_by', wm.updated_by_id,
      'created_at', CASE WHEN wm.created_at IS NULL THEN NULL ELSE
        to_char(wm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.created_at, 'US') END ||
        CASE WHEN to_char(wm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.created_at, 'TZH:TZM') END
      END,
      'updated_at', CASE WHEN wm.updated_at IS NULL THEN NULL ELSE
        to_char(wm.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.updated_at, 'US') END ||
        CASE WHEN to_char(wm.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.updated_at, 'TZH:TZM') END
      END,
      'deleted_at', CASE WHEN wm.deleted_at IS NULL THEN NULL ELSE
        to_char(wm.deleted_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.deleted_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.deleted_at, 'US') END ||
        CASE WHEN to_char(wm.deleted_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.deleted_at, 'TZH:TZM') END
      END,
      'workspace', wm.workspace_id,
      'member', CASE WHEN requester.role > 5 THEN
        jsonb_build_object(
          'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
          'email', member.email, 'avatar', member.avatar,
          'avatar_url', CASE
            WHEN member.avatar_asset_id IS NOT NULL
              AND member_avatar_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
              THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
            WHEN member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/issues/' ||
                member_avatar_asset.issue_id::text || '/attachments/' || member_avatar_asset.id::text || '/'
            WHEN member_avatar_asset.entity_type IN
              ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/' || member_avatar_asset.id::text || '/'
            WHEN member.avatar_asset_id IS NOT NULL THEN NULL
            ELSE NULLIF(member.avatar, '') END,
          'last_login_medium', member.last_login_medium, 'is_bot', member.is_bot,
          'display_name', member.display_name
        )
      ELSE
        jsonb_build_object(
          'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
          'avatar', member.avatar,
          'avatar_url', CASE
            WHEN member.avatar_asset_id IS NOT NULL
              AND member_avatar_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
              THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
            WHEN member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/issues/' ||
                member_avatar_asset.issue_id::text || '/attachments/' || member_avatar_asset.id::text || '/'
            WHEN member_avatar_asset.entity_type IN
              ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/' || member_avatar_asset.id::text || '/'
            WHEN member.avatar_asset_id IS NOT NULL THEN NULL
            ELSE NULLIF(member.avatar, '') END,
          'is_bot', member.is_bot, 'display_name', member.display_name
        )
      END
    ) ORDER BY wm.created_at DESC
  )
  FROM workspace_members wm
  JOIN workspaces w ON w.id = wm.workspace_id
  JOIN users member ON member.id = wm.member_id
  LEFT JOIN file_assets member_avatar_asset ON member_avatar_asset.id = member.avatar_asset_id
  LEFT JOIN workspaces member_avatar_asset_workspace ON member_avatar_asset_workspace.id = member_avatar_asset.workspace_id
  WHERE w.slug = $2 AND wm.deleted_at IS NULL
    AND NOT EXISTS (
      SELECT 1 FROM unnest($3::text[]) AS search_term
      WHERE NOT (
        UPPER(member.display_name::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\'
        OR UPPER(member.first_name::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\'
      )
    )
), '[]'::jsonb)
FROM requester
WHERE requester.user_timezone IS NOT NULL
`, userID, slug, searchTerms)
}

func (repository *Repository) WorkspaceMember(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	membershipID uuid.UUID,
	searchTerms []string,
) (json.RawMessage, error) {
	return repository.authorizedNullableJSON(ctx, `
WITH requester AS MATERIALIZED (
  SELECT wm.role,
    set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM workspace_members wm
  JOIN workspaces w ON w.id = wm.workspace_id
  JOIN users requester_user ON requester_user.id = wm.member_id
  WHERE w.slug = $2 AND wm.member_id = $1 AND wm.is_active = true
    AND wm.role IN (5, 15, 20) AND wm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT
    (to_jsonb(wm) - ARRAY['created_by_id','updated_by_id','member_id','workspace_id']) ||
    jsonb_build_object(
      'created_by', wm.created_by_id,
      'updated_by', wm.updated_by_id,
      'created_at', CASE WHEN wm.created_at IS NULL THEN NULL ELSE
        to_char(wm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.created_at, 'US') END ||
        CASE WHEN to_char(wm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.created_at, 'TZH:TZM') END
      END,
      'updated_at', CASE WHEN wm.updated_at IS NULL THEN NULL ELSE
        to_char(wm.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.updated_at, 'US') END ||
        CASE WHEN to_char(wm.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.updated_at, 'TZH:TZM') END
      END,
      'deleted_at', CASE WHEN wm.deleted_at IS NULL THEN NULL ELSE
        to_char(wm.deleted_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(wm.deleted_at, 'US') = '000000' THEN '' ELSE '.' || to_char(wm.deleted_at, 'US') END ||
        CASE WHEN to_char(wm.deleted_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(wm.deleted_at, 'TZH:TZM') END
      END,
      'workspace', wm.workspace_id,
      'member', CASE WHEN requester.role > 5 THEN
        jsonb_build_object(
          'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
          'email', member.email, 'avatar', member.avatar,
          'avatar_url', CASE
            WHEN member.avatar_asset_id IS NOT NULL
              AND member_avatar_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
              THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
            WHEN member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/issues/' ||
                member_avatar_asset.issue_id::text || '/attachments/' || member_avatar_asset.id::text || '/'
            WHEN member_avatar_asset.entity_type IN
              ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/' || member_avatar_asset.id::text || '/'
            WHEN member.avatar_asset_id IS NOT NULL THEN NULL
            ELSE NULLIF(member.avatar, '') END,
          'last_login_medium', member.last_login_medium, 'is_bot', member.is_bot,
          'display_name', member.display_name
        )
      ELSE
        jsonb_build_object(
          'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
          'avatar', member.avatar,
          'avatar_url', CASE
            WHEN member.avatar_asset_id IS NOT NULL
              AND member_avatar_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
              THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
            WHEN member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/issues/' ||
                member_avatar_asset.issue_id::text || '/attachments/' || member_avatar_asset.id::text || '/'
            WHEN member_avatar_asset.entity_type IN
              ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
              THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
                '/projects/' || member_avatar_asset.project_id::text || '/' || member_avatar_asset.id::text || '/'
            WHEN member.avatar_asset_id IS NOT NULL THEN NULL
            ELSE NULLIF(member.avatar, '') END,
          'is_bot', member.is_bot, 'display_name', member.display_name
        )
      END
    )
  FROM workspace_members wm
  JOIN workspaces w ON w.id = wm.workspace_id
  JOIN users member ON member.id = wm.member_id
  LEFT JOIN file_assets member_avatar_asset ON member_avatar_asset.id = member.avatar_asset_id
  LEFT JOIN workspaces member_avatar_asset_workspace ON member_avatar_asset_workspace.id = member_avatar_asset.workspace_id
  WHERE w.slug = $2 AND wm.id = $3 AND wm.deleted_at IS NULL
    AND NOT EXISTS (
      SELECT 1 FROM unnest($4::text[]) AS search_term
      WHERE NOT (
        UPPER(member.display_name::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\'
        OR UPPER(member.first_name::text) LIKE UPPER('%' || search_term || '%') ESCAPE '\'
      )
    )
), 'null'::jsonb)
FROM requester
WHERE requester.user_timezone IS NOT NULL
`, userID, slug, membershipID, searchTerms)
}

func (repository *Repository) WorkspaceProjectMembers(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
) (json.RawMessage, error) {
	return repository.authorizedJSON(ctx, `
WITH requester AS MATERIALIZED (
  SELECT wm.role,
    set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM workspace_members wm
  JOIN workspaces w ON w.id = wm.workspace_id
  JOIN users requester_user ON requester_user.id = wm.member_id
  WHERE w.slug = $2 AND wm.member_id = $1 AND wm.is_active = true
    AND wm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT jsonb_object_agg(grouped.project_id::text, grouped.members)
  FROM (
    SELECT pm.project_id, jsonb_agg(
      jsonb_build_object(
        'id', pm.id, 'role', pm.role, 'member', pm.member_id,
        'original_role', pm.role,
        'created_at', to_char(pm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
          CASE WHEN to_char(pm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.created_at, 'US') END ||
          CASE WHEN to_char(pm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.created_at, 'TZH:TZM') END
      ) ORDER BY pm.created_at DESC
    ) AS members
    FROM project_members pm
    JOIN workspaces w ON w.id = pm.workspace_id
    WHERE w.slug = $2 AND pm.is_active = true AND pm.deleted_at IS NULL
      AND pm.project_id IN (
        SELECT mine.project_id FROM project_members mine
        WHERE mine.member_id = $1 AND mine.is_active = true AND mine.deleted_at IS NULL
      )
    GROUP BY pm.project_id
  ) grouped
), '{}'::jsonb)
FROM requester
WHERE requester.user_timezone IS NOT NULL
`, userID, slug)
}

func (repository *Repository) ProjectMembers(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	return repository.authorizedJSON(ctx, `
WITH requester AS MATERIALIZED (
  SELECT pm.role,
    set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM project_members pm
  JOIN workspaces w ON w.id = pm.workspace_id
  JOIN users requester_user ON requester_user.id = pm.member_id
  WHERE w.slug = $2 AND pm.project_id = $3 AND pm.member_id = $1
    AND pm.is_active = true
    AND (
      pm.role IN (5, 15, 20)
      OR EXISTS (
        SELECT 1 FROM workspace_members requester_workspace_member
        WHERE requester_workspace_member.workspace_id = pm.workspace_id
          AND requester_workspace_member.member_id = pm.member_id
          AND requester_workspace_member.role = 20
          AND requester_workspace_member.is_active = true
          AND requester_workspace_member.deleted_at IS NULL
      )
    )
    AND pm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT jsonb_agg(
    jsonb_build_object(
      'id', pm.id, 'role', pm.role, 'member', pm.member_id,
      'project', pm.project_id, 'original_role', pm.role,
      'created_at', to_char(pm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(pm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.created_at, 'US') END ||
        CASE WHEN to_char(pm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.created_at, 'TZH:TZM') END
    ) ORDER BY pm.created_at DESC
  )
  FROM project_members pm
  JOIN workspaces w ON w.id = pm.workspace_id
  JOIN users member ON member.id = pm.member_id
  JOIN workspace_members wm ON wm.workspace_id = pm.workspace_id AND wm.member_id = pm.member_id
  WHERE w.slug = $2 AND pm.project_id = $3 AND pm.is_active = true
    AND pm.deleted_at IS NULL AND member.is_bot = false
    AND wm.is_active = true AND wm.deleted_at IS NULL
), '[]'::jsonb)
FROM requester
WHERE requester.user_timezone IS NOT NULL
`, userID, slug, projectID)
}

func (repository *Repository) ProjectMember(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	membershipID uuid.UUID,
) (json.RawMessage, error) {
	return repository.authorizedNullableJSON(ctx, `
WITH requester AS MATERIALIZED (
  SELECT pm.role,
    set_config('TimeZone', COALESCE(NULLIF(requester_user.user_timezone, ''), 'UTC'), true) AS user_timezone
  FROM project_members pm
  JOIN workspaces w ON w.id = pm.workspace_id
  JOIN users requester_user ON requester_user.id = pm.member_id
  WHERE w.slug = $2 AND pm.project_id = $3 AND pm.member_id = $1
    AND pm.is_active = true
    AND (
      pm.role IN (5, 15, 20)
      OR EXISTS (
        SELECT 1 FROM workspace_members requester_workspace_member
        WHERE requester_workspace_member.workspace_id = pm.workspace_id
          AND requester_workspace_member.member_id = pm.member_id
          AND requester_workspace_member.role = 20
          AND requester_workspace_member.is_active = true
          AND requester_workspace_member.deleted_at IS NULL
      )
    )
    AND pm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT CASE WHEN requester.role > 5 THEN
    (to_jsonb(pm) - ARRAY['created_by_id','updated_by_id','member_id','project_id','workspace_id']) ||
    jsonb_build_object(
      'created_by', pm.created_by_id, 'updated_by', pm.updated_by_id,
      'created_at', CASE WHEN pm.created_at IS NULL THEN NULL ELSE
        to_char(pm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(pm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.created_at, 'US') END ||
        CASE WHEN to_char(pm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.created_at, 'TZH:TZM') END
      END,
      'updated_at', CASE WHEN pm.updated_at IS NULL THEN NULL ELSE
        to_char(pm.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(pm.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.updated_at, 'US') END ||
        CASE WHEN to_char(pm.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.updated_at, 'TZH:TZM') END
      END,
      'deleted_at', CASE WHEN pm.deleted_at IS NULL THEN NULL ELSE
        to_char(pm.deleted_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(pm.deleted_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.deleted_at, 'US') END ||
        CASE WHEN to_char(pm.deleted_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.deleted_at, 'TZH:TZM') END
      END,
      'member', jsonb_build_object(
        'id', member.id, 'first_name', member.first_name, 'last_name', member.last_name,
        'email', member.email, 'avatar', member.avatar,
        'avatar_url', CASE
          WHEN member.avatar_asset_id IS NOT NULL
            AND member_avatar_asset.entity_type IN ('WORKSPACE_LOGO', 'USER_AVATAR', 'USER_COVER', 'PROJECT_COVER')
            THEN '/api/assets/v2/static/' || member.avatar_asset_id::text || '/'
          WHEN member_avatar_asset.entity_type = 'ISSUE_ATTACHMENT'
            THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
              '/projects/' || member_avatar_asset.project_id::text || '/issues/' ||
              member_avatar_asset.issue_id::text || '/attachments/' || member_avatar_asset.id::text || '/'
          WHEN member_avatar_asset.entity_type IN
            ('ISSUE_DESCRIPTION', 'COMMENT_DESCRIPTION', 'PAGE_DESCRIPTION', 'DRAFT_ISSUE_DESCRIPTION')
            THEN '/api/assets/v2/workspaces/' || member_avatar_asset_workspace.slug ||
              '/projects/' || member_avatar_asset.project_id::text || '/' || member_avatar_asset.id::text || '/'
          WHEN member.avatar_asset_id IS NOT NULL THEN NULL
          ELSE NULLIF(member.avatar, '') END,
        'last_login_medium', member.last_login_medium, 'is_bot', member.is_bot,
        'display_name', member.display_name
      ),
      'project', jsonb_build_object(
        'id', project.id, 'identifier', project.identifier, 'name', project.name,
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
          ELSE NULLIF(project.cover_image, '') END,
        'logo_props', project.logo_props, 'description', project.description
      ),
      'workspace', jsonb_build_object(
        'id', workspace.id, 'name', workspace.name, 'slug', workspace.slug,
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
          ELSE NULLIF(workspace.logo, '') END
      )
    )
  ELSE
    jsonb_build_object(
      'id', pm.id, 'role', pm.role, 'member', pm.member_id,
      'project', pm.project_id, 'original_role', pm.role,
      'created_at', to_char(pm.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
        CASE WHEN to_char(pm.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(pm.created_at, 'US') END ||
        CASE WHEN to_char(pm.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(pm.created_at, 'TZH:TZM') END
    )
  END
  FROM project_members pm
  JOIN projects project ON project.id = pm.project_id
  JOIN workspaces workspace ON workspace.id = pm.workspace_id
  JOIN users member ON member.id = pm.member_id
  LEFT JOIN file_assets member_avatar_asset ON member_avatar_asset.id = member.avatar_asset_id
  LEFT JOIN file_assets project_cover_asset ON project_cover_asset.id = project.cover_image_asset_id
  LEFT JOIN file_assets workspace_logo_asset ON workspace_logo_asset.id = workspace.logo_asset_id
  LEFT JOIN workspaces member_avatar_asset_workspace ON member_avatar_asset_workspace.id = member_avatar_asset.workspace_id
  LEFT JOIN workspaces project_cover_asset_workspace ON project_cover_asset_workspace.id = project_cover_asset.workspace_id
  LEFT JOIN workspaces workspace_logo_asset_workspace ON workspace_logo_asset_workspace.id = workspace_logo_asset.workspace_id
  WHERE workspace.slug = $2 AND pm.project_id = $3 AND pm.id = $4
    AND pm.is_active = true AND pm.deleted_at IS NULL AND member.is_bot = false
), 'null'::jsonb)
FROM requester
WHERE requester.user_timezone IS NOT NULL
`, userID, slug, projectID, membershipID)
}

func (repository *Repository) Preference(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	memberID uuid.UUID,
) (json.RawMessage, error) {
	return repository.authorizedNullableJSON(ctx, `
WITH requester AS (
  SELECT 1
  FROM project_members pm
  JOIN workspaces w ON w.id = pm.workspace_id
  WHERE w.slug = $2 AND pm.project_id = $3 AND pm.member_id = $1
    AND pm.is_active = true
    AND (
      pm.role IN (5, 15, 20)
      OR EXISTS (
        SELECT 1 FROM workspace_members requester_workspace_member
        WHERE requester_workspace_member.workspace_id = pm.workspace_id
          AND requester_workspace_member.member_id = pm.member_id
          AND requester_workspace_member.role = 20
          AND requester_workspace_member.is_active = true
          AND requester_workspace_member.deleted_at IS NULL
      )
    )
    AND pm.deleted_at IS NULL AND w.deleted_at IS NULL
)
SELECT COALESCE((
  SELECT jsonb_build_object(
    'preferences', pm.preferences, 'project_id', pm.project_id,
    'member_id', pm.member_id, 'workspace_id', pm.workspace_id
  )
  FROM project_members pm
  JOIN workspaces w ON w.id = pm.workspace_id
  WHERE w.slug = $2 AND pm.project_id = $3 AND pm.member_id = $4
    AND pm.deleted_at IS NULL
), 'null'::jsonb)
FROM requester
`, userID, slug, projectID, memberID)
}

func (repository *Repository) authorizedJSON(
	ctx context.Context,
	query string,
	args ...any,
) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, query, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, fmt.Errorf("read member directory: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) authorizedNullableJSON(
	ctx context.Context,
	query string,
	args ...any,
) (json.RawMessage, error) {
	data, err := repository.authorizedJSON(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return nil, ErrNotFound
	}
	return data, nil
}
