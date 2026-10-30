package assets

import (
	"context"
	"errors"
	"fmt"
	"io"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (service *LifecycleService) bulkAssociateDraftAssets(ctx context.Context, slug, rawDraftID string, body io.ReadCloser) error {
	actor, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspace, err := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return err
	}
	draft, err := uuid.Parse(rawDraftID)
	if err != nil {
		return notFoundAssetError()
	}
	if body == nil {
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "No asset ids provided."}}
	}
	defer func() { _ = body.Close() }()
	contentType := RequestContentType(ctx)
	if contentType == "" {
		contentType = "application/json"
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return err
	}
	encoded, ok := input.Fields["asset_ids"].([]any)
	if !ok || len(encoded) == 0 {
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Provide an array of asset ids."}}
	}
	ids := make([]uuid.UUID, 0, len(encoded))
	for _, value := range encoded {
		raw, ok := value.(string)
		if !ok {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Asset ids must be UUIDs."}}
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Asset ids must be UUIDs."}}
		}
		ids = append(ids, id)
	}
	return service.assets.repository.BulkAssociateDraftAssets(ctx, workspace, actor, draft, ids)
}

// Publication takes this same draft row lock before transferring its assets.
// Former project membership does not determine access to a private draft.
func lockOwnedAssetDraft(ctx context.Context, tx pgx.Tx, workspace, actor, draft uuid.UUID) (*uuid.UUID, error) {
	var project *uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT d.project_id FROM draft_issues d
		JOIN workspaces w ON w.id=d.workspace_id AND w.deleted_at IS NULL
		WHERE d.id=$1 AND d.workspace_id=$2 AND d.created_by_id=$3 AND d.deleted_at IS NULL
		AND EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=w.id AND m.member_id=$3
		  AND m.is_active AND m.deleted_at IS NULL AND m.role IN (5,15,20))
		FOR UPDATE OF d`, draft, workspace, actor).Scan(&project)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFoundAssetError()
	}
	return project, err
}

func (repository *PostgresRepository) BulkAssociateDraftAssets(ctx context.Context, workspace, actor, draft uuid.UUID, ids []uuid.UUID) (resultErr error) {
	unique := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		unique[id] = struct{}{}
	}
	if len(unique) == 0 {
		return notFoundAssetError()
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	project, err := lockOwnedAssetDraft(ctx, tx, workspace, actor, draft)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL
		 AND is_uploaded AND NOT is_deleted AND NOT is_archived
		 AND entity_type IN ('ISSUE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION')
		 AND issue_id IS NULL AND comment_id IS NULL AND page_id IS NULL AND user_id IS NULL
		 AND (draft_issue_id IS NULL OR draft_issue_id=$4)
		 AND (entity_identifier IS NULL OR entity_identifier='' OR entity_identifier=$4::uuid::text),false)
		FROM file_assets WHERE id=ANY($1::uuid[]) ORDER BY id FOR UPDATE`, ids, workspace, actor, draft)
	if err != nil {
		return err
	}
	count, allEligible := 0, true
	for rows.Next() {
		var eligible bool
		if err = rows.Scan(&eligible); err != nil {
			rows.Close()
			return err
		}
		count++
		allEligible = allEligible && eligible
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(unique) || !allEligible {
		return notFoundAssetError()
	}
	if _, err = tx.Exec(ctx, `UPDATE file_assets SET draft_issue_id=$2::uuid,
		entity_identifier=$2::uuid::text,entity_type='DRAFT_ISSUE_DESCRIPTION',project_id=$3,
		updated_at=NOW(),updated_by_id=$4 WHERE id=ANY($1::uuid[])`, ids, draft, project, actor); err != nil {
		return fmt.Errorf("associate private draft assets: %w", err)
	}
	return tx.Commit(ctx)
}
