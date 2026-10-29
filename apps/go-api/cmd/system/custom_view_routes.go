package main

import (
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountCustomViewCreationRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions) {
	if pending.customViewCreation != nil {
		mountCustomViewCreation(mux, pending.customViewCreation, options, pending.readServicesSession, pending.readServicesCookie, readServicesCORSPolicy)
	}
}
