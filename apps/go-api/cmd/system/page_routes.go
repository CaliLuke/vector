package main

import (
	"net/http"

	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func pageLifecycleWriteTransportMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		bounded := activityReadsRequestSizeMiddleware(options.bodySizeLimit)(activityReadsAuditFallbackMiddleware(options)(next))
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, ok := resolveReadServiceProtocol(request.URL.Path)
			if !ok || !protocol.pageLifecycle || protocol.methodAllow[request.Method] == "" || request.Method == http.MethodGet || request.Method == http.MethodHead {
				next.ServeHTTP(response, request)
				return
			}
			bounded.ServeHTTP(response, request)
		})
	}
}

func mountPageReadsProtocolRoutes(mux loomhttp.Muxer, server *pagereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/projects/{project_id}/pages-summary/", server.Summary},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", server.Detail},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", server.Description},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/", server.VersionList},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/", server.VersionDetail},
	}, corsPolicy)
}

func mountPageReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.pageReads != nil {
		server := pagereadssvr.New(pending.pageReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		if activityOptions != nil {
			options = *activityOptions
		}
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(sessionServicePrincipalLifecycle(pending.readServicesSession), pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		// Keep write transport boundaries outside middleware that can return
		// early, while leaving gzip outermost so audit records the plain response.
		server.Use(pageLifecycleWriteTransportMiddleware(options))
		server.Use(readServicesGzipMiddleware)
		pagereadssvr.Mount(mux, server)
		mountPageReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
