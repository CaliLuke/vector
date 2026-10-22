package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkItemRequestSizeFailurePrecedesViewAndSessionHeaders(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/acme/projects/11111111-1111-1111-1111-111111111111/issues/",
		"/api/v1/workspaces/acme/projects/11111111-1111-1111-1111-111111111111/issues/",
	} {
		t.Run(path, func(t *testing.T) {
			handler := readServicesCompatibilityMiddleware(activityReadsRequestSizeMiddleware(4)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("oversized request reached authentication or view")
			})))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader("12345")))
			if response.Code != http.StatusRequestEntityTooLarge || response.Body.String() != string(djangoRequestTooLargeBody) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "" || response.Header().Get("Vary") != "origin" {
				t.Fatalf("middleware rejection contains view/session headers: %v", response.Header())
			}
			if response.Header().Get("X-Frame-Options") != "DENY" {
				t.Fatalf("middleware rejection lost security headers: %v", response.Header())
			}
		})
	}
}
