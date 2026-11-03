package main

import (
	"net/http"

	workspacesidebarpreferencessvr "github.com/CaliLuke/go-plane/gen/http/workspace_sidebar_preferences/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceSidebarPreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return nativeProtocolCompatibilityMiddleware(func(string) string { return "GET, PATCH, HEAD, OPTIONS" })(next)
}

func mountWorkspaceSidebarPreferencesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.workspaceSidebarPreferences != nil {
		server := workspacesidebarpreferencessvr.New(pending.workspaceSidebarPreferences, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceSidebarPreferencesCompatibilityMiddleware)
		workspacesidebarpreferencessvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/sidebar-preferences/", nativeProtocolHeadHandler(workspaceSidebarPreferencesCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/sidebar-preferences/", nativeProtocolOptionsHandler(workspaceSidebarPreferencesCompatibilityMiddleware, "Workspace User Preference Endpoint", true))
	}
}
