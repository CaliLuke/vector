package main

import (
	"net/http"

	workspacemembermesvr "github.com/CaliLuke/go-plane/gen/http/workspace_member_me/server"
	workspacememberme "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceMemberMeCompatibilityMiddleware(next http.Handler) http.Handler {
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

func workspaceMemberMeHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceMemberMeCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceMemberMeOptionsHandler() http.Handler {
	return workspaceMemberMeCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "Workspace Member User Endpoint",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func mountWorkspaceMemberMeRoutes(memberMeEndpoints *workspacememberme.Endpoints, mux loomhttp.Muxer) {
	if memberMeEndpoints != nil {
		server := workspacemembermesvr.New(memberMeEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceMemberMeCompatibilityMiddleware)
		workspacemembermesvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/workspace-members/me/", workspaceMemberMeHeadHandler(server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/workspace-members/me/", workspaceMemberMeOptionsHandler())
	}
}
