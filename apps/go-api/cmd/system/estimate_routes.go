package main

import (
	estimatereadssvr "github.com/CaliLuke/go-plane/gen/http/estimate_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountEstimateReadsProtocolRoutes(mux loomhttp.Muxer, server *estimatereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/estimates/", server.WorkspaceSelected},
		{"/api/workspaces/{slug}/projects/{project_id}/estimates/", server.ProjectList},
		{"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/", server.ProjectDetail},
	}, corsPolicy)
}

func mountEstimateReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.estimateReads != nil {
		server := estimatereadssvr.New(pending.estimateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		estimatereadssvr.Mount(mux, server)
		mountEstimateReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
