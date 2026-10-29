package main

import (
	"context"
	"net/http"

	useraccountssvr "github.com/CaliLuke/go-plane/gen/http/user_accounts/server"
	useraccounts "github.com/CaliLuke/go-plane/gen/user_accounts"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userAccountsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserAccountsHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userAccountsHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userAccountsCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userAccountsOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserAccountsHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	})
}

func setUserAccountsHeaders(headers http.Header) {
	headers.Set("Allow", "GET, DELETE, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func mountUserAccountsRoutes(ctx context.Context, userAccountsEndpoints *useraccounts.Endpoints, mux loomhttp.Muxer) {
	if userAccountsEndpoints != nil {
		server := useraccountssvr.New(userAccountsEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userAccountsCompatibilityMiddleware)
		useraccountssvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/accounts/", userAccountsHeadHandler(server.List))
		mountProtocolHandler(mux, "HEAD", "/api/users/me/accounts/{pk}/", userAccountsHeadHandler(server.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/accounts/", userAccountsOptionsHandler("Account Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/accounts/{pk}/", userAccountsOptionsHandler("Account Endpoint"))
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "user_accounts", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
