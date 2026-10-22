package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	api "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
)

// APIV1StickyProtocolAuthorizer runs the ViewSet's initial permission and throttle
// checks for unsupported methods, without performing a view's object lookup.
type APIV1StickyProtocolAuthorizer interface {
	AuthorizeAPIV1StickyProtocol(context.Context, string, string) (int, map[string]any)
}

func (s *apiV1StickiesService) AuthorizeAPIV1StickyProtocol(ctx context.Context, key, slug string) (int, map[string]any) {
	authorized, err := s.APIKeyAuth(ctx, key, nil)
	if err == nil {
		principal := authorized.Value(apiV1StickiesPrincipalKey{}).(apiV1StickiesPrincipal)
		_, lookupErr := s.repository.ResolveAPIWorkspace(authorized, slug, principal.UserID)
		switch {
		case errors.Is(lookupErr, stickystore.ErrWorkspaceAccessDenied):
			err = apiV1StickyError(http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."})
		case lookupErr != nil && !errors.Is(lookupErr, stickystore.ErrStickyWorkspaceDeleted):
			err = apiV1StickyStickyInternalError()
		default:
			// WorkspaceUserPermission checks membership, including on a soft-deleted
			// workspace. Unsupported methods never enter the view's workspace lookup.
			err = s.throttleRequest(authorized)
		}
	}
	if err == nil {
		return 0, nil
	}
	var response *api.APIV1StickyErrorResponse
	if errors.As(err, &response) {
		status := map[string]int{"status_401": 401, "status_403": 403, "status_429": 429, "status_500": 500}[response.Name]
		var body map[string]any
		if status != 0 && json.Unmarshal(response.Body, &body) == nil {
			return status, body
		}
	}
	return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
}
