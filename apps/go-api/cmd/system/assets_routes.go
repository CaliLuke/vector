package main

import (
	"context"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	assetslifecyclesvr "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/plane_internal_django_migration_api/server"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

// assetLifecycleProtocolHeaders applies the Django REST framework response
// headers at the mounted assets boundary. Allow comes from source protocol
// observations and documented method repairs, including on application errors.
func assetLifecycleProtocolHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		allow := assetLifecycleAllowedMethods(request.URL.Path)
		if allow != "" {
			response.Header().Set("Allow", allow)
		}
		response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		response.Header().Set("Referrer-Policy", "same-origin")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		mergeVary(response.Header(), "Cookie", "origin")
		next.ServeHTTP(response, request)
	})
}

func assetLifecycleAllowedMethods(path string) string {
	for pattern, protocol := range assetsProtocolMetadata {
		if assetLifecyclePathMatches(pattern, path) {
			return protocol.Allow
		}
	}
	return ""
}

func assetLifecyclePathMatches(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if pathParts[index] == "" {
				return false
			}
			parameter := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			if strings.HasSuffix(parameter, "_id") || parameter == "id" || parameter == "pk" {
				if _, err := uuid.Parse(pathParts[index]); err != nil {
					return false
				}
			}
			continue
		}
		if part != pathParts[index] {
			return false
		}
	}
	return true
}

func assetRequestMetadataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		scheme := "http"
		if request.TLS != nil {
			scheme = "https"
		}
		if forwarded := request.Header.Get("X-Forwarded-Proto"); forwarded != "" {
			scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
		ctx := assetsstore.WithRequestOrigin(request.Context(), scheme, request.Host)
		ctx = assetsstore.WithRequestContentType(ctx, request.Header.Get("Content-Type"))
		if protocol, ok := resolveReadServiceProtocol(request.URL.Path); ok {
			ctx = systemapi.WithProtocolResource(ctx, sessionProtocolResource(request, protocol))
		}
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// assetLifecycleAPIV1Middleware gives the legacy V1 presign operations the
// same API activity, session, body-limit, and response-rate boundary as the
// other V1 replacements. V2 asset and public routes retain their own generated
// authentication shape and do not pass through the V1 compatibility policy.
func assetLifecycleAPIV1Middleware(apiV1 func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		apiHandler := apiV1(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				apiHandler.ServeHTTP(response, request)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func mountAssetsLifecycleRoutes(ctx context.Context, pending pendingReplacementEndpoints, mux loomhttp.Muxer, assetsMountedMethods map[assetsLifecycleMountedRoute]struct{}, activityOptions *apiV1CompatibilityOptions) {
	if pending.assetsLifecycle != nil {
		server := assetslifecyclesvr.New(pending.assetsLifecycle, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(assetLifecycleProtocolHeaders)
		server.Use(assetRequestMetadataMiddleware)
		if activityOptions != nil {
			apiStack := assetLifecycleAPIV1Middleware(activityReadsMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
			sessionStack := sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
			server.Use(func(next http.Handler) http.Handler {
				api := apiStack(next)
				session := sessionStack(assetDuplicateThrottleMiddleware(pending.assetsDuplicateThrottle)(assetsSessionPermissionMiddleware(pending.assetsSessionMetadata)(draftAssetCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins)(next))))
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					if strings.HasPrefix(request.URL.Path, "/api/v1/") {
						api.ServeHTTP(response, request)
					} else {
						session.ServeHTTP(response, request)
					}
				})
			})
		}
		assetslifecyclesvr.Mount(mux, server)
		mountAssetsLifecycleDerivedHeads(mux, server)
		for route := range assetsLifecycleMountedRoutes(server) {
			assetsMountedMethods[route] = struct{}{}
		}
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "assets_lifecycle", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
