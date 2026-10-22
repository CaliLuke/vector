// Package projectpublication persists project deploy-board publication state.
package projectpublication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrForbidden     = errors.New("project publication permission denied")
	ErrNotFound      = errors.New("project deploy board not found")
	ErrInvalidIntake = errors.New("invalid intake for project")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type SoftDeletePublisher interface {
	SoftDelete(context.Context, string, string, string) error
}
type Repository struct {
	db        Database
	publisher SoftDeletePublisher
}

func NewRepository(db Database, publisher SoftDeletePublisher) (*Repository, error) {
	if db == nil || publisher == nil {
		return nil, errors.New("project publication database and publisher are required")
	}
	return &Repository{db: db, publisher: publisher}, nil
}

// Authorize mirrors ProjectMemberPermission before timezone activation.
func (r *Repository) Authorize(ctx context.Context, userID, method, slug, projectID string) error {
	var allowed bool
	var err error
	switch method {
	case "GET", "HEAD", "OPTIONS":
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE w.slug=$1 AND p.id=$2::uuid AND pm.member_id=$3::uuid AND pm.is_active=true AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, slug, projectID, userID).Scan(&allowed)
	case "POST":
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2::uuid AND wm.is_active=true AND wm.role IN (15,20) AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, slug, userID).Scan(&allowed)
	default:
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE w.slug=$1 AND p.id=$2::uuid AND pm.member_id=$3::uuid AND pm.is_active=true AND pm.role IN (15,20) AND pm.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, slug, projectID, userID).Scan(&allowed)
	}
	if err != nil {
		return fmt.Errorf("authorize project publication: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

type CreateInput struct {
	Comments, Reactions, Votes bool
	Intake                     any
	Views                      any
}

func (r *Repository) Create(ctx context.Context, userID, slug, projectID string, input CreateInput) (json.RawMessage, error) {
	intake, err := r.intakeID(ctx, slug, projectID, input.Intake)
	if err != nil {
		return nil, err
	}
	views, err := json.Marshal(input.Views)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	anchor := strings.ReplaceAll(uuid.NewString(), "-", "")
	var boardID string
	err = r.db.QueryRow(ctx, `INSERT INTO deploy_boards (id,workspace_id,project_id,entity_name,entity_identifier,anchor,is_comments_enabled,is_reactions_enabled,intake_id,is_votes_enabled,view_props,is_activity_enabled,is_disabled,created_at,updated_at,created_by_id,updated_by_id)
SELECT $1::uuid,p.workspace_id,p.id,'project',p.id,$2,$3,$4,$5::uuid,$6,$7::jsonb,true,false,NOW(),NOW(),$8::uuid,$8::uuid FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$9::uuid AND w.slug=$10 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
ON CONFLICT (entity_name,entity_identifier) WHERE deleted_at IS NULL DO UPDATE SET intake_id=NULLIF(EXCLUDED.intake_id::text,'')::uuid,view_props=EXCLUDED.view_props,is_votes_enabled=EXCLUDED.is_votes_enabled,is_comments_enabled=EXCLUDED.is_comments_enabled,is_reactions_enabled=EXCLUDED.is_reactions_enabled,updated_at=NOW(),updated_by_id=EXCLUDED.updated_by_id
RETURNING id`, id, anchor, input.Comments, input.Reactions, intake, input.Votes, views, userID, projectID, slug).Scan(&boardID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("save project deploy board: %w", err)
	}
	return r.serialized(ctx, userID, slug, projectID, boardID)
}

// Validate a referenced intake before mutating the deploy board. Project scope
// prevents attaching an intake from a different project or workspace.
func (r *Repository) intakeID(ctx context.Context, slug, projectID string, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, ErrInvalidIntake
	}
	id, err := djangoinput.UUID(text)
	if err != nil {
		return nil, ErrInvalidIntake
	}
	var exists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM intakes i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE i.id=$1::uuid AND p.id=$2::uuid AND w.slug=$3 AND i.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, id.String(), projectID, slug).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidIntake
	}
	return id.String(), nil
}

type Patch struct{ Fields map[string]any }

func (r *Repository) Exists(ctx context.Context, slug, projectID, boardID string) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM deploy_boards b JOIN projects p ON p.id=b.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE b.id=$1::uuid AND b.project_id=p.id AND p.id=$2::uuid AND w.slug=$3 AND b.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, boardID, projectID, slug).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) Update(ctx context.Context, userID, slug, projectID, boardID string, patch Patch) (json.RawMessage, error) {
	if _, err := uuid.Parse(boardID); err != nil {
		return nil, ErrNotFound
	}
	parameter := func(field string) ([]byte, bool, error) {
		value, present := patch.Fields[field]
		if !present {
			return nil, false, nil
		}
		data, err := json.Marshal(value)
		return data, true, err
	}
	viewProps, hasViewProps, err := parameter("view_props")
	if err != nil {
		return nil, err
	}
	if value, present := patch.Fields["intake"]; present {
		if _, err := r.intakeID(ctx, slug, projectID, value); err != nil {
			return nil, err
		}
	}
	intake, hasIntake, err := parameter("intake")
	if err != nil {
		return nil, err
	}
	entityName, hasEntityName, err := parameter("entity_name")
	if err != nil {
		return nil, err
	}
	entityIdentifier, hasEntityIdentifier, err := parameter("entity_identifier")
	if err != nil {
		return nil, err
	}
	booleans := make(map[string]bool, 5)
	for _, field := range []string{"is_activity_enabled", "is_comments_enabled", "is_reactions_enabled", "is_votes_enabled", "is_disabled"} {
		value, ok := patch.Fields[field]
		if ok {
			booleans[field] = value.(bool)
		}
	}
	command, err := r.db.Exec(ctx, `UPDATE deploy_boards b SET
view_props=CASE WHEN @has_view_props THEN @view_props::jsonb ELSE b.view_props END,
intake_id=CASE WHEN @has_intake THEN NULLIF(@intake::jsonb #>> '{}','null')::uuid ELSE b.intake_id END,
entity_name=CASE WHEN @has_entity_name THEN @entity_name::jsonb #>> '{}' ELSE b.entity_name END,
entity_identifier=CASE WHEN @has_entity_identifier THEN NULLIF(@entity_identifier::jsonb #>> '{}','null')::uuid ELSE b.entity_identifier END,
is_activity_enabled=CASE WHEN @has_is_activity_enabled THEN @is_activity_enabled ELSE b.is_activity_enabled END,
is_comments_enabled=CASE WHEN @has_is_comments_enabled THEN @is_comments_enabled ELSE b.is_comments_enabled END,
is_reactions_enabled=CASE WHEN @has_is_reactions_enabled THEN @is_reactions_enabled ELSE b.is_reactions_enabled END,
is_votes_enabled=CASE WHEN @has_is_votes_enabled THEN @is_votes_enabled ELSE b.is_votes_enabled END,
is_disabled=CASE WHEN @has_is_disabled THEN @is_disabled ELSE b.is_disabled END,
updated_at=NOW(),updated_by_id=@user_id::uuid
FROM projects p JOIN workspaces w ON w.id=p.workspace_id
WHERE b.id=@board_id::uuid AND b.project_id=p.id AND p.id=@project_id::uuid AND w.slug=@slug AND b.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, pgx.StrictNamedArgs{
		"has_view_props": hasViewProps, "view_props": viewProps, "has_intake": hasIntake, "intake": intake,
		"has_entity_name": hasEntityName, "entity_name": entityName, "has_entity_identifier": hasEntityIdentifier, "entity_identifier": entityIdentifier,
		"has_is_activity_enabled": hasBoolean(booleans, "is_activity_enabled"), "is_activity_enabled": booleans["is_activity_enabled"],
		"has_is_comments_enabled": hasBoolean(booleans, "is_comments_enabled"), "is_comments_enabled": booleans["is_comments_enabled"],
		"has_is_reactions_enabled": hasBoolean(booleans, "is_reactions_enabled"), "is_reactions_enabled": booleans["is_reactions_enabled"],
		"has_is_votes_enabled": hasBoolean(booleans, "is_votes_enabled"), "is_votes_enabled": booleans["is_votes_enabled"],
		"has_is_disabled": hasBoolean(booleans, "is_disabled"), "is_disabled": booleans["is_disabled"],
		"user_id": userID, "board_id": boardID, "project_id": projectID, "slug": slug,
	})
	if err != nil {
		return nil, fmt.Errorf("patch project deploy board: %w", err)
	}
	if command.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.serialized(ctx, userID, slug, projectID, boardID)
}

func hasBoolean(values map[string]bool, key string) bool { _, ok := values[key]; return ok }

func (r *Repository) Delete(ctx context.Context, userID, slug, projectID, boardID string) error {
	command, err := r.db.Exec(ctx, `UPDATE deploy_boards b SET deleted_at=NOW(),updated_at=NOW(),updated_by_id=$1::uuid FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE b.id=$2::uuid AND b.project_id=p.id AND p.id=$3::uuid AND w.slug=$4 AND b.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, userID, boardID, projectID, slug)
	if err != nil {
		return fmt.Errorf("soft delete project deploy board: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return r.publisher.SoftDelete(ctx, "db", "deployboard", boardID)
}

func (r *Repository) serialized(ctx context.Context, userID, slug, projectID, boardID string) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, publicationBoardTimezoneSQL+`SELECT `+publicationBoardJSONSQL+`
FROM deploy_boards deploy_board
CROSS JOIN activated_timezone
JOIN workspaces workspace ON workspace.id=deploy_board.workspace_id
LEFT JOIN projects project ON project.id=deploy_board.project_id
LEFT JOIN file_assets project_cover_asset ON project_cover_asset.id=project.cover_image_asset_id
LEFT JOIN file_assets workspace_logo_asset ON workspace_logo_asset.id=workspace.logo_asset_id
LEFT JOIN workspaces project_cover_asset_workspace ON project_cover_asset_workspace.id=project_cover_asset.workspace_id
LEFT JOIN workspaces workspace_logo_asset_workspace ON workspace_logo_asset_workspace.id=workspace_logo_asset.workspace_id
WHERE deploy_board.id=$2::uuid AND deploy_board.project_id=$3::uuid AND project.id=$3::uuid AND workspace.slug=$4 AND deploy_board.deleted_at IS NULL AND project.deleted_at IS NULL AND workspace.deleted_at IS NULL
LIMIT 1`, userID, boardID, projectID, slug).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

const publicationBoardTimezoneSQL = `WITH requested_timezone AS MATERIALIZED (
SELECT COALESCE((SELECT NULLIF(user_timezone,'') FROM users WHERE id=$1::uuid),'UTC') AS name
), activated_timezone AS MATERIALIZED (
SELECT set_config('TimeZone',requested_timezone.name,true) AS name FROM requested_timezone
) `

// This is the DeployBoardSerializer JSON projection used by the existing read
// owner. Keep fields and asset URL rules in sync with that source-backed shape.
const publicationBoardJSONSQL = `json_build_object(
'anchor',deploy_board.anchor,
'created_at',to_char(deploy_board.created_at,'YYYY-MM-DD"T"HH24:MI:SS')||CASE WHEN to_char(deploy_board.created_at,'US')='000000' THEN '' ELSE '.'||to_char(deploy_board.created_at,'US') END||CASE WHEN to_char(deploy_board.created_at,'TZH:TZM')='+00:00' THEN 'Z' ELSE to_char(deploy_board.created_at,'TZH:TZM') END,
'created_by',deploy_board.created_by_id,'deleted_at',deploy_board.deleted_at,'entity_identifier',deploy_board.entity_identifier,'entity_name',deploy_board.entity_name,'id',deploy_board.id,'intake',deploy_board.intake_id,'is_activity_enabled',deploy_board.is_activity_enabled,'is_comments_enabled',deploy_board.is_comments_enabled,'is_disabled',deploy_board.is_disabled,'is_reactions_enabled',deploy_board.is_reactions_enabled,'is_votes_enabled',deploy_board.is_votes_enabled,'project',deploy_board.project_id,
'project_details',CASE WHEN project.id IS NULL THEN NULL ELSE json_build_object('id',project.id,'identifier',project.identifier,'name',project.name,'cover_image',project.cover_image,'cover_image_url',CASE WHEN project.cover_image_asset_id IS NOT NULL AND project_cover_asset.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||project.cover_image_asset_id::text||'/' WHEN project_cover_asset.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/'||project_cover_asset_workspace.slug||'/projects/'||project_cover_asset.project_id::text||'/issues/'||project_cover_asset.issue_id::text||'/attachments/'||project_cover_asset.id::text||'/' WHEN project_cover_asset.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/'||project_cover_asset_workspace.slug||'/projects/'||project_cover_asset.project_id::text||'/'||project_cover_asset.id::text||'/' WHEN project.cover_image_asset_id IS NOT NULL THEN NULL ELSE NULLIF(project.cover_image,'') END,'logo_props',project.logo_props,'description',project.description) END,
'updated_at',to_char(deploy_board.updated_at,'YYYY-MM-DD"T"HH24:MI:SS')||CASE WHEN to_char(deploy_board.updated_at,'US')='000000' THEN '' ELSE '.'||to_char(deploy_board.updated_at,'US') END||CASE WHEN to_char(deploy_board.updated_at,'TZH:TZM')='+00:00' THEN 'Z' ELSE to_char(deploy_board.updated_at,'TZH:TZM') END,
'updated_by',deploy_board.updated_by_id,'view_props',deploy_board.view_props,'workspace',deploy_board.workspace_id,
'workspace_detail',json_build_object('id',workspace.id,'name',workspace.name,'slug',workspace.slug,'logo_url',CASE WHEN workspace.logo_asset_id IS NOT NULL AND workspace_logo_asset.entity_type IN ('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||workspace.logo_asset_id::text||'/' WHEN workspace_logo_asset.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/'||workspace_logo_asset_workspace.slug||'/projects/'||workspace_logo_asset.project_id::text||'/issues/'||workspace_logo_asset.issue_id::text||'/attachments/'||workspace_logo_asset.id::text||'/' WHEN workspace_logo_asset.entity_type IN ('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/'||workspace_logo_asset_workspace.slug||'/projects/'||workspace_logo_asset.project_id::text||'/'||workspace_logo_asset.id::text||'/' WHEN workspace.logo_asset_id IS NOT NULL THEN NULL ELSE NULLIF(workspace.logo,'') END)
)`
