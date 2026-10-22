// Package publicprojectmetadatareads contains public deploy-board metadata reads.
package publicprojectmetadatareads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var (
	ErrInvalidAnchor       = errors.New("invalid deploy-board anchor")
	ErrProjectNotPublished = errors.New("project is not published")
	ErrRequiredObject      = errors.New("required object does not exist")
	ErrRender              = errors.New("render public project metadata response")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

// PublicBoard is the validated public scope shared by anchor-based routes.
type PublicBoard struct {
	WorkspaceSlug string
	WorkspaceID   uuid.UUID
	ProjectID     uuid.UUID
	IntakeID      *uuid.UUID
	VotesEnabled  bool
}

type userIDContextKey struct{}

// WithUserID carries the authenticated Django principal to reads whose
// serialization depends on TimezoneMixin. An empty value preserves anonymous UTC.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserID returns the authenticated Django principal carried by the HTTP
// compatibility layer, or an empty string for an anonymous request.
func UserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey{}).(string)
	return userID
}

type stateRecord struct {
	Name     string `json:"name"`
	Group    string `json:"group"`
	Color    string `json:"color"`
	ID       string `json:"id"`
	Sequence string `json:"sequence"`
}

type stateResponse struct {
	Name     string          `json:"name"`
	Group    string          `json:"group"`
	Color    string          `json:"color"`
	ID       string          `json:"id"`
	Sequence json.RawMessage `json:"sequence"`
}

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("public project metadata reads database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) Cycles(ctx context.Context, anchor string) (json.RawMessage, error) {
	workspaceSlug, projectID, err := repository.resolveAnchor(ctx, anchor)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'id', cycle.id,
  'name', cycle.name
) ORDER BY cycle.created_at DESC), '[]'::json)
FROM cycles cycle
JOIN workspaces workspace ON workspace.id = cycle.workspace_id
WHERE workspace.slug = $1
  AND cycle.project_id IS NOT DISTINCT FROM $2
  AND cycle.deleted_at IS NULL
`, workspaceSlug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: list public project cycles: %v", ErrRender, err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Labels(ctx context.Context, anchor string) (json.RawMessage, error) {
	workspaceSlug, projectID, err := repository.resolveAnchor(ctx, anchor)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'id', label.id,
  'name', label.name,
  'color', label.color,
  'parent', label.parent_id
) ORDER BY label.created_at DESC), '[]'::json)
FROM labels label
JOIN workspaces workspace ON workspace.id = label.workspace_id
WHERE workspace.slug = $1
  AND label.project_id IS NOT DISTINCT FROM $2
  AND label.deleted_at IS NULL
`, workspaceSlug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: list public project labels: %v", ErrRender, err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Modules(ctx context.Context, anchor string) (json.RawMessage, error) {
	workspaceSlug, projectID, err := repository.resolveAnchor(ctx, anchor)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'id', module.id,
  'name', module.name
) ORDER BY module.created_at DESC), '[]'::json)
FROM modules module
JOIN workspaces workspace ON workspace.id = module.workspace_id
WHERE workspace.slug = $1
  AND module.project_id IS NOT DISTINCT FROM $2
  AND module.deleted_at IS NULL
`, workspaceSlug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: list public project modules: %v", ErrRender, err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Meta(ctx context.Context, anchor string) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT json_build_object(
  'id', project.id,
  'identifier', project.identifier,
  'name', project.name,
  'cover_image', project.cover_image,
  'icon_prop', project.icon_prop,
  'emoji', project.emoji,
  'description', project.description
)
FROM deploy_boards deploy_board
JOIN projects project ON project.id = deploy_board.entity_identifier
WHERE deploy_board.anchor = $1
  AND deploy_board.entity_name = 'project'
  AND deploy_board.deleted_at IS NULL
  AND project.deleted_at IS NULL
`, anchor).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProjectNotPublished
	}
	if err != nil {
		return nil, fmt.Errorf("read public project metadata: %w", err)
	}
	return json.RawMessage(data), nil
}

// ProjectBoards returns the published project boards in a workspace. A board
// only exposes a project when both board identity columns agree; that keeps a
// stale or cross-tenant deploy-board row from widening the public directory.
func (repository *Repository) ProjectBoards(ctx context.Context, workspaceSlug string) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(project_row.item ORDER BY project_row.created_at DESC), '[]'::json)
FROM (
  SELECT DISTINCT ON (project.id)
    json_build_object(
      'id', project.id,
      'identifier', project.identifier,
      'name', project.name,
      'description', project.description,
      'emoji', project.emoji,
      'icon_prop', project.icon_prop,
      'cover_image', project.cover_image
    ) AS item,
    project.created_at
  FROM projects project
  JOIN workspaces workspace ON workspace.id = project.workspace_id
  JOIN deploy_boards board ON board.project_id = project.id
    AND board.entity_identifier = project.id
    AND board.workspace_id = workspace.id
  WHERE workspace.slug = $1
    AND workspace.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND board.deleted_at IS NULL
    AND board.entity_name = 'project'
    AND board.is_disabled = FALSE
  ORDER BY project.id, project.created_at DESC
) project_row
`, workspaceSlug).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list public project boards: %w", err)
	}
	return json.RawMessage(data), nil
}

// ResolvePublicBoard rejects unusable, disabled, wrong-kind, or inconsistent
// board rows before a public operation reads a project or performs a write.
func (repository *Repository) ResolvePublicBoard(ctx context.Context, anchor string) (PublicBoard, error) {
	var board PublicBoard
	err := repository.database.QueryRow(ctx, `
SELECT workspace.slug, workspace.id, project.id, deploy_board.intake_id, deploy_board.is_votes_enabled
FROM deploy_boards deploy_board
JOIN workspaces workspace ON workspace.id = deploy_board.workspace_id
JOIN projects project ON project.id = deploy_board.entity_identifier
  AND project.id = deploy_board.project_id
  AND project.workspace_id = deploy_board.workspace_id
WHERE deploy_board.anchor = $1
  AND deploy_board.entity_name = 'project'
  AND deploy_board.deleted_at IS NULL
  AND deploy_board.is_disabled = FALSE
  AND workspace.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND project.archived_at IS NULL
`, anchor).Scan(&board.WorkspaceSlug, &board.WorkspaceID, &board.ProjectID, &board.IntakeID, &board.VotesEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicBoard{}, ErrProjectNotPublished
	}
	if err != nil {
		return PublicBoard{}, fmt.Errorf("resolve public project board: %w", err)
	}
	return board, nil
}

// IssueDetail reads a public-visible issue under a previously validated board
// scope. Missing issues intentionally serialize as JSON null with HTTP 200.
func (repository *Repository) IssueDetail(ctx context.Context, board PublicBoard, issueID string) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE((
  SELECT json_build_object(
    'id', issue.id,
    'name', issue.name,
    'state_id', issue.state_id,
    'state__group', state."group",
    'sort_order', issue.sort_order,
    'description_json', issue.description_json,
    'description_html', issue.description_html,
    'description_stripped', issue.description_stripped,
    'description_binary', issue.description_binary,
    'module_ids', COALESCE((SELECT json_agg(DISTINCT relation.module_id)
      FROM module_issues relation JOIN modules module ON module.id = relation.module_id
      WHERE relation.issue_id = issue.id AND relation.deleted_at IS NULL AND module.archived_at IS NULL), '[]'::json),
    'label_ids', COALESCE((SELECT json_agg(DISTINCT relation.label_id)
      FROM issue_labels relation WHERE relation.issue_id = issue.id AND relation.deleted_at IS NULL), '[]'::json),
    'assignee_ids', COALESCE((SELECT json_agg(DISTINCT relation.assignee_id)
      FROM issue_assignees relation JOIN project_members member ON member.project_id = issue.project_id
        AND member.member_id = relation.assignee_id AND member.is_active = TRUE
      WHERE relation.issue_id = issue.id AND relation.deleted_at IS NULL), '[]'::json),
    'estimate_point', issue.estimate_point_id,
    'priority', issue.priority,
    'start_date', issue.start_date,
    'target_date', issue.target_date,
    'sequence_id', issue.sequence_id,
    'project_id', issue.project_id,
    'parent_id', issue.parent_id,
    'cycle_id', (SELECT relation.cycle_id FROM cycle_issues relation WHERE relation.issue_id = issue.id AND relation.deleted_at IS NULL LIMIT 1),
    'created_by', issue.created_by_id,
    'vote_items', (SELECT jsonb_agg(DISTINCT jsonb_build_object(
      'vote', vote.vote,
      'actor_details', json_build_object('id', actor.id, 'first_name', actor.first_name, 'last_name', actor.last_name,
        'avatar', actor.avatar, 'avatar_url', CASE WHEN actor.avatar_asset_id IS NOT NULL
          THEN '/api/assets/v2/static/' || actor.avatar_asset_id::text || '/' ELSE actor.avatar END,
        'display_name', actor.display_name)
    )) FROM issue_votes vote JOIN users actor ON actor.id = vote.actor_id
      WHERE vote.issue_id = issue.id AND vote.deleted_at IS NULL),
    'reaction_items', (SELECT jsonb_agg(DISTINCT jsonb_build_object(
      'reaction', reaction.reaction,
      'actor_details', json_build_object('id', actor.id, 'first_name', actor.first_name, 'last_name', actor.last_name,
        'avatar', actor.avatar, 'avatar_url', CASE WHEN actor.avatar_asset_id IS NOT NULL
          THEN '/api/assets/v2/static/' || actor.avatar_asset_id::text || '/' ELSE actor.avatar END,
        'display_name', actor.display_name)
    )) FROM issue_reactions reaction JOIN users actor ON actor.id = reaction.actor_id
      WHERE reaction.issue_id = issue.id AND reaction.deleted_at IS NULL)
  )
  FROM issues issue
  LEFT JOIN states state ON state.id = issue.state_id
  JOIN projects project ON project.id = issue.project_id
  JOIN workspaces workspace ON workspace.id = issue.workspace_id
  WHERE issue.id = $1::uuid
    AND issue.project_id = $2::uuid
    AND issue.workspace_id = $3::uuid
    AND issue.deleted_at IS NULL
    AND issue.archived_at IS NULL
    AND issue.is_draft = FALSE
    AND state."group" IS DISTINCT FROM 'triage'
    AND project.deleted_at IS NULL
    AND project.archived_at IS NULL
    AND workspace.deleted_at IS NULL
), 'null'::json)
`, issueID, board.ProjectID, board.WorkspaceID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("read public issue detail: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Members(ctx context.Context, anchor string) (json.RawMessage, error) {
	workspaceSlug, projectID, err := repository.resolveAnchor(ctx, anchor)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'id', project_member.id,
  'member', project_member.member_id,
  'member__display_name', member.display_name,
  'member__avatar', member.avatar
) ORDER BY project_member.created_at DESC), '[]'::json)
FROM project_members project_member
JOIN workspaces workspace ON workspace.id = project_member.workspace_id
LEFT JOIN users member ON member.id = project_member.member_id
WHERE workspace.slug = $1
  AND project_member.project_id IS NOT DISTINCT FROM $2
  AND project_member.is_active = TRUE
  AND project_member.deleted_at IS NULL
`, workspaceSlug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: list public project members: %v", ErrRender, err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Settings(ctx context.Context, anchor string) (json.RawMessage, error) {
	return repository.deployBoardSettings(ctx, "anchor", anchor, uuid.Nil())
}

func (repository *Repository) ProjectAnchor(ctx context.Context, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	return repository.deployBoardSettings(ctx, "project", slug, projectID)
}

func (repository *Repository) deployBoardSettings(ctx context.Context, lookup, key string, projectID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	var matches int64
	err := repository.database.QueryRow(ctx, `
WITH requested_timezone AS MATERIALIZED (
  SELECT CASE WHEN $4 = '' THEN 'UTC' ELSE COALESCE(
    (SELECT user_timezone FROM users WHERE id = $4::uuid),
    'UTC'
  ) END AS name
), activated_timezone AS MATERIALIZED (
  SELECT set_config('TimeZone', requested_timezone.name, true) AS name
  FROM requested_timezone
)
SELECT json_build_object(
  'id', deploy_board.id,
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
      ELSE NULLIF(project.cover_image, '') END,
    'logo_props', project.logo_props,
    'description', project.description
  ) END,
  'workspace_detail', json_build_object(
    'name', workspace.name,
    'slug', workspace.slug,
    'id', workspace.id,
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
  ),
  'created_at', to_char(deploy_board.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(deploy_board.created_at, 'US') = '000000' THEN '' ELSE '.' || to_char(deploy_board.created_at, 'US') END ||
    CASE WHEN to_char(deploy_board.created_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(deploy_board.created_at, 'TZH:TZM') END,
  'updated_at', to_char(deploy_board.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') ||
    CASE WHEN to_char(deploy_board.updated_at, 'US') = '000000' THEN '' ELSE '.' || to_char(deploy_board.updated_at, 'US') END ||
    CASE WHEN to_char(deploy_board.updated_at, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(deploy_board.updated_at, 'TZH:TZM') END,
  'deleted_at', deploy_board.deleted_at,
  'entity_identifier', deploy_board.entity_identifier,
  'entity_name', deploy_board.entity_name,
  'anchor', deploy_board.anchor,
  'is_comments_enabled', deploy_board.is_comments_enabled,
  'is_reactions_enabled', deploy_board.is_reactions_enabled,
  'is_votes_enabled', deploy_board.is_votes_enabled,
  'view_props', deploy_board.view_props,
  'is_activity_enabled', deploy_board.is_activity_enabled,
  'is_disabled', deploy_board.is_disabled,
  'created_by', deploy_board.created_by_id,
  'updated_by', deploy_board.updated_by_id,
  'workspace', deploy_board.workspace_id,
  'project', deploy_board.project_id,
  'intake', deploy_board.intake_id
), count(*) OVER ()
FROM deploy_boards deploy_board
CROSS JOIN activated_timezone
JOIN workspaces workspace ON workspace.id = deploy_board.workspace_id
LEFT JOIN projects project ON project.id = deploy_board.project_id
LEFT JOIN file_assets project_cover_asset ON project_cover_asset.id = project.cover_image_asset_id
LEFT JOIN file_assets workspace_logo_asset ON workspace_logo_asset.id = workspace.logo_asset_id
LEFT JOIN workspaces project_cover_asset_workspace ON project_cover_asset_workspace.id = project_cover_asset.workspace_id
LEFT JOIN workspaces workspace_logo_asset_workspace ON workspace_logo_asset_workspace.id = workspace_logo_asset.workspace_id
WHERE (($1 = 'anchor' AND deploy_board.anchor = $2)
    OR ($1 = 'project' AND workspace.slug = $2 AND deploy_board.project_id = $3::uuid))
  AND deploy_board.entity_name = 'project'
  AND deploy_board.deleted_at IS NULL
`, lookup, key, projectID, UserID(ctx)).Scan(&data, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequiredObject
	}
	if err != nil {
		return nil, fmt.Errorf("read public project settings: %w", err)
	}
	if matches > 1 {
		return nil, fmt.Errorf("multiple public project settings objects")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, fmt.Errorf("compact public project settings: %w", err)
	}
	return compact.Bytes(), nil
}

func (repository *Repository) States(ctx context.Context, anchor string) (json.RawMessage, error) {
	workspaceSlug, projectID, err := repository.resolveAnchor(ctx, anchor)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(json_build_object(
  'name', state.name,
  'group', state."group",
  'color', state.color,
  'id', state.id,
  'sequence', state.sequence::text
) ORDER BY state.sequence), '[]'::json)
FROM states state
JOIN workspaces workspace ON workspace.id = state.workspace_id
WHERE workspace.slug = $1
  AND state.project_id IS NOT DISTINCT FROM $2
  AND state.deleted_at IS NULL
  AND state."group" <> 'triage'
  AND state.name <> 'Triage'
`, workspaceSlug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%w: list public project states: %v", ErrRender, err)
	}
	var records []stateRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("%w: decode public project states: %v", ErrRender, err)
	}
	responses := make([]stateResponse, 0, len(records))
	for _, record := range records {
		sequence, err := strconv.ParseFloat(record.Sequence, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: decode public project state sequence: %v", ErrRender, err)
		}
		encodedSequence, err := pythonFloatJSON(sequence)
		if err != nil {
			return nil, fmt.Errorf("%w: encode public project state sequence: %v", ErrRender, err)
		}
		responses = append(responses, stateResponse{
			Name:     record.Name,
			Group:    record.Group,
			Color:    record.Color,
			ID:       record.ID,
			Sequence: encodedSequence,
		})
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(responses); err != nil {
		return nil, fmt.Errorf("%w: encode public project states: %v", ErrRender, err)
	}
	return bytes.TrimSuffix(encoded.Bytes(), []byte("\n")), nil
}

func pythonFloatJSON(value float64) (json.RawMessage, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("non-finite float %v", value)
	}
	abs := math.Abs(value)
	format := byte('f')
	if abs != 0 && (abs < 1e-4 || abs >= 1e16) {
		format = 'e'
	}
	encoded := strconv.FormatFloat(value, format, -1, 64)
	if format == 'f' && !strings.Contains(encoded, ".") {
		encoded += ".0"
	}
	return json.RawMessage(encoded), nil
}

func (repository *Repository) resolveAnchor(ctx context.Context, anchor string) (string, *uuid.UUID, error) {
	var workspaceSlug string
	var projectID *uuid.UUID
	err := repository.database.QueryRow(ctx, `
SELECT workspace.slug, deploy_board.project_id
FROM deploy_boards deploy_board
JOIN workspaces workspace ON workspace.id = deploy_board.workspace_id
WHERE deploy_board.anchor = $1
  AND deploy_board.deleted_at IS NULL
ORDER BY deploy_board.created_at DESC
LIMIT 1
`, anchor).Scan(&workspaceSlug, &projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrInvalidAnchor
	}
	if err != nil {
		return "", nil, fmt.Errorf("resolve public project anchor: %w", err)
	}
	return workspaceSlug, projectID, nil
}
