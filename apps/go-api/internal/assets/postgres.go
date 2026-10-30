package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
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

func assetParentVisibilitySQL(asset, actor string) string {
	return parentVisibilitySQL(asset+".workspace_id", asset+".project_id", asset+".issue_id", asset+".comment_id", asset+".page_id", actor)
}

func parentVisibilitySQL(workspace, project, issue, comment, page, actor string) string {
	return `(` + issue + ` IS NULL OR EXISTS(SELECT 1 FROM issues asset_issue WHERE asset_issue.id=` + issue + `
	 AND (` + project + ` IS NULL OR asset_issue.project_id=` + project + `)
	 AND ` + djangoapi.IssueVisibleSQL("asset_issue", actor, workspace) + `))
	AND (` + comment + ` IS NULL OR EXISTS(SELECT 1 FROM issue_comments asset_comment
	 JOIN issues asset_comment_issue ON asset_comment_issue.id=asset_comment.issue_id
	 WHERE asset_comment.id=` + comment + ` AND asset_comment.workspace_id=` + workspace + `
	 AND (` + project + ` IS NULL OR asset_comment.project_id=` + project + `)
	 AND (` + issue + ` IS NULL OR asset_comment.issue_id=` + issue + `)
	 AND asset_comment.deleted_at IS NULL
	 AND ` + djangoapi.IssueVisibleSQL("asset_comment_issue", actor, workspace) + `))
	AND (` + page + ` IS NULL OR EXISTS(SELECT 1 FROM project_pages asset_page_link
	 JOIN pages asset_page ON asset_page.id=asset_page_link.page_id
	 WHERE asset_page.id=` + page + ` AND (` + project + ` IS NULL OR asset_page_link.project_id=` + project + `)
	 AND ` + djangoapi.PageVisibleSQL("asset_page", actor, workspace, "asset_page_link.project_id") + `))`
}

func targetVisibilitySQL(entityType, actor, workspace, project, issue, comment, page string) string {
	switch entityType {
	case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION":
		return `EXISTS(SELECT 1 FROM issues target_issue WHERE target_issue.id=` + issue + ` AND ` + djangoapi.IssueVisibleSQL("target_issue", actor, workspace) + ` AND (` + project + ` IS NULL OR target_issue.project_id=` + project + `))`
	case "COMMENT_DESCRIPTION":
		return `EXISTS(SELECT 1 FROM issue_comments target_comment JOIN issues target_issue ON target_issue.id=target_comment.issue_id WHERE target_comment.id=` + comment + ` AND target_comment.workspace_id=` + workspace + ` AND target_comment.deleted_at IS NULL AND (` + issue + ` IS NULL OR target_comment.issue_id=` + issue + `) AND ` + djangoapi.IssueVisibleSQL("target_issue", actor, workspace) + ` AND (` + project + ` IS NULL OR target_comment.project_id=` + project + `))`
	case "PAGE_DESCRIPTION":
		return `EXISTS(SELECT 1 FROM project_pages target_page_link JOIN pages target_page ON target_page.id=target_page_link.page_id WHERE target_page.id=` + page + ` AND target_page_link.deleted_at IS NULL AND (` + project + ` IS NULL OR target_page_link.project_id=` + project + `) AND ` + djangoapi.PageVisibleSQL("target_page", actor, workspace, "target_page_link.project_id") + `)`
	default:
		return "TRUE"
	}
}

func (repository *PostgresRepository) AssetParentVisible(ctx context.Context, actorID, workspaceID, assetID uuid.UUID) (bool, error) {
	var visible bool
	query := `SELECT EXISTS(SELECT 1 FROM file_assets asset WHERE asset.id=$1 AND asset.workspace_id=$2 AND ` + assetParentVisibilitySQL("asset", "$3::uuid") + `)`
	if err := repository.database.QueryRow(ctx, query, assetID, workspaceID, actorID).Scan(&visible); err != nil {
		return false, fmt.Errorf("check FileAsset parent visibility: %w", err)
	}
	return visible, nil
}

func insertWorkItemEffect(ctx context.Context, tx pgx.Tx, kind string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s FileAsset effect: %w", kind, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,$2,$3::jsonb)`, uuid.New(), kind, encoded); err != nil {
		return fmt.Errorf("enqueue %s FileAsset effect: %w", kind, err)
	}
	return nil
}

// ObjectKeyInUse prevents a delayed deletion from removing an object that has
// since been restored or referenced by another live FileAsset.
func (repository *PostgresRepository) ObjectKeyInUse(ctx context.Context, _ uuid.UUID, key string) (bool, error) {
	var inUse bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE asset=$1 AND deleted_at IS NULL)`, key).Scan(&inUse)
	if err != nil {
		return false, fmt.Errorf("check live FileAsset references for object key: %w", err)
	}
	return inUse, nil
}

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

func (repository *PostgresRepository) CreateWorkspaceEntityUpload(ctx context.Context, id, userID, workspaceID, parentProjectID uuid.UUID, key string, attributes []byte, size int64, entityType, entityID string, projectAsset bool) (resultErr error) {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	var projectID, userTarget, issueID, commentID, pageID, draftIssueID any
	database := repository.database
	var draftTransaction pgx.Tx
	identifier := ""
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
			draftTransaction, err = repository.database.Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { resultErr = dbutil.RollbackTransaction(ctx, draftTransaction.Rollback, resultErr) }()
			draftProject, err := lockOwnedAssetDraft(ctx, draftTransaction, workspaceID, userID, parsed)
			if err != nil {
				return err
			}
			if projectAsset && (draftProject == nil || *draftProject != parentProjectID) {
				return notFoundAssetError()
			}
			projectID, draftIssueID, database = draftProject, parsed, draftTransaction
			identifier = parsed.String()
		case "COMMENT_DESCRIPTION":
			commentID = parsed
		case "PAGE_DESCRIPTION":
			pageID = parsed
		}
	}
	result, err := database.Exec(ctx, `
INSERT INTO file_assets
 (id,created_at,updated_at,attributes,asset,user_id,workspace_id,project_id,draft_issue_id,issue_id,comment_id,page_id,entity_type,entity_identifier,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
SELECT $1,NOW(),NOW(),$2::jsonb,$3,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,''),$14,false,false,false,'{}'::jsonb,$4
WHERE `+targetVisibilitySQL(entityType, "$4::uuid", "$6::uuid", "$7::uuid", "$9::uuid", "$10::uuid", "$11::uuid")+`
`, id, attributes, key, userID, userTarget, workspaceID, projectID, draftIssueID, issueID, commentID, pageID, entityType, identifier, float64(size))
	if err != nil {
		return fmt.Errorf("insert workspace FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return notFoundAssetError()
	}
	if draftTransaction != nil {
		return draftTransaction.Commit(ctx)
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
	result, err := repository.database.Exec(ctx, `
		INSERT INTO file_assets
			 (id,created_at,updated_at,attributes,asset,workspace_id,project_id,issue_id,entity_type,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
		SELECT $1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,'ISSUE_ATTACHMENT',NULLIF($7,''),NULLIF($8,''),$9,false,false,false,'{}'::jsonb,$10
 WHERE EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$6 AND attachment_item.project_id=$5 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$10::uuid", "$4::uuid")+`)
	`, id, attributes, key, workspaceID, projectID, issueID, externalID, externalSource, float64(size), userID)
	if err != nil {
		return fmt.Errorf("insert issue attachment FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return notFoundAssetError()
	}
	return nil
}

func (repository *PostgresRepository) IssueAttachmentExternal(ctx context.Context, userID, workspaceID, projectID, issueID uuid.UUID, externalID, externalSource string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := repository.database.QueryRow(ctx, `SELECT file.id FROM file_assets file WHERE file.workspace_id=$1 AND file.project_id=$2 AND file.issue_id=$3 AND file.entity_type='ISSUE_ATTACHMENT' AND file.external_id=$4 AND file.external_source=$5 AND file.deleted_at IS NULL
		AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$3 AND attachment_item.project_id=$2 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$6::uuid", "$1::uuid")+`)
		ORDER BY file.created_at ASC LIMIT 1`, workspaceID, projectID, issueID, externalID, externalSource, userID).Scan(&id)
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
		SELECT CASE WHEN draft_issue_id IS NOT NULL THEN 'DRAFT_ISSUE_DESCRIPTION' ELSE entity_type END FROM file_assets
		WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL
		AND (project_id=$4 OR project_id IS NULL)
		ORDER BY (entity_type='DRAFT_ISSUE_DESCRIPTION' OR draft_issue_id IS NOT NULL) DESC, created_at DESC LIMIT 1`, assetIDs, workspaceID, userID, projectID).Scan(&entityType)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	if err != nil {
		return fmt.Errorf("find project bulk assets: %w", err)
	}
	if entityType == "DRAFT_ISSUE_DESCRIPTION" {
		return repository.BulkAssociateDraftAssets(ctx, workspaceID, userID, entityID, assetIDs)
	}
	if entityType == "PROJECT_COVER" {
		var ids []uuid.UUID
		if scanErr := repository.database.QueryRow(ctx, `SELECT COALESCE(array_agg(id ORDER BY created_at DESC), '{}'::uuid[]) FROM file_assets WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND entity_type='PROJECT_COVER' AND (project_id=$4 OR project_id IS NULL)`, assetIDs, workspaceID, userID, projectID).Scan(&ids); scanErr != nil {
			return fmt.Errorf("list project cover assets: %w", scanErr)
		}
		if _, execErr := repository.database.Exec(ctx, `UPDATE file_assets SET project_id=$1 WHERE id=ANY($2::uuid[]) AND workspace_id=$3 AND created_by_id=$4 AND deleted_at IS NULL AND draft_issue_id IS NULL AND entity_type='PROJECT_COVER' AND (project_id=$1 OR project_id IS NULL)`, projectID, assetIDs, workspaceID, userID); execErr != nil {
			return fmt.Errorf("associate project cover assets: %w", execErr)
		}
		var projectExists bool
		if scanErr := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND deleted_at IS NULL)`, projectID).Scan(&projectExists); scanErr != nil {
			return fmt.Errorf("load active project for cover association: %w", scanErr)
		}
		if !projectExists {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		for _, id := range ids {
			if _, execErr := repository.database.Exec(ctx, `UPDATE projects SET cover_image_asset_id=$1, updated_by_id=$3, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, id, projectID, userID); execErr != nil {
				return fmt.Errorf("set project cover asset: %w", execErr)
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
	default:
		return nil
	}
	targetIssue, targetComment, targetPage := "NULL::uuid", "NULL::uuid", "NULL::uuid"
	switch entityType {
	case "ISSUE_DESCRIPTION":
		targetIssue = "$1::uuid"
	case "COMMENT_DESCRIPTION":
		targetComment = "$1::uuid"
	case "PAGE_DESCRIPTION":
		targetPage = "$1::uuid"
	}
	query := `UPDATE file_assets SET ` + column + `=$1`
	args := []any{value, assetIDs, workspaceID, userID, projectID, entityType}
	query += ` WHERE id=ANY($2::uuid[]) AND workspace_id=$3 AND created_by_id=$4 AND deleted_at IS NULL AND (project_id=$5 OR project_id IS NULL) AND draft_issue_id IS NULL AND entity_type=$6`
	query += ` AND ` + targetVisibilitySQL(entityType, "$4::uuid", "$3::uuid", "$5::uuid", targetIssue, targetComment, targetPage)
	if entityType == "ISSUE_DESCRIPTION" {
		query = `UPDATE file_assets SET issue_id=$1, project_id=$5` + query[len(`UPDATE file_assets SET issue_id=$1`):]
	}
	result, err := repository.database.Exec(ctx, query, args...)
	if err != nil {
		// The source deliberately ignores a deleted target's FK violation for
		// issue, comment, and draft-issue association requests.
		var pgErr *pgconn.PgError
		if (entityType == "ISSUE_DESCRIPTION" || entityType == "COMMENT_DESCRIPTION") && errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "23") {
			return nil
		}
		return fmt.Errorf("associate project assets: %w", err)
	}
	if result.RowsAffected() == 0 {
		return notFoundAssetError()
	}
	return nil
}

func (repository *PostgresRepository) DeleteLegacyIssueAttachment(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID, activity map[string]any) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin legacy issue attachment deletion: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var objectKey string
	err = tx.QueryRow(ctx, `SELECT file.asset FROM file_assets file
		WHERE file.id=$1 AND file.workspace_id=$2 AND file.project_id=$3 AND file.issue_id=$4 AND file.deleted_at IS NULL
		AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$4 AND attachment_item.project_id=$3
		 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "$2::uuid")+`) FOR UPDATE`, assetID, workspaceID, projectID, issueID, userID).Scan(&objectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue attachment not found."}}
	}
	if err != nil {
		return fmt.Errorf("load legacy issue attachment for deletion: %w", err)
	}
	result, err := tx.Exec(ctx, `UPDATE file_assets SET asset='', is_deleted=true, deleted_at=NOW(), updated_by_id=$5, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4 AND deleted_at IS NULL
		AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$4 AND attachment_item.project_id=$3
		 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "$2::uuid")+`)`, assetID, workspaceID, projectID, issueID, userID)
	if err != nil {
		return fmt.Errorf("delete legacy issue attachment row: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue attachment not found."}}
	}
	if activity != nil {
		if err := insertWorkItemEffect(ctx, tx, "activity", activity); err != nil {
			return err
		}
	}
	if err := insertWorkItemEffect(ctx, tx, "soft_delete", map[string]string{"app": "db", "model": "fileasset", "id": assetID.String()}); err != nil {
		return err
	}
	if objectKey != "" {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_delete", map[string]string{"asset_id": assetID.String(), "object_key": objectKey}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit legacy issue attachment deletion: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) IssueAccess(ctx context.Context, userID, workspaceID, projectID, issueID uuid.UUID) (bool, bool, error) {
	var found, allowed bool
	err := repository.database.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND i.workspace_id=$2 AND i.project_id=$3 AND i.deleted_at IS NULL),
 EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND i.project_id=$3 AND `+djangoapi.IssueVisibleSQL("i", "$4::uuid", "$2::uuid")+`)`, issueID, workspaceID, projectID, userID).Scan(&found, &allowed)
	if err != nil {
		return false, false, fmt.Errorf("check issue attachment access: %w", err)
	}
	return found, allowed, nil
}

func (repository *PostgresRepository) UpdateAPIFileAssetUploaded(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, created_by_id=CASE WHEN is_uploaded THEN created_by_id ELSE $4 END, updated_by_id=$4, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
	AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=file_assets.issue_id AND attachment_item.project_id=file_assets.project_id AND `+djangoapi.IssueVisibleSQL("attachment_item", "$4::uuid", "$2::uuid")+`)`, assetID, workspaceID, projectID, userID)
	if err != nil {
		return fmt.Errorf("confirm API issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) DeleteAPIFileAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_by_id=$4, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
	AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=file_assets.issue_id AND attachment_item.project_id=file_assets.project_id AND `+djangoapi.IssueVisibleSQL("attachment_item", "$4::uuid", "$2::uuid")+`)`, assetID, workspaceID, projectID, userID)
	if err != nil {
		return fmt.Errorf("delete API issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) CreateLegacyFileAsset(ctx context.Context, id, userID uuid.UUID, fields map[string]any, key string) (map[string]any, error) {
	if draft, ok := fields["draft_issue"].(uuid.UUID); ok {
		return repository.createLegacyDraftAsset(ctx, id, userID, draft, fields, key)
	}
	return repository.insertLegacyFileAsset(ctx, id, userID, fields, key)
}

func (repository *PostgresRepository) insertLegacyFileAsset(ctx context.Context, id, userID uuid.UUID, fields map[string]any, key string) (map[string]any, error) {
	attributes := legacyJSONField(fields, "attributes")
	metadata := legacyJSONField(fields, "storage_metadata")
	result, err := repository.database.Exec(ctx, `
		INSERT INTO file_assets
			 (id,created_at,updated_at,attributes,asset,user_id,workspace_id,draft_issue_id,project_id,issue_id,comment_id,page_id,entity_type,entity_identifier,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id,deleted_at)
		SELECT $1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::jsonb,$20,$21
		WHERE `+parentVisibilitySQL("$5::uuid", "$7::uuid", "$8::uuid", "$9::uuid", "$10::uuid", "$20::uuid")+`
	`, id, attributes, key, legacyUUIDField(fields, "user"), legacyUUIDField(fields, "workspace"), legacyUUIDField(fields, "draft_issue"), legacyUUIDField(fields, "project"), legacyUUIDField(fields, "issue"), legacyUUIDField(fields, "comment"), legacyUUIDField(fields, "page"), legacyNullableStringField(fields, "entity_type"), legacyNullableStringField(fields, "entity_identifier"), legacyNullableStringField(fields, "external_id"), legacyNullableStringField(fields, "external_source"), legacyFloatField(fields, "size"), legacyBoolField(fields, "is_uploaded"), legacyBoolField(fields, "is_deleted"), legacyBoolField(fields, "is_archived"), metadata, userID, legacyTimeField(fields, "deleted_at"))
	if err != nil {
		return nil, fmt.Errorf("insert legacy FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return nil, notFoundAssetError()
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
		"user": "users", "workspace": "workspaces", "draft_issue": "draft_issues", "project": "projects",
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

func (repository *PostgresRepository) CreateLegacyIssueAttachment(ctx context.Context, id, userID, workspaceID, projectID, issueID uuid.UUID, fields map[string]any, key, assetURL string, activity map[string]any) (resultErr error) {
	if legacyUUIDField(fields, "draft_issue") != nil {
		return notFoundAssetError()
	}
	attributes, metadata := legacyJSONField(fields, "attributes"), legacyJSONField(fields, "storage_metadata")
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin legacy issue attachment creation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	result, err := tx.Exec(ctx, `
		INSERT INTO file_assets (id,created_at,updated_at,attributes,asset,user_id,workspace_id,draft_issue_id,project_id,issue_id,comment_id,page_id,entity_type,entity_identifier,external_id,external_source,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id,deleted_at)
		SELECT $1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,'ISSUE_ATTACHMENT',$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19,$20
 WHERE `+parentVisibilitySQL("$5::uuid", "$7::uuid", "$8::uuid", "$9::uuid", "$10::uuid", "$19::uuid")+`
	`, id, attributes, key, legacyUUIDField(fields, "user"), workspaceID, legacyUUIDField(fields, "draft_issue"), projectID, issueID, legacyUUIDField(fields, "comment"), legacyUUIDField(fields, "page"), legacyNullableStringField(fields, "entity_identifier"), legacyNullableStringField(fields, "external_id"), legacyNullableStringField(fields, "external_source"), legacyFloatField(fields, "size"), legacyBoolField(fields, "is_uploaded"), legacyBoolField(fields, "is_deleted"), legacyBoolField(fields, "is_archived"), metadata, userID, legacyTimeField(fields, "deleted_at"))
	if err != nil {
		return fmt.Errorf("insert legacy issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return notFoundAssetError()
	}
	if activity != nil {
		var encoded []byte
		if err := tx.QueryRow(ctx, issueAttachmentsQuery("f.id=$1 AND f.workspace_id=$4", false), id, projectID, issueID, workspaceID).Scan(&encoded); err != nil {
			return fmt.Errorf("read created legacy issue attachment for activity: %w", err)
		}
		var rows []map[string]any
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		if err := decoder.Decode(&rows); err != nil {
			return fmt.Errorf("decode created legacy issue attachment for activity: %w", err)
		}
		if len(rows) != 1 {
			return fmt.Errorf("created legacy issue attachment %s missing from activity snapshot", id)
		}
		rows[0]["asset_url"] = assetURL
		currentInstance, err := json.Marshal(pythonJSONNumbers(rows[0]))
		if err != nil {
			return fmt.Errorf("encode created legacy issue attachment activity: %w", err)
		}
		activity["current_instance"] = string(currentInstance)
		if err := insertWorkItemEffect(ctx, tx, "activity", activity); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit legacy issue attachment creation: %w", err)
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
	asset, err := repository.findAsset(ctx, assetID, workspaceID, true, !includeDeleted)
	if err != nil {
		return FileAsset{}, err
	}
	if privateDraftAsset(asset) || asset.ProjectID == nil || *asset.ProjectID != projectID {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if descriptionsOnly && asset.EntityType != "ISSUE_DESCRIPTION" && asset.EntityType != "COMMENT_DESCRIPTION" {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return asset, nil
}

func (repository *PostgresRepository) CreatePublicUpload(ctx context.Context, id, userID, workspaceID, projectID uuid.UUID, key string, attributes []byte, size int64, entityType, entityIdentifier string) (resultErr error) {
	if !json.Valid(attributes) {
		return fmt.Errorf("public asset attributes must be valid JSON")
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public FileAsset insert: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
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
			if scanErr := tx.QueryRow(ctx, `SELECT id FROM issue_comments WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND created_by_id=$4 AND deleted_at IS NULL FOR KEY SHARE`, parsedID, workspaceID, projectID, userID).Scan(&found); errors.Is(scanErr, pgx.ErrNoRows) {
				return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
			} else if scanErr != nil {
				return fmt.Errorf("validate public upload comment: %w", scanErr)
			}
		case "ISSUE_DESCRIPTION":
			relationColumn = "issue_id"
			var found uuid.UUID
			if scanErr := tx.QueryRow(ctx, `SELECT id FROM issues WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL FOR KEY SHARE`, parsedID, workspaceID, projectID).Scan(&found); errors.Is(scanErr, pgx.ErrNoRows) {
				return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
			} else if scanErr != nil {
				return fmt.Errorf("validate public upload issue: %w", scanErr)
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
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND deleted_at IS NULL
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
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND deleted_at IS NULL
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
		WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
	`, pgx.StrictNamedArgs{"id": assetID, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("restore public FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) BulkAssociatePublicAssetComments(ctx context.Context, userID, workspaceID, projectID, commentID uuid.UUID, assetIDs []uuid.UUID) (resultErr error) {
	ids := uniquePublicAssetIDs(assetIDs)
	if len(ids) == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public FileAsset association: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var authorizedComment uuid.UUID
	if scanErr := tx.QueryRow(ctx, `SELECT id FROM issue_comments WHERE id=@comment AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND deleted_at IS NULL FOR KEY SHARE`, pgx.StrictNamedArgs{"comment": commentID, "workspace": workspaceID, "project": projectID, "actor": userID}).Scan(&authorizedComment); errors.Is(scanErr, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	} else if scanErr != nil {
		return fmt.Errorf("validate public asset comment association: %w", scanErr)
	}
	rows, err := tx.Query(ctx, `SELECT id FROM file_assets
		WHERE id=ANY(@ids) AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor
			AND draft_issue_id IS NULL AND entity_type='COMMENT_DESCRIPTION' AND deleted_at IS NULL
		FOR UPDATE`, pgx.StrictNamedArgs{"ids": ids, "workspace": workspaceID, "project": projectID, "actor": userID})
	if err != nil {
		return fmt.Errorf("validate public FileAsset association: %w", err)
	}
	matched := make(map[uuid.UUID]struct{}, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			rows.Close()
			return fmt.Errorf("scan public FileAsset association: %w", scanErr)
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
			AND draft_issue_id IS NULL AND entity_type='COMMENT_DESCRIPTION' AND deleted_at IS NULL`, pgx.StrictNamedArgs{"comment": commentID, "ids": ids, "workspace": workspaceID, "project": projectID, "actor": userID})
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
		WHERE id=ANY($1) AND workspace_id=$2 AND project_id=$3 AND created_by_id=$4 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
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

const issueAttachmentProjection = `jsonb_build_object(
		'asset', f.asset,
		'attributes', f.attributes, 'comment', f.comment_id, 'created_at', f.created_at,
		'created_by', f.created_by_id, 'deleted_at', f.deleted_at, 'draft_issue', f.draft_issue_id,
		'entity_identifier', f.entity_identifier, 'entity_type', f.entity_type,
		'external_id', f.external_id, 'external_source', f.external_source, 'id', f.id,
		'is_archived', f.is_archived, 'is_deleted', f.is_deleted, 'is_uploaded', f.is_uploaded,
		'issue', f.issue_id, 'page', f.page_id, 'project', f.project_id, 'size', f.size,
		'storage_metadata', f.storage_metadata, 'updated_at', f.updated_at,
		'updated_by', f.updated_by_id, 'user', f.user_id, 'workspace', f.workspace_id
	)`

func issueAttachmentsQuery(workspacePredicate string, uploadedOnly bool) string {
	query := `SELECT COALESCE(jsonb_agg(` + issueAttachmentProjection + ` ORDER BY f.created_at DESC), '[]'::jsonb)
	FROM file_assets f
	WHERE ` + workspacePredicate + ` AND f.project_id=$2 AND f.issue_id=$3 AND f.deleted_at IS NULL AND f.draft_issue_id IS NULL AND COALESCE(f.entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')`
	if uploadedOnly {
		query += ` AND f.entity_type='ISSUE_ATTACHMENT' AND f.is_uploaded=true`
	}
	return query
}

func (repository *PostgresRepository) readIssueAttachmentRows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	var encoded []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&encoded); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
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

func (repository *PostgresRepository) IssueAttachments(ctx context.Context, workspaceID, projectID, issueID uuid.UUID, uploadedOnly bool) ([]map[string]any, error) {
	return repository.readIssueAttachmentRows(ctx, issueAttachmentsQuery("f.workspace_id=$1", uploadedOnly), workspaceID, projectID, issueID)
}

// VisibleIssueAttachments checks current item visibility in the statement returning
// metadata, so an earlier project permission check cannot authorize a stale read.
func (repository *PostgresRepository) VisibleIssueAttachments(ctx context.Context, userID, workspaceID, projectID, issueID uuid.UUID, uploadedOnly bool) ([]map[string]any, error) {
	query := `SELECT attachment_result.data FROM (` + issueAttachmentsQuery("f.workspace_id=$1", uploadedOnly) + `) AS attachment_result(data)
 WHERE EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$3 AND attachment_item.project_id=$2 AND ` + djangoapi.IssueVisibleSQL("attachment_item", "$4::uuid", "$1::uuid") + `)`
	return repository.readIssueAttachmentRows(ctx, query, workspaceID, projectID, issueID, userID)
}

// IssueAttachment reads the object key and metadata from one authorized snapshot.
func (repository *PostgresRepository) IssueAttachment(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID) (FileAsset, map[string]any, error) {
	var encoded []byte
	query := `SELECT ` + fileAssetColumns + `, ` + issueAttachmentProjection + ` FROM file_assets f
 WHERE f.id=$1 AND f.workspace_id=$2 AND f.project_id=$3 AND f.issue_id=$4
 AND f.deleted_at IS NULL AND f.draft_issue_id IS NULL
 AND COALESCE(f.entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
 AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=f.issue_id AND attachment_item.project_id=f.project_id
 AND ` + djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "f.workspace_id") + `)`
	asset, err := scanFileAsset(repository.database.QueryRow(ctx, query, assetID, workspaceID, projectID, issueID, userID), &encoded)
	if err != nil {
		return FileAsset{}, nil, err
	}
	row := make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&row); err != nil {
		return FileAsset{}, nil, fmt.Errorf("decode issue attachment: %w", err)
	}
	return asset, row, nil
}

func (repository *PostgresRepository) UpdateIssueAttachmentUploaded(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID, activity map[string]any, enqueueMetadata bool) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin issue attachment confirmation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var uploaded bool
	err = tx.QueryRow(ctx, `SELECT file.is_uploaded FROM file_assets file WHERE file.id=$1 AND file.workspace_id=$2 AND file.project_id=$3 AND file.issue_id=$4
		AND file.deleted_at IS NULL AND file.draft_issue_id IS NULL AND COALESCE(file.entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
		AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$4 AND attachment_item.project_id=$3
		 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "$2::uuid")+`) FOR UPDATE`, assetID, workspaceID, projectID, issueID, userID).Scan(&uploaded)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("load issue attachment confirmation: %w", err)
	}
	if !uploaded {
		result, err := tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, updated_by_id=$5, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4
		AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$4 AND attachment_item.project_id=$3
		 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "$2::uuid")+`)`, assetID, workspaceID, projectID, issueID, userID)
		if err != nil {
			return fmt.Errorf("confirm issue attachment upload: %w", err)
		}
		if result.RowsAffected() == 0 {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
	}
	if !uploaded && activity != nil {
		if err := insertWorkItemEffect(ctx, tx, "activity", activity); err != nil {
			return err
		}
	}
	if enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit issue attachment confirmation: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) DeleteIssueAttachment(ctx context.Context, userID, workspaceID, projectID, issueID, assetID uuid.UUID, activity map[string]any) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin issue attachment deletion: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	result, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_by_id=$5, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4 AND deleted_at IS NULL AND draft_issue_id IS NULL
		AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')
		AND EXISTS(SELECT 1 FROM issues attachment_item WHERE attachment_item.id=$4 AND attachment_item.project_id=$3
		 AND `+djangoapi.IssueVisibleSQL("attachment_item", "$5::uuid", "$2::uuid")+`)`, assetID, workspaceID, projectID, issueID, userID)
	if err != nil {
		return fmt.Errorf("delete issue attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if activity != nil {
		if err := insertWorkItemEffect(ctx, tx, "activity", activity); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit issue attachment deletion: %w", err)
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
	return repository.findAsset(ctx, id, uuid.Nil, false, true)
}

func (repository *PostgresRepository) UserAssetByID(ctx context.Context, userID, assetID uuid.UUID) (FileAsset, error) {
	return scanFileAsset(repository.database.QueryRow(ctx, `SELECT `+fileAssetColumns+` FROM file_assets f WHERE f.id=$1 AND f.user_id=$2 AND f.deleted_at IS NULL AND `+assetParentVisibilitySQL("f", "$2::uuid"), assetID, userID))
}

func (repository *PostgresRepository) UserAssetsByKey(ctx context.Context, userID uuid.UUID, key string) ([]map[string]any, error) {
	return repository.assetsByKey(ctx, key, &userID, nil, &userID)
}

func (repository *PostgresRepository) WorkspaceAssetsByKey(ctx context.Context, key string, workspaceID, actorID uuid.UUID) ([]map[string]any, error) {
	return repository.assetsByKey(ctx, key, nil, &workspaceID, &actorID)
}

func (repository *PostgresRepository) assetsByKey(ctx context.Context, key string, createdBy, workspaceID, actorID *uuid.UUID) ([]map[string]any, error) {
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
		FROM file_assets asset
		WHERE asset.asset=$1 AND asset.deleted_at IS NULL AND ($2::uuid IS NULL OR asset.created_by_id=$2)
		AND ($3::uuid IS NULL OR asset.workspace_id=$3)
		AND ($4::uuid IS NULL OR `+assetParentVisibilitySQL("asset", "$4::uuid")+`)
	`, key, createdBy, workspaceID, actorID).Scan(&encoded)
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

func (repository *PostgresRepository) UpdateAssetDeletedByKey(ctx context.Context, key string, createdBy, workspaceID *uuid.UUID, actorID uuid.UUID, deleted bool) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin FileAsset key mutation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	query := `SELECT asset.id FROM file_assets asset WHERE asset.asset=$1 AND asset.deleted_at IS NULL
	 AND ($2::uuid IS NULL OR asset.created_by_id=$2) AND asset.draft_issue_id IS NULL
	 AND COALESCE(asset.entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')`
	args := []any{key, createdBy}
	if workspaceID != nil {
		query += ` AND asset.workspace_id=$3 AND ` + assetParentVisibilitySQL("asset", "$4::uuid")
		args = append(args, workspaceID, actorID)
	} else if createdBy != nil {
		query += ` AND ` + assetParentVisibilitySQL("asset", "$3::uuid")
		args = append(args, actorID)
	}
	query += ` ORDER BY asset.created_at DESC LIMIT 2 FOR UPDATE`
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("find FileAsset by key for mutation: %w", err)
	}
	ids := make([]uuid.UUID, 0, 2)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			rows.Close()
			return fmt.Errorf("scan FileAsset key mutation target: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if ErrErr := rows.Err(); ErrErr != nil {
		rows.Close()
		return fmt.Errorf("read FileAsset key mutation targets: %w", ErrErr)
	}
	rows.Close()
	if len(ids) > 1 {
		// Django's .get() raises MultipleObjectsReturned and the request makes
		// no mutation when duplicate storage keys exist.
		return fmt.Errorf("multiple FileAssets match storage key %q", key)
	}
	if len(ids) == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	update := `UPDATE file_assets SET is_deleted=$1 WHERE id=$2 AND asset=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')`
	updateArgs := []any{deleted, ids[0], key}
	if workspaceID != nil {
		update += ` AND workspace_id=$4 AND ` + assetParentVisibilitySQL("file_assets", "$5::uuid")
		updateArgs = append(updateArgs, workspaceID, actorID)
	} else if createdBy != nil {
		update += ` AND ` + assetParentVisibilitySQL("file_assets", "$4::uuid")
		updateArgs = append(updateArgs, actorID)
	}
	result, err := tx.Exec(ctx, update, updateArgs...)
	if err != nil {
		return fmt.Errorf("update FileAsset deletion by key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return tx.Commit(ctx)
}

func (repository *PostgresRepository) UpdateLegacyUserAsset(ctx context.Context, userID, assetID uuid.UUID, attributes []byte, enqueueMetadata bool) (resultErr error) {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin legacy user FileAsset confirmation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	result, err := tx.Exec(ctx, `
UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb, updated_at=NOW()
WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$3::uuid")+`
`, attributes, assetID, userID)
	if err != nil {
		return fmt.Errorf("confirm legacy user FileAsset upload: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// UpdateUserAsset applies the user-avatar/cover relationship update and the
// FileAsset upload confirmation in one database transaction. Reconfirming the
// current asset preserves it; replacing it retires only the previous asset.
// The domain service handles cache invalidation and metadata tasks.
func (repository *PostgresRepository) UpdateUserAsset(ctx context.Context, userID, assetID uuid.UUID, attributes []byte, uploaded, enqueueMetadata bool) (resultErr error) {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user FileAsset update: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var entityType string
	var previous *uuid.UUID
	err = tx.QueryRow(ctx, `
SELECT COALESCE(entity_type,''), CASE WHEN COALESCE(entity_type,'')='USER_AVATAR' THEN avatar_asset_id WHEN COALESCE(entity_type,'')='USER_COVER' THEN cover_image_asset_id ELSE NULL END
FROM file_assets f JOIN users u ON u.id=f.user_id
WHERE f.id=$1 AND f.user_id=$2 AND f.deleted_at IS NULL AND `+assetParentVisibilitySQL("f", "$2::uuid")+`
FOR UPDATE OF f
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
			if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND id<>$2`, *previous, assetID); err != nil {
				return fmt.Errorf("logically delete previous user FileAsset: %w", err)
			}
		}
	}
	if uploaded {
		if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb, updated_at=NOW() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$3::uuid"), attributes, assetID, userID); err != nil {
			return fmt.Errorf("confirm user FileAsset upload: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$2::uuid"), assetID, userID); err != nil {
			return fmt.Errorf("delete user FileAsset: %w", err)
		}
	}
	if uploaded && enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user FileAsset update: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) DeleteUserAsset(ctx context.Context, userID, assetID uuid.UUID) (resultErr error) {
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin user FileAsset deletion: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var entityType string
	err = tx.QueryRow(ctx, `SELECT COALESCE(entity_type,'') FROM file_assets WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$2::uuid")+` FOR UPDATE`, assetID, userID).Scan(&entityType)
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
		if _, err := tx.Exec(ctx, `UPDATE users SET `+column+`=NULL, updated_at=NOW() WHERE id=$1 AND `+column+`=$2`, userID, assetID); err != nil {
			return fmt.Errorf("clear user asset relationship: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$2::uuid"), assetID, userID); err != nil {
		return fmt.Errorf("logically delete user FileAsset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit user FileAsset deletion: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) AssetByWorkspaceAndID(ctx context.Context, workspaceID, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, workspaceID, true, true)
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

func (repository *PostgresRepository) UpdateWorkspaceAssetUploaded(ctx context.Context, actorID, workspaceID, assetID uuid.UUID, uploaded, enqueueMetadata bool) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workspace FileAsset upload-state update: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	result, err := tx.Exec(ctx, `
UPDATE file_assets SET is_uploaded=$1
WHERE id=$2 AND workspace_id=$3 AND is_deleted=false AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$4::uuid")+`
`, uploaded, assetID, workspaceID, actorID)
	if err != nil {
		return fmt.Errorf("update workspace FileAsset upload state: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	if uploaded && enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (repository *PostgresRepository) UpdateWorkspaceAsset(ctx context.Context, userID, workspaceID, assetID uuid.UUID, attributes []byte, enqueueMetadata bool) (_ bool, resultErr error) {
	if !json.Valid(attributes) {
		return false, fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin workspace FileAsset confirmation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var entityType string
	var projectID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(entity_type,''), project_id FROM file_assets
		WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$3::uuid")+` FOR UPDATE
	`, assetID, workspaceID, userID).Scan(&entityType, &projectID)
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
		scanErr := tx.QueryRow(ctx, `SELECT logo_asset_id FROM workspaces WHERE id=$1 AND deleted_at IS NULL`, workspaceID).Scan(&previous)
		if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
			return false, fmt.Errorf("load previous workspace logo asset: %w", scanErr)
		}
		if scanErr == nil {
			if _, execErr := tx.Exec(ctx, `UPDATE workspaces SET logo='', logo_asset_id=$1, updated_at=NOW(), updated_by_id=$3 WHERE id=$2 AND deleted_at IS NULL`, assetID, workspaceID, userID); execErr != nil {
				return false, fmt.Errorf("set workspace logo asset: %w", execErr)
			}
			workspaceHookApplied = true
		}
	case "PROJECT_COVER":
		if projectID != nil {
			if scanErr := tx.QueryRow(ctx, `SELECT cover_image_asset_id FROM projects WHERE id=$1 AND deleted_at IS NULL`, *projectID).Scan(&previous); scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
				return false, fmt.Errorf("load previous project cover asset: %w", scanErr)
			}
			if _, execErr := tx.Exec(ctx, `UPDATE projects SET cover_image='', cover_image_asset_id=$1, updated_at=NOW(), updated_by_id=$3 WHERE id=$2 AND deleted_at IS NULL`, assetID, *projectID, userID); execErr != nil {
				return false, fmt.Errorf("set project cover asset: %w", execErr)
			}
		}
	}
	// A retried confirmation must not retire the asset it just attached.
	if previous != nil {
		if _, execErr := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND id<>$2 AND deleted_at IS NULL`, *previous, assetID); execErr != nil {
			return false, fmt.Errorf("logically delete previous workspace asset: %w", execErr)
		}
	}
	result, err := tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true, attributes=$1::jsonb WHERE id=$2 AND workspace_id=$3 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$4::uuid"), attributes, assetID, workspaceID, userID)
	if err != nil {
		return false, fmt.Errorf("confirm workspace FileAsset upload: %w", err)
	}
	if result.RowsAffected() == 0 {
		return false, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit workspace FileAsset confirmation: %w", err)
	}
	return workspaceHookApplied, nil
}

func (repository *PostgresRepository) DeleteWorkspaceAsset(ctx context.Context, userID, workspaceID, assetID uuid.UUID) (resultErr error) {
	tx, err := repository.database.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}).Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workspace FileAsset deletion: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var entityType string
	var projectID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(entity_type,''), project_id FROM file_assets
		WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$3::uuid")+` FOR UPDATE
	`, assetID, workspaceID, userID).Scan(&entityType, &projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return fmt.Errorf("load workspace FileAsset deletion: %w", err)
	}
	switch entityType {
	case "WORKSPACE_LOGO":
		var exists bool
		if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1 AND deleted_at IS NULL)`, workspaceID).Scan(&exists); scanErr != nil {
			return fmt.Errorf("check active workspace for logo deletion: %w", scanErr)
		}
		if !exists {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		if _, execErr := tx.Exec(ctx, `UPDATE workspaces SET logo_asset_id=NULL, updated_at=NOW() WHERE id=$1 AND logo_asset_id=$2 AND deleted_at IS NULL`, workspaceID, assetID); execErr != nil {
			return fmt.Errorf("clear workspace logo asset: %w", execErr)
		}
	case "PROJECT_COVER":
		if projectID != nil {
			if _, execErr := tx.Exec(ctx, `UPDATE projects SET cover_image_asset_id=NULL, updated_by_id=$2, updated_at=NOW() WHERE id=$1 AND cover_image_asset_id=$3 AND deleted_at IS NULL`, *projectID, userID, assetID); execErr != nil {
				return fmt.Errorf("clear project cover asset: %w", execErr)
			}
		}
	}
	result, err := tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$3::uuid"), assetID, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("logically delete workspace FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit workspace FileAsset deletion: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) UpdateProjectAsset(ctx context.Context, actorID, workspaceID, projectID, assetID uuid.UUID, attributes []byte, enqueueMetadata bool) (resultErr error) {
	if !json.Valid(attributes) {
		return fmt.Errorf("asset attributes must be valid JSON")
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin project FileAsset confirmation: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	command, err := tx.Exec(ctx, `
		UPDATE file_assets SET is_uploaded=true, attributes=$5::jsonb
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$4::uuid")+`
	`, assetID, workspaceID, projectID, actorID, attributes)
	if err != nil {
		return fmt.Errorf("confirm project FileAsset upload: %w", err)
	}
	if command.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if enqueueMetadata {
		if err := insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": assetID.String()}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (repository *PostgresRepository) DeleteProjectAsset(ctx context.Context, actorID, workspaceID, projectID, assetID uuid.UUID) error {
	command, err := repository.database.Exec(ctx, `
		UPDATE file_assets SET is_deleted=true, deleted_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$4::uuid")+`
	`, assetID, workspaceID, projectID, actorID)
	if err != nil {
		return fmt.Errorf("delete project FileAsset: %w", err)
	}
	if command.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) RestoreWorkspaceAsset(ctx context.Context, actorID, workspaceID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `
UPDATE file_assets SET is_deleted=false, deleted_at=NULL, updated_at=NOW()
WHERE id=$1 AND workspace_id=$2 AND draft_issue_id IS NULL AND COALESCE(entity_type,'') NOT IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT') AND `+assetParentVisibilitySQL("file_assets", "$3::uuid")+`
`, assetID, workspaceID, actorID)
	if err != nil {
		return fmt.Errorf("restore workspace FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return nil
}

func (repository *PostgresRepository) AssetByWorkspaceAndIDIncludingDeleted(ctx context.Context, workspaceID, id uuid.UUID) (FileAsset, error) {
	return repository.findAsset(ctx, id, workspaceID, true, false)
}

// WorkspaceAssetExists matches AssetCheckEndpoint, which uses all_objects and
// filters only deleted_at. Its observable result intentionally ignores the
// legacy is_deleted and upload flags.
func (repository *PostgresRepository) WorkspaceAssetExists(ctx context.Context, actorID, workspaceID, id uuid.UUID) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM file_assets asset WHERE asset.id=$1 AND asset.workspace_id=$2 AND asset.deleted_at IS NULL AND `+assetParentVisibilitySQL("asset", "$3::uuid")+`)
	`, id, workspaceID, actorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace FileAsset %s: %w", id, err)
	}
	return exists, nil
}

func (repository *PostgresRepository) AssetForDuplicate(ctx context.Context, actorID, workspaceID, id uuid.UUID) (FileAsset, error) {
	query := `SELECT ` + fileAssetColumns + ` FROM file_assets f WHERE f.id=$1 AND f.workspace_id=$2 AND f.deleted_at IS NULL AND ` + assetParentVisibilitySQL("f", "$3::uuid")
	return scanFileAsset(repository.database.QueryRow(ctx, query, id, workspaceID, actorID))
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
		parsed, parseErr := uuid.Parse(projectID)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"project_id": []string{"Must be a valid UUID."}}}
		}
		project = parsed
	}
	if entityID != "" {
		switch entityType {
		case "WORKSPACE_LOGO":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil {
				return parseErr
			}
			relatedWorkspace = parsed
		case "PROJECT_COVER":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil || entityID == "" {
				return projectCoverEntityMismatchError()
			}
			if requested, ok := project.(uuid.UUID); ok && requested != parsed {
				return projectCoverEntityMismatchError()
			}
			if project == nil {
				project = parsed
			}
		case "USER_AVATAR", "USER_COVER":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil {
				return parseErr
			}
			user = parsed
		case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil {
				return parseErr
			}
			issue = parsed
		case "COMMENT_DESCRIPTION":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil {
				return parseErr
			}
			comment = parsed
		case "PAGE_DESCRIPTION":
			parsed, parseErr := uuid.Parse(entityID)
			if parseErr != nil {
				return parseErr
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
	if entityType == "DRAFT_ISSUE_DESCRIPTION" {
		parent := uuid.Nil
		if project != nil {
			parent = project.(uuid.UUID)
		}
		return repository.CreateWorkspaceEntityUpload(ctx, id, creatorID, workspaceID, parent, destinationKey, attributes, int64(source.Size), entityType, entityID, parent != uuid.Nil)
	}
	metadata, err := json.Marshal(source.StorageMetadata)
	if err != nil {
		return err
	}
	result, err := repository.database.Exec(ctx, `
INSERT INTO file_assets
 (id,created_at,updated_at,attributes,asset,workspace_id,project_id,user_id,issue_id,comment_id,page_id,entity_type,entity_identifier,size,is_uploaded,is_deleted,is_archived,storage_metadata,created_by_id)
SELECT $1,NOW(),NOW(),$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,NULL,$11,false,false,false,$12::jsonb,$13
WHERE `+targetVisibilitySQL(entityType, "$13::uuid", "$4::uuid", "$5::uuid", "$7::uuid", "$8::uuid", "$9::uuid")+`
`, id, attributes, destinationKey, targetWorkspace, project, user, issue, comment, page, entityType, source.Size, metadata, creatorID)
	if err != nil {
		return fmt.Errorf("insert duplicated FileAsset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return notFoundAssetError()
	}
	return nil
}

func (repository *PostgresRepository) MarkDuplicateUploaded(ctx context.Context, actorID, workspaceID, assetID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `UPDATE file_assets SET is_uploaded=true WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND `+assetParentVisibilitySQL("file_assets", "$3::uuid"), assetID, workspaceID, actorID)
	if err != nil {
		return fmt.Errorf("mark duplicated FileAsset uploaded: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("duplicated FileAsset %s disappeared before upload confirmation", assetID)
	}
	return nil
}

const fileAssetColumns = `f.id::text, f.asset, COALESCE(f.entity_type,''), f.attributes, f.workspace_id, f.project_id, f.issue_id, f.user_id, f.draft_issue_id, f.is_uploaded, f.is_deleted, f.size, COALESCE(f.storage_metadata,'{}'::jsonb)`

func (repository *PostgresRepository) findAsset(ctx context.Context, id, workspaceFilter uuid.UUID, scopeWorkspace, filterDeletedAt bool) (FileAsset, error) {
	query := `SELECT ` + fileAssetColumns + ` FROM file_assets f WHERE id=$1`
	args := []any{id}
	if filterDeletedAt {
		query += ` AND deleted_at IS NULL`
	}
	if scopeWorkspace {
		query += ` AND workspace_id=$` + strconv.Itoa(len(args)+1)
		args = append(args, workspaceFilter)
	}
	return scanFileAsset(repository.database.QueryRow(ctx, query, args...))
}

func scanFileAsset(row pgx.Row, extra ...any) (FileAsset, error) {
	var asset FileAsset
	var attributes, metadata []byte
	var workspaceID, projectID, issueID, userID *uuid.UUID
	destinations := []any{&asset.ID, &asset.Asset, &asset.EntityType, &attributes, &workspaceID, &projectID, &issueID, &userID, &asset.DraftIssueID, &asset.IsUploaded, &asset.IsDeleted, &asset.Size, &metadata}
	err := row.Scan(append(destinations, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		return FileAsset{}, fmt.Errorf("load FileAsset: %w", err)
	}
	asset.Attributes, err = decodeFileAssetAttributes(attributes)
	if err != nil {
		return FileAsset{}, fmt.Errorf("decode FileAsset %s attributes: %w", asset.ID, err)
	}
	if err := json.Unmarshal(metadata, &asset.StorageMetadata); err != nil {
		return FileAsset{}, fmt.Errorf("decode FileAsset %s storage metadata: %w", asset.ID, err)
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
