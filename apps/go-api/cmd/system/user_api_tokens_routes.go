package main

import (
	"context"
	"net/http"

	userapitokenssvr "github.com/CaliLuke/go-plane/gen/http/user_api_tokens/server"
	userapitokens "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userAPITokensCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserAPITokensHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userAPITokensHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userAPITokensCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userAPITokensOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserAPITokensHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	})
}

func setUserAPITokensHeaders(headers http.Header) {
	headers.Set("Allow", "GET, POST, PATCH, DELETE, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func mountUserAPITokensRoutes(ctx context.Context, userAPITokensEndpoints *userapitokens.Endpoints, mux loomhttp.Muxer) {
	if userAPITokensEndpoints != nil {
		server := userapitokenssvr.New(userAPITokensEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userAPITokensCompatibilityMiddleware)
		userapitokenssvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/api-tokens/", userAPITokensHeadHandler(server.List))
		mountProtocolHandler(mux, "HEAD", "/api/users/api-tokens/{pk}/", userAPITokensHeadHandler(server.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/api-tokens/", userAPITokensOptionsHandler("API Token Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/api-tokens/{pk}/", userAPITokensOptionsHandler("API Token Endpoint"))
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "user_api_tokens", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
