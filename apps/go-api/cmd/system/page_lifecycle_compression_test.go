package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageDescriptionCompressionExceptionAppliesOnlyToReads(t *testing.T) {
	path := "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/00000000-0000-4000-8000-000000000002/description/"
	for _, test := range []struct {
		method string
		body   string
		vary   bool
	}{
		{method: http.MethodPatch, body: strings.Repeat("x", 164)},
		{method: http.MethodOptions, body: strings.Repeat("x", 164)},
		{method: http.MethodGet, body: strings.Repeat("x", 164), vary: true},
	} {
		t.Run(test.method, func(t *testing.T) {
			handler := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				mergeVary(response.Header(), "Cookie", "origin")
				_, _ = response.Write([]byte(test.body))
			}))
			request := httptest.NewRequest(test.method, path, nil)
			request.Header.Set("Accept-Encoding", "gzip")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			got := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
			if got != test.vary {
				t.Fatalf("Vary=%q, want Accept-Encoding present=%t", response.Header().Get("Vary"), test.vary)
			}
			if test.method != http.MethodGet && response.Header().Get("Content-Encoding") != "" {
				t.Fatalf("Content-Encoding=%q, small write response should not be compressed", response.Header().Get("Content-Encoding"))
			}
		})
	}
}
