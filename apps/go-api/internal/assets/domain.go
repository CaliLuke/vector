package assets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"path"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("asset not found")

type assetRepository interface {
	CreateUserUpload(context.Context, uuid.UUID, uuid.UUID, string, []byte, int64, string) error
	AuthorizedWorkspaceID(context.Context, uuid.UUID, string) (uuid.UUID, error)
	WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error)
	RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error)
	WorkspaceMemberBySlug(context.Context, uuid.UUID, string) (bool, error)
	WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error)
	CreateWorkspaceUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string, string) error
	AssetByID(context.Context, uuid.UUID) (FileAsset, error)
	UserAssetByID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error)
	UserAssetsByKey(context.Context, uuid.UUID, string) ([]map[string]any, error)
	WorkspaceAssetsByKey(context.Context, string) ([]map[string]any, error)
	UpdateAssetDeletedByKey(context.Context, string, *uuid.UUID, bool) error
	UpdateLegacyUserAsset(context.Context, uuid.UUID, uuid.UUID, []byte) error
	UpdateUserAsset(context.Context, uuid.UUID, uuid.UUID, []byte, bool) error
	DeleteUserAsset(context.Context, uuid.UUID, uuid.UUID) error
	AssetByWorkspaceAndID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error)
	AssetByProjectAndID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, error)
	UpdateWorkspaceAssetUploaded(context.Context, uuid.UUID, uuid.UUID, bool) error
	UpdateWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []byte) (bool, error)
	DeleteWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	UpdateProjectAsset(context.Context, uuid.UUID, uuid.UUID, []byte) error
	DeleteProjectAsset(context.Context, uuid.UUID, uuid.UUID) error
	RestoreWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID) error
	AssetByWorkspaceAndIDIncludingDeleted(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error)
	WorkspaceAssetExists(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	AssetForDuplicate(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error)
	ProjectBelongsToWorkspace(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error)
	ProjectMemberActive(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error)
	ProjectMemberActiveByProject(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	WorkspaceMemberByID(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	WorkspaceRole(context.Context, uuid.UUID, uuid.UUID) (int, error)
	CreateWorkspaceEntityUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string, bool) error
	CreateDuplicate(context.Context, FileAsset, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string) error
	CreateIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string) error
	IssueAttachmentExternal(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (uuid.UUID, bool, error)
	IssuePermission(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (bool, bool, bool, error)
	IssueAttachments(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool) ([]map[string]any, error)
	IssueAttachmentsBySlug(context.Context, string, uuid.UUID, uuid.UUID, bool) ([]map[string]any, error)
	IssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, map[string]any, error)
	ProjectIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, map[string]any, error)
	UpdateIssueAttachmentUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	DeleteIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	UpdateAPIFileAssetUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	DeleteAPIFileAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	IssueAttachmentDeleteAllowed(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (bool, error)
	CreateLegacyFileAsset(context.Context, uuid.UUID, uuid.UUID, map[string]any, string) (map[string]any, error)
	LegacyRelationExists(context.Context, string, uuid.UUID) (bool, error)
	CreateLegacyIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, map[string]any, string) error
	PublicProjectBoard(context.Context, string, bool) (uuid.UUID, uuid.UUID, string, bool, error)
	PublicProjectAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool, bool) (FileAsset, error)
	PublicOwnedProjectAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, bool) (FileAsset, error)
	CreatePublicUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string) error
	UpdatePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []byte) error
	DeletePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	RestorePublicAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	PublicBulkAssetRows(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) ([]uuid.UUID, string, error)
	BulkAssociatePublicAssetComments(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) error
	BulkAssociateProjectAssets(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, []uuid.UUID) error
	DeleteLegacyIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
	MarkDuplicateUploaded(context.Context, uuid.UUID) error
}

type FileAsset struct {
	ID, Asset, EntityType string
	Attributes            any
	WorkspaceID           *uuid.UUID
	ProjectID             *uuid.UUID
	IssueID               *uuid.UUID
	UserID                *uuid.UUID
	IsUploaded            bool
	IsDeleted             bool
	Size                  float64
	StorageMetadata       map[string]any
}

func fileAssetAttributesObject(value any) (map[string]any, error) {
	attributes, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("FileAsset attributes are not an object")
	}
	return attributes, nil
}

func projectCoverEntityMismatchError() error {
	return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Entity project must match the requested project.", "status": false}}
}

type apiKeyValidator interface {
	Validate(context.Context, string) (apikey.Principal, error)
}

type taskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

type cacheInvalidator interface {
	Delete(context.Context, string) error
}

// Service owns the FileAsset persistence and S3 signing behavior used by the
// assets lifecycle routes. The generated service interface is implemented in
// service.go.
type Service struct {
	repository assetRepository
	storage    *Storage
	apiKeys    apiKeyValidator
	throttle   *apikey.Throttle
	fileLimit  int64
	tasks      taskPublisher
	cache      cacheInvalidator
	appBaseURL string
}

func NewService(repository assetRepository, storage *Storage, apiKeys apiKeyValidator, throttle *apikey.Throttle, fileLimit int64, effects ...AssetEffects) (*Service, error) {
	if repository == nil || storage == nil || apiKeys == nil || throttle == nil {
		return nil, fmt.Errorf("assets repository, storage, API key validator, and throttle are required")
	}
	if fileLimit < 1 {
		return nil, fmt.Errorf("assets file size limit must be positive")
	}
	if len(effects) != 1 {
		return nil, fmt.Errorf("an assets task publisher and cache invalidator are required")
	}
	if effects[0].Tasks == nil || effects[0].Cache == nil {
		return nil, fmt.Errorf("an assets task publisher and cache invalidator are required")
	}
	service := &Service{repository: repository, storage: storage, apiKeys: apiKeys, throttle: throttle, fileLimit: fileLimit, tasks: effects[0].Tasks, cache: effects[0].Cache, appBaseURL: effects[0].AppBaseURL}
	return service, nil
}

type AssetEffects struct {
	Tasks      taskPublisher
	Cache      cacheInvalidator
	AppBaseURL string
}

func (service *Service) taskOrigin(ctx context.Context) string {
	if service.appBaseURL != "" {
		return strings.TrimRight(service.appBaseURL, "/")
	}
	scheme, host := RequestOrigin(ctx)
	return scheme + "://" + host
}

func pythonJSONNumbers(value any) any {
	switch typed := value.(type) {
	case float64:
		return json.Number(pythoncompat.FloatString(typed))
	case map[string]any:
		for key, child := range typed {
			typed[key] = pythonJSONNumbers(child)
		}
		return typed
	case []any:
		for index, child := range typed {
			typed[index] = pythonJSONNumbers(child)
		}
		return typed
	default:
		return value
	}
}

func (service *Service) publishTask(ctx context.Context, task string, args []any, kwargs map[string]any) error {
	if service.tasks == nil {
		return errors.New("assets task publisher is required for this effect")
	}
	return service.tasks.Task(ctx, task, args, kwargs)
}

type assetUserKey struct{}

func (service *Service) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme == nil {
		return ctx, &djangoapi.HTTPError{Status: http.StatusUnauthorized, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	if scheme.Name == "PlaneSessionCookie" {
		userID, ok := systemapi.ReadServiceSessionUserID(ctx)
		parsed, err := uuid.Parse(userID)
		if !ok || err != nil {
			return ctx, &djangoapi.HTTPError{Status: http.StatusUnauthorized, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
		}
		return context.WithValue(ctx, assetUserKey{}, parsed), nil
	}
	if scheme.Name != "ApiKeyAuth" {
		return ctx, &djangoapi.HTTPError{Status: http.StatusUnauthorized, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	if key == "" {
		return ctx, &djangoapi.HTTPError{Status: http.StatusUnauthorized, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	principal, err := service.apiKeys.Validate(ctx, key)
	if err != nil {
		return ctx, &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"detail": "Given API token is not valid"}}
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	ctx = context.WithValue(ctx, assetUserKey{}, principal.UserID)
	// GenericAssetEndpoint uses WorkspaceUserPermission as a class-level
	// permission. DRF evaluates that permission after authentication but before
	// throttling, including for requests whose method is not implemented.
	// Keep the check here so denied requests do not consume the API-key rate
	// bucket and so generated methods and the protocol fallback share ordering.
	if resource, ok := systemapi.ProtocolResourceFromContext(ctx); ok && isGenericAssetResource(resource) {
		slug := resource.Slug
		if slug == "" {
			slug = resource.PathParameters["slug"]
		}
		member, permissionErr := service.repository.WorkspaceMemberBySlug(ctx, principal.UserID, slug)
		if permissionErr != nil {
			return ctx, fmt.Errorf("check generic asset workspace permission: %w", permissionErr)
		}
		if !member {
			return ctx, &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"detail": "You do not have permission to perform this action."}}
		}
	}
	rate, err := service.throttle.Allow(ctx, key)
	if err != nil {
		return ctx, fmt.Errorf("throttle asset API key: %w", err)
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return ctx, &djangoapi.HTTPError{Status: http.StatusTooManyRequests, Body: map[string]any{"detail": "Request was throttled."}}
	}
	return ctx, nil
}

func isGenericAssetResource(resource systemapi.SessionProtocolResource) bool {
	return resource.Pattern == "/api/v1/workspaces/{slug}/assets/" ||
		resource.Pattern == "/api/v1/workspaces/{slug}/assets/{asset_id}/"
}

func assetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(assetUserKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil, &djangoapi.HTTPError{Status: http.StatusUnauthorized, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	return userID, nil
}

func (service *Service) createUserUpload(ctx context.Context, body uploadRequest) (*uploadResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := sanitizeFilename(stringValue(body.Name))
	if name == "" {
		name = "unnamed"
	}
	contentType := stringValue(body.Type)
	if !body.TypePresent {
		contentType = "image/jpeg"
	}
	entityType := stringValue(body.EntityType)
	if entityType != "USER_AVATAR" && entityType != "USER_COVER" {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Invalid entity type.", "status": false}}
	}
	if !allowedProfileMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Invalid file type. Only JPEG and PNG files are allowed.", "status": false}}
	}
	size, err := body.IntSize(service.fileLimit)
	if err != nil {
		return nil, err
	}
	sizeLimit := size
	if sizeLimit > service.fileLimit {
		sizeLimit = service.fileLimit
	}
	assetID := uuid.New()
	key := compactUUID(uuid.NewString()) + "-" + name
	if err := service.repository.CreateUserUpload(ctx, assetID, userID, key, body.Attributes(name, contentType, sizeLimit), sizeLimit, entityType); err != nil {
		return nil, fmt.Errorf("create user FileAsset: %w", err)
	}
	return service.signUpload(ctx, assetID.String(), key, contentType, sizeLimit, "/api/assets/v2/static/"+assetID.String()+"/")
}

func (service *Service) createUserUploadV2(ctx context.Context, body uploadRequest) (*uploadResponse, error) {
	userID, err := assetUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := sanitizeFilename(stringValue(body.Name))
	if name == "" {
		name = "unnamed"
	}
	contentType := stringValue(body.Type)
	if !body.TypePresent {
		contentType = "image/jpeg"
	}
	size, err := body.IntSize(service.fileLimit)
	if err != nil {
		return nil, err
	}
	entityType := stringValue(body.EntityType)
	if entityType != "USER_AVATAR" && entityType != "USER_COVER" {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Invalid entity type.", "status": false}}
	}
	if !allowedV2ProfileMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Invalid file type. Only JPEG, PNG, WebP, JPG and GIF files are allowed.", "status": false}}
	}
	sizeLimit := size
	if sizeLimit > service.fileLimit {
		sizeLimit = service.fileLimit
	}
	assetID := uuid.New()
	key := compactUUID(uuid.NewString()) + "-" + name
	if err := service.repository.CreateUserUpload(ctx, assetID, userID, key, body.Attributes(name, contentType, sizeLimit), sizeLimit, entityType); err != nil {
		return nil, fmt.Errorf("create V2 user FileAsset: %w", err)
	}
	return service.signUpload(ctx, assetID.String(), key, contentType, sizeLimit, "/api/assets/v2/static/"+assetID.String()+"/")
}

func (service *Service) createWorkspaceUpload(ctx context.Context, userID, workspaceID uuid.UUID, slug string, body uploadRequest) (*uploadResponse, error) {
	name := sanitizeFilename(stringValue(body.Name))
	size, err := body.IntSize(service.fileLimit)
	if err != nil {
		return nil, err
	}
	if name == "" || size == 0 {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Name and size are required fields.", "status": false}}
	}
	contentType := stringValue(body.Type)
	if contentType == "" || !allowedAttachmentMIME(contentType) {
		return nil, &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "Invalid file type.", "status": false}}
	}
	sizeLimit := size
	if sizeLimit > service.fileLimit {
		sizeLimit = service.fileLimit
	}
	assetID := uuid.New()
	key := workspaceID.String() + "/" + compactUUID(uuid.NewString()) + "-" + name
	if err := service.repository.CreateWorkspaceUpload(ctx, assetID, userID, workspaceID, key, body.Attributes(name, contentType, sizeLimit), sizeLimit, stringValue(body.ProjectID), stringValue(body.ExternalID), stringValue(body.ExternalSource)); err != nil {
		return nil, fmt.Errorf("create workspace FileAsset: %w", err)
	}
	assetURL := fmt.Sprintf("/api/assets/v2/workspaces/%s/projects/%s/issues/None/attachments/%s/", slug, nullableText(stringValue(body.ProjectID), "None"), assetID)
	return service.signUpload(ctx, assetID.String(), key, contentType, sizeLimit, assetURL)
}

func (service *Service) signUpload(ctx context.Context, assetID, key, contentType string, size int64, assetURL string) (*uploadResponse, error) {
	scheme, host := RequestOrigin(ctx)
	url, fields, err := service.storage.PresignPost(ctx, key, contentType, size, scheme, host)
	if err != nil {
		return nil, fmt.Errorf("presign asset upload: %w", err)
	}
	data, err := json.Marshal(map[string]any{"url": url, "fields": fields})
	if err != nil {
		return nil, fmt.Errorf("encode presigned upload data: %w", err)
	}
	var nullableURL loom.Nullable[string]
	if assetURL != "" {
		nullableURL = loom.NullableValue(assetURL)
	}
	return &uploadResponse{AssetID: assetID, AssetURL: nullableURL, UploadData: map[string]loom.JSONValue{"url": dataField(data, "url"), "fields": dataField(data, "fields")}}, nil
}

func dataField(data []byte, key string) loom.JSONValue {
	var values map[string]json.RawMessage
	_ = json.Unmarshal(data, &values)
	return loom.JSONValue(values[key])
}

func nullableText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func compactUUID(value string) string {
	return strings.ReplaceAll(value, "-", "")
}

func sanitizeFilename(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	value = strings.ReplaceAll(value, "\\", "/")
	value = path.Base(value)
	value = strings.ReplaceAll(value, "..", "")
	value = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(value), "."))
	return strings.TrimSpace(value)
}

func allowedProfileMIME(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/jpg", "image/gif":
		return true
	default:
		return false
	}
}

func allowedV2ProfileMIME(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/jpg", "image/gif":
		return true
	default:
		return false
	}
}

func allowedAttachmentMIME(value string) bool {
	return attachmentMIMETypes[value]
}

var attachmentMIMETypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/svg+xml": true, "image/webp": true, "image/tiff": true, "image/bmp": true,
	"application/pdf": true, "application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain": true, "text/markdown": true, "application/rtf": true,
	"application/vnd.oasis.opendocument.spreadsheet": true, "application/vnd.oasis.opendocument.text": true,
	"application/vnd.oasis.opendocument.presentation": true, "application/vnd.oasis.opendocument.graphics": true,
	"application/vnd.visio": true, "image/x-portable-graymap": true, "image/x-portable-bitmap": true,
	"image/x-portable-pixmap": true, "application/vnd.oasis.opendocument.database": true,
	"audio/mpeg": true, "audio/wav": true, "audio/ogg": true, "audio/midi": true, "audio/x-midi": true,
	"audio/aac": true, "audio/flac": true, "audio/x-m4a": true,
	"video/mp4": true, "video/mpeg": true, "video/ogg": true, "video/webm": true, "video/quicktime": true,
	"video/x-msvideo": true, "video/x-ms-wmv": true,
	"application/zip": true, "application/x-rar": true, "application/x-rar-compressed": true,
	"application/x-tar": true, "application/gzip": true, "application/x-zip": true,
	"application/x-zip-compressed": true, "application/x-7z-compressed": true,
	"application/x-compressed": true, "application/x-compressed-tar": true,
	"application/x-compressed-tar-gz": true, "application/x-compressed-tar-bz2": true,
	"application/x-compressed-tar-zip": true, "application/x-compressed-tar-7z": true,
	"application/x-compressed-tar-rar": true, "model/gltf-binary": true, "model/gltf+json": true,
	"application/octet-stream": true, "font/ttf": true, "font/otf": true, "font/woff": true,
	"font/woff2": true, "text/css": true, "text/javascript": true, "application/json": true,
	"text/xml": true, "text/csv": true, "application/xml": true, "application/x-sql": true, "application/x-gzip": true,
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func (request uploadRequest) IntSize(defaultValue int64) (int64, error) {
	if !request.SizePresent {
		return defaultValue, nil
	}
	var number *big.Int
	switch value := request.Size.(type) {
	case bool:
		if value {
			return 1, nil
		}
		return 0, nil
	case json.Number:
		if strings.ContainsAny(string(value), ".eE") {
			parsed, err := strconv.ParseFloat(string(value), 64)
			if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || parsed < math.MinInt64 || parsed >= math.MaxInt64 {
				return 0, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
			}
			return int64(parsed), nil
		}
		parsed, ok := new(big.Int).SetString(string(value), 10)
		if !ok {
			return 0, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
		}
		number = parsed
	case string:
		parsed, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
		if !ok {
			return 0, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
		}
		number = parsed
	default:
		return 0, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	if !number.IsInt64() {
		if number.Sign() > 0 {
			return math.MaxInt64, nil
		}
		return math.MinInt64, nil
	}
	return number.Int64(), nil
}
