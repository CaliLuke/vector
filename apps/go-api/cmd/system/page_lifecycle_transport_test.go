package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestPageLifecycleGeneratedTransportsPreserveAuthenticationError(t *testing.T) {
	mux := loomhttp.NewMuxer()
	server := pagereadssvr.New(pagereads.NewEndpoints(pageUnauthorizedService{}), mux,
		loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	pagereadssvr.Mount(mux, server)

	const collection = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/"
	const detail = collection + "00000000-0000-4000-8000-000000000002/"
	for _, request := range []struct{ method, path string }{
		{http.MethodPost, collection},
		{http.MethodPatch, detail},
		{http.MethodDelete, detail},
		{http.MethodPost, detail + "access/"},
		{http.MethodPost, detail + "archive/"},
		{http.MethodDelete, detail + "archive/"},
		{http.MethodPatch, detail + "description/"},
		{http.MethodPost, detail + "duplicate/"},
		{http.MethodPost, detail + "lock/"},
		{http.MethodDelete, detail + "lock/"},
	} {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(request.method, request.path, nil))
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%q, want 401", response.Code, response.Body.String())
			}
			if got := strings.TrimSpace(response.Body.String()); got != `{"detail":"Authentication credentials were not provided."}` {
				t.Errorf("authentication response=%s", got)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type=%q, want application/json", got)
			}
		})
	}
}

func TestPageLifecycleProtocolKeepsWriteAllowsAndAuthenticatesOrdinaryOptions(t *testing.T) {
	mux := loomhttp.NewMuxer()
	cors, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	server := pagereadssvr.New(pagereads.NewEndpoints(pageUnauthorizedService{}), mux,
		loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	pagereadssvr.Mount(mux, server)
	mountPageReadsProtocolRoutes(mux, server, cors)
	handler := readServicesCompatibilityMiddleware(mux)

	const collection = "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/"
	const detail = collection + "00000000-0000-4000-8000-000000000002/"
	for _, route := range []struct {
		path, allow string
		methods     []string
	}{
		{collection, "POST, OPTIONS", []string{http.MethodPost}},
		{detail, "PATCH, DELETE, OPTIONS", []string{http.MethodPatch, http.MethodDelete}},
		{detail + "description/", "PATCH, OPTIONS", []string{http.MethodPatch}},
		{detail + "access/", "POST, OPTIONS", []string{http.MethodPost}},
		{detail + "archive/", "POST, DELETE, OPTIONS", []string{http.MethodPost, http.MethodDelete}},
		{detail + "duplicate/", "POST, OPTIONS", []string{http.MethodPost}},
		{detail + "lock/", "POST, DELETE, OPTIONS", []string{http.MethodPost, http.MethodDelete}},
	} {
		t.Run(route.path, func(t *testing.T) {
			for _, method := range append(route.methods, http.MethodOptions) {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, route.path, nil))
				if response.Code != http.StatusUnauthorized {
					t.Errorf("%s status=%d body=%q, want 401", method, response.Code, response.Body.String())
				}
				if got := response.Header().Get("Allow"); got != route.allow {
					t.Errorf("%s Allow=%q, want %q", method, got, route.allow)
				}
			}
			preflight := httptest.NewRequest(http.MethodOptions, route.path, nil)
			preflight.Header.Set("Origin", "https://app.example.test")
			preflight.Header.Set("Access-Control-Request-Method", route.methods[0])
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, preflight)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Errorf("preflight status=%d body=%q, want empty 204", response.Code, response.Body.String())
			}
			if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
				t.Errorf("preflight headers=%v", response.Header())
			}
		})
	}
}
