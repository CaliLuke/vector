package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

func (service *LifecycleService) bulkAssociateProjectAssets(ctx context.Context, slug, rawProjectID, rawEntityID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project not found"}}
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return err
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return err
	}
	entityID, err := uuid.Parse(rawEntityID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
	}
	fields := map[string]any{}
	if body != nil {
		defer body.Close()
		contentType := RequestContentType(ctx)
		if contentType == "" {
			contentType = "application/json"
		}
		parsed, parseErr := djangoinput.Parse(body, contentType)
		if parseErr != nil {
			return parseErr
		}
		fields = parsed.Fields
	}
	values, ok := fields["asset_ids"].([]any)
	if !ok {
		if strings, ok2 := fields["asset_ids"].([]string); ok2 {
			values = make([]any, 0, len(strings))
			for _, value := range strings {
				values = append(values, value)
			}
		}
	}
	if len(values) == 0 {
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "No asset ids provided."}}
	}
	ids := make([]uuid.UUID, 0, len(values))
	for _, raw := range values {
		value, ok := raw.(string)
		if !ok {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
		}
		id, parseErr := uuid.Parse(value)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
		}
		ids = append(ids, id)
	}
	return service.assets.repository.BulkAssociateProjectAssets(ctx, workspaceID, projectID, userID, entityID, ids)
}

func (service *LifecycleService) listLegacyIssueAttachments(ctx context.Context, slug, rawProjectID, rawIssueID string) ([]*api.PlaneAppSerializersIssueIssueattachmentserializerResponse, error) {
	return service.listScopedIssueAttachments(ctx, slug, rawProjectID, rawIssueID, false)
}

func (service *LifecycleService) listScopedIssueAttachments(ctx context.Context, slug, rawProjectID, rawIssueID string, uploadedOnly bool) ([]*api.PlaneAppSerializersIssueIssueattachmentserializerResponse, error) {
	userID, workspaceID, err := service.attachmentScope(ctx, slug, rawProjectID)
	if err != nil {
		return nil, err
	}
	projectID, _ := uuid.Parse(rawProjectID)
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	rows, err := service.assets.repository.VisibleIssueAttachments(ctx, userID, workspaceID, projectID, issueID, uploadedOnly)
	if err != nil {
		return nil, err
	}
	return decodeAttachmentRows[api.PlaneAppSerializersIssueIssueattachmentserializerResponse](decorateIssueAttachmentRows(rows, slug, false))
}

func (service *LifecycleService) createLegacyIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID string, body io.ReadCloser) (*api.PlaneAppSerializersIssueIssueattachmentserializerResponse, error) {
	userID, workspaceID, err := service.attachmentScope(ctx, slug, rawProjectID)
	if err != nil {
		return nil, err
	}
	projectID, _ := uuid.Parse(rawProjectID)
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if body == nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	defer body.Close()
	file, err := parseLegacyAssetFile(body, RequestContentType(ctx))
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	activeWorkspaceID, err := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	workspaceID = activeWorkspaceID
	found, allowed, err := service.assets.repository.IssueAccess(ctx, userID, workspaceID, projectID, issueID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue not found."}}
	}
	if !allowed {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	if file.Name == "" {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"asset": []string{"No file was submitted."}}}
	}
	if int64(len(file.Contents)) > service.assets.fileLimit {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"asset": []string{"File too large. Size should not exceed 5 MB."}}}
	}
	fields, err := legacyWritableFields(file.Fields)
	if err != nil {
		return nil, err
	}
	fields["workspace"], fields["project"], fields["issue"] = workspaceID, projectID, issueID
	fields["entity_type"] = "ISSUE_ATTACHMENT"
	if validateLegacyRelationsErr := service.validateLegacyRelations(ctx, fields); validateLegacyRelationsErr != nil {
		return nil, validateLegacyRelationsErr
	}
	name := sanitizeFilename(file.Name)
	if name == "" {
		name = randomHex32()
	}
	key := workspaceID.String() + "/" + randomHex32() + "-" + name
	if putErr := service.assets.storage.Put(ctx, key, file.ContentType, file.Contents); putErr != nil {
		return nil, fmt.Errorf("store legacy issue attachment: %w", putErr)
	}
	assetID := uuid.New()
	assetURL := issueAttachmentURL(slug, projectID, issueID, assetID)
	activity := map[string]any{"type": "attachment.activity.created", "requested_data": nil, "actor_id": userID.String(), "issue_id": issueID.String(), "project_id": projectID.String(), "epoch": time.Now().Unix(), "notification": true, "origin": service.assets.taskOrigin(ctx)}
	if createLegacyErr := service.assets.repository.CreateLegacyIssueAttachment(ctx, assetID, userID, workspaceID, projectID, issueID, fields, key, assetURL, activity); createLegacyErr != nil {
		return nil, createLegacyErr
	}
	rows, err := service.assets.repository.IssueAttachments(ctx, workspaceID, projectID, issueID, false)
	if err != nil {
		return nil, err
	}
	var serialized map[string]any
	for _, row := range rows {
		if row["id"] == assetID.String() {
			serialized = row
			break
		}
	}
	if serialized == nil {
		return nil, fmt.Errorf("legacy issue attachment %s missing after creation", assetID)
	}
	serialized = decorateIssueAttachmentRows([]map[string]any{serialized}, slug, false)[0]
	return decodeAttachmentResult[api.PlaneAppSerializersIssueIssueattachmentserializerResponse](decorateIssueAttachmentRows([]map[string]any{serialized}, slug, false)[0])
}

func (service *LifecycleService) deleteLegacyIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue attachment not found."}}
	}
	allowed, err := service.assets.repository.IssueAttachmentDeleteAllowed(ctx, userID, slug, projectID, assetID)
	if err != nil {
		return err
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return err
	}
	_, _, err = service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue attachment not found."}}
	}
	activity := map[string]any{"type": "attachment.activity.deleted", "requested_data": nil, "actor_id": userID.String(), "issue_id": issueID.String(), "project_id": projectID.String(), "current_instance": nil, "epoch": time.Now().Unix(), "notification": true, "origin": service.assets.taskOrigin(ctx)}
	return service.assets.repository.DeleteLegacyIssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID, activity)
}

func (service *LifecycleService) attachmentScope(ctx context.Context, slug, rawProjectID string) (uuid.UUID, uuid.UUID, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if !allowed {
		return uuid.Nil, uuid.Nil, &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return userID, workspaceID, nil
}

func (service *LifecycleService) listIssueAttachments(ctx context.Context, slug, rawProjectID, rawIssueID string) ([]*api.PlaneAppSerializersIssueIssueattachmentserializerResponse, error) {
	return service.listScopedIssueAttachments(ctx, slug, rawProjectID, rawIssueID, true)
}

// API V1's collection GET uses the same issue authorization as its upload and
// mutation operations before returning attachment metadata.
func (service *LifecycleService) listIssueAttachmentsAPI(ctx context.Context, slug, rawProjectID, rawIssueID string) ([]*api.PlaneAppSerializersIssueIssueattachmentserializerResponse, error) {
	if _, err := assetUserID(ctx); err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	userID, workspaceID, _, found, allowed, err := service.issuePermissionV1(ctx, slug, rawProjectID, rawIssueID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
	}
	if !allowed {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You are not allowed to download this attachment"}}
	}
	rows, err := service.assets.repository.VisibleIssueAttachments(ctx, userID, workspaceID, projectID, issueID, true)
	if err != nil {
		return nil, err
	}
	return decodeAttachmentRows[api.PlaneAppSerializersIssueIssueattachmentserializerResponse](decorateIssueAttachmentRows(rows, slug, true))
}

func (service *LifecycleService) createIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID string, body io.ReadCloser) (*api.RestFrameworkSerializersIssueattachmentuploadresponseResponse, error) {
	return service.createIssueAttachmentAPI(ctx, slug, rawProjectID, rawIssueID, body, false)
}

func (service *LifecycleService) createIssueAttachmentAPI(ctx context.Context, slug, rawProjectID, rawIssueID string, body io.ReadCloser, apiV1 bool) (*api.RestFrameworkSerializersIssueattachmentuploadresponseResponse, error) {
	var userID, workspaceID, projectID, issueID uuid.UUID
	var err error
	if apiV1 {
		var found, allowed bool
		var permissionErr error
		userID, workspaceID, projectID, found, allowed, permissionErr = service.issuePermissionV1(ctx, slug, rawProjectID, rawIssueID)
		if permissionErr != nil {
			return nil, permissionErr
		}
		if !found {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
		}
		if !allowed {
			return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You are not allowed to upload this attachment"}}
		}
		issueID, _ = uuid.Parse(rawIssueID)
	} else {
		userID, workspaceID, err = service.attachmentScope(ctx, slug, rawProjectID)
		if err != nil {
			return nil, err
		}
		projectID, _ = uuid.Parse(rawProjectID)
		issueID, err = uuid.Parse(rawIssueID)
		if err != nil {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
		}
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
	size, err := request.IntSize(service.assets.fileLimit)
	if err != nil {
		return nil, err
	}
	if apiV1 && (sanitizeFilename(stringValue(request.Name)) == "" || !request.SizePresent || size == 0) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid request.", "status": false}}
	}
	if !allowedAttachmentMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid file type.", "status": false}}
	}
	if size > service.assets.fileLimit {
		size = service.assets.fileLimit
	}
	if apiV1 {
		activeWorkspaceID, workspaceErr := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
		if workspaceErr != nil {
			return nil, sourceLookupError(workspaceErr, map[string]any{"error": "The requested resource does not exist."})
		}
		workspaceID = activeWorkspaceID
	} else {
		activeWorkspaceID, workspaceErr := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
		if workspaceErr != nil {
			return nil, workspaceErr
		}
		workspaceID = activeWorkspaceID
	}
	externalID, externalSource := "", ""
	if apiV1 {
		externalID = stringValue(request.ExternalID)
		externalSource = stringValue(request.ExternalSource)
		if externalID != "" && externalSource != "" {
			id, exists, checkErr := service.assets.repository.IssueAttachmentExternal(ctx, userID, workspaceID, projectID, issueID, externalID, externalSource)
			if checkErr != nil {
				return nil, checkErr
			}
			if exists {
				return nil, &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": "Issue with the same external id and external source already exists", "id": id.String()}}
			}
		}
	}
	assetID := uuid.New()
	key := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	attributes := uploadRequest{Name: name, Type: contentType}.Attributes(name, contentType, size)
	if createIssueAttachmentErr := service.assets.repository.CreateIssueAttachment(ctx, assetID, userID, workspaceID, projectID, issueID, key, attributes, size, externalID, externalSource); createIssueAttachmentErr != nil {
		return nil, fmt.Errorf("create issue attachment FileAsset: %w", createIssueAttachmentErr)
	}
	assetURL := issueAttachmentURL(slug, projectID, issueID, assetID)
	created, serialized, err := service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return nil, err
	}
	_ = created
	if apiV1 {
		serialized = apiIssueAttachmentRow(serialized)
	} else {
		serialized = decorateIssueAttachmentRows([]map[string]any{serialized}, slug, false)[0]
	}
	upload, err := service.assets.signUpload(ctx, assetID.String(), key, contentType, size, assetURL)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"asset_id": assetID.String(), "asset_url": assetURL, "upload_data": upload.UploadData, "attachment": serialized}
	return decodeAttachmentResult[api.RestFrameworkSerializersIssueattachmentuploadresponseResponse](result)
}

func (service *LifecycleService) getIssueAttachmentV1(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) (*string, error) {
	userID, workspaceID, projectID, found, allowed, err := service.issuePermissionV1(ctx, slug, rawProjectID, rawIssueID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
	}
	if !allowed {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You are not allowed to download this attachment"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	issueID, _ := uuid.Parse(rawIssueID)
	asset, _, err := service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return nil, sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	if !asset.IsUploaded {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The asset is not uploaded.", "status": false}}
	}
	var filename *string
	attributes, err := fileAssetAttributesObject(asset.Attributes)
	if err != nil {
		return nil, err
	}
	if name, ok := attributes["name"].(string); ok {
		filename = &name
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, "attachment", filename, scheme, host)
	if err != nil {
		return nil, fmt.Errorf("presign issue attachment %s: %w", asset.ID, err)
	}
	return &location, nil
}

func (service *LifecycleService) issuePermissionV1(ctx context.Context, slug, rawProjectID, rawIssueID string) (uuid.UUID, uuid.UUID, uuid.UUID, bool, bool, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, false, false, err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, false, false, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, false, false, sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, false, false, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue not found"}}
	}
	found, allowed, err := service.assets.repository.IssueAccess(ctx, userID, workspaceID, projectID, issueID)
	return userID, workspaceID, projectID, found, allowed, err
}

func (service *LifecycleService) confirmIssueAttachmentV1(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) error {
	userID, workspaceID, projectID, found, allowed, err := service.issuePermissionV1(ctx, slug, rawProjectID, rawIssueID)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You are not allowed to upload this attachment"}}
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue not found"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, row, err := service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	var activity map[string]any
	if !asset.IsUploaded {
		current := decorateIssueAttachmentRows([]map[string]any{row}, slug, true)[0]
		currentJSON, marshalErr := json.Marshal(pythonJSONNumbers(current))
		if marshalErr != nil {
			return marshalErr
		}
		activity = map[string]any{
			"type": "attachment.activity.created", "requested_data": nil, "actor_id": userID.String(), "issue_id": rawIssueID,
			"project_id": projectID.String(), "current_instance": string(currentJSON), "epoch": time.Now().Unix(),
			"notification": true, "origin": service.assets.taskOrigin(ctx),
		}
	}
	return service.assets.repository.UpdateIssueAttachmentUploaded(ctx, userID, workspaceID, projectID, issueID, assetID, activity, len(asset.StorageMetadata) == 0)
}

func (service *LifecycleService) deleteIssueAttachmentV1(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) error {
	userID, workspaceID, projectID, found, allowed, err := service.issuePermissionV1(ctx, slug, rawProjectID, rawIssueID)
	if err != nil {
		return err
	}
	if !found {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You are not allowed to delete this attachment"}}
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Issue not found"}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	_, _, err = service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	activity := map[string]any{"type": "attachment.activity.deleted", "requested_data": nil, "actor_id": userID.String(), "issue_id": rawIssueID,
		"project_id": projectID.String(), "current_instance": nil, "epoch": time.Now().Unix(), "notification": true, "origin": service.assets.taskOrigin(ctx)}
	if err := service.assets.repository.DeleteIssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID, activity); err != nil {
		return err
	}
	return nil
}

func (service *LifecycleService) getIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) (*api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkResult, error) {
	userID, workspaceID, err := service.attachmentScope(ctx, slug, rawProjectID)
	if err != nil {
		return nil, err
	}
	projectID, _ := uuid.Parse(rawProjectID)
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, _, err := service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return nil, sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if !asset.IsUploaded {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The asset is not uploaded.", "status": false}}
	}
	var filename *string
	attributes, err := fileAssetAttributesObject(asset.Attributes)
	if err != nil {
		return nil, err
	}
	if name, ok := attributes["name"].(string); ok {
		filename = &name
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, "attachment", filename, scheme, host)
	if err != nil {
		return nil, fmt.Errorf("presign issue attachment %s: %w", asset.ID, err)
	}
	return &api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
}

func (service *LifecycleService) confirmIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) error {
	userID, workspaceID, err := service.attachmentScope(ctx, slug, rawProjectID)
	if err != nil {
		return err
	}
	projectID, _ := uuid.Parse(rawProjectID)
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, row, err := service.assets.repository.IssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	var activity map[string]any
	if !asset.IsUploaded {
		row = decorateIssueAttachmentRows([]map[string]any{row}, slug, false)[0]
		currentInstance, marshalErr := json.Marshal(pythonJSONNumbers(row))
		if marshalErr != nil {
			return marshalErr
		}
		activity = map[string]any{
			"type": "attachment.activity.created", "requested_data": nil, "actor_id": userID.String(),
			"issue_id": issueID.String(), "project_id": projectID.String(), "current_instance": string(currentInstance),
			"epoch": time.Now().Unix(), "notification": true, "origin": service.assets.taskOrigin(ctx),
		}
	}
	return service.assets.repository.UpdateIssueAttachmentUploaded(ctx, userID, workspaceID, projectID, issueID, assetID, activity, len(asset.StorageMetadata) == 0)
}

func (service *LifecycleService) deleteIssueAttachment(ctx context.Context, slug, rawProjectID, rawIssueID, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	allowed, err := service.assets.repository.IssueAttachmentDeleteAllowed(ctx, userID, slug, projectID, assetID)
	if err != nil {
		return err
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return err
	}
	issueID, err := uuid.Parse(rawIssueID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	activity := map[string]any{"type": "attachment.activity.deleted", "requested_data": nil, "actor_id": userID.String(),
		"issue_id": issueID.String(), "project_id": projectID.String(), "current_instance": nil,
		"epoch": time.Now().Unix(), "notification": true, "origin": service.assets.taskOrigin(ctx)}
	if err := service.assets.repository.DeleteIssueAttachment(ctx, userID, workspaceID, projectID, issueID, assetID, activity); err != nil {
		return sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	return nil
}

func issueAttachmentURL(slug string, projectID, issueID, assetID uuid.UUID) string {
	return fmt.Sprintf("/api/assets/v2/workspaces/%s/projects/%s/issues/%s/attachments/%s/", slug, projectID, issueID, assetID)
}

func decodeAttachmentRows[T any](rows []map[string]any) ([]*T, error) {
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	var result []*T
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, fmt.Errorf("decode issue attachment response: %w", err)
	}
	return result, nil
}

func decorateIssueAttachmentRows(rows []map[string]any, slug string, apiSerializer bool) []map[string]any {
	decorated := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		copy := make(map[string]any, len(row)+1)
		for key, value := range row {
			copy[key] = value
		}
		if size, ok := row["size"].(json.Number); ok {
			if parsed, err := strconv.ParseFloat(size.String(), 64); err == nil {
				copy["size"] = json.Number(pythoncompat.FloatString(parsed))
			}
		}
		if apiSerializer {
			delete(copy, "asset_url")
		} else {
			entityType, _ := row["entity_type"].(string)
			assetID, _ := row["id"].(string)
			projectID, _ := row["project"].(string)
			issueID, _ := row["issue"].(string)
			if entityType == "ISSUE_ATTACHMENT" && assetID != "" {
				copy["asset_url"] = fmt.Sprintf("/api/assets/v2/workspaces/%s/projects/%s/issues/%s/attachments/%s/", slug, projectID, issueID, assetID)
			}
		}
		decorated = append(decorated, copy)
	}
	return decorated
}

func apiIssueAttachmentRow(row map[string]any) map[string]any {
	copy := make(map[string]any, len(row))
	for key, value := range row {
		if key != "asset_url" {
			copy[key] = value
		}
	}
	return copy
}

func decodeAttachmentResult[T any](value any) (*T, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, fmt.Errorf("decode issue attachment upload response: %w", err)
	}
	return &result, nil
}
