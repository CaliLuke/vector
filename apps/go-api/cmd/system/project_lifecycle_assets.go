package main

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
)

type projectLifecycleAssetReader interface {
	AssetByProjectAndID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (assets.FileAsset, error)
}

func projectLifecycleCoverValidator(reader projectLifecycleAssetReader) workspaceprojectreads.ProjectCoverAssetValidator {
	return func(ctx context.Context, workspace, project, id uuid.UUID) (bool, error) {
		asset, err := reader.AssetByProjectAndID(ctx, workspace, project, id)
		var response *djangoapi.HTTPError
		if errors.As(err, &response) && response.Status == 404 {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return !asset.IsDeleted && asset.EntityType == "PROJECT_COVER", nil
	}
}
