package main

import (
	"net/http"
	"strings"

	standardviewssvr "github.com/CaliLuke/go-plane/gen/http/standard_views/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func standardViewsCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(path string) string {
		if strings.HasSuffix(strings.TrimSuffix(path, "/"), "/views") {
			return "GET, POST, HEAD, OPTIONS"
		}
		return "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
	})(next)
}

func mountStandardViewsProtocolRoutes(mux loomhttp.Muxer, server *standardviewssvr.Server) {
	routes := []struct {
		path string
		name string
		get  http.Handler
	}{
		{"/api/workspaces/{slug}/projects/{project_id}/views/", "Issue View Endpoint", server.ListProject},
		{"/api/workspaces/{slug}/projects/{project_id}/views/{pk}/", "Issue View Endpoint", server.RetrieveProject},
		{"/api/workspaces/{slug}/views/", "Workspace View Endpoint", server.ListWorkspace},
		{"/api/workspaces/{slug}/views/{pk}/", "Workspace View Endpoint", server.RetrieveWorkspace},
	}
	for _, route := range routes {
		mountProtocolHandler(mux, http.MethodHead, route.path, djangoProtocolHeadHandler(standardViewsCompatibilityMiddleware, route.get))
		mountProtocolHandler(mux, http.MethodOptions, route.path, djangoProtocolOptionsHandler(standardViewsCompatibilityMiddleware, route.name, strings.Contains(route.path, "{pk}")))
	}
}

func mountStandardViewsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.standardViews != nil {
		server := standardviewssvr.New(pending.standardViews, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(standardViewsCompatibilityMiddleware)
		standardviewssvr.Mount(mux, server)
		mountStandardViewsProtocolRoutes(mux, server)
	}
}
