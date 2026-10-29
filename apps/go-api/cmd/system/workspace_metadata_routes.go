package main

import (
	"net/http"

	workspacemetadatasvr "github.com/CaliLuke/go-plane/gen/http/workspace_metadata/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceMetadataCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, HEAD, OPTIONS" })(next)
}

func workspaceMetadataOptionsHandler(name string) http.Handler {
	return djangoProtocolOptionsHandler(workspaceMetadataCompatibilityMiddleware, name, false)
}

func mountWorkspaceMetadataRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.workspaceMetadata != nil {
		server := workspacemetadatasvr.New(pending.workspaceMetadata, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceMetadataCompatibilityMiddleware)
		workspacemetadatasvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/labels/", djangoProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.Labels))
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/states/", djangoProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.States))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/labels/", workspaceMetadataOptionsHandler("Workspace Labels Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/states/", workspaceMetadataOptionsHandler("Workspace States Endpoint"))
	}
}
