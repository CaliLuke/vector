package assets

import (
	"context"
	"errors"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

// AuthorizeSessionProtocol preserves source permission precedence for the
// production shared fallback that answers native OPTIONS and unmounted
// methods. Generated methods still use APIKeyAuth and operation-specific
// errors; this method only answers the permission check before fallback.
func (service *LifecycleService) AuthorizeSessionProtocol(ctx context.Context, _ string) (int, map[string]any) {
	return service.AuthorizeSessionResourceProtocol(ctx, "", systemapi.SessionProtocolResource{})
}

func (service *LifecycleService) AuthorizeSessionResourceProtocol(ctx context.Context, _ string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	pattern := resource.Pattern
	if strings.HasPrefix(pattern, "/api/assets/v2/static/") || strings.HasPrefix(pattern, "/api/public/assets/v2/anchor/") && resource.Method == http.MethodGet {
		return 0, nil
	}
	userID, err := protocolSessionUserID(ctx)
	if err != nil {
		return 401, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	if pattern == "/api/workspaces/{slug}/file-assets/" ||
		pattern == "/api/workspaces/file-assets/{workspace_id}/{asset_key}/" ||
		pattern == "/api/workspaces/file-assets/{workspace_id}/{asset_key}/restore/" {
		if pattern == "/api/workspaces/{slug}/file-assets/" {
			member, err := service.assets.repository.WorkspaceMemberBySlug(ctx, userID, resource.Slug)
			if err != nil {
				return protocolHTTPError(err)
			}
			if !member {
				return 403, map[string]any{"detail": "You do not have permission to perform this action."}
			}
		} else {
			rawID := resource.PathParameters["workspace_id"]
			id, err := uuid.Parse(rawID)
			if err != nil {
				return 403, map[string]any{"detail": "You do not have permission to perform this action."}
			}
			member, err := service.assets.repository.WorkspaceMemberByID(ctx, id, userID)
			if err != nil {
				return 500, map[string]any{"detail": "A server error occurred."}
			}
			if !member {
				return 403, map[string]any{"detail": "You do not have permission to perform this action."}
			}
		}
	}
	return 0, nil
}

// AuthorizeAssetsAPIProtocol is used by the shared compatibility boundary for
// native API protocol methods (HEAD, OPTIONS, and unsupported verbs). Generated
// API-key authentication has already validated the credential; this hook
// preserves the class-level workspace permission that DRF runs before method
// dispatch. GenericAssetEndpoint's permission applies to its whole resource.
func (service *LifecycleService) AuthorizeAssetsAPIProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	// This fallback runs before Loom's generated APIKeyAuth hook, so it owns the
	// complete DRF authentication -> permission -> throttle sequence. Reuse the
	// same service method as generated operations to keep effects and ordering
	// identical, including GenericAssetEndpoint's pre-throttle membership check.
	ctx = systemapi.WithProtocolResource(ctx, resource)
	_, err := service.assets.APIKeyAuth(ctx, key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		return protocolHTTPError(err)
	}
	return 0, nil
}

func protocolSessionUserID(ctx context.Context) (uuid.UUID, error) {
	value, ok := systemapi.ReadServiceSessionUserID(ctx)
	if !ok {
		return uuid.Nil, http.ErrNoCookie
	}
	return uuid.Parse(value)
}

func protocolHTTPError(err error) (int, map[string]any) {
	var httpErr *djangoapi.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status, httpErr.Body
	}
	return 500, map[string]any{"detail": "A server error occurred."}
}
