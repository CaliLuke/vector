package main

import (
	"context"
	"fmt"
	"net/http"

	workspacerecentvisitssvr "github.com/CaliLuke/go-plane/gen/http/workspace_recent_visits/server"
	workspacerecentvisits "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceRecentVisitsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func mountWorkspaceRecentVisitsProtocolRoutes(mux loomhttp.Muxer, list http.Handler) {
	collection := "/api/workspaces/{slug}/recent-visits/"
	mountProtocolHandler(mux, http.MethodHead, collection, workspaceRecentVisitsHeadHandler(list))
	mountProtocolHandler(mux, http.MethodOptions, collection, workspaceRecentVisitsOptionsHandler())
	mountProtocolHandler(mux, http.MethodPost, collection, workspaceRecentVisitsMethodNotAllowed())
}

func workspaceRecentVisitsHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceRecentVisitsCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceRecentVisitsOptionsHandler() http.Handler {
	return workspaceRecentVisitsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "Workspace Recent Visit List",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func workspaceRecentVisitsMethodNotAllowed() http.Handler {
	return workspaceRecentVisitsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{"detail": fmt.Sprintf("Method %q not allowed.", request.Method)})
	}))
}

func mountWorkspaceRecentVisitsRoutes(ctx context.Context, recentVisitEndpoints *workspacerecentvisits.Endpoints, mux loomhttp.Muxer) {
	if recentVisitEndpoints != nil {
		server := workspacerecentvisitssvr.New(recentVisitEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceRecentVisitsCompatibilityMiddleware)
		workspacerecentvisitssvr.Mount(mux, server)
		mountWorkspaceRecentVisitsProtocolRoutes(mux, server.List)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "workspace_recent_visits", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
