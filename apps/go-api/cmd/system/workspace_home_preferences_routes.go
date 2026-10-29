package main

import (
	"net/http"

	workspacehomepreferencessvr "github.com/CaliLuke/go-plane/gen/http/workspace_home_preferences/server"
	workspacehomepreferences "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceHomePreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, PATCH, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func mountWorkspaceHomePreferencesProtocolRoutes(mux loomhttp.Muxer, list, retrieve http.Handler) {
	collection := "/api/workspaces/{slug}/home-preferences/"
	detail := collection + "{key}/"
	mountProtocolHandler(mux, "HEAD", collection, workspaceHomePreferencesHeadHandler(list))
	mountProtocolHandler(mux, "HEAD", detail, workspaceHomePreferencesHeadHandler(retrieve))
	mountProtocolHandler(mux, "OPTIONS", collection, workspaceHomePreferencesOptionsHandler("Workspace Home Preference List"))
	mountProtocolHandler(mux, "OPTIONS", detail, workspaceHomePreferencesOptionsHandler("Workspace Home Preference"))
}

func workspaceHomePreferencesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceHomePreferencesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceHomePreferencesOptionsHandler(name string) http.Handler {
	return workspaceHomePreferencesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions":     map[string]any{"PATCH": map[string]any{}},
		})
	}))
}

func mountWorkspaceHomePreferencesRoutes(homePreferenceEndpoints *workspacehomepreferences.Endpoints, mux loomhttp.Muxer) {
	if homePreferenceEndpoints != nil {
		server := workspacehomepreferencessvr.New(homePreferenceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceHomePreferencesCompatibilityMiddleware)
		workspacehomepreferencessvr.Mount(mux, server)
		mountWorkspaceHomePreferencesProtocolRoutes(mux, server.List, server.Retrieve)
	}
}
