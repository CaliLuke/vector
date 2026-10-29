package main

import (
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkItemHistoryRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.workItemHistory != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountWorkItemHistory(mux, pending.workItemHistory, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}
