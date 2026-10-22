package main

import (
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	customviewcreationsvr "github.com/CaliLuke/go-plane/customviewcreation/gen/http/custom_view_creation/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func customViewCreationProtocols() []readServiceProtocol {
	return []readServiceProtocol{
		newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/project-views/", "POST, OPTIONS", "Project User Views Endpoint", false),
		newSessionServiceProtocol("/api/workspaces/{slug}/workspace-views/", "POST, OPTIONS", "Workspace Member User Views Endpoint", false),
	}
}

func isCustomViewCreationProtocol(pattern string) bool {
	for _, protocol := range customViewCreationProtocols() {
		if protocol.pattern == pattern {
			return true
		}
	}
	return false
}

func mountCustomViewCreation(
	mux loomhttp.Muxer,
	endpoints *customviewcreationapi.Endpoints,
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
	policy loomhttp.RuntimeCORSPolicy,
) *customviewcreationsvr.Server {
	server := customviewcreationsvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, policy)
	server.Use(sessionServiceMiddlewareStack(options, sessions, cookie))
	customviewcreationsvr.Mount(mux, server)
	return server
}
