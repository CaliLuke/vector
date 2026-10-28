package assets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type LifecycleService struct {
	assets *Service
}

func NewLifecycleService(assets *Service) (*LifecycleService, error) {
	if assets == nil {
		return nil, fmt.Errorf("assets lifecycle service is required")
	}
	return &LifecycleService{assets: assets}, nil
}

func (service *LifecycleService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	updated, err := service.assets.APIKeyAuth(ctx, key, scheme)
	if err == nil {
		return updated, nil
	}
	operation, _ := ctx.Value(loom.MethodKey).(string)
	return updated, mapAssetsLifecycleError(operation, err)
}

// invoke is the single binding boundary for the reverse-generated service
// methods. Each case delegates to the source-shaped operation implementation;
// keeping dispatch here lets the generated interface remain mechanically
// reproducible without embedding a nil Service implementation.
func (service *LifecycleService) invoke(ctx context.Context, name string, payload any, body io.ReadCloser) (any, error) {
	switch name {
	case "GetAPIAssetsV2WorkspacesBySlugCheckByAssetID":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugCheckByAssetIDPayload)
		userID, err := assetUserID(ctx)
		if err != nil {
			return nil, err
		}
		workspaceID, err := service.v2WorkspaceID(ctx, userID, p.Slug)
		if err != nil {
			return nil, err
		}
		assetID, err := uuid.Parse(p.AssetID)
		if err != nil {
			return &api.RestFrameworkSerializersFileassetexistsresponseResponse{Exists: false}, nil
		}
		exists, err := service.assets.repository.WorkspaceAssetExists(ctx, userID, workspaceID, assetID)
		if err != nil {
			return nil, err
		}
		if exists {
			asset, lookupErr := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
				var denied *djangoapi.HTTPError
				if errors.As(accessErr, &denied) && denied.Status == 404 {
					exists = false
				} else {
					return nil, accessErr
				}
			}
		}
		return &api.RestFrameworkSerializersFileassetexistsresponseResponse{Exists: exists}, nil
	case "GetAPIAssetsV2WorkspacesBySlugDownloadByAssetID":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugDownloadByAssetIDPayload)
		location, err := service.workspaceDownload(ctx, p.Slug, p.AssetID)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIAssetsV2WorkspacesBySlugDownloadByAssetIDResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "GetAPIAssetsV2WorkspacesBySlugByAssetID":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugByAssetIDPayload)
		location, err := service.workspaceAssetRedirect(ctx, p.Slug, p.AssetID)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIAssetsV2WorkspacesBySlugByAssetIDResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPk":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkPayload)
		location, err := service.projectAssetRedirect(ctx, p.Slug, p.ProjectID, p.Pk)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDDownloadByAssetID":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDDownloadByAssetIDPayload)
		location, err := service.projectDownload(ctx, p.Slug, p.ProjectID, p.AssetID)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDDownloadByAssetIDResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "PostAPIAssetsV2WorkspacesBySlugDuplicateAssetsByAssetID":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugDuplicateAssetsByAssetIDPayload)
		result, err := service.duplicateWorkspaceAsset(ctx, p.Slug, p.AssetID, body)
		return result, err
	case "PostAPIAssetsV2WorkspacesBySlug":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugPayload)
		result, err := service.createV2WorkspaceUpload(ctx, p.Slug, body)
		return result, err
	case "PostAPIAssetsV2WorkspacesBySlugProjectsByProjectID":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDPayload)
		result, err := service.createV2ProjectUpload(ctx, p.Slug, p.ProjectID, body)
		return result, err
	case "PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByEntityIDBulk":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByEntityIDBulkPayload)
		return nil, service.bulkAssociateProjectAssets(ctx, p.Slug, p.ProjectID, p.EntityID, body)
	case "PostAPIAssetsV2WorkspacesBySlugByEntityIDBulk":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugByEntityIDBulkPayload)
		return nil, service.bulkAssociateDraftAssets(ctx, p.Slug, p.EntityID, body)
	case "PatchAPIAssetsV2UserAssetsByAssetID":
		p := payload.(*api.PatchAPIAssetsV2UserAssetsByAssetIDPayload)
		return nil, service.confirmUserAssetUpload(ctx, p.AssetID, body)
	case "DeleteAPIAssetsV2UserAssetsByAssetID":
		p := payload.(*api.DeleteAPIAssetsV2UserAssetsByAssetIDPayload)
		return nil, service.deleteUserAsset(ctx, p.AssetID)
	case "PatchAPIV1AssetsUserAssetsByAssetID":
		p := payload.(*api.PatchAPIV1AssetsUserAssetsByAssetIDPayload)
		return nil, service.confirmLegacyUserAssetUpload(ctx, p.AssetID, body)
	case "DeleteAPIV1AssetsUserAssetsByAssetID":
		p := payload.(*api.DeleteAPIV1AssetsUserAssetsByAssetIDPayload)
		return nil, service.deleteLegacyUserAsset(ctx, p.AssetID)
	case "PatchAPIV1AssetsUserAssetsByAssetIDServer":
		p := payload.(*api.PatchAPIV1AssetsUserAssetsByAssetIDServerPayload)
		return nil, service.confirmLegacyUserAssetUpload(ctx, p.AssetID, body)
	case "DeleteAPIV1AssetsUserAssetsByAssetIDServer":
		p := payload.(*api.DeleteAPIV1AssetsUserAssetsByAssetIDServerPayload)
		return nil, service.deleteLegacyUserAsset(ctx, p.AssetID)
	case "PatchAPIV1WorkspacesBySlugAssetsByAssetID":
		p := payload.(*api.PatchAPIV1WorkspacesBySlugAssetsByAssetIDPayload)
		return nil, service.confirmLegacyWorkspaceAssetUpload(ctx, p.Slug, p.AssetID, body)
	case "PostAPIAssetsV2WorkspacesBySlugRestoreByAssetID":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugRestoreByAssetIDPayload)
		return nil, service.restoreWorkspaceAsset(ctx, p.Slug, p.AssetID)
	case "PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPk":
		p := payload.(*api.PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkPayload)
		return nil, service.patchProjectAsset(ctx, p.Slug, p.ProjectID, p.Pk, body)
	case "DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPk":
		p := payload.(*api.DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkPayload)
		return nil, service.deleteProjectAsset(ctx, p.Slug, p.ProjectID, p.Pk)
	case "PatchAPIAssetsV2WorkspacesBySlugByAssetID":
		p := payload.(*api.PatchAPIAssetsV2WorkspacesBySlugByAssetIDPayload)
		return nil, service.patchWorkspaceAsset(ctx, p.Slug, p.AssetID, body)
	case "DeleteAPIAssetsV2WorkspacesBySlugByAssetID":
		p := payload.(*api.DeleteAPIAssetsV2WorkspacesBySlugByAssetIDPayload)
		return nil, service.deleteWorkspaceAsset(ctx, p.Slug, p.AssetID)
	case "GetAPIUsersFileAssetsByAssetKey":
		p := payload.(*api.GetAPIUsersFileAssetsByAssetKeyPayload)
		return service.getLegacyUserAssetByKey(ctx, p.AssetKey)
	case "DeleteAPIUsersFileAssetsByAssetKey":
		p := payload.(*api.DeleteAPIUsersFileAssetsByAssetKeyPayload)
		return nil, service.deleteLegacyUserAssetByKey(ctx, p.AssetKey)
	case "GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKey":
		p := payload.(*api.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyPayload)
		return service.getLegacyWorkspaceAssetByKey(ctx, p.WorkspaceID, p.AssetKey)
	case "DeleteAPIWorkspacesFileAssetsByWorkspaceIDByAssetKey":
		p := payload.(*api.DeleteAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyPayload)
		return nil, service.deleteLegacyWorkspaceAssetByKey(ctx, p.WorkspaceID, p.AssetKey, true)
	case "PostAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyRestore":
		p := payload.(*api.PostAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyRestorePayload)
		return nil, service.deleteLegacyWorkspaceAssetByKey(ctx, p.WorkspaceID, p.AssetKey, false)
	case "PostAPIPublicAssetsV2AnchorByAnchor":
		p := payload.(*api.PostAPIPublicAssetsV2AnchorByAnchorPayload)
		return service.createPublicAsset(ctx, p.Anchor, body)
	case "GetAPIPublicAssetsV2AnchorByAnchorByPk":
		p := payload.(*api.GetAPIPublicAssetsV2AnchorByAnchorByPkPayload)
		return service.getPublicAsset(ctx, p.Anchor, p.Pk)
	case "PatchAPIPublicAssetsV2AnchorByAnchorByPk":
		p := payload.(*api.PatchAPIPublicAssetsV2AnchorByAnchorByPkPayload)
		return nil, service.confirmPublicAsset(ctx, p.Anchor, p.Pk, body)
	case "DeleteAPIPublicAssetsV2AnchorByAnchorByPk":
		p := payload.(*api.DeleteAPIPublicAssetsV2AnchorByAnchorByPkPayload)
		return nil, service.deletePublicAsset(ctx, p.Anchor, p.Pk)
	case "PostAPIPublicAssetsV2AnchorByAnchorRestoreByPk":
		p := payload.(*api.PostAPIPublicAssetsV2AnchorByAnchorRestoreByPkPayload)
		return nil, service.restorePublicAsset(ctx, p.Anchor, p.Pk)
	case "PostAPIPublicAssetsV2AnchorByAnchorByEntityIDBulk":
		p := payload.(*api.PostAPIPublicAssetsV2AnchorByAnchorByEntityIDBulkPayload)
		return nil, service.bulkAssociatePublicAssets(ctx, p.Anchor, p.EntityID, body)
	case "GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments":
		p := payload.(*api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsPayload)
		return service.listIssueAttachmentsAPI(ctx, p.Slug, p.ProjectID, p.IssueID)
	case "PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments":
		p := payload.(*api.PostAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsPayload)
		return service.createIssueAttachmentAPI(ctx, p.Slug, p.ProjectID, p.IssueID, body, true)
	case "GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk":
		p := payload.(*api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload)
		location, err := service.getIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkResult{Location: location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk":
		p := payload.(*api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload)
		return nil, service.confirmIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk":
		p := payload.(*api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload)
		return nil, service.deleteIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachments":
		p := payload.(*api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsPayload)
		return service.listIssueAttachmentsAPI(ctx, p.Slug, p.ProjectID, p.IssueID)
	case "PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachments":
		p := payload.(*api.PostAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsPayload)
		return service.createIssueAttachmentAPI(ctx, p.Slug, p.ProjectID, p.IssueID, body, true)
	case "GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPk":
		p := payload.(*api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkPayload)
		location, err := service.getIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
		if err != nil {
			return nil, err
		}
		return &api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkResult{Location: location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
	case "PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPk":
		p := payload.(*api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkPayload)
		return nil, service.confirmIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPk":
		p := payload.(*api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkPayload)
		return nil, service.deleteIssueAttachmentV1(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsPayload)
		return service.listIssueAttachments(ctx, p.Slug, p.ProjectID, p.IssueID)
	case "PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments":
		p := payload.(*api.PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsPayload)
		return service.createIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, body)
	case "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk":
		p := payload.(*api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload)
		return service.getIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk":
		p := payload.(*api.PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload)
		return nil, service.confirmIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk":
		p := payload.(*api.DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload)
		return nil, service.deleteIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	case "PostAPIUsersFileAssets":
		return service.createLegacyUserAsset(ctx, body)
	case "PostAPIWorkspacesBySlugFileAssets":
		p := payload.(*api.PostAPIWorkspacesBySlugFileAssetsPayload)
		return service.createLegacyWorkspaceAsset(ctx, p.Slug, body)
	case "GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments":
		p := payload.(*api.GetAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsPayload)
		return service.listLegacyIssueAttachments(ctx, p.Slug, p.ProjectID, p.IssueID)
	case "PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments":
		p := payload.(*api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsPayload)
		return service.createLegacyIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, body)
	case "DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk":
		p := payload.(*api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload)
		return nil, service.deleteLegacyIssueAttachment(ctx, p.Slug, p.ProjectID, p.IssueID, p.Pk)
	default:
		if err := service.invalidMethod(ctx, name, payload); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("assets lifecycle operation %s is not implemented", name)
	}
}

func (service *LifecycleService) getLegacyUserAssetByKey(ctx context.Context, key string) (*api.GetAPIUsersFileAssetsByAssetKeyResponseDataOrGetAPIUsersFileAssetsByAssetKeyResponseError, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	assets, err := service.assets.repository.UserAssetsByKey(ctx, userID, key)
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		result := api.NewGetAPIUsersFileAssetsByAssetKeyResponseDataOrGetAPIUsersFileAssetsByAssetKeyResponseErrorGetAPIUsersFileAssetsByAssetKeyResponseError(&api.GetAPIUsersFileAssetsByAssetKeyResponseError{Error: "Asset key does not exist", Status: false})
		return &result, nil
	}
	for index := range assets {
		assetKey, _ := assets[index]["asset"].(string)
		assets[index]["asset"] = legacyFileAssetURL(ctx, "/api/users/file-assets/"+key+"/", assetKey)
	}
	data, err := fileAssetSerializerResponses(assets)
	if err != nil {
		return nil, err
	}
	resultData := api.GetAPIUsersFileAssetsByAssetKeyResponseData{Data: data, Status: true}
	result := api.NewGetAPIUsersFileAssetsByAssetKeyResponseDataOrGetAPIUsersFileAssetsByAssetKeyResponseErrorGetAPIUsersFileAssetsByAssetKeyResponseData(&resultData)
	return &result, nil
}

func (service *LifecycleService) deleteLegacyUserAssetByKey(ctx context.Context, key string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	return sourceLookupError(service.assets.repository.UpdateAssetDeletedByKey(ctx, key, &userID, nil, userID, true), map[string]any{"error": "The required object does not exist."})
}

func (service *LifecycleService) getLegacyWorkspaceAssetByKey(ctx context.Context, rawWorkspaceID, key string) (*api.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseDataOrGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID, err := uuid.Parse(rawWorkspaceID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	member, err := service.assets.repository.WorkspaceMemberByID(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	assets, err := service.assets.repository.WorkspaceAssetsByKey(ctx, workspaceAssetKey(rawWorkspaceID, key), workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		result := api.NewGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseDataOrGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseErrorGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError(&api.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError{Error: "Asset key does not exist", Status: false})
		return &result, nil
	}
	visibleAssets := make([]map[string]any, 0, len(assets))
	for index := range assets {
		kind, _ := assets[index]["entity_type"].(string)
		id, _ := assets[index]["id"].(string)
		asset := FileAsset{ID: id, EntityType: kind}
		if raw, ok := assets[index]["draft_issue"].(string); ok {
			draft, parseErr := uuid.Parse(raw)
			if parseErr != nil {
				return nil, notFoundAssetError()
			}
			asset.DraftIssueID = &draft
		}
		if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
			var denied *djangoapi.HTTPError
			if errors.As(accessErr, &denied) && denied.Status == 404 {
				continue
			}
			return nil, accessErr
		}
		assetKey, _ := assets[index]["asset"].(string)
		assets[index]["asset"] = legacyFileAssetURL(ctx, "/api/workspaces/file-assets/"+rawWorkspaceID+"/"+key+"/", assetKey)
		visibleAssets = append(visibleAssets, assets[index])
	}
	if len(visibleAssets) == 0 {
		result := api.NewGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseDataOrGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseErrorGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError(&api.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError{Error: "Asset key does not exist", Status: false})
		return &result, nil
	}
	data, err := fileAssetSerializerResponses(visibleAssets)
	if err != nil {
		return nil, err
	}
	resultData := api.GetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseData{Data: data, Status: true}
	result := api.NewGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseDataOrGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseErrorGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseData(&resultData)
	return &result, nil
}

func legacyFileAssetURL(ctx context.Context, requestPath, assetKey string) string {
	scheme, host := RequestOrigin(ctx)
	if scheme == "" {
		scheme = "http"
	}
	if host == "" {
		host = "testserver"
	}
	base := &url.URL{Scheme: scheme, Host: host, Path: requestPath}
	return base.ResolveReference(&url.URL{Path: assetKey}).String()
}

func (service *LifecycleService) deleteLegacyWorkspaceAssetByKey(ctx context.Context, rawWorkspaceID, key string, deleted bool) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := uuid.Parse(rawWorkspaceID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	member, err := service.assets.repository.WorkspaceMemberByID(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if !member {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
	}
	return sourceLookupError(service.assets.repository.UpdateAssetDeletedByKey(ctx, workspaceAssetKey(rawWorkspaceID, key), nil, &workspaceID, userID, deleted), map[string]any{"error": "The required object does not exist."})
}

func workspaceAssetKey(rawWorkspaceID, key string) string {
	return rawWorkspaceID + "/" + key
}

func fileAssetSerializerResponses(assets []map[string]any) ([]*api.PlaneAppSerializersAssetFileassetserializerResponse, error) {
	encoded, err := json.Marshal(assets)
	if err != nil {
		return nil, err
	}
	var response []*api.PlaneAppSerializersAssetFileassetserializerResponse
	if err := json.Unmarshal(encoded, &response); err != nil {
		return nil, fmt.Errorf("decode FileAsset serializer response: %w", err)
	}
	return response, nil
}

func (service *LifecycleService) authorizeV2Workspace(ctx context.Context, userID uuid.UUID, slug string) error {
	allowed, err := service.assets.repository.WorkspacePermissionAllowed(ctx, userID, slug)
	if err != nil {
		return err
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}

func (service *LifecycleService) v2WorkspaceID(ctx context.Context, userID uuid.UUID, slug string) (uuid.UUID, error) {
	if err := service.authorizeV2Workspace(ctx, userID, slug); err != nil {
		return uuid.Nil, err
	}
	return service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
}

func (service *LifecycleService) projectAssetForMutation(ctx context.Context, slug, rawProjectID, rawAssetID string) (uuid.UUID, uuid.UUID, FileAsset, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, notFoundAssetError()
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	if !allowed {
		return uuid.Nil, uuid.Nil, FileAsset{}, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, notFoundAssetError()
	}
	asset, err := service.assets.repository.AssetByProjectAndID(ctx, workspaceID, projectID, assetID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if err := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	return workspaceID, projectID, asset, nil
}

func (service *LifecycleService) patchProjectAsset(ctx context.Context, slug, rawProjectID, rawAssetID string, body io.ReadCloser) error {
	workspaceID, projectID, asset, err := service.projectAssetForMutation(ctx, slug, rawProjectID, rawAssetID)
	if err != nil {
		return err
	}
	attributes, err := assetAttributes(ctx, body, asset.Attributes)
	if err != nil {
		return err
	}
	assetID, _ := uuid.Parse(rawAssetID)
	if privateDraftAsset(asset) {
		actor, assetUserIDErr := assetUserID(ctx)
		if assetUserIDErr != nil {
			return assetUserIDErr
		}
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, actor, assetID, "confirm", attributes, true)
	}
	actor, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	return service.assets.repository.UpdateProjectAsset(ctx, actor, workspaceID, projectID, assetID, attributes, len(asset.StorageMetadata) == 0)
}

func (service *LifecycleService) deleteProjectAsset(ctx context.Context, slug, rawProjectID, rawAssetID string) error {
	workspaceID, projectID, asset, err := service.projectAssetForMutation(ctx, slug, rawProjectID, rawAssetID)
	if err != nil {
		return err
	}
	assetID, _ := uuid.Parse(rawAssetID)
	if privateDraftAsset(asset) {
		actor, assetUserIDErr := assetUserID(ctx)
		if assetUserIDErr != nil {
			return assetUserIDErr
		}
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, actor, assetID, "delete", nil, false)
	}
	actor, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	return service.assets.repository.DeleteProjectAsset(ctx, actor, workspaceID, projectID, assetID)
}

func (service *LifecycleService) workspaceAssetForMutation(ctx context.Context, slug, rawAssetID string) (uuid.UUID, uuid.UUID, FileAsset, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	workspaceID, err := service.v2WorkspaceID(ctx, userID, slug)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, notFoundAssetError()
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if err := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); err != nil {
		return uuid.Nil, uuid.Nil, FileAsset{}, err
	}
	return userID, workspaceID, asset, nil
}

func (service *LifecycleService) patchWorkspaceAsset(ctx context.Context, slug, rawAssetID string, body io.ReadCloser) error {
	userID, workspaceID, asset, err := service.workspaceAssetForMutation(ctx, slug, rawAssetID)
	if err != nil {
		return err
	}
	attributes, err := assetAttributes(ctx, body, asset.Attributes)
	if err != nil {
		return err
	}
	assetID, _ := uuid.Parse(rawAssetID)
	if privateDraftAsset(asset) {
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, userID, assetID, "confirm", attributes, true)
	}
	workspaceHookApplied, err := service.assets.repository.UpdateWorkspaceAsset(ctx, userID, workspaceID, assetID, attributes, len(asset.StorageMetadata) == 0)
	if err != nil {
		return err
	}
	if asset.EntityType == "WORKSPACE_LOGO" {
		if !workspaceHookApplied {
			return nil
		}
		return service.invalidateWorkspaceAssetCaches(ctx, userID)
	}
	return nil
}

func (service *LifecycleService) deleteWorkspaceAsset(ctx context.Context, slug, rawAssetID string) error {
	userID, workspaceID, asset, err := service.workspaceAssetForMutation(ctx, slug, rawAssetID)
	if err != nil {
		return err
	}
	assetID, _ := uuid.Parse(rawAssetID)
	if privateDraftAsset(asset) {
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, userID, assetID, "delete", nil, false)
	}
	if err := service.assets.repository.DeleteWorkspaceAsset(ctx, userID, workspaceID, assetID); err != nil {
		return err
	}
	if asset.EntityType == "WORKSPACE_LOGO" {
		return service.invalidateWorkspaceAssetCaches(ctx, userID)
	}
	return nil
}

func (service *LifecycleService) invalidateWorkspaceAssetCaches(ctx context.Context, userID uuid.UUID) error {
	if service.assets.cache == nil {
		return fmt.Errorf("assets cache invalidator is required")
	}
	for _, key := range []string{":1:/api/workspaces/", ":1:/api/users/me/workspaces/:" + userID.String(), ":1:/api/instances/"} {
		if err := service.assets.cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("invalidate workspace logo cache %s: %w", key, err)
		}
	}
	return nil
}

func assetAttributes(ctx context.Context, body io.ReadCloser, current any) ([]byte, error) {
	value := current
	if body != nil {
		defer body.Close()
		contentType := RequestContentType(ctx)
		if contentType == "" {
			contentType = "application/json"
		}
		input, err := djangoinput.Parse(body, contentType)
		if err != nil {
			return nil, err
		}
		if next, present := input.Fields["attributes"]; present {
			value = next
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"attributes": []string{"Value must be valid JSON."}}}
	}
	return encoded, nil
}

func (service *LifecycleService) restoreWorkspaceAsset(ctx context.Context, slug, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := service.v2WorkspaceID(ctx, userID, slug)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndIDIncludingDeleted(ctx, workspaceID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return accessErr
	}
	if privateDraftAsset(asset) {
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, userID, assetID, "restore", nil, false)
	}
	return sourceLookupError(service.assets.repository.RestoreWorkspaceAsset(ctx, userID, workspaceID, assetID), map[string]any{"error": "The required object does not exist."})
}

func (service *LifecycleService) confirmLegacyWorkspaceAssetUpload(ctx context.Context, slug, rawAssetID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	workspaceID, err := service.assets.repository.AuthorizedWorkspaceID(ctx, userID, slug)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "Asset not found"})
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return accessErr
	}
	if asset.IsDeleted {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	uploaded := asset.IsUploaded
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
		if value, exists := input.Fields["is_uploaded"]; exists {
			parsed, ok := djangoBool(value)
			if !ok {
				return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"is_uploaded": []string{"Must be a valid boolean."}}}
			}
			uploaded = parsed
		}
	}
	if privateDraftAsset(asset) {
		return service.assets.repository.MutateDraftAsset(ctx, workspaceID, userID, assetID, "confirm", nil, uploaded)
	}
	return service.assets.repository.UpdateWorkspaceAssetUploaded(ctx, userID, workspaceID, assetID, uploaded, uploaded && len(asset.StorageMetadata) == 0)
}

func djangoBool(value any) (bool, bool) {
	switch value := value.(type) {
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "t", "yes", "on":
			return true, true
		case "0", "false", "f", "no", "off", "":
			return false, true
		default:
			return false, false
		}
	case float64:
		return value != 0, true
	default:
		return false, false
	}
}

func (service *LifecycleService) confirmLegacyUserAssetUpload(ctx context.Context, rawAssetID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.UserAssetByID(ctx, userID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	attributes := asset.Attributes
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
		if value, exists := input.Fields["attributes"]; exists {
			attributes = value
		}
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("encode FileAsset attributes: %w", err)
	}
	return service.assets.repository.UpdateLegacyUserAsset(ctx, userID, assetID, encoded, len(asset.StorageMetadata) == 0)
}

func (service *LifecycleService) deleteLegacyUserAsset(ctx context.Context, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	if _, err := service.assets.repository.UserAssetByID(ctx, userID, assetID); err != nil {
		return sourceLookupError(err, map[string]any{"error": "The requested resource does not exist."})
	}
	return service.assets.repository.DeleteUserAsset(ctx, userID, assetID)
}

func (service *LifecycleService) confirmUserAssetUpload(ctx context.Context, rawAssetID string, body io.ReadCloser) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.UserAssetByID(ctx, userID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	attributes := asset.Attributes
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
		if value, exists := input.Fields["attributes"]; exists {
			attributes = value
		}
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("encode FileAsset attributes: %w", err)
	}
	if err := service.assets.repository.UpdateUserAsset(ctx, userID, assetID, encoded, true, len(asset.StorageMetadata) == 0); err != nil {
		return err
	}
	if asset.EntityType == "USER_AVATAR" || asset.EntityType == "USER_COVER" {
		return service.invalidateUserAssetCaches(ctx, userID)
	}
	return nil
}

func (service *LifecycleService) deleteUserAsset(ctx context.Context, rawAssetID string) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	asset, err := service.assets.repository.UserAssetByID(ctx, userID, assetID)
	if err != nil {
		return sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if err := service.assets.repository.DeleteUserAsset(ctx, userID, assetID); err != nil {
		return err
	}
	if asset.EntityType == "USER_AVATAR" || asset.EntityType == "USER_COVER" {
		return service.invalidateUserAssetCaches(ctx, userID)
	}
	return nil
}

func (service *LifecycleService) invalidateUserAssetCaches(ctx context.Context, userID uuid.UUID) error {
	if service.assets.cache == nil {
		return fmt.Errorf("assets cache invalidator is required")
	}
	for _, path := range []string{"/api/users/me/", "/api/users/me/settings/"} {
		key := ":1:" + path + ":" + userID.String()
		if err := service.assets.cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("invalidate user profile cache %s: %w", path, err)
		}
	}
	return nil
}

func (service *LifecycleService) workspaceAssetRedirect(ctx context.Context, slug, rawAssetID string) (string, error) {
	return service.workspaceAssetLocation(ctx, slug, rawAssetID, "The required object does not exist.")
}

func (service *LifecycleService) workspaceAssetLocation(ctx context.Context, slug, rawAssetID, missingMessage string) (string, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return "", err
	}
	workspaceID, err := service.v2WorkspaceID(ctx, userID, slug)
	if err != nil {
		return "", err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return "", notFoundAssetError()
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		return "", sourceLookupError(err, map[string]any{"error": missingMessage})
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return "", accessErr
	}
	if !asset.IsUploaded {
		return "", notFoundAssetError()
	}
	filename, err := assetFilenameDefault(asset.Attributes, true)
	if err != nil {
		return "", err
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, "attachment", filename, scheme, host)
	if err != nil {
		return "", fmt.Errorf("presign workspace V2 FileAsset %s: %w", asset.ID, err)
	}
	return location, nil
}

func (service *LifecycleService) projectAssetRedirect(ctx context.Context, slug, rawProjectID, rawAssetID string) (string, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return "", err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return "", notFoundAssetError()
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return "", err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return "", notFoundAssetError()
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		return "", sourceLookupError(err, map[string]any{"error": "The required object does not exist."})
	}
	if asset.ProjectID == nil || *asset.ProjectID != projectID {
		return "", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return "", accessErr
	}
	if !asset.IsUploaded {
		return "", notFoundAssetError()
	}
	filename, err := assetFilenameDefault(asset.Attributes, true)
	if err != nil {
		return "", err
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, "attachment", filename, scheme, host)
	if err != nil {
		return "", fmt.Errorf("presign project V2 FileAsset %s: %w", asset.ID, err)
	}
	return location, nil
}

var invalidAssetMethods = map[string]string{
	"DeleteAPIAssetsV2UserAssets": "DELETE", "PatchAPIAssetsV2UserAssets": "PATCH",
	"PostAPIAssetsV2UserAssetsByAssetID": "POST", "DeleteAPIAssetsV2WorkspacesBySlug": "DELETE",
	"GetAPIAssetsV2WorkspacesBySlug": "GET", "PatchAPIAssetsV2WorkspacesBySlug": "PATCH",
	"DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectID": "DELETE",
	"GetAPIAssetsV2WorkspacesBySlugProjectsByProjectID":    "GET", "PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectID": "PATCH",
	"DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments": "DELETE",
	"PostAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPk":                         "POST", "PostAPIAssetsV2WorkspacesBySlugByAssetID": "POST",
	"DeleteAPIPublicAssetsV2AnchorByAnchor": "DELETE", "GetAPIPublicAssetsV2AnchorByAnchor": "GET",
	"PatchAPIPublicAssetsV2AnchorByAnchor": "PATCH", "PostAPIPublicAssetsV2AnchorByAnchorByPk": "POST",
	"DeleteAPIUsersFileAssets": "DELETE", "GetAPIUsersFileAssets": "GET",
	"PostAPIUsersFileAssetsByAssetKey": "POST", "PostAPIWorkspacesFileAssetsByWorkspaceIDByAssetKey": "POST",
	"DeleteAPIWorkspacesBySlugFileAssets": "DELETE", "GetAPIWorkspacesBySlugFileAssets": "GET",
	"DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments": "DELETE",
}

func (service *LifecycleService) invalidMethod(ctx context.Context, name string, payload any) error {
	method, invalid := invalidAssetMethods[name]
	if !invalid {
		return nil
	}
	// Source permission decorators run before the Python method signature is
	// invoked. Preserve their tenant checks before returning the intentional
	// routing correction for this invalid method/path pair.
	if name == "DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments" ||
		name == "DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments" {
		if err := service.authorizeWorkspaceFor405(ctx, name, payload); err != nil {
			return err
		}
	} else if strings.Contains(name, "ProjectsByProjectID") {
		if err := service.authorizeProjectFor405(ctx, payload); err != nil {
			return err
		}
	} else if strings.Contains(name, "WorkspacesBySlug") || strings.Contains(name, "WorkspacesFileAssets") {
		if err := service.authorizeWorkspaceFor405(ctx, name, payload); err != nil {
			return err
		}
	}
	return &djangoapi.HTTPError{Status: 405, Body: map[string]any{"detail": fmt.Sprintf("Method %q not allowed.", method)}}
}

func payloadString(payload any, field string) string {
	value := reflect.ValueOf(payload)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	member := value.FieldByName(field)
	if member.IsValid() && member.Kind() == reflect.String {
		return member.String()
	}
	return ""
}

func (service *LifecycleService) authorizeWorkspaceFor405(ctx context.Context, name string, payload any) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	slug := payloadString(payload, "Slug")
	if slug != "" {
		if name == "DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments" || name == "DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachments" {
			_, err = service.assets.repository.AuthorizedWorkspaceID(ctx, userID, slug)
		} else {
			_, err = service.v2WorkspaceID(ctx, userID, slug)
		}
		if err == nil {
			return nil
		}
		var httpErr *djangoapi.HTTPError
		if errors.As(err, &httpErr) && httpErr.Status == 403 {
			return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
		}
		return err
	}
	if rawWorkspaceID := payloadString(payload, "WorkspaceID"); rawWorkspaceID != "" {
		workspaceID, parseErr := uuid.Parse(rawWorkspaceID)
		if parseErr != nil {
			return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
		}
		member, checkErr := service.assets.repository.WorkspaceMemberByID(ctx, workspaceID, userID)
		if checkErr != nil {
			return checkErr
		}
		if !member {
			return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
		}
	}
	return nil
}

func (service *LifecycleService) authorizeProjectFor405(ctx context.Context, payload any) error {
	userID, err := assetUserID(ctx)
	if err != nil {
		return err
	}
	slug, rawProjectID := payloadString(payload, "Slug"), payloadString(payload, "ProjectID")
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return err
	}
	if !allowed {
		return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}

func (service *LifecycleService) duplicateWorkspaceAsset(ctx context.Context, slug, rawAssetID string, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetduplicateresponseResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeV2WorkspaceErr := service.authorizeV2Workspace(ctx, userID, slug); authorizeV2WorkspaceErr != nil {
		return nil, authorizeV2WorkspaceErr
	}
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	entityType := stringValue(request.EntityType)
	if !validEntityType(entityType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity type or entity id"}}
	}
	workspaceID, err := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	projectID := stringValue(request.ProjectID)
	var requestedProjectID uuid.UUID
	if projectID != "" {
		parsed, parseErr := uuid.Parse(projectID)
		if parseErr != nil {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project not found"}}
		}
		belongs, checkErr := service.assets.repository.ProjectBelongsToWorkspace(ctx, workspaceID, parsed)
		if checkErr != nil {
			return nil, checkErr
		}
		if !belongs {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project not found"}}
		}
		requestedProjectID = parsed
	}
	entityID := stringValue(request.EntityIDField)
	if entityType == "PROJECT_COVER" {
		if projectID != "" && projectID != entityID {
			return nil, projectCoverEntityMismatchError()
		}
		coverProjectID, valid := projectCoverEntityID(request.EntityIDField)
		if !valid {
			return nil, projectCoverEntityMismatchError()
		}
		if requestedProjectID != uuid.Nil && requestedProjectID != coverProjectID {
			return nil, projectCoverEntityMismatchError()
		}
		if requestedProjectID == uuid.Nil {
			belongs, checkErr := service.assets.repository.ProjectBelongsToWorkspace(ctx, workspaceID, coverProjectID)
			if checkErr != nil {
				return nil, checkErr
			}
			if !belongs {
				return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project not found"}}
			}
		}
		projectID = coverProjectID.String()
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	source, err := service.assets.repository.AssetForDuplicate(ctx, userID, workspaceID, assetID)
	if err != nil {
		return nil, sourceLookupError(err, map[string]any{"error": "Asset not found"})
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, source); accessErr != nil {
		return nil, accessErr
	}
	if !source.IsUploaded {
		return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}}
	}
	attributes, err := fileAssetAttributesObject(source.Attributes)
	if err != nil {
		return nil, err
	}
	name := sanitizeFilename(stringValue(attributes["name"]))
	if name == "" {
		name = "unnamed"
	}
	destinationKey := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	duplicateID := uuid.New()
	if err := service.assets.repository.CreateDuplicate(ctx, source, duplicateID, workspaceID, userID, destinationKey, projectID, entityType, entityID); err != nil {
		return nil, err
	}
	if err := service.assets.storage.Copy(ctx, source.Asset, destinationKey); err != nil {
		return nil, fmt.Errorf("copy workspace FileAsset %s: %w", source.ID, err)
	}
	if entityType == "DRAFT_ISSUE_DESCRIPTION" {
		if err := service.assets.repository.MutateDraftAsset(ctx, workspaceID, userID, duplicateID, "confirm", nil, true); err != nil {
			return nil, err
		}
	} else if err := service.assets.repository.MarkDuplicateUploaded(ctx, userID, workspaceID, duplicateID); err != nil {
		return nil, err
	}
	return &api.RestFrameworkSerializersFileassetduplicateresponseResponse{AssetID: duplicateID.String()}, nil
}

func validEntityType(value string) bool {
	switch value {
	case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION", "COMMENT_DESCRIPTION", "PAGE_DESCRIPTION", "USER_COVER", "USER_AVATAR", "WORKSPACE_LOGO", "PROJECT_COVER", "DRAFT_ISSUE_ATTACHMENT", "DRAFT_ISSUE_DESCRIPTION":
		return true
	default:
		return false
	}
}

func (service *LifecycleService) createV2WorkspaceUpload(ctx context.Context, slug string, body io.ReadCloser) (*uploadResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeV2WorkspaceErr := service.authorizeV2Workspace(ctx, userID, slug); authorizeV2WorkspaceErr != nil {
		return nil, authorizeV2WorkspaceErr
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
		return nil, err
	}
	entityType := stringValue(request.EntityType)
	if !validEntityType(entityType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity type.", "status": false}}
	}
	if entityType == "WORKSPACE_LOGO" {
		relatedWorkspaceID, lookupErr := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
		if lookupErr != nil {
			return nil, lookupErr
		}
		role, roleErr := service.assets.repository.WorkspaceRole(ctx, relatedWorkspaceID, userID)
		if roleErr != nil {
			return nil, roleErr
		}
		if role != 20 {
			return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "Only workspace admins can upload a workspace logo."}}
		}
	}
	if !allowedV2ProfileMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid file type. Only JPEG, PNG, WebP, JPG and GIF files are allowed.", "status": false}}
	}
	if size > service.assets.fileLimit {
		size = service.assets.fileLimit
	}
	workspaceID, err := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	assetID := uuid.New()
	key := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	attributes := uploadRequest{Name: name, Type: contentType}.Attributes(name, contentType, size)
	entityID := stringValue(request.EntityID)
	if err := service.assets.repository.CreateWorkspaceEntityUpload(ctx, assetID, userID, workspaceID, uuid.Nil, key, attributes, size, entityType, entityID, false); err != nil {
		return nil, fmt.Errorf("create workspace V2 FileAsset: %w", err)
	}
	assetURL := v2WorkspaceAssetURL(slug, assetID.String(), entityType, entityID)
	return service.assets.signUpload(ctx, assetID.String(), key, contentType, size, assetURL)
}

func (service *LifecycleService) createV2ProjectUpload(ctx context.Context, slug, rawProjectID string, body io.ReadCloser) (*uploadResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
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
		return nil, err
	}
	entityType := stringValue(request.EntityType)
	if !validEntityType(entityType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity type.", "status": false}}
	}
	if !allowedV2ProfileMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid file type. Only JPEG, PNG, WebP, JPG and GIF files are allowed.", "status": false}}
	}
	if size > service.assets.fileLimit {
		size = service.assets.fileLimit
	}
	workspaceID, err := service.assets.repository.WorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	assetID := uuid.New()
	key := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	attributes := uploadRequest{Name: name, Type: contentType}.Attributes(name, contentType, size)
	entityID := stringValue(request.EntityIdentifier)
	if entityType == "PROJECT_COVER" {
		if entityID != projectID.String() {
			return nil, projectCoverEntityMismatchError()
		}
		coverProjectID, valid := projectCoverEntityID(request.EntityIdentifier)
		if !valid || coverProjectID != projectID {
			return nil, projectCoverEntityMismatchError()
		}
		projectExists, checkErr := service.assets.repository.ProjectBelongsToWorkspace(ctx, workspaceID, projectID)
		if checkErr != nil {
			return nil, checkErr
		}
		if !projectExists {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Project not found", "status": false}}
		}
		entityID = coverProjectID.String()
	}
	if err := service.assets.repository.CreateWorkspaceEntityUpload(ctx, assetID, userID, workspaceID, projectID, key, attributes, size, entityType, entityID, true); err != nil {
		return nil, fmt.Errorf("create project V2 FileAsset: %w", err)
	}
	assetURL := v2ProjectAssetURL(slug, projectID, assetID.String(), entityType, entityID)
	return service.assets.signUpload(ctx, assetID.String(), key, contentType, size, assetURL)
}

func v2ProjectAssetURL(slug string, projectID uuid.UUID, assetID, entityType, entityID string) string {
	switch entityType {
	case "WORKSPACE_LOGO", "USER_AVATAR", "USER_COVER", "PROJECT_COVER":
		return "/api/assets/v2/static/" + assetID + "/"
	case "ISSUE_ATTACHMENT":
		return "/api/assets/v2/workspaces/" + slug + "/projects/" + projectID.String() + "/issues/" + nullableText(entityID, "None") + "/attachments/" + assetID + "/"
	case "ISSUE_DESCRIPTION", "COMMENT_DESCRIPTION", "PAGE_DESCRIPTION", "DRAFT_ISSUE_DESCRIPTION":
		return "/api/assets/v2/workspaces/" + slug + "/projects/" + projectID.String() + "/" + assetID + "/"
	default:
		return ""
	}
}

func v2WorkspaceAssetURL(slug, assetID, entityType, entityID string) string {
	switch entityType {
	case "WORKSPACE_LOGO", "USER_AVATAR", "USER_COVER", "PROJECT_COVER":
		return "/api/assets/v2/static/" + assetID + "/"
	case "ISSUE_ATTACHMENT":
		return "/api/assets/v2/workspaces/" + slug + "/projects/None/issues/" + nullableText(entityID, "None") + "/attachments/" + assetID + "/"
	case "DRAFT_ISSUE_DESCRIPTION":
		return "/api/assets/v2/workspaces/" + slug + "/" + assetID + "/"
	case "ISSUE_DESCRIPTION", "COMMENT_DESCRIPTION", "PAGE_DESCRIPTION":
		return "/api/assets/v2/workspaces/" + slug + "/projects/None/" + assetID + "/"
	default:
		return ""
	}
}

func (service *LifecycleService) workspaceDownload(ctx context.Context, slug, rawAssetID string) (string, error) {
	return service.workspaceAssetLocation(ctx, slug, rawAssetID, "The requested asset could not be found.")
}

func (service *LifecycleService) projectDownload(ctx context.Context, slug, rawProjectID, rawAssetID string) (string, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return "", err
	}
	projectID, err := uuid.Parse(rawProjectID)
	if err != nil {
		return "", notFoundAssetError()
	}
	allowed, err := service.assets.repository.ProjectPermissionAllowed(ctx, userID, slug, projectID)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	workspaceID, err := service.assets.repository.RelatedWorkspaceIDBySlug(ctx, slug)
	if err != nil {
		return "", err
	}
	assetID, err := uuid.Parse(rawAssetID)
	if err != nil {
		return "", notFoundAssetError()
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		return "", sourceLookupError(err, map[string]any{"error": "The requested asset could not be found."})
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return "", accessErr
	}
	if asset.ProjectID == nil || *asset.ProjectID != projectID || !asset.IsUploaded {
		return "", notFoundAssetError()
	}
	filename, err := assetFilenameDefault(asset.Attributes, true)
	if err != nil {
		return "", err
	}
	scheme, host := RequestOrigin(ctx)
	location, err := service.assets.storage.PresignGet(ctx, asset.Asset, "attachment", filename, scheme, host)
	if err != nil {
		return "", fmt.Errorf("presign project FileAsset %s download: %w", asset.ID, err)
	}
	return location, nil
}

func assetFilenameDefault(attributes any, generatedWhenMissing bool) (*string, error) {
	object, err := fileAssetAttributesObject(attributes)
	if err != nil {
		return nil, err
	}
	value, exists := object["name"]
	if !exists && generatedWhenMissing {
		value = compactUUID(uuid.NewString())
	}
	if value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, nil
	}
	return &text, nil
}

func notFoundAssetError() error {
	return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}}
}

// sourceLookupError converts only an actual missing-row result. Django's bare
// ORM .get() calls surface database failures as server errors, so repository
// failures other than the explicit 404 must retain their original status.
func sourceLookupError(err error, missingBody map[string]any) error {
	var httpErr *djangoapi.HTTPError
	if errors.As(err, &httpErr) && httpErr.Status == 404 {
		return &djangoapi.HTTPError{Status: 404, Body: missingBody}
	}
	return err
}

var _ api.Authorizer = (*LifecycleService)(nil)

func (service *LifecycleService) decodeUploadRequest(ctx context.Context, body io.ReadCloser) (uploadRequest, error) {
	if body == nil {
		return uploadRequest{}, nil
	}
	defer body.Close()
	contentType := RequestContentType(ctx)
	if contentType == "" {
		contentType = "application/json"
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return uploadRequest{}, err
	}
	return uploadRequestFromFields(input.Fields), nil
}

func (service *LifecycleService) PostAPIV1AssetsUserAssets(ctx context.Context, payload *api.PostAPIV1AssetsUserAssetsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetuploadresponseResponse, error) {
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1AssetsUserAssets", err)
	}
	result, err := service.assets.createUserUpload(ctx, request)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1AssetsUserAssets", err)
	}
	return result, nil
}

func (service *LifecycleService) PostAPIV1AssetsUserAssetsServer(ctx context.Context, payload *api.PostAPIV1AssetsUserAssetsServerPayload, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetuploadresponseResponse, error) {
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1AssetsUserAssetsServer", err)
	}
	result, err := service.assets.createUserUpload(ctx, request)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1AssetsUserAssetsServer", err)
	}
	return result, nil
}

func (service *LifecycleService) PostAPIV1WorkspacesBySlugAssets(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugAssetsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetuploadresponseResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1WorkspacesBySlugAssets", err)
	}
	workspaceID, err := service.assets.repository.AuthorizedWorkspaceID(ctx, userID, payload.Slug)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1WorkspacesBySlugAssets", err)
	}
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1WorkspacesBySlugAssets", err)
	}
	result, err := service.assets.createWorkspaceUpload(ctx, userID, workspaceID, payload.Slug, request)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIV1WorkspacesBySlugAssets", err)
	}
	return result, nil
}

func (service *LifecycleService) GetAPIV1WorkspacesBySlugAssetsByAssetID(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugAssetsByAssetIDPayload) (*api.GetAPIV1WorkspacesBySlugAssetsByAssetIDResult, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", err)
	}
	workspaceID, err := service.assets.repository.AuthorizedWorkspaceID(ctx, userID, payload.Slug)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", err)
	}
	assetID, err := uuid.Parse(payload.AssetID)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}})
	}
	asset, err := service.assets.repository.AssetByWorkspaceAndID(ctx, workspaceID, assetID)
	if err != nil {
		var httpErr *djangoapi.HTTPError
		if errors.As(err, &httpErr) && httpErr.Status == 404 {
			return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}})
		}
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", err)
	}
	if accessErr := service.authorizeAssetAttachment(ctx, workspaceID, userID, asset); accessErr != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", accessErr)
	}
	if asset.IsDeleted {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Asset not found"}})
	}
	if !asset.IsUploaded {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Asset not yet uploaded"}})
	}
	attributes, err := fileAssetAttributesObject(asset.Attributes)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", err)
	}
	contentType, _ := attributes["type"].(string)
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	disposition := "inline"
	switch contentType {
	case "image/svg+xml", "text/javascript", "application/javascript", "text/html", "application/xhtml+xml", "text/xml", "application/xml":
		disposition = "attachment"
	}
	var filename *string
	if value, present := attributes["name"]; present {
		if text, ok := value.(string); ok {
			filename = &text
		}
	}
	scheme, host := RequestOrigin(ctx)
	url, err := service.assets.storage.PresignGet(ctx, asset.Asset, disposition, filename, scheme, host)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", fmt.Errorf("presign generic FileAsset %s: %w", asset.ID, err))
	}
	name, _ := attributes["name"].(string)
	typeName, _ := attributes["type"].(string)
	return &api.GetAPIV1WorkspacesBySlugAssetsByAssetIDResult{AssetID: asset.ID, AssetName: name, AssetType: typeName, AssetURL: url}, nil
}

func (service *LifecycleService) GetAPIAssetsV2StaticByAssetID(ctx context.Context, payload *api.GetAPIAssetsV2StaticByAssetIDPayload) (*api.GetAPIAssetsV2StaticByAssetIDResult, error) {
	assetID, err := uuid.Parse(payload.AssetID)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}})
	}
	asset, err := service.assets.repository.AssetByID(ctx, assetID)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", sourceLookupError(err, map[string]any{"error": "The required object does not exist."}))
	}
	if privateDraftAsset(asset) || !asset.IsUploaded {
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested asset could not be found."}})
	}
	switch asset.EntityType {
	case "USER_AVATAR", "USER_COVER", "WORKSPACE_LOGO", "PROJECT_COVER":
	default:
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Invalid entity type.", "status": false}})
	}
	attributes, err := fileAssetAttributesObject(asset.Attributes)
	if err != nil {
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", err)
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
		return nil, mapAssetsLifecycleError("GetAPIAssetsV2StaticByAssetID", fmt.Errorf("presign static FileAsset %s: %w", asset.ID, err))
	}
	return &api.GetAPIAssetsV2StaticByAssetIDResult{Location: &location, ContentType: stringPointer("text/html; charset=utf-8")}, nil
}

func (service *LifecycleService) PostAPIAssetsV2UserAssets(ctx context.Context, payload *api.PostAPIAssetsV2UserAssetsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersFileassetuploadresponseResponse, error) {
	request, err := service.decodeUploadRequest(ctx, body)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIAssetsV2UserAssets", err)
	}
	result, err := service.assets.createUserUploadV2(ctx, request)
	if err != nil {
		return nil, mapAssetsLifecycleError("PostAPIAssetsV2UserAssets", err)
	}
	return result, nil
}

type uploadRequest struct {
	Name             any
	Type             any
	Size             any
	EntityType       any
	EntityID         any
	EntityIDField    any
	EntityIdentifier any
	ProjectID        any
	ExternalID       any
	ExternalSource   any
	SizePresent      bool
	TypePresent      bool
}

func uploadRequestFromFields(fields map[string]any) uploadRequest {
	entityID, exists := fields["entity_identifier"]
	if !exists {
		entityID = fields["entity_id"]
	}
	return uploadRequest{
		Name: fields["name"], Type: fields["type"], Size: fields["size"], EntityType: fields["entity_type"], EntityID: entityID,
		EntityIDField: fields["entity_id"], EntityIdentifier: fields["entity_identifier"],
		ProjectID: fields["project_id"], ExternalID: fields["external_id"], ExternalSource: fields["external_source"], SizePresent: hasField(fields, "size"), TypePresent: hasField(fields, "type"),
	}
}

func projectCoverEntityID(value any) (uuid.UUID, bool) {
	text, ok := value.(string)
	if !ok || text == "" {
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(text)
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}

func (request uploadRequest) Attributes(name, contentType string, size int64) []byte {
	data, _ := json.Marshal(map[string]any{"name": name, "type": contentType, "size": size})
	return data
}

func hasField(fields map[string]any, key string) bool { _, ok := fields[key]; return ok }

func stringPointer(value string) *string { return &value }

type uploadResponse = api.RestFrameworkSerializersFileassetuploadresponseResponse

// RequestOrigin lets the system HTTP mount preserve the request host used by
// Django's MinIO public-link fallback without coupling the domain to net/http.
type originKey struct{}

func WithRequestOrigin(ctx context.Context, scheme, host string) context.Context {
	return context.WithValue(ctx, originKey{}, requestOrigin{scheme: scheme, host: host})
}

type requestOrigin struct{ scheme, host string }

func RequestOrigin(ctx context.Context) (string, string) {
	origin, _ := ctx.Value(originKey{}).(requestOrigin)
	return origin.scheme, origin.host
}

type contentTypeKey struct{}

func WithRequestContentType(ctx context.Context, contentType string) context.Context {
	return context.WithValue(ctx, contentTypeKey{}, contentType)
}

func RequestContentType(ctx context.Context) string {
	value, _ := ctx.Value(contentTypeKey{}).(string)
	return value
}

var _ api.Service = (*LifecycleService)(nil)
