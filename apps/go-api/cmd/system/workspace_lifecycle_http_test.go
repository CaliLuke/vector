package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkspaceLifecycleWriteMethodsCheckTimezoneBeforeAuthorization(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		protocol := workspaceLifecycleProtocolForMethod(newSessionServiceProtocol("/api/workspaces/{slug}/", "GET, HEAD, OPTIONS", "Workspace", true), method)
		if !protocol.authenticateMethod || protocol.permissionBeforeTimezone {
			t.Errorf("%s protocol = authenticate:%t permission-before-timezone:%t", method, protocol.authenticateMethod, protocol.permissionBeforeTimezone)
		}
	}
}

func TestWorkspaceLifecycleReadMethodsKeepExistingProtocol(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		protocol := workspaceLifecycleProtocolForMethod(newSessionServiceProtocol("/api/workspaces/{slug}/", "GET, HEAD, OPTIONS", "Workspace", true), method)
		if !protocol.authenticateMethod || protocol.permissionBeforeTimezone || protocol.allow != "GET, HEAD, OPTIONS" {
			t.Errorf("%s changed existing read protocol: %+v", method, protocol)
		}
	}
}

func TestWorkspaceLifecycleUnsupportedMethodsStillUseSessionProtocol(t *testing.T) {
	for _, method := range []string{http.MethodTrace, http.MethodConnect} {
		protocol, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/", method)
		if !ok || !protocol.authenticateMethod {
			t.Errorf("%s protocol = %+v, matched=%t; want authenticated method boundary", method, protocol, ok)
		}
	}
	protocol, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/", http.MethodOptions)
	if !ok || !protocol.authenticateMethod || protocol.authoredOptions {
		t.Fatalf("OPTIONS protocol = %+v, matched=%t; want authenticated DRF metadata handling", protocol, ok)
	}
}

func TestWorkspaceLifecycleTransportUsesDjangoSessionProtocol(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}
	stack := sessionServiceMiddlewareStack(options, nil, djangoSessionCookieConfig{})
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	protocol := newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{allowed: true}, policy, stack, djangoSessionCookieConfig{}, true, options)
	handler := readServicesCompatibilityMiddleware(protocol(next))

	for _, test := range []struct {
		name, method, path string
		preflight          bool
		status             int
		location           string
	}{
		{name: "canonical metadata", method: http.MethodOptions, path: "/api/workspaces/demo/", status: http.StatusOK},
		{name: "canonical preflight", method: http.MethodOptions, path: "/api/workspaces/demo/", preflight: true, status: http.StatusOK},
		{name: "slashless get", method: http.MethodGet, path: "/api/workspaces/demo", status: http.StatusMovedPermanently, location: "/api/workspaces/demo/"},
		{name: "slashless head", method: http.MethodHead, path: "/api/workspaces/demo", status: http.StatusMovedPermanently, location: "/api/workspaces/demo/"},
		{name: "slashless options", method: http.MethodOptions, path: "/api/workspaces/demo", status: http.StatusMovedPermanently, location: "/api/workspaces/demo/"},
		{name: "slashless trace", method: http.MethodTrace, path: "/api/workspaces/demo", status: http.StatusMovedPermanently, location: "/api/workspaces/demo/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			if test.preflight {
				request.Header.Set("Origin", "https://app.example.test")
				request.Header.Set("Access-Control-Request-Method", http.MethodPatch)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || response.Header().Get("Location") != test.location {
				t.Fatalf("status=%d location=%q body=%q; want status=%d location=%q", response.Code, response.Header().Get("Location"), response.Body.String(), test.status, test.location)
			}
		})
	}
}
