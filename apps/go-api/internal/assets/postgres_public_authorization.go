package assets

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (repository *PostgresRepository) PublicOwnedProjectAsset(ctx context.Context, userID, workspaceID, projectID, assetID uuid.UUID, includeDeleted bool) (FileAsset, error) {
	asset, err := repository.PublicProjectAsset(ctx, workspaceID, projectID, assetID, includeDeleted, false)
	if err != nil {
		return FileAsset{}, err
	}
	var ownedID uuid.UUID
	if err := repository.database.QueryRow(ctx, `SELECT id FROM file_assets WHERE id=@id AND workspace_id=@workspace AND project_id=@project AND created_by_id=@actor AND (@include_deleted OR deleted_at IS NULL)`, pgx.StrictNamedArgs{"id": assetID, "workspace": workspaceID, "project": projectID, "actor": userID, "include_deleted": includeDeleted}).Scan(&ownedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FileAsset{}, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
		return FileAsset{}, fmt.Errorf("load public FileAsset owner: %w", err)
	}
	return asset, nil
}
