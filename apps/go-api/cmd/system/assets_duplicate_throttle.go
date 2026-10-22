package main

import (
	"math"
	"net/http"
	"strconv"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/google/uuid"
)

const duplicateAssetThrottlePath = "/api/assets/v2/workspaces/{slug}/duplicate-assets/{asset_id}/"

// assetDuplicateThrottleMiddleware runs inside session authentication and
// before the assets permission check, matching DuplicateAssetEndpoint's DRF
// throttle ordering. Only the duplicate endpoint uses this asset-ID scope.
func assetDuplicateThrottleMiddleware(throttle *apikey.Throttle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, matched := resolveReadServiceProtocol(request.URL.Path)
			if !matched || protocol.pattern != duplicateAssetThrottlePath {
				next.ServeHTTP(response, request)
				return
			}
			userID, authenticated := systemapi.ReadServiceSessionUserID(request.Context())
			if !authenticated {
				writeReadServiceJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			if _, err := uuid.Parse(userID); err != nil {
				writeReadServiceJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			if throttle == nil {
				writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{"detail": "A server error occurred."})
				return
			}
			resource := sessionProtocolResource(request, protocol)
			assetID, err := uuid.Parse(resource.PathParameters["asset_id"])
			if err != nil {
				next.ServeHTTP(response, request)
				return
			}
			rate, err := throttle.AllowCacheKey(request.Context(), ":1:throttle_asset_"+assetID.String(), "asset-ID")
			if err != nil {
				writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{"detail": "A server error occurred."})
				return
			}
			if !rate.Allowed {
				if rate.Wait > 0 {
					response.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(rate.Wait.Seconds()))))
				}
				writeReadServiceJSON(response, http.StatusTooManyRequests, map[string]any{
					"error_code": 5900, "error_message": "RATE_LIMIT_EXCEEDED",
				})
				return
			}
			timezone, _ := systemapi.ReadServiceSessionTimezone(request.Context())
			if err := djangosession.ValidateUserTimezone(timezone); err != nil {
				writeReadServicesSessionInspectionError(response, request, err)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}
