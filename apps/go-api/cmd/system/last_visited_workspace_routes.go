package main

import (
	"net/http"

	lastvisitedworkspacesvr "github.com/CaliLuke/go-plane/gen/http/last_visited_workspace/server"
	lastvisitedworkspace "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	loomhttp "github.com/CaliLuke/loom/http"
)

func lastVisitedWorkspaceCompatibilityMiddleware(next http.Handler) http.Handler {
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

func lastVisitedWorkspaceHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		lastVisitedWorkspaceCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func lastVisitedWorkspaceOptionsHandler() http.Handler {
	return lastVisitedWorkspaceCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "User Last Project With Workspace Endpoint",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func mountLastVisitedWorkspaceRoutes(lastVisitedEndpoints *lastvisitedworkspace.Endpoints, mux loomhttp.Muxer) {
	if lastVisitedEndpoints != nil {
		server := lastvisitedworkspacesvr.New(lastVisitedEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(lastVisitedWorkspaceCompatibilityMiddleware)
		lastvisitedworkspacesvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/last-visited-workspace/", lastVisitedWorkspaceHeadHandler(server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/last-visited-workspace/", lastVisitedWorkspaceOptionsHandler())
	}
}
