package main

import (
	"net/http"
	"strings"

	invitationssvr "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountInvitationsMembershipRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.invitationsMembership != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		server := invitationssvr.New(pending.invitationsMembership, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		apiStack := activityReadsMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		sessionStack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server.Use(func(next http.Handler) http.Handler {
			api, session := apiStack(next), sessionStack(next)
			return readServicesCORSPolicy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if strings.HasPrefix(request.URL.Path, "/api/v1/") {
					api.ServeHTTP(response, request)
				} else {
					session.ServeHTTP(response, request)
				}
			}))
		})
		invitationssvr.Mount(mux, server)
		mountInvitationsMembershipHeads(mux, server, readServicesCORSPolicy)
	}
}
