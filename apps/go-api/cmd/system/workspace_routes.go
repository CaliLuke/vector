package main

import (
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceLifecycleRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.workspaceLifecycle != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountWorkspaceLifecycle(mux, pending.workspaceLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}
