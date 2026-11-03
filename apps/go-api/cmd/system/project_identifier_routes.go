package main

import (
	"net/http"

	projectidentifierssvr "github.com/CaliLuke/go-plane/gen/http/project_identifiers/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func projectIdentifiersCompatibilityMiddleware(next http.Handler) http.Handler {
	return nativeProtocolCompatibilityMiddleware(func(string) string { return "GET, DELETE, HEAD, OPTIONS" })(next)
}

func mountProjectIdentifiersRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.projectIdentifiers != nil {
		server := projectidentifierssvr.New(pending.projectIdentifiers, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectIdentifiersCompatibilityMiddleware)
		projectidentifierssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/project-identifiers/", nativeProtocolHeadHandler(projectIdentifiersCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/project-identifiers/", nativeProtocolOptionsHandler(projectIdentifiersCompatibilityMiddleware, "Project Identifier Endpoint", false))
	}
}
