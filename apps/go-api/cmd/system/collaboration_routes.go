package main

import (
	collaborationsvr "github.com/CaliLuke/go-plane/ticket_collaboration/gen/http/ticket_collaboration/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountTicketCollaborationRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.ticketCollaboration != nil {
		collaboration := collaborationsvr.New(pending.ticketCollaboration, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		collaboration.Use(sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
		collaborationsvr.Mount(mux, collaboration)
		mountCollaborationHeads(mux, collaboration, readServicesCORSPolicy)
	}
}
