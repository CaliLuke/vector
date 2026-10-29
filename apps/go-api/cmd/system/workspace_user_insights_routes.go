package main

import (
	"net/http"

	workspaceuserinsightssvr "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/http/workspace_user_insights/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceUserInsightsRoutes(dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) func(http.Handler) http.Handler {
	var sessionProtocolMiddleware func(http.Handler) http.Handler
	if pending.workspaceUserInsights != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		stack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server := workspaceuserinsightssvr.New(pending.workspaceUserInsights, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		workspaceuserinsightssvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/workspaces/{slug}/user-activity/{user_id}/", server.Activity},
			{"/api/workspaces/{slug}/user-activity/{user_id}/export/", server.ExportActivity},
			{"/api/workspaces/{slug}/user-profile/{user_id}/", server.Profile},
			{"/api/workspaces/{slug}/user-stats/{user_id}/", server.Stats},
		}, readServicesCORSPolicy)
		authorizer := labelSessionProtocolDispatch{userInsights: pending.workspaceUserInsightsMetadata}
		sessionProtocolMiddleware = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, authorizer, readServicesCORSPolicy, stack, pending.readServicesCookie, dbg, options)
	}
	return sessionProtocolMiddleware
}
