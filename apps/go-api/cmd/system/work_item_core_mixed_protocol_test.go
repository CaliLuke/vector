package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loom "github.com/CaliLuke/loom/pkg"
)

// These paths were split between a retired Go read and a Django write. Once
// writes retire, their previously Django-owned redirects and preflights must
// still pass through the session compatibility boundary.
func TestWorkItemMixedPathsPreserveSessionProtocol(t *testing.T) {
	const project = "00000000-0000-4000-8000-000000020104"
	const child = "99999999-9999-4999-8999-999999999999"
	paths := []string{
		"/api/workspaces/demo/draft-issues/" + child + "/",
		"/api/workspaces/demo/projects/" + project + "/cycles/" + child + "/cycle-issues/",
		"/api/workspaces/demo/projects/" + project + "/modules/" + child + "/issues/" + child + "/",
		"/api/workspaces/demo/projects/" + project + "/intake-issues/",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
			if err != nil {
				t.Fatal(err)
			}
			options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}
			stack := sessionServiceMiddlewareStack(options, nil, djangoSessionCookieConfig{})
			auth := &workItemProtocolAuthorizerStub{err: &workitemapi.WorkItemError{Kind: "status_403", Body: loom.MustJSONValueFrom(map[string]any{"detail": "You do not have permission to perform this action."})}}
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
			handler = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, sessionProtocolAuthorizerStub{false}, policy, stack, djangoSessionCookieConfig{}, false, options)(handler)
			handler = workItemProtocolMiddleware(auth, policy, stack, djangoSessionCookieConfig{})(handler)
			handler = readServicesCompatibilityMiddleware(handler)
			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions, http.MethodTrace} {
				response := httptest.NewRecorder()
				request := httptest.NewRequest(method, strings.TrimSuffix(path, "/")+"?x=1", nil)
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != path+"?x=1" || auth.calls != 0 {
					t.Fatalf("slashless %s: status=%d location=%q auth=%d", method, response.Code, response.Header().Get("Location"), auth.calls)
				}
			}
			for _, target := range []string{path, strings.TrimSuffix(path, "/")} {
				request := httptest.NewRequest(http.MethodOptions, target, nil)
				request.Header.Set("Origin", "https://app.example.test")
				request.Header.Set("Access-Control-Request-Method", http.MethodPost)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" || auth.calls != 0 {
					t.Fatalf("preflight %s: status=%d headers=%v auth=%d", target, response.Code, response.Header(), auth.calls)
				}
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodTrace, path, nil))
			if response.Code != http.StatusForbidden || auth.calls != 1 {
				t.Fatalf("unsupported TRACE: status=%d auth=%d body=%s", response.Code, auth.calls, response.Body.String())
			}
		})
	}
}
