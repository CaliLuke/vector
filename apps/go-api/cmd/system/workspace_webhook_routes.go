package main

import (
	"net/http"

	workspacewebhookssvr "github.com/CaliLuke/go-plane/workspace_webhooks/gen/http/workspace_webhooks/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceWebhooksRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.workspaceWebhooks != nil {
		server := workspacewebhookssvr.New(pending.workspaceWebhooks, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		workspacewebhookssvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhooks/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.List)))
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhooks/{pk}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Detail)))
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhook-logs/{webhook_id}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Logs)))
	}
}
