package main

import (
	deployboardreadssvr "github.com/CaliLuke/go-plane/gen/http/deploy_board_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountDeployBoardReadsProtocolRoutes(mux loomhttp.Muxer, server *deployboardreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/", server.Collection},
		{"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/", server.Detail},
	}, corsPolicy)
}

func mountDeployBoardReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.deployBoardReads != nil {
		server := deployboardreadssvr.New(pending.deployBoardReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		deployboardreadssvr.Mount(mux, server)
		mountDeployBoardReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
