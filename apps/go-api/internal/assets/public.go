package assets

import (
	"context"
	"fmt"
	"io"
	"strings"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
)

func (service *LifecycleService) createPublicAsset(ctx context.Context, anchor string, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetuploadresponseResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID, projectID, workspaceSlug, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, false)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project is not published"}}
	}
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	name := sanitizeFilename(stringValue(request.Name))
	if name == "" {
		name = "unnamed"
	}
	contentType := stringValue(request.Type)
	if !request.TypePresent {
		contentType = "image/jpeg"
	}
	size, err := request.IntSize(service.assets.fileLimit)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid size.", "status": false}}
	}
	if size < 1 {
		size = 1
	}
	if size > service.assets.fileLimit {
		size = service.assets.fileLimit
	}
	entityType := stringValue(request.EntityType)
	if !validEntityType(entityType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity type.", "status": false}}
	}
	if !allowedV2ProfileMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid file type. Only JPEG, PNG, WebP, JPG and GIF files are allowed.", "status": false}}
	}
	entityIdentifier := stringValue(request.EntityID)
	assetID := uuid.New()
	key := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	attributes := uploadRequest{Name: name, Type: contentType}.Attributes(name, contentType, size)
	if err := service.assets.repository.CreatePublicUpload(ctx, assetID, userID, workspaceID, projectID, key, attributes, size, entityType, entityIdentifier); err != nil {
		return nil, fmt.Errorf("create public FileAsset: %w", err)
	}
	assetURL := publicAssetURL(workspaceSlug, projectID, assetID, entityType)
	return service.assets.signUpload(ctx, assetID.String(), key, contentType, size, assetURL)
}

func publicAssetURL(workspaceSlug string, projectID, assetID uuid.UUID, entityType string) string {
	switch entityType {
	case "WORKSPACE_LOGO", "USER_AVATAR", "USER_COVER", "PROJECT_COVER":
		return "/api/assets/v2/static/" + assetID.String() + "/"
	case "ISSUE_ATTACHMENT":
		return fmt.Sprintf("/api/assets/v2/workspaces/%s/projects/%s/issues/None/attachments/%s/", workspaceSlug, projectID, assetID)
	case "ISSUE_DESCRIPTION", "COMMENT_DESCRIPTION", "PAGE_DESCRIPTION", "DRAFT_ISSUE_DESCRIPTION":
		return fmt.Sprintf("/api/assets/v2/workspaces/%s/projects/%s/%s/", workspaceSlug, projectID, assetID)
	default:
		return ""
	}
}

func (service *LifecycleService) getPublicAsset(ctx context.Context, anchor, rawAssetID string) (*api.GetAPIPublicAssetsV2AnchorByAnchorByPkResult, error) {
	workspaceID, projectID, _, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, false)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Requested resource could not be found."}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.PublicProjectAsset(ctx, workspaceID, projectID, assetID, false, true)
	if err != nil {
		return nil, sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if !asset.IsUploaded {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	attributes, err := fileAssetAttributesObject(asset.Attributes)
	if err != nil {
		return nil, err
	}
	contentType, _ := attributes["type"].(string)
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	disposition := "inline"
	switch contentType {
	case "image/svg+xml", "text/javascript", "application/javascript", "text/html", "application/xhtml+xml", "text/xml", "application/xml":
		disposition = "attachment"
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, disposition, nil, scheme, host)
	if err != nil {
		return nil, fmt.Errorf("presign public FileAsset %s: %w", asset.ID, err)
	}
	return &api.GetAPIPublicAssetsV2AnchorByAnchorByPkResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
}

func (service *LifecycleService) confirmPublicAsset(ctx context.Context, anchor, rawAssetID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, projectID, _, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, false)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project is not published"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.PublicOwnedProjectAsset(ctx, userID, workspaceID, projectID, assetID, false)
	if err != nil {
		return err
	}
	attributes, err := assetAttributes(ctx, body, asset.Attributes)
	if err != nil {
		return err
	}
	if len(asset.StorageMetadata) == 0 {
		if err := service.assets.publishTask(ctx, "plane.bgtasks.storage_metadata_task.get_asset_object_metadata", []any{assetID.String()}, nil); err != nil {
			return fmt.Errorf("queue public FileAsset storage metadata: %w", err)
		}
	}
	return service.assets.repository.UpdatePublicAsset(ctx, userID, workspaceID, projectID, assetID, attributes)
}

func (service *LifecycleService) deletePublicAsset(ctx context.Context, anchor, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, projectID, _, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, true)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project is not published"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return service.assets.repository.DeletePublicAsset(ctx, userID, workspaceID, projectID, assetID)
}

func (service *LifecycleService) restorePublicAsset(ctx context.Context, anchor, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, projectID, _, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, true)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project is not published"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	return service.assets.repository.RestorePublicAsset(ctx, userID, workspaceID, projectID, assetID)
}

func (service *LifecycleService) bulkAssociatePublicAssets(ctx context.Context, anchor, rawEntityID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, projectID, _, found, err := service.assets.repository.PublicProjectBoard(ctx, anchor, true)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project is not published"}}
	}
	var fields map[string]any
	if body != nil {
		defer body.Close()
		contentType := RequestContentType(ctx)
		if contentType == "" {
			contentType = "application/json"
		}
		input, parseErr := djangoinput.Parse(body, contentType)
		if parseErr != nil {
			return parseErr
		}
		fields = input.Fields
	}
	var rawIDs []any
	switch value := fields["asset_ids"].(type) {
	case []any:
		rawIDs = value
	case []string:
		for _, item := range value {
			rawIDs = append(rawIDs, item)
		}
	}
	if len(rawIDs) == 0 {
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "No asset ids provided."}}
	}
	assetIDs := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		value, ok := raw.(string)
		if !ok {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The requested asset could not be found."}}
		}
		id, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The requested asset could not be found."}}
		}
		assetIDs = append(assetIDs, id)
	}
	matchedIDs, entityType, err := service.assets.repository.PublicBulkAssetRows(ctx, userID, workspaceID, projectID, assetIDs)
	if err != nil {
		return err
	}
	if len(matchedIDs) == 0 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	if entityType == "COMMENT_DESCRIPTION" {
		commentID, parseErr := uuid.Parse(rawEntityID)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
		return service.assets.repository.BulkAssociatePublicAssetComments(ctx, userID, workspaceID, projectID, commentID, matchedIDs)
	}
	return nil
}
