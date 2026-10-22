package main

import (
	_ "embed"
	"encoding/json"
	"net/http"

	svr "github.com/CaliLuke/go-plane/workspacelifecycle/gen/http/workspace_lifecycle/server"
	workspaceapi "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed workspace_collection_options.json
var workspaceCollectionOptionsMetadata []byte

// Reads and OPTIONS retain the existing shared-path protocol owner. Generated
// writes use the same session finalization and request compatibility boundary.
func workspaceLifecycleProtocolForMethod(protocol readServiceProtocol, method string) readServiceProtocol {
	if protocol.pattern != "/api/workspaces/" && protocol.pattern != "/api/workspaces/{slug}/" {
		return protocol
	}
	if protocol.pattern == "/api/workspaces/" && method == http.MethodOptions {
		protocol.optionsMetadata = json.RawMessage(workspaceCollectionOptionsMetadata)
	}
	// Django resolves the workspace view before method dispatch. Authentication,
	// OPTIONS metadata, and APPEND_SLASH therefore apply to unsupported verbs
	// and reads too, rather than only to the methods implemented by Go.
	protocol.authenticateMethod = true
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		protocol.allow = "PUT, PATCH, DELETE, OPTIONS"
		if protocol.pattern == "/api/workspaces/" {
			protocol.allow = "POST, OPTIONS"
		}
	}
	return protocol
}

func mountWorkspaceLifecycle(mux loomhttp.Muxer, endpoints *workspaceapi.Endpoints, cors loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) {
	server := svr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, cors)
	server.Use(sessionServiceMiddlewareStack(options, sessions, cookie))
	svr.Mount(mux, server)
}
