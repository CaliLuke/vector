package main

import (
	"context"

	authredirectssvr "github.com/CaliLuke/go-plane/authredirects/gen/http/auth_redirects/server"
	oauthredirectssvr "github.com/CaliLuke/go-plane/oauthredirects/gen/http/oauth_redirects/server"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountAuthRedirectsRoutes(ctx context.Context, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.authRedirects != nil {
		server := authredirectssvr.New(pending.authRedirects, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(authRedirectSecurityMiddleware)
		server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, authRedirectSessionPath))
		authredirectssvr.Mount(mux, server)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "auth_redirects", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}

func mountOAuthRedirectsRoutes(ctx context.Context, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.oauthRedirects != nil {
		server := oauthredirectssvr.New(pending.oauthRedirects, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(authRedirectSecurityMiddleware)
		server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, authRedirectSessionPath))
		oauthredirectssvr.Mount(mux, server)
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "oauth_redirects"}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
}
