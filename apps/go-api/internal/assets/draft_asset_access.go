package assets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func privateDraftAsset(asset FileAsset) bool {
	return asset.DraftIssueID != nil || asset.EntityType == "DRAFT_ISSUE_DESCRIPTION" || asset.EntityType == "DRAFT_ISSUE_ATTACHMENT"
}

// Draft ownership overrides project membership, including administrator access.
func (service *LifecycleService) authorizeAssetAttachment(ctx context.Context, workspace, actor uuid.UUID, asset FileAsset) error {
	if privateDraftAsset(asset) {
		id, err := uuid.Parse(asset.ID)
		if err != nil {
			return notFoundAssetError()
		}
		allowed, err := service.assets.repository.DraftAssetAllowed(ctx, workspace, actor, id)
		if err != nil {
			return err
		}
		if !allowed {
			return notFoundAssetError()
		}
		return nil
	}
	assetID, err := uuid.Parse(asset.ID)
	if err != nil {
		return notFoundAssetError()
	}
	visible, err := service.assets.repository.AssetParentVisible(ctx, actor, workspace, assetID)
	if err != nil {
		return err
	}
	if !visible {
		return notFoundAssetError()
	}
	if asset.ProjectID != nil {
		allowed, err := service.assets.repository.ProjectMemberActive(ctx, actor, workspace, *asset.ProjectID)
		if err != nil {
			return err
		}
		if !allowed {
			return notFoundAssetError()
		}
	}
	return nil
}

func (repository *PostgresRepository) DraftAssetAllowed(ctx context.Context, workspace, actor, asset uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM file_assets f JOIN workspaces w ON w.id=f.workspace_id AND w.deleted_at IS NULL
		WHERE f.id=$1 AND f.workspace_id=$2 AND f.created_by_id=$3
		AND (f.draft_issue_id IS NOT NULL OR f.entity_type IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT')) AND NOT f.is_archived
		AND EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=w.id AND m.member_id=$3
		 AND m.is_active AND m.deleted_at IS NULL AND m.role IN (5,15,20))
		AND (f.draft_issue_id IS NULL OR EXISTS(SELECT 1 FROM draft_issues d WHERE d.id=f.draft_issue_id
		 AND d.workspace_id=f.workspace_id AND d.created_by_id=$3 AND d.deleted_at IS NULL)))`, asset, workspace, actor).Scan(&allowed)
	return allowed, err
}

// Take the draft lock before the asset lock, matching publication's lock order.
// A stale confirmation cannot change a newly published work item's asset.
func (repository *PostgresRepository) MutateDraftAsset(ctx context.Context, workspace, actor, asset uuid.UUID, action string, attributes []byte, uploaded bool) error {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var draft *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT draft_issue_id FROM file_assets WHERE id=$1 AND workspace_id=$2 AND created_by_id=$3 AND (draft_issue_id IS NOT NULL OR entity_type IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT'))`, asset, workspace, actor).Scan(&draft)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundAssetError()
	}
	if err != nil {
		return err
	}
	if draft != nil {
		if _, err = lockOwnedAssetDraft(ctx, tx, workspace, actor, *draft); err != nil {
			return err
		}
	}
	var locked uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM file_assets WHERE id=$1 AND workspace_id=$2 AND created_by_id=$3
		AND draft_issue_id IS NOT DISTINCT FROM $4 AND (draft_issue_id IS NOT NULL OR entity_type IN ('DRAFT_ISSUE_DESCRIPTION','DRAFT_ISSUE_ATTACHMENT'))
		AND NOT is_archived AND ($5='restore' OR (deleted_at IS NULL AND NOT is_deleted)) FOR UPDATE`, asset, workspace, actor, draft, action).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundAssetError()
	}
	if err != nil {
		return err
	}
	writer := &PostgresRepository{database: tx}
	allowed, err := writer.DraftAssetAllowed(ctx, workspace, actor, asset)
	if err != nil {
		return err
	}
	if !allowed {
		return notFoundAssetError()
	}
	switch action {
	case "confirm":
		_, err = tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=$2,attributes=COALESCE($3::jsonb,attributes),updated_at=now(),updated_by_id=$4 WHERE id=$1`, asset, uploaded, attributes, actor)
	case "delete":
		_, err = tx.Exec(ctx, `UPDATE file_assets SET is_deleted=true,deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, asset, actor)
	case "restore":
		_, err = tx.Exec(ctx, `UPDATE file_assets SET is_deleted=false,deleted_at=NULL,updated_at=now(),updated_by_id=$2 WHERE id=$1`, asset, actor)
	default:
		return notFoundAssetError()
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
