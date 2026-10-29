package main

import (
	"net/http"

	externalsvr "github.com/CaliLuke/go-plane/externalintegrations/gen/http/external_integrations/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountExternalIntegrationsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.externalIntegrations != nil {
		server := externalsvr.New(pending.externalIntegrations, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		externalsvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/unsplash/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Unsplash)))
	}
}
