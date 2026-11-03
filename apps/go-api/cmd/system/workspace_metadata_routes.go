package main

import (
	"net/http"

	workspacemetadatasvr "github.com/CaliLuke/go-plane/gen/http/workspace_metadata/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceMetadataCompatibilityMiddleware(next http.Handler) http.Handler {
	return nativeProtocolCompatibilityMiddleware(func(string) string { return "GET, HEAD, OPTIONS" })(next)
}

func workspaceMetadataOptionsHandler(name string) http.Handler {
	return nativeProtocolOptionsHandler(workspaceMetadataCompatibilityMiddleware, name, false)
}

func mountWorkspaceMetadataRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.workspaceMetadata != nil {
		server := workspacemetadatasvr.New(pending.workspaceMetadata, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceMetadataCompatibilityMiddleware)
		workspacemetadatasvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/labels/", nativeProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.Labels))
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/states/", nativeProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.States))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/labels/", workspaceMetadataOptionsHandler("Workspace Labels Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/states/", workspaceMetadataOptionsHandler("Workspace States Endpoint"))
	}
}
