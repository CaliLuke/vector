package main

import (
	workspaceprojectreadssvr "github.com/CaliLuke/go-plane/gen/http/workspace_project_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceProjectReadsProtocolRoutes(mux loomhttp.Muxer, server *workspaceprojectreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/users/me/workspaces/", server.UserWorkspaces},
		{"/api/workspaces/", server.Workspaces},
		{"/api/workspaces/{slug}/", server.Workspace},
		{"/api/workspaces/{slug}/project-stats/", server.ProjectStats},
		{"/api/workspaces/{slug}/projects/", server.Projects},
		{"/api/workspaces/{slug}/projects/details/", server.ProjectDetails},
		{"/api/workspaces/{slug}/projects/{pk}/", server.Project},
	}, corsPolicy)
}

func mountWorkspaceProjectReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.workspaceProjectReads != nil {
		server := workspaceprojectreadssvr.New(pending.workspaceProjectReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		workspaceprojectreadssvr.Mount(mux, server)
		mountWorkspaceProjectReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
