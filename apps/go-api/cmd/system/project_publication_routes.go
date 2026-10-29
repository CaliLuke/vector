package main

import (
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectPublicationRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.projectPublication != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountProjectPublication(mux, pending.projectPublication, pending.projectPublicationMetadata, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}
