package main

import (
	"context"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	milestonessvr "github.com/CaliLuke/go-plane/milestones/gen/http/milestones/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountMilestoneHTTP(mux loomhttp.Muxer, pending pendingReplacementEndpoints, handleError func(context.Context, http.ResponseWriter, error), cors loomhttp.RuntimeCORSPolicy) {
	if pending.milestones == nil {
		return
	}
	server := milestonessvr.New(pending.milestones, mux, systemapi.MilestoneRequestDecoder, loomhttp.ResponseEncoder, handleError, nil, cors)
	options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
	server.Use(activityReadsMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie))
	milestonessvr.Mount(mux, server)
}
