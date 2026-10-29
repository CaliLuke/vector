package main

import (
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectLifecycleRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.projectLifecycle != nil {
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		if activityOptions != nil {
			options = *activityOptions
		}
		mountProjectLifecycle(mux, pending.projectLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}
