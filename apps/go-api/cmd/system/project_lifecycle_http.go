package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	svr "github.com/CaliLuke/go-plane/projectlifecycle/gen/http/project_lifecycle/server"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed project_lifecycle_options_metadata.json
var projectLifecycleOptionsMetadataJSON []byte
var projectLifecycleOptionsMetadata = func() map[string]struct {
	Metadata        json.RawMessage `json:"metadata"`
	MissingMetadata json.RawMessage `json:"missing_metadata"`
} {
	var result map[string]struct {
		Metadata        json.RawMessage `json:"metadata"`
		MissingMetadata json.RawMessage `json:"missing_metadata"`
	}
	if err := json.Unmarshal(projectLifecycleOptionsMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

type projectLifecycleProtocol struct {
	readServiceProtocol
	methods []string
}

func newProjectLifecycleAPIProtocol(pattern, allow, name string, methods ...string) projectLifecycleProtocol {
	p := newReadServiceProtocol(pattern, allow, name, "", false, "project_lifecycle")
	if pattern == "/api/v1/workspaces/{slug}/projects/" || pattern == "/api/v1/workspaces/{slug}/projects/{pk}/" {
		p.methodAllow = map[string]string{http.MethodGet: "GET"}
	}
	p.generatedMethods = true
	return projectLifecycleProtocol{readServiceProtocol: p, methods: methods}
}
func newProjectLifecycleSessionProtocol(pattern, allow, name string, methods ...string) projectLifecycleProtocol {
	p := newSessionServiceProtocol(pattern, allow, name, false)
	p.generatedMethods = true
	if metadata, ok := projectLifecycleOptionsMetadata[pattern]; ok {
		p.optionsMetadata = metadata.Metadata
		p.optionsMetadataMissing = metadata.MissingMetadata
	}
	if !strings.HasSuffix(pattern, "/archive/") {
		p.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	}
	return projectLifecycleProtocol{readServiceProtocol: p, methods: methods}
}

var projectLifecycleProtocols = []projectLifecycleProtocol{
	newProjectLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects-lite/", "GET", "Project List Lite API Endpoint", http.MethodGet),
	newProjectLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/", "GET, POST", "Project List Create API Endpoint", http.MethodGet, http.MethodPost),
	newProjectLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{pk}/", "GET, PATCH, DELETE", "Project Detail API Endpoint", http.MethodGet, http.MethodPatch, http.MethodDelete),
	newProjectLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archive/", "POST, DELETE", "Project Archive Unarchive API Endpoint", http.MethodPost, http.MethodDelete),
	newProjectLifecycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/summary/", "GET", "Project Summary API Endpoint", http.MethodGet),
	newProjectLifecycleSessionProtocol("/api/workspaces/{slug}/projects/", "POST, OPTIONS", "Project", http.MethodPost),
	newProjectLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Project", http.MethodPut, http.MethodPatch, http.MethodDelete),
	newProjectLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archive/", "POST, DELETE, OPTIONS", "Project Archive Unarchive Endpoint", http.MethodPost, http.MethodDelete),
}

func projectLifecycleProtocolForMethod(existing readServiceProtocol, method string) readServiceProtocol {
	for _, p := range projectLifecycleProtocols {
		if existing.pattern != p.pattern {
			continue
		}
		if p.authenticateMethod && (method == http.MethodGet || method == http.MethodHead) {
			// Keep canonical retired reads on their existing handlers while
			// slashless requests use the shared session append-slash boundary.
			redirectBoundary := existing
			redirectBoundary.authenticateMethod = true
			redirectBoundary.passthroughMethods = []string{http.MethodGet, http.MethodHead}
			return redirectBoundary
		}
		return p.readServiceProtocol
	}
	return existing
}

func projectLifecycleProtocolForPath(path string) (readServiceProtocol, bool) {
	for _, p := range projectLifecycleProtocols {
		if readServiceProtocolMatchesPath(path, p.readServiceProtocol) {
			return p.readServiceProtocol, true
		}
	}
	return readServiceProtocol{}, false
}
func projectLifecycleResponseCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := projectLifecycleProtocolForPath(r.URL.Path); ok && strings.HasPrefix(r.URL.Path, "/api/v1/") {
			allow := p.allow
			if value := p.methodAllow[r.Method]; value != "" {
				allow = value
			}
			w.Header().Set("Allow", allow)
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Referrer-Policy", "same-origin")
			mergeVary(w.Header(), "origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
		}
		next.ServeHTTP(w, r)
	})
}
func mountProjectLifecycle(mux loomhttp.Muxer, endpoints *api.Endpoints, cors loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) {
	server := svr.New(endpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
	server.Use(func(next http.Handler) http.Handler {
		apiStack := activityReadsMiddlewareStack(options, sessions, cookie)(projectLifecycleResponseCompatibilityMiddleware(projectLifecycleQueryCompatibilityMiddleware(next)))
		sessionStack := sessionServiceMiddlewareStack(options, sessions, cookie)(next)
		return cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/") {
				apiStack.ServeHTTP(w, r)
			} else {
				sessionStack.ServeHTTP(w, r)
			}
		}))
	})
	server.Mount(mux)
}

// DRF reads include_archived as text and considers only true/1 enabled.
// Preserve the original query in the shared request context for repository use.
func projectLifecycleQueryCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/projects-lite/") {
			query := r.URL.Query()
			if values, ok := query["include_archived"]; ok {
				enabled := false
				if len(values) > 0 {
					value := strings.ToLower(values[len(values)-1])
					enabled = value == "true" || value == "1"
				}
				query.Set("include_archived", fmt.Sprint(enabled))
				clone := r.Clone(r.Context())
				url := *r.URL
				url.RawQuery = query.Encode()
				clone.URL = &url
				r = clone
			}
		}
		next.ServeHTTP(w, r)
	})
}

type projectLifecycleProtocolAuthorizer interface {
	systemapi.SessionResourceProtocolAuthorizer
	systemapi.SessionProtocolAuthorizer
	AuthorizeProjectLifecycleAPIProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any)
}

func isProjectLifecycleSessionProtocol(pattern string) bool {
	for _, p := range projectLifecycleProtocols {
		if p.authenticateMethod && p.pattern == pattern {
			return true
		}
	}
	return false
}
