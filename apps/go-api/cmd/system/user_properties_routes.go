package main

import (
	"net/http"

	userpropertiessvr "github.com/CaliLuke/go-plane/gen/http/user_properties/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userPropertiesCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, PATCH, HEAD, OPTIONS" })(next)
}

func mountUserPropertiesProtocolRoutes(mux loomhttp.Muxer, server *userpropertiessvr.Server) {
	routes := []struct {
		path string
		name string
		get  http.Handler
	}{
		{"/api/workspaces/{slug}/user-properties/", "Workspace User Properties Endpoint", server.WorkspaceGet},
		{"/api/workspaces/{slug}/projects/{project_id}/user-properties/", "Project User Display Property Endpoint", server.ProjectGet},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/user-properties/", "Cycle User Properties Endpoint", server.CycleGet},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/user-properties/", "Module User Properties Endpoint", server.ModuleGet},
	}
	for _, route := range routes {
		mountProtocolHandler(mux, http.MethodHead, route.path, djangoProtocolHeadHandler(userPropertiesCompatibilityMiddleware, route.get))
		mountProtocolHandler(mux, http.MethodOptions, route.path, djangoProtocolOptionsHandler(userPropertiesCompatibilityMiddleware, route.name, true))
	}
}

func mountUserPropertiesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.userProperties != nil {
		server := userpropertiessvr.New(pending.userProperties, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userPropertiesCompatibilityMiddleware)
		userpropertiessvr.Mount(mux, server)
		mountUserPropertiesProtocolRoutes(mux, server)
	}
}
