package main

import (
	"net/http"

	exportssvr "github.com/CaliLuke/go-plane/workspaceexports/gen/http/workspace_exports/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountWorkspaceExportsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.workspaceExports != nil {
		server := exportssvr.New(pending.workspaceExports, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		exportssvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/export-issues/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.GetIssueExports)))
	}
}
