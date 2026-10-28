package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

func (service *LifecycleService) createLegacyUserAsset(ctx context.Context, body io.ReadCloser) (*api.PlaneAppSerializersAssetFileassetserializerResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	return service.createLegacyAsset(ctx, userID, nil, body)
}

func (service *LifecycleService) createLegacyWorkspaceAsset(ctx context.Context, slug string, body io.ReadCloser) (*api.PlaneAppSerializersAssetFileassetserializerResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID, err := service.assets.repository.AuthorizedWorkspaceID(ctx, userID, slug)
	if err != nil {
		return nil, err
	}
	return service.createLegacyAsset(ctx, userID, &workspaceID, body)
}

func (service *LifecycleService) createLegacyAsset(ctx context.Context, userID uuid.UUID, workspaceID *uuid.UUID, body io.ReadCloser) (*api.PlaneAppSerializersAssetFileassetserializerResponse, error) {
	if body == nil {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	defer body.Close()
	contentType := RequestContentType(ctx)
	file, err := parseLegacyAssetFile(body, contentType)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	if file.Name == "" {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	if int64(len(file.Contents)) > service.assets.fileLimit {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"asset": []string{"File too large. Size should not exceed 5 MB."}}}
	}
	fields, err := legacyWritableFields(file.Fields)
	if err != nil {
		return nil, err
	}
	if workspaceID != nil {
		fields["workspace"] = *workspaceID
	}
	if workspaceID == nil {
		if id, ok := fields["workspace"].(uuid.UUID); ok {
			workspaceID = &id
		}
	}
	if validateLegacyRelationsErr := service.validateLegacyRelations(ctx, fields); validateLegacyRelationsErr != nil {
		return nil, validateLegacyRelationsErr
	}
	name := sanitizeFilename(file.Name)
	if name == "" {
		name = randomHex32()
	}
	key := "user-" + randomHex32() + "-" + name
	if workspaceID != nil {
		key = workspaceID.String() + "/" + randomHex32() + "-" + name
	}
	if putErr := service.assets.storage.Put(ctx, key, file.ContentType, file.Contents); putErr != nil {
		return nil, fmt.Errorf("store legacy FileAsset %s: %w", key, putErr)
	}
	asset, err := service.assets.repository.CreateLegacyFileAsset(ctx, uuid.New(), userID, fields, key)
	if err != nil {
		return nil, err
	}
	return decodeAttachmentResult[api.PlaneAppSerializersAssetFileassetserializerResponse](asset)
}

func (service *LifecycleService) validateLegacyRelations(ctx context.Context, fields map[string]any) error {
	for _, field := range []string{"user", "workspace", "draft_issue", "project", "issue", "comment", "page"} {
		id, ok := fields[field].(uuid.UUID)
		if !ok {
			continue
		}
		exists, err := service.assets.repository.LegacyRelationExists(ctx, field, id)
		if err != nil {
			return err
		}
		if !exists {
			return legacyRelationError(field, id.String())
		}
	}
	return nil
}
