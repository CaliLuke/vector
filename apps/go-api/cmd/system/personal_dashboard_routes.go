package main

import (
	"net/http"

	personalsvr "github.com/CaliLuke/go-plane/workspacepersonal/gen/http/personal_dashboard/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountPersonalDashboardRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.personalDashboard != nil {
		server := personalsvr.New(pending.personalDashboard, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		personalsvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/users/me/workspaces/{slug}/activity-graph/", server.ActivityGraph},
			{"/api/users/me/workspaces/{slug}/dashboard/", server.Dashboard},
			{"/api/users/me/workspaces/{slug}/issues-completed-graph/", server.IssuesCompletedGraph},
		}, readServicesCORSPolicy)
	}
}
