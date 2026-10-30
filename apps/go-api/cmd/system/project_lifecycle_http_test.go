package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestProjectLifecycleProtocolsResolveAPICompanionsAndRetainedSessionReads(t *testing.T) {
	protocol, ok := resolveReadServiceProtocolForMethod("/api/v1/workspaces/demo/projects-lite/", http.MethodOptions)
	if !ok || protocol.apiAuthorization != "project_lifecycle" || protocol.allow != "GET" {
		t.Fatalf("picker OPTIONS protocol=%+v matched=%v", protocol, ok)
	}
	write, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/", http.MethodPatch)
	if !ok || !write.authenticateMethod || write.allow != "PUT, PATCH, DELETE, OPTIONS" {
		t.Fatalf("session write protocol=%+v matched=%v", write, ok)
	}
	read, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/", http.MethodGet)
	if !ok || read.allow == write.allow {
		t.Fatalf("retained session read inherited write contract: %+v", read)
	}
}
func TestProjectLifecycleProtocolsRejectNeighboringPaths(t *testing.T) {
	for _, path := range []string{"/api/v1/workspaces/demo/projects-extra/", "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/summary-extra/", "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/modules/"} {
		if protocol, ok := projectLifecycleProtocolForPath(path); ok {
			t.Errorf("unowned path %s resolved to %s", path, protocol.pattern)
		}
	}
}

func TestProjectLifecycleOptionsRetainSourceActionsAndMissingTargetVariant(t *testing.T) {
	collection := newProjectLifecycleSessionProtocol("/api/workspaces/{slug}/projects/", "POST, OPTIONS", "Project")
	if !strings.Contains(string(collection.optionsMetadata), `"POST"`) || !strings.Contains(string(collection.optionsMetadata), `"name"`) {
		t.Fatalf("collection metadata lacks source actions: %s", collection.optionsMetadata)
	}
	detail := newProjectLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Project")
	if !strings.Contains(string(detail.optionsMetadata), `"PUT"`) || strings.Contains(string(detail.optionsMetadataMissing), `"actions"`) {
		t.Fatalf("missing/existing target metadata collapsed: present=%s missing=%s", detail.optionsMetadata, detail.optionsMetadataMissing)
	}
}

func TestProjectLifecycleArchiveHeadUsesAuthenticatedUnsupportedMethodBoundary(t *testing.T) {
	protocol, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/archive/", http.MethodHead)
	if !ok || !protocol.authenticateMethod || slices.Contains(protocol.passthroughMethods, http.MethodHead) {
		t.Fatalf("archive HEAD bypassed authentication: %+v matched=%v", protocol, ok)
	}
}

func TestAPIProjectReadsKeepSourceAllowWithoutChangingWriteMethods(t *testing.T) {
	for _, path := range []string{"/api/v1/workspaces/demo/projects/", "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/"} {
		protocol, ok := resolveReadServiceProtocolForMethod(path, http.MethodGet)
		if !ok || protocol.allow != "GET" {
			t.Fatalf("GET protocol=%+v matched=%v", protocol, ok)
		}
		response := httptest.NewRecorder()
		projectLifecycleResponseCompatibilityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Header().Get("Allow") != "GET" {
			t.Fatalf("Allow=%s", response.Header().Get("Allow"))
		}
		write, ok := resolveReadServiceProtocolForMethod(path, http.MethodPatch)
		if !ok || write.allow == "GET" {
			t.Fatalf("write protocol lost its methods: %+v", write)
		}
	}
}

func TestProjectLiteArchivedQueryPreservesNativeBooleanRules(t *testing.T) {
	for _, tc := range []struct{ name, query, want string }{
		{"omitted", "order_by=name", ""},
		{"true", "include_archived=true&order_by=name", "true"},
		{"case insensitive", "include_archived=TRUE&order_by=name", "true"},
		{"one", "include_archived=1&order_by=name", "true"},
		{"false", "include_archived=false&order_by=name", "false"},
		{"zero", "include_archived=0&order_by=name", "false"},
		{"empty", "include_archived=&order_by=name", "false"},
		{"other text", "include_archived=yes&order_by=name", "false"},
		{"last repeated value wins", "include_archived=true&include_archived=false&order_by=name", "false"},
		{"last repeated true wins", "include_archived=false&include_archived=1&order_by=name", "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects-lite/?"+tc.query, nil)
			handled := false
			handler := projectLifecycleQueryCompatibilityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handled = true
				if got := r.URL.Query().Get("include_archived"); got != tc.want {
					t.Errorf("include_archived=%q, want %q", got, tc.want)
				}
				if got := r.URL.Query().Get("order_by"); got != "name" {
					t.Errorf("unrelated order_by changed to %q", got)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, original)
			if !handled || response.Code != http.StatusNoContent {
				t.Fatalf("request not forwarded: handled=%v status=%d", handled, response.Code)
			}
			if original.URL.RawQuery != tc.query {
				t.Fatalf("shared request query changed: %q", original.URL.RawQuery)
			}
		})
	}
}
