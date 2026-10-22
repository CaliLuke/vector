package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type assetDatabase interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type PostgresRepository struct{ database assetDatabase }

func NewPostgresRepository(database assetDatabase) (*PostgresRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("assets database is required")
	}
	return &PostgresRepository{database: database}, nil
}

func (repository *PostgresRepository) AssetKey(ctx context.Context, assetID uuid.UUID) (string, bool, error) {
	var key string
	err := repository.database.QueryRow(ctx, `SELECT asset FROM file_assets WHERE id=$1 AND deleted_at IS NULL`, assetID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load FileAsset %s for metadata refresh: %w", assetID, err)
	}
	return key, true, nil
}

// SaveMetadataForKey keeps the asset and observed object key paired. A deleted
// or replaced asset makes the fetched metadata stale and therefore a no-op.
func (repository *PostgresRepository) SaveMetadataForKey(ctx context.Context, assetID uuid.UUID, key string, metadata any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode FileAsset %s storage metadata: %w", assetID, err)
	}
	if _, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET storage_metadata=$3::jsonb
		WHERE id=$1 AND asset=$2 AND deleted_at IS NULL
	`, assetID, key, encoded); err != nil {
		return fmt.Errorf("save FileAsset %s storage metadata: %w", assetID, err)
	}
	return nil
}

func (repository *PostgresRepository) AuthorizedWorkspaceID(ctx context.Context, userID uuid.UUID, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `
		SELECT id FROM workspaces WHERE slug=$1 AND deleted_at IS NULL
	`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Workspace not found"}}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve asset workspace: %w", err)
	}
	var member bool
	if err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active=true AND deleted_at IS NULL)
	`, id, userID).Scan(&member); err != nil {
		return uuid.Nil, fmt.Errorf("check asset workspace membership: %w", err)
	}
	if !member {
		return uuid.Nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	return id, nil
}

func (repository *PostgresRepository) WorkspaceIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve asset workspace: %w", err)
	}
	return id, nil
}

// RelatedWorkspaceIDBySlug resolves a workspace through a related-row scope.
// Django's related joins do not apply the workspace default-manager tombstone
// filter, so surviving asset rows remain addressable after workspace deletion.
func (repository *PostgresRepository) RelatedWorkspaceIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug=$1`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve related asset workspace: %w", err)
	}
	return id, nil
}

// WorkspaceMemberBySlug mirrors WorkspaceUserPermission: the permission is
// scoped by the URL slug and requires an active membership. A missing slug or
// inactive workspace membership is a permission denial at this boundary.
func (repository *PostgresRepository) WorkspaceMemberBySlug(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	var member bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members wm
			JOIN workspaces w ON w.id=wm.workspace_id
			WHERE w.slug=$1
			  AND wm.member_id=$2 AND wm.is_active=true AND wm.deleted_at IS NULL
		)
	`, slug, userID).Scan(&member)
	if err != nil {
		return false, fmt.Errorf("check asset workspace slug membership: %w", err)
	}
	return member, nil
}

// WorkspacePermissionAllowed mirrors V2 allow_permission(..., level="WORKSPACE")
// with the source's ADMIN/MEMBER/GUEST role set. As with WorkspaceMemberBySlug,
// the related Workspace lookup does not apply Workspace's default manager.
func (repository *PostgresRepository) WorkspacePermissionAllowed(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members wm
			JOIN workspaces w ON w.id=wm.workspace_id
			WHERE w.slug=$1 AND wm.member_id=$2 AND wm.role IN (5,15,20)
			  AND wm.is_active=true AND wm.deleted_at IS NULL
		)
	`, slug, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check V2 asset workspace permission: %w", err)
	}
	return allowed, nil
}

func (repository *PostgresRepository) WorkspaceMemberByID(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var member bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active=true AND deleted_at IS NULL)`, workspaceID, userID).Scan(&member)
	if err != nil {
		return false, fmt.Errorf("check asset workspace membership: %w", err)
	}
	return member, nil
}

func (repository *PostgresRepository) WorkspaceRole(ctx context.Context, workspaceID, userID uuid.UUID) (int, error) {
	var role int
	err := repository.database.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active=true AND deleted_at IS NULL`, workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load asset workspace role: %w", err)
	}
	return role, nil
}

func (repository *PostgresRepository) CreateWorkspaceEntityUpload(ctx context.Context, id, userID, workspaceID, parentProjectID uuid.UUID, key string, attributes []byte, size int64, entityType, entityID string, projectAsset bool) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	var projectID, userTarget, issueID, commentID, pageID, draftIssueID any
	if parentProjectID != uuid.Nil {
		projectID = parentProjectID
	}
	if projectAsset && entityType == "PROJECT_COVER" {
		parsed, err := uuid.Parse(entityID)
		if err != nil || parsed != parentProjectID {
			return projectCoverEntityMismatchError()
		}
	}
	if entityID != "" {
		parsed, err := uuid.Parse(entityID)
		if err != nil {
			return err
		}
		switch entityType {
		case "PROJECT_COVER":
			if projectAsset {
				if parsed != parentProjectID {
					return projectCoverEntityMismatchError()
				}
				projectID = parentProjectID
			} else {
				projectID = parsed
			}
		case "USER_AVATAR", "USER_COVER":
			userTarget = parsed
		case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION":
			issueID = parsed
		case "DRAFT_ISSUE_DESCRIPTION":
			if projectAsset {
				draftIssueID = parsed
			}
		case "COMMENT_DESCRIPTION":
			commentID = parsed
		case "PAGE_DESCRIPTION":
			pageID = parsed
		}
	}
	_, err := repository.database.Exec(ctx, `
INSERT INTO file_assets
 (id,created_at,updated_at,attributes,asset,user_id,workspace_id,project_id,draft_issue_id,issue_id,comment_id,page_id,entity_type,entity_identifier,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
VALUES ($1,NOW(),NOW(),$2::jsonb,$3,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,''),$14,false,false,false,'{}'::jsonb,$4)
`, id, attributes, key, userID, userTarget, workspaceID, projectID, draftIssueID, issueID, commentID, pageID, entityType, "", float64(size))
	if err != nil {
		return fmt.Errorf("insert workspace FileAsset: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) CreateUserUpload(ctx context.Context, id, userID uuid.UUID, key string, attributes []byte, size int64, entityType string) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	_, err := repository.database.Exec(ctx, `
INSERT INTO file_assets
 (id, created_at, updated_at, attributes, asset, user_id, entity_type, size, is_uploaded, is_deleted, is_archived, storage_metadata, created_by_id)
VALUES ($1, NOW(), NOW(), $2::jsonb, $3, $4, $5, $6, false, false, false, '{}'::jsonb, $4)
`, id, attributes, key, userID, entityType, float64(size))
	if err != nil {
		return fmt.Errorf("insert user file asset: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) CreateWorkspaceUpload(ctx context.Context, id, userID, workspaceID uuid.UUID, key string, attributes []byte, size int64, projectID, externalID, externalSource string) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	var project any
	if projectID != "" {
		parsed, err := uuid.Parse(projectID)
		if err != nil {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"project_id": []string{"Must be a valid UUID."}}}
		}
		project = parsed
	}
	_, err := repository.database.Exec(ctx, `
INSERT INTO file_assets
 (id, created_at, updated_at, attributes, asset, workspace_id, project_id, external_id, external_source, entity_type, size, is_uploaded, is_deleted, is_archived, storage_metadata, created_by_id)
VALUES ($1, NOW(), NOW(), $2::jsonb, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), 'ISSUE_ATTACHMENT', $8, false, false, false, '{}'::jsonb, $9)
`, id, attributes, key, workspaceID, project, externalID, externalSource, float64(size), userID)
	if err != nil {
		return fmt.Errorf("insert workspace file asset: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) CreateIssueAttachment(ctx context.Context, id, userID, workspaceID, projectID, issueID uuid.UUID, key string, attributes []byte, size int64, externalID, externalSource string) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("issue attachment attributes must be valid JSON")
	}
	_, err := repository.database.Exec(ctx, `
		INSERT INTO file_assets
			 (id,created_at,updated_at,attributes,asset,workspace_id,project_id,issue_id,entity_type,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
		VALUES ($1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,'ISSUE_ATTACHMENT',NULLIF($7,''),NULLIF($8,''),$9,false,false,false,'{}'::jsonb,$10)
	`, id, attributes, key, workspaceID, projectID, issueID, externalID, externalSource, float64(size), userID)
	if err != nil {
		return fmt.Errorf("insert issue attachment FileAsset: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) IssueAttachmentExternal(ctx context.Context, workspaceID, projectID, issueID uuid.UUID, externalID, externalSource string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `SELECT id FROM file_assets WHERE workspace_id=$1 AND project_id=$2 AND issue_id=$3 AND entity_type='ISSUE_ATTACHMENT' AND external_id=$4 AND external_source=$5 AND deleted_at IS NULL ORDER BY created_at ASC LIMIT 1`, workspaceID, projectID, issueID, externalID, externalSource).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("find duplicate issue attachment: %w", err)
	}
	return id, true, nil
}

func (repository *PostgresRepository) BulkAssociateProjectAssets(ctx context.Context, workspaceID, projectID, userID, entityID uuid.UUID, assetIDs []uuid.UUID) error {
	var entityType string
	err := repository.database.QueryRow(ctx, `
		SELECT entity_type FROM file_assets
		WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL
		AND (project_id=$4 OR project_id IS NULL)
		ORDER BY created_at DESC LIMIT 1`, assetIDs, workspaceID, userID, projectID).Scan(&entityType)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	if err != nil {
		return fmt.Errorf("find project bulk assets: %w", err)
	}
	if entityType == "PROJECT_COVER" {
		var ids []uuid.UUID
		if err := repository.database.QueryRow(ctx, `SELECT COALESCE(array_agg(id ORDER BY created_at DESC), '{}'::uuid[]) FROM file_assets WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL AND (project_id=$4 OR project_id IS NULL)`, assetIDs, workspaceID, userID, projectID).Scan(&ids); err != nil {
			return fmt.Errorf("list project cover assets: %w", err)
		}
		if _, err := repository.database.Exec(ctx, `UPDATE file_assets SET project_id=$1 WHERE id=ANY($2::uuid[]) AND workspace_id=$3 AND created_by_id=$4 AND deleted_at IS NULL AND (project_id=$1 OR project_id IS NULL)`, projectID, assetIDs, workspaceID, userID); err != nil {
			return fmt.Errorf("associate project cover assets: %w", err)
		}
		var projectExists bool
		if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND deleted_at IS NULL)`, projectID).Scan(&projectExists); err != nil {
			return fmt.Errorf("load active project for cover association: %w", err)
		}
		if !projectExists {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		for _, id := range ids {
			if _, err := repository.database.Exec(ctx, `UPDATE projects SET cover_image_asset_id=$1, updated_by_id=$3, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, id, projectID, userID); err != nil {
				return fmt.Errorf("set project cover asset: %w", err)
			}
		}
		return nil
	}
	column, value := "", any(entityID)
	switch entityType {
	case "ISSUE_DESCRIPTION":
		column = "issue_id"
	case "COMMENT_DESCRIPTION":
		column = "comment_id"
	case "PAGE_DESCRIPTION":
		column = "page_id"
	case "DRAFT_ISSUE_DESCRIPTION":
		column = "draft_issue_id"
	default:
		return nil
	}
	query := `UPDATE file_assets SET ` + column + `=$1`
	args := []any{value, assetIDs, workspaceID, userID, projectID}
	query += ` WHERE id=ANY($2::uuid[]) AND workspace_id=$3 AND created_by_id=$4 AND deleted_at IS NULL AND (project_id=$5 OR project_id IS NULL)`
	if entityType == "ISSUE_DESCRIPTION" {
		query = `UPDATE file_assets SET issue_id=$1, project_id=$5` + query[len(`UPDATE file_assets SET issue_id=$1`):]
	}
	if _, err := repository.database.Exec(ctx, query, args...); err != nil {
		// The source deliberately ignores a deleted target's FK violation for
		// issue, comment, and draft-issue association requests.
		var pgErr *pgconn.PgError
		if (entityType == "ISSUE_DESCRIPTION" || entityType == "COMMENT_DESCRIPTION" || entityType == "DRAFT_ISSUE_DESCRIPTION") && errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "23") {
			return nil
		}
		return fmt.Errorf("associate project assets: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) DeleteLegacyIssueAttachment(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET asset='', deleted_at=NOW(), updated_by_id=$5, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4 AND deleted_at IS NULL`, assetID, workspaceID, projectID, issueID, userID)
	if err != nil {
		return fmt.Errorf("delete legacy issue attachment row: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue attachment not found."}}
	}
	return nil
}

func (repository *PostgresRepository) IssuePermission(ctx context.Context, userID, workspaceID, projectID, issueID uuid.UUID) (bool, bool, bool, error) {
	var found, member, creator bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND i.workspace_id=$2 AND i.project_id=$3 AND i.deleted_at IS NULL),
		 EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=$3 AND pm.member_id=$4 AND pm.is_active=true AND pm.deleted_at IS NULL AND pm.role IN (5,15,20)),
		 EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND i.workspace_id=$2 AND i.project_id=$3 AND i.deleted_at IS NULL AND i.created_by_id=$4)
	`, issueID, workspaceID, projectID, userID).Scan(&found, &member, &creator)
	if err != nil {
		return false, false, false, fmt.Errorf("check issue attachment permission: %w", err)
	}
	return found, member, creator, nil
}

func (repository *PostgresRepository) UpdateAPIFileAssetUploaded(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, created_by_id=CASE WHEN is_uploaded THEN created_by_id ELSE $4 END, updated_by_id=$4, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL`, assetID, workspaceID, projectID, userID)
	if err != nil {
		return fmt.Errorf("confirm API issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) DeleteAPIFileAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_by_id=$4, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL`, assetID, workspaceID, projectID, userID)
	if err != nil {
		return fmt.Errorf("delete API issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) CreateLegacyFileAsset(ctx context.Context, id, userID uuid.UUID, fields map[string]any, key string) (map[string]any, error) {
	attributes := legacyJSONField(fields, "attributes", `{}`)
	metadata := legacyJSONField(fields, "storage_metadata", `{}`)
	_, err := repository.database.Exec(ctx, `
		INSERT INTO file_assets
			 (id,created_at,updated_at,attributes,asset,user_id,workspace_id,draft_issue_id,project_id,issue_id,comment_id,page_id,entity_type,entity_identifier,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id,deleted_at)
		VALUES ($1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::jsonb,$20,$21)
	`, id, attributes, key, legacyUUIDField(fields, "user"), legacyUUIDField(fields, "workspace"), legacyUUIDField(fields, "draft_issue"), legacyUUIDField(fields, "project"), legacyUUIDField(fields, "issue"), legacyUUIDField(fields, "comment"), legacyUUIDField(fields, "page"), legacyNullableStringField(fields, "entity_type"), legacyNullableStringField(fields, "entity_identifier"), legacyNullableStringField(fields, "external_id"), legacyNullableStringField(fields, "external_source"), legacyFloatField(fields, "size"), legacyBoolField(fields, "is_uploaded"), legacyBoolField(fields, "is_deleted"), legacyBoolField(fields, "is_archived"), metadata, userID, legacyTimeField(fields, "deleted_at"))
	if err != nil {
		return nil, fmt.Errorf("insert legacy FileAsset: %w", err)
	}
	var encoded []byte
	if err := repository.database.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'asset', asset, 'attributes', attributes, 'comment', comment_id,
			'created_at', created_at, 'created_by', created_by_id, 'deleted_at', deleted_at,
			'draft_issue', draft_issue_id, 'entity_identifier', entity_identifier,
			'entity_type', entity_type, 'external_id', external_id, 'external_source', external_source,
			'id', id, 'is_archived', is_archived, 'is_deleted', is_deleted, 'is_uploaded', is_uploaded,
			'issue', issue_id, 'page', page_id, 'project', project_id, 'size', size,
			'storage_metadata', COALESCE(storage_metadata,'{}'::jsonb), 'updated_at', updated_at,
			'updated_by', updated_by_id, 'user', user_id, 'workspace', workspace_id
		) FROM file_assets WHERE id=$1
	`, id).Scan(&encoded); err != nil {
		return nil, fmt.Errorf("serialize legacy FileAsset: %w", err)
	}
	var response map[string]any
	if err := json.Unmarshal(encoded, &response); err != nil {
		return nil, fmt.Errorf("decode legacy FileAsset: %w", err)
	}
	return response, nil
}

func (repository *PostgresRepository) LegacyRelationExists(ctx context.Context, field string, id uuid.UUID) (bool, error) {
	tables := map[string]string{
		"user": "users", "workspace": "workspaces", "draft_issue": "issues", "project": "projects",
		"issue": "issues", "comment": "issue_comments", "page": "pages",
	}
	table := tables[field]
	if table == "" {
		return false, fmt.Errorf("unknown FileAsset relation %q", field)
	}
	filterDeleted := field != "user"
	query := "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE id=$1"
	if filterDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += ")"
	var exists bool
	if err := repository.database.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (repository *PostgresRepository) CreateLegacyIssueAttachment(ctx context.Context, id, userID, workspaceID, projectID, issueID uuid.UUID, fields map[string]any, key string) error {
	attributes, metadata := legacyJSONField(fields, "attributes", `{}`), legacyJSONField(fields, "storage_metadata", `{}`)
	_, err := repository.database.Exec(ctx, `
		INSERT INTO file_assets (id,created_at,updated_at,attributes,asset,user_id,workspace_id,draft_issue_id,project_id,issue_id,comment_id,page_id,entity_type,entity_identifier,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id,deleted_at)
		VALUES ($1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,'ISSUE_ATTACHMENT',$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19,$20)
	`, id, attributes, key, legacyUUIDField(fields, "user"), workspaceID, legacyUUIDField(fields, "draft_issue"), projectID, issueID, legacyUUIDField(fields, "comment"), legacyUUIDField(fields, "page"), legacyNullableStringField(fields, "entity_identifier"), legacyNullableStringField(fields, "external_id"), legacyNullableStringField(fields, "external_source"), legacyFloatField(fields, "size"), legacyBoolField(fields, "is_uploaded"), legacyBoolField(fields, "is_deleted"), legacyBoolField(fields, "is_archived"), metadata, userID, legacyTimeField(fields, "deleted_at"))
	if err != nil {
		return fmt.Errorf("insert legacy issue attachment: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) PublicProjectBoard(ctx context.Context, anchor string, projectOnly bool) (uuid.UUID, uuid.UUID, string, bool, error) {
	var workspaceID uuid.UUID
	var projectID uuid.UUID
	var workspaceSlug string
	err := repository.database.QueryRow(ctx, `SELECT workspace.id, project.id, workspace.slug
		FROM deploy_boards board
		JOIN workspaces workspace ON workspace.id=board.workspace_id
		JOIN projects project ON project.id=board.entity_identifier
			AND project.id=board.project_id AND project.workspace_id=board.workspace_id
		WHERE board.anchor=$1 AND board.entity_name='project'
			AND board.deleted_at IS NULL AND board.is_disabled=FALSE
			AND workspace.deleted_at IS NULL
			AND project.deleted_at IS NULL AND project.archived_at IS NULL
		ORDER BY board.created_at DESC LIMIT 1`, anchor).Scan(&workspaceID, &projectID, &workspaceSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false, fmt.Errorf("resolve public asset board: %w", err)
	}
	_ = projectOnly // Public assets are only valid under a live project publication.
	return workspaceID, projectID, workspaceSlug, true, nil
}

func (repository *PostgresRepository) PublicProjectAsset(ctx context.Context, workspaceID, projectID, assetID uuid.UUID, includeDeleted, descriptionsOnly bool) (FileAsset, error) {
	asset, err := repository.findAsset(ctx, assetID, workspaceID, true, !includeDeleted, false)
	if err != nil {
		return FileAsset{}, err
	}
	if asset.ProjectID == nil || *asset.ProjectID != projectID {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if descriptionsOnly && asset.EntityType != "ISSUE_DESCRIPTION" && asset.EntityType != "COMMENT_DESCRIPTION" {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return asset, nil
}

func (repository *PostgresRepository) CreatePublicUpload(ctx context.Context, id, userID, workspaceID, projectID uuid.UUID, key string, attributes []byte, size int64, entityType, entityIdentifier string) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("public asset attributes must be valid JSON")
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public FileAsset insert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var relationID any
	relationColumn := "comment_id"
	if entityIdentifier != "" {
		parsedID, parseErr := uuid.Parse(entityIdentifier)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
		relationID = parsedID
		switch entityType {
		case "COMMENT_DESCRIPTION":
			var found uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM issue_comments WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND created_by_id=$4 AND deleted_at IS NULL FOR KEY SHARE`, parsedID, workspaceID, projectID, userID).Scan(&found); errors.Is(err, pgx.ErrNoRows) {
				return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
			} else if err != nil {
				return fmt.Errorf("validate public upload comment: %w", err)
			}
		case "ISSUE_DESCRIPTION":
			relationColumn = "issue_id"
			var found uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT id FROM issues WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL FOR KEY SHARE`, parsedID, workspaceID, projectID).Scan(&found); errors.Is(err, pgx.ErrNoRows) {
				return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
			} else if err != nil {
				return fmt.Errorf("validate public upload issue: %w", err)
			}
		default:
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity identifier.", "status": false}}
		}
	}
	query := `INSERT INTO file_assets
		 (id,created_at,updated_at,attributes,asset,workspace_id,project_id,` + relationColumn + `,entity_type,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
		VALUES (@id,NOW(),NOW(),@attributes::jsonb,@asset,@workspace,@project,@relation,@entity_type,@size,false,false,false,'{}'::jsonb,@actor)`
	_, err = tx.Exec(ctx, query,
		pgx.StrictNamedArgs{"id": id, "attributes": attributes, "asset": key, "workspace": workspaceID,
			"project": projectID, "relation": relationID, "entity_type": entityType, "size": float64(size), "actor": userID})
	if err != nil {
		return fmt.Errorf("insert public FileAsset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit public FileAsset insert: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) UpdatePublicAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID, attributes []byte) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("public asset attributes must be valid JSON")
	}
	result, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_uploaded=true, attributes=@attributes::jsonb
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND deleted_at IS NULL
	`, pgx.StrictNamedArgs{"id": assetID, "workspace": workspaceID, "project": projectID, "attributes": attributes, "actor": userID})
	if err != nil {
		return fmt.Errorf("confirm public FileAsset upload: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) DeletePublicAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_deleted=true, deleted_at=NOW()
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND deleted_at IS NULL
	`, pgx.StrictNamedArgs{"id": assetID, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("delete public FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) RestorePublicAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_deleted=false, deleted_at=NULL
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor
	`, pgx.StrictNamedArgs{"id": assetID, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("restore public FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) BulkAssociatePublicAssetComments(ctx context.Context, userID, workspaceID, projectID, commentID uuid.UUID, assetIDs []uuid.UUID) error {
	ids := uniquePublicAssetIDs(assetIDs)
	if len(ids) == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public FileAsset association: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var authorizedComment uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM issue_comments WHERE id=@comment AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND deleted_at IS NULL FOR KEY SHARE`, pgx.StrictNamedArgs{"comment": commentID, "workspace": workspaceID, "project": projectID, "actor": userID}).Scan(&authorizedComment); errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	} else if err != nil {
		return fmt.Errorf("validate public asset comment association: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT id FROM file_assets
		WHERE id=ANY(@ids) AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor
			AND entity_type='COMMENT_DESCRIPTION' AND deleted_at IS NULL
		FOR UPDATE`, pgx.StrictNamedArgs{"ids": ids, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("validate public FileAsset association: %w", err)
	}
	matched := make(map[uuid.UUID]struct{}, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan public FileAsset association: %w", err)
		}
		matched[id] = struct{}{}
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return fmt.Errorf("read public FileAsset association: %w", rowsErr)
	}
	if len(matched) != len(ids) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	result, err := tx.Exec(ctx, `UPDATE file_assets SET comment_id=@comment
		WHERE id=ANY(@ids) AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor
			AND entity_type='COMMENT_DESCRIPTION' AND deleted_at IS NULL`, pgx.StrictNamedArgs{"comment": commentID, "ids": ids, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("associate public FileAssets with comment: %w", err)
	}
	if result.RowsAffected() != int64(len(ids)) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit public FileAsset association: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) PublicBulkAssetRows(ctx context.Context, userID, workspaceID, projectID uuid.UUID, assetIDs []uuid.UUID) ([]uuid.UUID, string, error) {
	ids := uniquePublicAssetIDs(assetIDs)
	var encoded []byte
	err := repository.database.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(jsonb_build_object('id',id,'entity_type',entity_type) ORDER BY created_at DESC),'[]'::jsonb)
		FROM file_assets
		WHERE id=ANY($1) AND workspace_id=$2 AND project_id=$3 AND created_by_id=$4 AND deleted_at IS NULL
	`, ids, workspaceID, projectID, userID).Scan(&encoded)
	if err != nil {
		return nil, "", fmt.Errorf("list public bulk FileAssets: %w", err)
	}
	var rows []struct {
		ID         uuid.UUID `json:"id"`
		EntityType string    `json:"entity_type"`
	}
	if err := json.Unmarshal(encoded, &rows); err != nil {
		return nil, "", fmt.Errorf("decode public bulk FileAssets: %w", err)
	}
	matchedIDs := make([]uuid.UUID, 0, len(rows))
	entityType := ""
	for i, row := range rows {
		if i == 0 {
			entityType = row.EntityType
		} else if row.EntityType != entityType {
			return nil, "", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
		}
		matchedIDs = append(matchedIDs, row.ID)
	}
	if len(matchedIDs) != len(ids) {
		return nil, "", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	return matchedIDs, entityType, nil
}

func uniquePublicAssetIDs(assetIDs []uuid.UUID) []uuid.UUID {
	unique := make([]uuid.UUID, 0, len(assetIDs))
	seen := make(map[uuid.UUID]struct{}, len(assetIDs))
	for _, id := range assetIDs {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func (repository *PostgresRepository) IssueAttachments(ctx context.Context, workspaceID, projectID, issueID uuid.UUID, uploadedOnly bool) ([]map[string]any, error) {
	var encoded []byte
	query := `SELECT COALESCE(jsonb_agg(jsonb_build_object(
		'asset', f.asset,
		'attributes', f.attributes, 'comment', f.comment_id, 'created_at', f.created_at,
		'created_by', f.created_by_id, 'deleted_at', f.deleted_at, 'draft_issue', f.draft_issue_id,
		'entity_identifier', f.entity_identifier, 'entity_type', f.entity_type,
		'external_id', f.external_id, 'external_source', f.external_source, 'id', f.id,
		'is_archived', f.is_archived, 'is_deleted', f.is_deleted, 'is_uploaded', f.is_uploaded,
		'issue', f.issue_id, 'page', f.page_id, 'project', f.project_id, 'size', f.size,
		'storage_metadata', f.storage_metadata, 'updated_at', f.updated_at,
		'updated_by', f.updated_by_id, 'user', f.user_id, 'workspace', f.workspace_id
	) ORDER BY f.created_at DESC), '[]'::jsonb)
	FROM file_assets f
	WHERE f.workspace_id=$1 AND f.project_id=$2 AND f.issue_id=$3 AND f.deleted_at IS NULL`
	if uploadedOnly {
		query += ` AND f.entity_type='ISSUE_ATTACHMENT' AND f.is_uploaded=true`
	}
	if err := repository.database.QueryRow(ctx, query, workspaceID, projectID, issueID).Scan(&encoded); err != nil {
		return nil, fmt.Errorf("list issue FileAssets: %w", err)
	}
	var rows []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode issue FileAssets: %w", err)
	}
	return rows, nil
}

func (repository *PostgresRepository) IssueAttachmentsBySlug(ctx context.Context, slug string, projectID, issueID uuid.UUID, uploadedOnly bool) ([]map[string]any, error) {
	var encoded []byte
	query := `SELECT COALESCE(jsonb_agg(jsonb_build_object(
		'asset', f.asset,
		'attributes', f.attributes, 'comment', f.comment_id, 'created_at', f.created_at,
		'created_by', f.created_by_id, 'deleted_at', f.deleted_at, 'draft_issue', f.draft_issue_id,
		'entity_identifier', f.entity_identifier, 'entity_type', f.entity_type,
		'external_id', f.external_id, 'external_source', f.external_source, 'id', f.id,
		'is_archived', f.is_archived, 'is_deleted', f.is_deleted, 'is_uploaded', f.is_uploaded,
		'issue', f.issue_id, 'page', f.page_id, 'project', f.project_id, 'size', f.size,
		'storage_metadata', f.storage_metadata, 'updated_at', f.updated_at,
		'updated_by', f.updated_by_id, 'user', f.user_id, 'workspace', f.workspace_id
	) ORDER BY f.created_at DESC), '[]'::jsonb)
	FROM file_assets f
	JOIN workspaces w ON w.id=f.workspace_id
	WHERE w.slug=$1 AND f.project_id=$2 AND f.issue_id=$3 AND f.deleted_at IS NULL`
	if uploadedOnly {
		query += ` AND f.entity_type='ISSUE_ATTACHMENT' AND f.is_uploaded=true`
	}
	if err := repository.database.QueryRow(ctx, query, slug, projectID, issueID).Scan(&encoded); err != nil {
		return nil, fmt.Errorf("list issue FileAssets by workspace slug: %w", err)
	}
	var rows []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode issue FileAssets by workspace slug: %w", err)
	}
	return rows, nil
}

func (repository *PostgresRepository) IssueAttachment(ctx context.Context, workspaceID, projectID, issueID, assetID uuid.UUID) (FileAsset, map[string]any, error) {
	asset, err := repository.findAsset(ctx, assetID, workspaceID, true, true, false)
	if err != nil {
		return FileAsset{}, nil, err
	}
	if asset.ProjectID == nil || *asset.ProjectID != projectID || asset.IssueID == nil || *asset.IssueID != issueID {
		return FileAsset{}, nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	rows, err := repository.IssueAttachments(ctx, workspaceID, projectID, issueID, false)
	if err != nil {
		return FileAsset{}, nil, err
	}
	for _, row := range rows {
		if rowID, _ := row["id"].(string); rowID == assetID.String() {
			return asset, row, nil
		}
	}
	return FileAsset{}, nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
}

func (repository *PostgresRepository) ProjectIssueAttachment(ctx context.Context, workspaceID, projectID, assetID uuid.UUID) (FileAsset, map[string]any, error) {
	asset, err := repository.AssetByProjectAndID(ctx, workspaceID, projectID, assetID)
	if err != nil {
		return FileAsset{}, nil, err
	}
	var encoded []byte
	err = repository.database.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'asset', f.asset,
			'attributes', f.attributes, 'comment', f.comment_id, 'created_at', f.created_at,
			'created_by', f.created_by_id, 'deleted_at', f.deleted_at, 'draft_issue', f.draft_issue_id,
			'entity_identifier', f.entity_identifier, 'entity_type', f.entity_type,
			'external_id', f.external_id, 'external_source', f.external_source, 'id', f.id,
			'is_archived', f.is_archived, 'is_deleted', f.is_deleted, 'is_uploaded', f.is_uploaded,
			'issue', f.issue_id, 'page', f.page_id, 'project', f.project_id, 'size', f.size,
			'storage_metadata', f.storage_metadata, 'updated_at', f.updated_at,
			'updated_by', f.updated_by_id, 'user', f.user_id, 'workspace', f.workspace_id
		)
		FROM file_assets f
		WHERE f.workspace_id=$1 AND f.project_id=$2 AND f.id=$3 AND f.deleted_at IS NULL
	`, workspaceID, projectID, assetID).Scan(&encoded)
	if errors.Is(err, pgx.ErrNoRows) {
		return FileAsset{}, nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return FileAsset{}, nil, fmt.Errorf("serialize project issue FileAsset: %w", err)
	}
	row := make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&row); err != nil {
		return FileAsset{}, nil, fmt.Errorf("decode project issue FileAsset: %w", err)
	}
	return asset, row, nil
}

func (repository *PostgresRepository) UpdateIssueAttachmentUploaded(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_uploaded=true, updated_by_id=$5, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4 AND deleted_at IS NULL
	`, assetID, workspaceID, projectID, issueID, userID)
	if err != nil {
		return fmt.Errorf("confirm issue attachment upload: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) DeleteIssueAttachment(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_by_id=$5, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4 AND deleted_at IS NULL
	`, assetID, workspaceID, projectID, issueID, userID)
	if err != nil {
		return fmt.Errorf("delete issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) IssueAttachmentDeleteAllowed(ctx context.Context, userID uuid.UUID, slug string, projectID, assetID uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members wm
			JOIN workspaces w ON w.id=wm.workspace_id
			WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active=true AND wm.deleted_at IS NULL
			  AND EXISTS (SELECT 1 FROM file_assets f WHERE f.id=$4 AND f.created_by_id=$2 AND f.deleted_at IS NULL)
		) OR EXISTS(
			SELECT 1 FROM project_members pm
			JOIN workspaces w ON w.id=pm.workspace_id
			WHERE w.slug=$1 AND pm.project_id=$3 AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL
			  AND (pm.role=20 OR EXISTS (
				SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$2
				  AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
			  ))
		)
	`, slug, userID, projectID, assetID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check issue attachment delete permission: %w", err)
	}
	return allowed, nil
}

func (repository *PostgresRepository) AssetByID(ctx context.Context, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, uuid.Nil, false, true, false)
}

func (repository *PostgresRepository) UserAssetByID(ctx context.Context, userID, assetID uuid.UUID) (FileAsset, error) {
	asset, err := repository.findAsset(ctx, assetID, uuid.Nil, false, true, false)
	if err != nil {
		return FileAsset{}, err
	}
	if asset.UserID == nil || *asset.UserID != userID {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return asset, nil
}

func (repository *PostgresRepository) UserAssetsByKey(ctx context.Context, userID uuid.UUID, key string) ([]map[string]any, error) {
	return repository.assetsByKey(ctx, key, &userID)
}

func (repository *PostgresRepository) WorkspaceAssetsByKey(ctx context.Context, key string) ([]map[string]any, error) {
	return repository.assetsByKey(ctx, key, nil)
}

func (repository *PostgresRepository) assetsByKey(ctx context.Context, key string, createdBy *uuid.UUID) ([]map[string]any, error) {
	var encoded []byte
	err := repository.database.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(jsonb_build_object(
			'asset', asset, 'attributes', attributes, 'comment', comment_id,
			'created_at', created_at, 'created_by', created_by_id, 'deleted_at', deleted_at,
			'draft_issue', draft_issue_id, 'entity_identifier', entity_identifier,
			'entity_type', entity_type, 'external_id', external_id, 'external_source', external_source,
			'id', id, 'is_archived', is_archived, 'is_deleted', is_deleted, 'is_uploaded', is_uploaded,
			'issue', issue_id, 'page', page_id, 'project', project_id, 'size', size,
			'storage_metadata', COALESCE(storage_metadata,'{}'::jsonb), 'updated_at', updated_at,
			'updated_by', updated_by_id, 'user', user_id, 'workspace', workspace_id
		) ORDER BY created_at DESC), '[]'::jsonb)
		FROM file_assets
		WHERE asset=$1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR created_by_id=$2)
	`, key, createdBy).Scan(&encoded)
	if err != nil {
		return nil, fmt.Errorf("list FileAssets by key: %w", err)
	}
	var assets []map[string]any
	if err := json.Unmarshal(encoded, &assets); err != nil {
		return nil, fmt.Errorf("decode FileAssets by key: %w", err)
	}
	return assets, nil
}

// UpdateAssetDeletedByKey mirrors the legacy endpoints' exact asset-key
// lookup. A nil owner means the workspace route, which relies on the class
// workspace-membership permission; a non-nil owner matches UserAssetsEndpoint.
func (repository *PostgresRepository) UpdateAssetDeletedByKey(ctx context.Context, key string, createdBy *uuid.UUID, deleted bool) error {
	var matches int
	if err := repository.database.QueryRow(ctx, `
		SELECT COUNT(*) FROM file_assets
		WHERE asset=$1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR created_by_id=$2)
	`, key, createdBy).Scan(&matches); err != nil {
		return fmt.Errorf("count FileAssets by key for update: %w", err)
	}
	if matches > 1 {
		// Django's .get() raises MultipleObjectsReturned and the request makes
		// no mutation when duplicate storage keys exist.
		return fmt.Errorf("multiple FileAssets match storage key %q", key)
	}
	if matches == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `
		SELECT id FROM file_assets
		WHERE asset=$1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR created_by_id=$2)
	`, key, createdBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("find FileAsset by key for update: %w", err)
	}
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_deleted=$1 WHERE id=$2`, deleted, id)
	if err != nil {
		return fmt.Errorf("update FileAsset deletion by key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) UpdateLegacyUserAsset(ctx context.Context, userID, assetID uuid.UUID, attributes []byte) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	result, err := repository.database.Exec(ctx, `
UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb, updated_at=NOW()
WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL
`, attributes, assetID, userID)
	if err != nil {
		return fmt.Errorf("confirm legacy user FileAsset upload: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

// UpdateUserAsset applies the user-avatar/cover relationship update and the
// FileAsset upload confirmation in one database transaction. The source view
// marks the previous referenced asset as logically deleted before saving the
// new upload, while cache invalidation and the metadata task are handled by
// the domain service.
func (repository *PostgresRepository) UpdateUserAsset(ctx context.Context, userID, assetID uuid.UUID, attributes []byte, uploaded bool) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user FileAsset update: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var entityType string
	var previous *uuid.UUID
	err = tx.QueryRow(ctx, `
SELECT COALESCE(entity_type,''), CASE WHEN COALESCE(entity_type,'')='USER_AVATAR' THEN avatar_asset_id WHEN COALESCE(entity_type,'')='USER_COVER' THEN cover_image_asset_id ELSE NULL END
FROM file_assets f JOIN users u ON u.id=f.user_id
WHERE f.id=$1 AND f.user_id=$2 AND f.deleted_at IS NULL
`, assetID, userID).Scan(&entityType, &previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("load user FileAsset for update: %w", err)
	}
	if entityType == "USER_AVATAR" || entityType == "USER_COVER" {
		column := "cover_image_asset_id"
		legacyClear := "cover_image=NULL"
		if entityType == "USER_AVATAR" {
			column = "avatar_asset_id"
			legacyClear = "avatar=''"
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET `+column+`=$1, `+legacyClear+`, updated_at=NOW() WHERE id=$2`, assetID, userID); err != nil {
			return fmt.Errorf("set user asset relationship: %w", err)
		}
		if previous != nil {
			if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1`, *previous); err != nil {
				return fmt.Errorf("logically delete previous user FileAsset: %w", err)
			}
		}
	}
	if uploaded {
		if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb, updated_at=NOW() WHERE id=$2`, attributes, assetID); err != nil {
			return fmt.Errorf("confirm user FileAsset upload: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1`, assetID); err != nil {
			return fmt.Errorf("delete user FileAsset: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user FileAsset update: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) DeleteUserAsset(ctx context.Context, userID, assetID uuid.UUID) error {
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user FileAsset deletion: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var entityType string
	err = tx.QueryRow(ctx, `SELECT COALESCE(entity_type,'') FROM file_assets WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, assetID, userID).Scan(&entityType)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("load user FileAsset for deletion: %w", err)
	}
	if entityType == "USER_AVATAR" || entityType == "USER_COVER" {
		column := "cover_image_asset_id"
		if entityType == "USER_AVATAR" {
			column = "avatar_asset_id"
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET `+column+`=NULL, updated_at=NOW() WHERE id=$1`, userID); err != nil {
			return fmt.Errorf("clear user asset relationship: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1`, assetID); err != nil {
		return fmt.Errorf("logically delete user FileAsset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user FileAsset deletion: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) AssetByWorkspaceAndID(ctx context.Context, workspaceID, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, workspaceID, true, true, false)
}

func (repository *PostgresRepository) AssetByProjectAndID(ctx context.Context, workspaceID, projectID, id uuid.UUID) (FileAsset, error) {
	asset, err := repository.AssetByWorkspaceAndID(ctx, workspaceID, id)
	if err != nil {
		return FileAsset{}, err
	}
	if asset.ProjectID == nil || *asset.ProjectID != projectID {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return asset, nil
}

func (repository *PostgresRepository) UpdateWorkspaceAssetUploaded(ctx context.Context, workspaceID, assetID uuid.UUID, uploaded bool) error {
	result, err := repository.database.Exec(ctx, `
UPDATE file_assets SET is_uploaded=$1
WHERE id=$2 AND workspace_id=$3 AND is_deleted=false AND deleted_at IS NULL
`, uploaded, assetID, workspaceID)
	if err != nil {
		return fmt.Errorf("update workspace FileAsset upload state: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	return nil
}

func (repository *PostgresRepository) UpdateWorkspaceAsset(ctx context.Context, userID, workspaceID, assetID uuid.UUID, attributes []byte) (bool, error) {
	if !json.Valid(attributes) {
		return false, fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin workspace FileAsset confirmation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var entityType string
	var projectID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(entity_type,''), project_id FROM file_assets
		WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL
	`, assetID, workspaceID).Scan(&entityType, &projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return false, fmt.Errorf("load workspace FileAsset confirmation: %w", err)
	}
	var previous *uuid.UUID
	workspaceHookApplied := false
	switch entityType {
	case "WORKSPACE_LOGO":
		err := tx.QueryRow(ctx, `SELECT logo_asset_id FROM workspaces WHERE id=$1 AND deleted_at IS NULL`, workspaceID).Scan(&previous)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("load previous workspace logo asset: %w", err)
		}
		if err == nil {
			if _, err := tx.Exec(ctx, `UPDATE workspaces SET logo='', logo_asset_id=$1, updated_at=NOW(), updated_by_id=$3 WHERE id=$2 AND deleted_at IS NULL`, assetID, workspaceID, userID); err != nil {
				return false, fmt.Errorf("set workspace logo asset: %w", err)
			}
			workspaceHookApplied = true
		}
	case "PROJECT_COVER":
		if projectID != nil {
			if err := tx.QueryRow(ctx, `SELECT cover_image_asset_id FROM projects WHERE id=$1 AND deleted_at IS NULL`, *projectID).Scan(&previous); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return false, fmt.Errorf("load previous project cover asset: %w", err)
			}
			if _, err := tx.Exec(ctx, `UPDATE projects SET cover_image='', cover_image_asset_id=$1, updated_at=NOW(), updated_by_id=$3 WHERE id=$2 AND deleted_at IS NULL`, assetID, *projectID, userID); err != nil {
				return false, fmt.Errorf("set project cover asset: %w", err)
			}
		}
	}
	// WorkspaceFileAssetEndpoint.entity_asset_save deletes the currently
	// referenced asset before reassigning the uploaded asset. When the upload
	// is already the referenced row, Django leaves that same row logically
	// deleted after confirmation; preserve that source lifecycle quirk.
	if previous != nil {
		if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, *previous); err != nil {
			return false, fmt.Errorf("logically delete previous workspace asset: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb WHERE id=$2`, attributes, assetID); err != nil {
		return false, fmt.Errorf("confirm workspace FileAsset upload: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit workspace FileAsset confirmation: %w", err)
	}
	return workspaceHookApplied, nil
}

func (repository *PostgresRepository) DeleteWorkspaceAsset(ctx context.Context, userID, workspaceID, assetID uuid.UUID) error {
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workspace FileAsset deletion: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var entityType string
	var projectID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(entity_type,''), project_id FROM file_assets
		WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL
	`, assetID, workspaceID).Scan(&entityType, &projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("load workspace FileAsset deletion: %w", err)
	}
	switch entityType {
	case "WORKSPACE_LOGO":
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1 AND deleted_at IS NULL)`, workspaceID).Scan(&exists); err != nil {
			return fmt.Errorf("check active workspace for logo deletion: %w", err)
		}
		if !exists {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		if _, err := tx.Exec(ctx, `UPDATE workspaces SET logo_asset_id=NULL, updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, workspaceID); err != nil {
			return fmt.Errorf("clear workspace logo asset: %w", err)
		}
	case "PROJECT_COVER":
		if projectID != nil {
			if _, err := tx.Exec(ctx, `UPDATE projects SET cover_image_asset_id=NULL, updated_by_id=$2, updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, *projectID, userID); err != nil {
				return fmt.Errorf("clear project cover asset: %w", err)
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1`, assetID); err != nil {
		return fmt.Errorf("logically delete workspace FileAsset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit workspace FileAsset deletion: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) UpdateProjectAsset(ctx context.Context, projectID, assetID uuid.UUID, attributes []byte) error {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	command, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_uploaded=true, attributes=$3::jsonb
		WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL
	`, assetID, projectID, attributes)
	if err != nil {
		return fmt.Errorf("confirm project FileAsset upload: %w", err)
	}
	if command.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) DeleteProjectAsset(ctx context.Context, projectID, assetID uuid.UUID) error {
	command, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL
	`, assetID, projectID)
	if err != nil {
		return fmt.Errorf("delete project FileAsset: %w", err)
	}
	if command.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) RestoreWorkspaceAsset(ctx context.Context, workspaceID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
UPDATE file_assets SET is_deleted=false, deleted_at=NULL, updated_at=NOW()
WHERE id=$1 AND workspace_id=$2
`, assetID, workspaceID)
	if err != nil {
		return fmt.Errorf("restore workspace FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) AssetByWorkspaceAndIDIncludingDeleted(ctx context.Context, workspaceID, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, workspaceID, true, false, false)
}

// WorkspaceAssetExists matches AssetCheckEndpoint, which uses all_objects and
// filters only deleted_at. Its observable result intentionally ignores the
// legacy is_deleted and upload flags.
func (repository *PostgresRepository) WorkspaceAssetExists(ctx context.Context, workspaceID, id uuid.UUID) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL)
	`, id, workspaceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace FileAsset %s: %w", id, err)
	}
	return exists, nil
}

func (repository *PostgresRepository) AssetForDuplicate(ctx context.Context, workspaceID, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, workspaceID, true, true, false)
}

func (repository *PostgresRepository) ProjectBelongsToWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL)`, projectID, workspaceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check project %s workspace: %w", projectID, err)
	}
	return exists, nil
}

func (repository *PostgresRepository) ProjectPermissionAllowed(ctx context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS(
 SELECT 1 FROM project_members pm
 JOIN workspaces w ON w.id=pm.workspace_id
 WHERE w.slug=$1 AND pm.project_id=$2 AND pm.member_id=$3 AND pm.is_active=true AND pm.deleted_at IS NULL
 AND (pm.role IN (5,15,20) OR EXISTS (
   SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$3 AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
 ))
)
`, slug, projectID, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check asset project permission: %w", err)
	}
	return allowed, nil
}

func (repository *PostgresRepository) ProjectMemberActive(ctx context.Context, userID, workspaceID, projectID uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS(
 SELECT 1 FROM project_members pm
 WHERE pm.workspace_id=$1 AND pm.project_id=$2 AND pm.member_id=$3
 AND pm.is_active=true AND pm.deleted_at IS NULL
)
`, workspaceID, projectID, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check asset project membership: %w", err)
	}
	return allowed, nil
}

func (repository *PostgresRepository) ProjectMemberActiveByProject(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS(
 SELECT 1 FROM project_members pm
 WHERE pm.project_id=$1 AND pm.member_id=$2
 AND pm.is_active=true AND pm.deleted_at IS NULL
)
`, projectID, userID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check asset project membership: %w", err)
	}
	return allowed, nil
}

func (repository *PostgresRepository) CreateDuplicate(ctx context.Context, source FileAsset, id, workspaceID, creatorID uuid.UUID, destinationKey, projectID, entityType, entityID string) error {
	attributesSource, err := fileAssetAttributesObject(source.Attributes)
	if err != nil {
		return err
	}
	var project, user, relatedWorkspace, issue, comment, page any
	if projectID != "" {
		parsed, err := uuid.Parse(projectID)
		if err != nil {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"project_id": []string{"Must be a valid UUID."}}}
		}
		project = parsed
	}
	if entityID != "" {
		switch entityType {
		case "WORKSPACE_LOGO":
			parsed, err := uuid.Parse(entityID)
			if err != nil {
				return err
			}
			relatedWorkspace = parsed
		case "PROJECT_COVER":
			parsed, err := uuid.Parse(entityID)
			if err != nil || entityID == "" {
				return projectCoverEntityMismatchError()
			}
			if requested, ok := project.(uuid.UUID); ok && requested != parsed {
				return projectCoverEntityMismatchError()
			}
			if project == nil {
				project = parsed
			}
		case "USER_AVATAR", "USER_COVER":
			parsed, err := uuid.Parse(entityID)
			if err != nil {
				return err
			}
			user = parsed
		case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION":
			parsed, err := uuid.Parse(entityID)
			if err != nil {
				return err
			}
			issue = parsed
		case "COMMENT_DESCRIPTION":
			parsed, err := uuid.Parse(entityID)
			if err != nil {
				return err
			}
			comment = parsed
		case "PAGE_DESCRIPTION":
			parsed, err := uuid.Parse(entityID)
			if err != nil {
				return err
			}
			page = parsed
		}
	}
	targetWorkspace := any(workspaceID)
	if relatedWorkspace != nil {
		targetWorkspace = relatedWorkspace
	}
	attributes, err := json.Marshal(map[string]any{"name": attributesSource["name"], "type": attributesSource["type"], "size": attributesSource["size"]})
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(source.StorageMetadata)
	if err != nil {
		return err
	}
	_, err = repository.database.Exec(ctx, `
INSERT INTO file_assets
 (id,created_at,updated_at,attributes,asset,workspace_id,project_id,user_id,issue_id,comment_id,page_id,entity_type,entity_identifier,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
VALUES ($1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,NULL,$11,false,false,false,$12::jsonb,$13)
`, id, attributes, destinationKey, targetWorkspace, project, user, issue, comment, page, entityType, source.Size, metadata, creatorID)
	if err != nil {
		return fmt.Errorf("insert duplicated FileAsset: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) MarkDuplicateUploaded(ctx context.Context, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_uploaded=true WHERE id=$1`, assetID)
	if err != nil {
		return fmt.Errorf("mark duplicated FileAsset uploaded: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("duplicated FileAsset %s disappeared before upload confirmation", assetID)
	}
	return nil
}

func (repository *PostgresRepository) findAsset(ctx context.Context, id, workspaceFilter uuid.UUID, scopeWorkspace, filterDeletedAt, filterIsDeleted bool) (FileAsset, error) {
	var asset FileAsset
	var attributes []byte
	var workspaceID, projectID, issueID, userID *uuid.UUID
	var metadata []byte
	query := `SELECT id::text, asset, COALESCE(entity_type,''), attributes, workspace_id, project_id, issue_id, user_id, is_uploaded, is_deleted, size, COALESCE(storage_metadata,'{}'::jsonb) FROM file_assets WHERE id=$1`
	args := []any{id}
	if filterDeletedAt {
		query += ` AND deleted_at IS NULL`
	}
	if scopeWorkspace {
		query += ` AND workspace_id=$` + strconv.Itoa(len(args)+1)
		args = append(args, workspaceFilter)
	}
	if filterIsDeleted {
		query += ` AND is_deleted=false`
	}
	err := repository.database.QueryRow(ctx, query, args...).Scan(&asset.ID, &asset.Asset, &asset.EntityType, &attributes, &workspaceID, &projectID, &issueID, &userID, &asset.IsUploaded, &asset.IsDeleted, &asset.Size, &metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return FileAsset{}, fmt.Errorf("load FileAsset %s: %w", id, err)
	}
	asset.Attributes, err = decodeFileAssetAttributes(attributes)
	if err != nil {
		return FileAsset{}, fmt.Errorf("decode FileAsset %s attributes: %w", id, err)
	}
	if err := json.Unmarshal(metadata, &asset.StorageMetadata); err != nil {
		return FileAsset{}, fmt.Errorf("decode FileAsset %s storage metadata: %w", id, err)
	}
	asset.WorkspaceID, asset.ProjectID, asset.IssueID, asset.UserID = workspaceID, projectID, issueID, userID
	return asset, nil
}

func decodeFileAssetAttributes(encoded []byte) (any, error) {
	var attributes any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}
