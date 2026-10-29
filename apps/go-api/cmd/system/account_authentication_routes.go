package main

import (
	"context"

	accountauthsvr "github.com/CaliLuke/go-plane/accountauthentication/gen/http/account_authentication/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountAccountAuthenticationRoutes(ctx context.Context, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.accountAuthentication != nil {
		server := accountauthsvr.New(pending.accountAuthentication, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		bodySizeLimit := pending.activityReadsBodyLimit
		if bodySizeLimit <= 0 {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityOptions := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: bodySizeLimit}
		server.Use(accountAuthenticationGzipMiddleware)
		server.Use(passwordLoginCSRFCookieMiddleware(pending.csrfTokenCookie))
		server.Use(accountAuthenticationProtocolMiddleware)
		server.Use(accountAuthenticationCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins))
		server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, accountAuthenticationSessionPath))
		server.Use(activityReadsAuditFallbackMiddleware(compatibilityOptions))
		server.Use(activityReadsRequestSizeMiddleware(bodySizeLimit))
		accountauthsvr.Mount(mux, server)
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "account_authentication", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
