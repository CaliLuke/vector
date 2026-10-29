package main

import (
	"context"
	"net/http"
	"strings"

	workspacequicklinkssvr "github.com/CaliLuke/go-plane/gen/http/workspace_quick_links/server"
	workspacethemesvr "github.com/CaliLuke/go-plane/gen/http/workspace_themes/server"
	workspacequicklinks "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacethemes "github.com/CaliLuke/go-plane/gen/workspace_themes"
	loomhttp "github.com/CaliLuke/loom/http"
)

type workspaceContentResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (writer *workspaceContentResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.Header().Del("loom-error")
	if status == http.StatusOK || status == http.StatusCreated {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	if status == http.StatusUnauthorized {
		mergeVary(writer.Header(), "Cookie")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *workspaceContentResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func workspaceContentCompatibilityMiddleware(resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			if strings.HasSuffix(strings.TrimSuffix(request.URL.Path, "/"), "/"+resource) {
				headers.Set("Allow", "GET, POST, HEAD, OPTIONS")
			} else {
				headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
			}
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			mergeVary(headers, "origin")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
		})
	}
}

func mountWorkspaceContentProtocolRoutes(mux loomhttp.Muxer, resource, name string, list, retrieve http.Handler) {
	collection := "/api/workspaces/{slug}/" + resource + "/"
	detail := collection + "{pk}/"
	mountProtocolHandler(mux, "HEAD", collection, workspaceContentHeadHandler(resource, list))
	mountProtocolHandler(mux, "HEAD", detail, workspaceContentHeadHandler(resource, retrieve))
	mountProtocolHandler(mux, "OPTIONS", collection, workspaceContentOptionsHandler(resource, name+" List", true))
	mountProtocolHandler(mux, "OPTIONS", detail, workspaceContentOptionsHandler(resource, name, false))
}

func workspaceContentHeadHandler(resource string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceContentCompatibilityMiddleware(resource)(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceContentOptionsHandler(resource, name string, collection bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceContentCompatibilityMiddleware(resource)(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			method := "PATCH"
			if collection {
				method = "POST"
			}
			writeStickyJSON(response, http.StatusOK, map[string]any{
				"name":        name,
				"description": "",
				"renders":     []string{"application/json"},
				"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
				"actions":     map[string]any{method: map[string]any{}},
			})
		})).ServeHTTP(response, request)
	})
}

func mountWorkspaceQuickLinksRoutes(ctx context.Context, quickLinkEndpoints *workspacequicklinks.Endpoints, mux loomhttp.Muxer) {
	if quickLinkEndpoints != nil {
		server := workspacequicklinkssvr.New(quickLinkEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceContentCompatibilityMiddleware("quick-links"))
		workspacequicklinkssvr.Mount(mux, server)
		mountWorkspaceContentProtocolRoutes(mux, "quick-links", "Workspace Quick Link", server.List, server.Retrieve)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "workspace_quick_links", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}

func mountWorkspaceThemesRoutes(ctx context.Context, themeEndpoints *workspacethemes.Endpoints, mux loomhttp.Muxer) {
	if themeEndpoints != nil {
		server := workspacethemesvr.New(themeEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceContentCompatibilityMiddleware("workspace-themes"))
		workspacethemesvr.Mount(mux, server)
		mountWorkspaceContentProtocolRoutes(mux, "workspace-themes", "Workspace Theme", server.List, server.Retrieve)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "workspace_themes", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
