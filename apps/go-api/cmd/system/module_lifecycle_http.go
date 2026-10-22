package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	modulelifecyclesvr "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/server"
	modulelifecycleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed module_lifecycle_options_metadata.json
var moduleLifecycleOptionsMetadataJSON []byte

var moduleLifecycleOptionsMetadata = func() map[string]struct {
	Status          int             `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	MissingMetadata json.RawMessage `json:"missing_metadata"`
	GuestMetadata   json.RawMessage `json:"guest_metadata"`
} {
	var result map[string]struct {
		Status          int             `json:"status"`
		Metadata        json.RawMessage `json:"metadata"`
		MissingMetadata json.RawMessage `json:"missing_metadata"`
		GuestMetadata   json.RawMessage `json:"guest_metadata"`
	}
	if err := json.Unmarshal(moduleLifecycleOptionsMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

type moduleLifecycleProtocol struct {
	readServiceProtocol
	methods []string
}

func newModuleLifecycleSessionProtocol(pattern, allow, name string, methods []string) moduleLifecycleProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, name, false)
	protocol.generatedMethods = true
	protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	if metadata, ok := moduleLifecycleOptionsMetadata[pattern]; ok {
		protocol.optionsStatus = metadata.Status
		protocol.optionsMetadata = metadata.Metadata
		protocol.optionsMetadataMissing = metadata.MissingMetadata
		protocol.optionsMetadataGuest = metadata.GuestMetadata
	}
	return moduleLifecycleProtocol{readServiceProtocol: protocol, methods: methods}
}

func newModuleLifecycleAPIProtocol(pattern, allow string, methods []string) moduleLifecycleProtocol {
	protocol := newReadServiceProtocol(pattern, allow, "Module API Endpoint", "", false, "module_lifecycle")
	protocol.generatedMethods = true
	// Collection/detail paths still have a retained Go GET surface. Action
	// aliases are write-only Django routes, so GET must reach the shared
	// method-not-allowed boundary instead of the generated mux's empty 405.
	if !strings.HasSuffix(pattern, "/archive/") && !strings.HasSuffix(pattern, "/unarchive/") {
		protocol.passthroughMethods = []string{http.MethodGet}
	}
	return moduleLifecycleProtocol{readServiceProtocol: protocol, methods: methods}
}

var moduleLifecycleProtocols = []moduleLifecycleProtocol{
	newModuleLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/", "POST", []string{http.MethodPost}),
	newModuleLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "PATCH, DELETE", []string{http.MethodPatch, http.MethodDelete}),
	newModuleLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/archive/", "POST", []string{http.MethodPost}),
	newModuleLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/unarchive/", "DELETE", []string{http.MethodDelete}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/", "POST, OPTIONS", "Module", []string{http.MethodPost}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Module", []string{http.MethodPut, http.MethodPatch, http.MethodDelete}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/", "POST, OPTIONS", "Module Link", []string{http.MethodPost}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Module Link", []string{http.MethodPut, http.MethodPatch, http.MethodDelete}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/", "POST, DELETE, OPTIONS", "Module Archive Unarchive Endpoint", []string{http.MethodPost, http.MethodDelete}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/", "POST, DELETE, OPTIONS", "Module Archive Unarchive Endpoint", []string{http.MethodPost, http.MethodDelete}),
	newModuleLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/", "POST, DELETE, OPTIONS", "Module Archive Unarchive Endpoint", []string{http.MethodPost, http.MethodDelete}),
}

func moduleLifecycleProtocolForMethod(existing readServiceProtocol, method string) readServiceProtocol {
	for _, protocol := range moduleLifecycleProtocols {
		if protocol.pattern != existing.pattern {
			continue
		}
		if method == http.MethodGet || method == http.MethodHead {
			// Session lifecycle routes wrap the existing read route so the
			// canonical request keeps its prior Go handler while slashless GET
			// and HEAD use Django's append-slash redirect boundary.
			if protocol.authenticateMethod {
				redirectBoundary := existing
				redirectBoundary.authenticateMethod = true
				redirectBoundary.passthroughMethods = []string{http.MethodGet, http.MethodHead}
				return redirectBoundary
			}
			return existing
		}
		// Django's Allow header describes the route's writable contract even
		// when the requested verb is unsupported. GET/HEAD are handled by the
		// protocol's passthrough list, preserving the retired Go read behavior.
		return protocol.readServiceProtocol
	}
	return existing
}

func moduleLifecycleProtocolForPath(requestPath string) (readServiceProtocol, bool) {
	var selected readServiceProtocol
	found := false
	for _, candidate := range moduleLifecycleProtocols {
		protocol := candidate.readServiceProtocol
		if !readServiceProtocolMatchesPath(requestPath, protocol) {
			continue
		}
		if !found || readServiceProtocolSpecificity(protocol.pattern) > readServiceProtocolSpecificity(selected.pattern) {
			selected = protocol
			found = true
		}
	}
	return selected, found
}

// moduleLifecycleAPIResponseCompatibilityMiddleware applies Django's common
// protocol headers to generated API-v1 lifecycle responses, including errors
// returned before the service method runs (for example missing API keys).
func moduleLifecycleAPIResponseCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, matched := moduleLifecycleProtocolForPath(request.URL.Path)
		apiV1 := strings.HasPrefix(request.URL.Path, "/api/v1/")
		if matched && apiV1 {
			response.Header().Set("Allow", protocol.allow)
			response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			response.Header().Set("Referrer-Policy", "same-origin")
			mergeVary(response.Header(), "origin")
			response.Header().Set("X-Content-Type-Options", "nosniff")
			response.Header().Set("X-Frame-Options", "DENY")
		}
		next.ServeHTTP(response, request)
	})
}

func mountModuleLifecycle(mux loomhttp.Muxer, endpoints *modulelifecycleapi.Endpoints, cors loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) {
	server := modulelifecyclesvr.New(endpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
	apiStack := activityReadsMiddlewareStack(options, sessions, cookie)
	sessionStack := sessionServiceMiddlewareStack(options, sessions, cookie)
	server.Use(func(next http.Handler) http.Handler {
		api := apiStack(moduleLifecycleAPIResponseCompatibilityMiddleware(next))
		session := sessionStack(moduleLifecycleAPIResponseCompatibilityMiddleware(next))
		return cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/") {
				api.ServeHTTP(w, r)
				return
			}
			session.ServeHTTP(w, r)
		}))
	})
	server.Mount(mux)
}
