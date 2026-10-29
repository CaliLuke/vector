package main

import (
	workitemsvr "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkItemCoreRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.workItemCore != nil {
		core := workitemsvr.New(pending.workItemCore, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		core.Use(sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
		workitemsvr.Mount(mux, core)
		mountWorkItemHeads(mux, core, readServicesCORSPolicy)
	}
}
