package main

import (
	"net/http"
	"strings"

	apiv1stickiessvr "github.com/CaliLuke/go-plane/apiv1stickies/gen/http/api_v1_stickies/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

// API v1 Sticky query values remain raw until the service validates them. The
// shared activity stack normalizes invalid integer filters for older APIs.
func apiV1StickyActivityMiddlewareStack(
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = apiV1StickyProtocolHeadersMiddleware(next)
		next = readServicesQueryPresenceMiddleware(next)
		next = activityReadsCompatibilityMiddleware(options)(next)
		next = activityReadsSessionMiddleware(sessions, cookie)(next)
		next = readServicesContentNegotiationMiddleware(next)
		next = activityReadsAuditFallbackMiddleware(options)(next)
		next = activityReadsRequestSizeMiddleware(options.bodySizeLimit)(next)
		next = readServicesGzipMiddleware(next)
		return next
	}
}

func apiV1StickyProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		segments := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
		allow := "GET, POST, HEAD, OPTIONS"
		if len(segments) == 6 {
			allow = "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
		}
		setDjangoProtocolCompatibilityHeaders(response.Header(), allow)
		next.ServeHTTP(response, request)
	})
}

func mountAPIV1StickiesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.apiV1Stickies != nil {
		stickies := apiv1stickiessvr.New(pending.apiV1Stickies, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		stickyStack := apiV1StickyActivityMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
		stickies.Use(stickyStack)
		apiv1stickiessvr.Mount(mux, stickies)
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies.json", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies.json/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}.json", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
		loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}.json/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
	}
}
