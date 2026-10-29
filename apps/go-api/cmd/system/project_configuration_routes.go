package main

import (
	"net/http"
	"strings"

	projectconfigurationsvr "github.com/CaliLuke/go-plane/projectconfiguration/gen/http/project_configuration/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectConfigurationRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions, stack func(http.Handler) http.Handler) {
	if pending.projectConfiguration != nil {
		server := projectconfigurationsvr.New(pending.projectConfiguration, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		apiStack := activityReadsMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server.Use(func(next http.Handler) http.Handler {
			sessionHandler, apiHandler := stack(next), apiStack(next)
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if strings.HasPrefix(request.URL.Path, "/api/v1/") {
					apiHandler.ServeHTTP(response, request)
					return
				}
				sessionHandler.ServeHTTP(response, request)
			})
		})
		projectconfigurationsvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/project-estimates/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.ListProjectEstimates)))
	}
}
