package main

import (
	"net/http"

	apicyclereadssvr "github.com/CaliLuke/go-plane/apicyclereads/gen/http/api_cycle_reads/server"
	cyclereadssvr "github.com/CaliLuke/go-plane/gen/http/cycle_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountCycleReadsProtocolRoutes(mux loomhttp.Muxer, server *cyclereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/cycles/", server.WorkspaceList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/", server.ArchivedList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/", server.ArchivedDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/", server.Analytics},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", server.ArchiveDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", server.Issues},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", server.IssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/", server.Progress},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", server.Detail},
	}, corsPolicy)
}

func mountAPICycleReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.apiCycleReads != nil {
		cycles := apicyclereadssvr.New(pending.apiCycleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		cycles.Use(stack)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCycles, readServicesCORSPolicy)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLiteHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLite, readServicesCORSPolicy)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCycles, readServicesCORSPolicy)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues, readServicesCORSPolicy)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID, readServicesCORSPolicy)
		apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPkHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk, readServicesCORSPolicy)
	}
}

func mountCycleReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.cycleReads != nil {
		server := cyclereadssvr.New(pending.cycleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		cyclereadssvr.Mount(mux, server)
		mountCycleReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}

func mountCycleLifecycleRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.cycleLifecycle != nil {
		var options apiV1CompatibilityOptions
		if activityOptions != nil {
			options = *activityOptions
		} else {
			options = apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		}
		mountCycleLifecycle(mux, pending.cycleLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}
