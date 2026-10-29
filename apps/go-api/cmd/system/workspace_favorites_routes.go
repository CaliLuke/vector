package main

import (
	"context"
	"net/http"
	"strings"

	entityfavorites "github.com/CaliLuke/go-plane/gen/entity_favorites"
	entityfavoritessvr "github.com/CaliLuke/go-plane/gen/http/entity_favorites/server"
	workspacefavoritessvr "github.com/CaliLuke/go-plane/gen/http/workspace_favorites/server"
	workspacefavorites "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	loomhttp "github.com/CaliLuke/loom/http"
)

func workspaceFavoritesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		if strings.HasSuffix(request.URL.Path, "/group/") {
			headers.Set("Allow", "GET, HEAD, OPTIONS")
		} else {
			headers.Set("Allow", "GET, POST, PATCH, DELETE, HEAD, OPTIONS")
		}
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func entityFavoritesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Allow", entityFavoritesAllow(r.URL.Path))
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Referrer-Policy", "same-origin")
		mergeVary(h, "Cookie", "origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: w}, r)
	})
}

func entityFavoritesAllow(path string) string {
	if strings.HasSuffix(path, "user-favorite-projects/") || strings.HasSuffix(path, "user-favorite-cycles/") || strings.HasSuffix(path, "user-favorite-modules/") || strings.HasSuffix(path, "user-favorite-views/") {
		return "GET, POST, HEAD, OPTIONS"
	}
	if strings.Contains(path, "favorite-pages/") {
		return "POST, DELETE, OPTIONS"
	}
	return "DELETE, OPTIONS"
}

func mountEntityFavoritesProtocolRoutes(mux loomhttp.Muxer, server *entityfavoritessvr.Server) {
	collections := []struct {
		path string
		list http.Handler
	}{
		{"/api/workspaces/{slug}/user-favorite-projects/", server.ProjectList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-cycles/", server.CycleList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-modules/", server.ModuleList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-views/", server.ViewList},
	}
	for _, route := range collections {
		mountProtocolHandler(mux, http.MethodHead, route.path, entityFavoritesHeadHandler(route.list))
		mountProtocolHandler(mux, http.MethodOptions, route.path, entityFavoritesOptionsHandler())
		mountProtocolHandler(mux, http.MethodOptions, route.path+"{id}/", entityFavoritesOptionsHandler())
	}
	page := "/api/workspaces/{slug}/projects/{project_id}/favorite-pages/{page_id}/"
	mountProtocolHandler(mux, http.MethodOptions, page, entityFavoritesOptionsHandler())
}

func entityFavoritesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityFavoritesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{w}, r)
	})
}

func entityFavoritesOptionsHandler() http.Handler {
	return entityFavoritesCompatibilityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(w, http.StatusOK, map[string]any{"name": "Entity Favorite Endpoint", "description": "", "renders": []string{"application/json"}, "parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}})
	}))
}

func mountWorkspaceFavoritesProtocolRoutes(mux loomhttp.Muxer, list, retrieve, group http.Handler) {
	collection := "/api/workspaces/{slug}/user-favorites/"
	detail := collection + "{favorite_id}/"
	groupPath := detail + "group/"
	mountProtocolHandler(mux, http.MethodHead, collection, workspaceFavoritesHeadHandler(list))
	mountProtocolHandler(mux, http.MethodHead, detail, workspaceFavoritesHeadHandler(retrieve))
	mountProtocolHandler(mux, http.MethodHead, groupPath, workspaceFavoritesHeadHandler(group))
	mountProtocolHandler(mux, http.MethodOptions, collection, workspaceFavoritesOptionsHandler("Workspace Favorite List", "POST"))
	mountProtocolHandler(mux, http.MethodOptions, detail, workspaceFavoritesOptionsHandler("Workspace Favorite", "PATCH"))
	mountProtocolHandler(mux, http.MethodOptions, groupPath, workspaceFavoritesOptionsHandler("Workspace Favorite Group", ""))
}

func workspaceFavoritesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceFavoritesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceFavoritesOptionsHandler(name, action string) http.Handler {
	return workspaceFavoritesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		body := map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		}
		if action != "" {
			body["actions"] = map[string]any{action: map[string]any{}}
		}
		writeStickyJSON(response, http.StatusOK, body)
	}))
}

func mountWorkspaceFavoritesRoutes(mux loomhttp.Muxer, favoriteEndpoint *workspacefavorites.Endpoints) {
	if favoriteEndpoint != nil {
		server := workspacefavoritessvr.New(favoriteEndpoint, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceFavoritesCompatibilityMiddleware)
		workspacefavoritessvr.Mount(mux, server)
		mountWorkspaceFavoritesProtocolRoutes(mux, server.List, server.Retrieve, server.Group)
	}
}

func mountEntityFavoritesRoutes(ctx context.Context, entityEndpoint *entityfavorites.Endpoints, mux loomhttp.Muxer) {
	if entityEndpoint != nil {
		server := entityfavoritessvr.New(entityEndpoint, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(entityFavoritesCompatibilityMiddleware)
		entityfavoritessvr.Mount(mux, server)
		mountEntityFavoritesProtocolRoutes(mux, server)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "entity_favorites", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
