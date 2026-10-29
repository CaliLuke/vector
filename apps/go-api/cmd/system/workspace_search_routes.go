package main

import (
	"net/http"

	searchsvr "github.com/CaliLuke/go-plane/workspacesearch/gen/http/workspace_search/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceSearchRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.workspaceSearch != nil {
		server := searchsvr.New(pending.workspaceSearch, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		searchsvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/workspaces/{slug}/search/", server.GlobalSearch},
			{"/api/workspaces/{slug}/entity-search/", server.EntitySearch},
		}, readServicesCORSPolicy)
	}
}
