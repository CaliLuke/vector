package main

import (
	"net/http"

	apimodulereadssvr "github.com/CaliLuke/go-plane/apimodulereads/gen/http/api_module_reads/server"
	modulereadssvr "github.com/CaliLuke/go-plane/gen/http/module_reads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountModuleReadsProtocolRoutes(mux loomhttp.Muxer, server *modulereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/modules/", server.WorkspaceList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-modules/", server.ArchivedList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/", server.ArchivedDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/", server.ArchiveDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/", server.Issues},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/", server.IssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/", server.Links},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/", server.LinkDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/", server.Detail},
	}, corsPolicy)
}

func mountAPIModuleReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.apiModuleReads != nil {
		modules := apimodulereadssvr.New(pending.apiModuleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		modules.Use(stack)
		apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModules, readServicesCORSPolicy)
		apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLiteHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLite, readServicesCORSPolicy)
		apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModules, readServicesCORSPolicy)
		apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues, readServicesCORSPolicy)
		apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk, readServicesCORSPolicy)
	}
}

func mountModuleLifecycleRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) {
	if pending.moduleLifecycle != nil {
		var options apiV1CompatibilityOptions
		if activityOptions != nil {
			options = *activityOptions
		} else {
			options = apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		}
		mountModuleLifecycle(mux, pending.moduleLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
}

func mountModuleReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.moduleReads != nil {
		server := modulereadssvr.New(pending.moduleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		modulereadssvr.Mount(mux, server)
		mountModuleReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
