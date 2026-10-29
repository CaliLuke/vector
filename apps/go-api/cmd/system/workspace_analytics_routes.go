package main

import (
	"net/http"

	analyticssvr "github.com/CaliLuke/go-plane/workspaceanalytics/gen/http/workspace_analytics/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceAnalyticsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.workspaceAnalytics != nil {
		server := analyticssvr.New(pending.workspaceAnalytics, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		analyticssvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/workspaces/{slug}/advance-analytics/", server.AdvanceAnalytics},
			{"/api/workspaces/{slug}/advance-analytics-stats/", server.AdvanceAnalyticsStats},
			{"/api/workspaces/{slug}/advance-analytics-charts/", server.AdvanceAnalyticsCharts},
			{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics/", server.ProjectAdvanceAnalytics},
			{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-stats/", server.ProjectAdvanceAnalyticsStats},
			{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-charts/", server.ProjectAdvanceAnalyticsCharts},
			{"/api/workspaces/{slug}/analytics/", server.Analytics},
			{"/api/workspaces/{slug}/default-analytics/", server.DefaultAnalytics},
			{"/api/workspaces/{slug}/saved-analytic-view/{analytic_id}/", server.SavedAnalyticView},
		}, readServicesCORSPolicy)
	}
}
