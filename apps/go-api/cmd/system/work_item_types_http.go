package main

import (
	"context"
	"net/http"

	workitemtypessvr "github.com/CaliLuke/go-plane/workitemtypes/gen/http/work_item_types/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

// This service uses the same CSRF token and origin checks as account writes.
func workItemTypesCSRFMiddleware(trustedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if accountAuthenticationUnsafeMethod(request.Method) {
				if reason := accountAuthenticationCSRFReason(request, trustedOrigins); reason != "" {
					writeReadServiceJSON(response, http.StatusForbidden, map[string]string{"message": "CSRF Failed: " + reason})
					return
				}
			}
			next.ServeHTTP(response, request)
		})
	}
}

func mountWorkItemTypes(mux loomhttp.Muxer, pending pendingReplacementEndpoints, cors loomhttp.RuntimeCORSPolicy, onError func(context.Context, http.ResponseWriter, error)) {
	if pending.workItemTypes == nil {
		return
	}
	server := workitemtypessvr.New(pending.workItemTypes, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, onError, nil, cors)
	options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
	server.Use(sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie))
	server.Use(workItemTypesCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins))
	workitemtypessvr.Mount(mux, server)
}
