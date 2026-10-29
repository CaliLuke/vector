package main

import (
	"context"
	"net/http"

	instanceworkspaceslugsvr "github.com/CaliLuke/go-plane/gen/http/instance_workspace_slug_check/server"
	workspaceslugsvr "github.com/CaliLuke/go-plane/gen/http/workspace_slug_check/server"
	instanceworkspaceslugcheck "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	loomhttp "github.com/CaliLuke/loom/http"
)

type workspaceSlugResponseWriter struct{ http.ResponseWriter }

func (writer workspaceSlugResponseWriter) WriteHeader(status int) {
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
}

func workspaceSlugCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if values, ok := request.URL.Query()["slug"]; ok && len(values) > 0 {
			request = request.Clone(request.Context())
			query := request.URL.Query()
			query.Set("slug", values[len(values)-1])
			request.URL.RawQuery = query.Encode()
		}
		djangoJSONCompatibilityMiddleware(next).ServeHTTP(response, request)
	})
}

type workspaceSlugHeadWriter struct{ http.ResponseWriter }

func (writer workspaceSlugHeadWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func workspaceSlugHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceSlugCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceSlugOptionsHandler(response http.ResponseWriter, _ *http.Request) {
	headers := response.Header()
	headers.Set("Allow", "GET, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
	response.WriteHeader(http.StatusOK)
}

func mountWorkspaceSlugRoutes(ctx context.Context, workspaceEndpoints *workspaceslugcheck.Endpoints, mux loomhttp.Muxer) {
	if workspaceEndpoints != nil {
		workspaceServer := workspaceslugsvr.New(workspaceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		workspaceServer.Use(workspaceSlugCompatibilityMiddleware)
		workspaceslugsvr.Mount(mux, workspaceServer)
		// Django exposes HEAD/OPTIONS as protocol-level companions to this GET
		// route. They are not separate application operations.
		mountProtocolHandler(mux, "HEAD", "/api/workspace-slug-check/", workspaceSlugHeadHandler(workspaceServer.Check))
		mountProtocolHandler(mux, "OPTIONS", "/api/workspace-slug-check/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range workspaceServer.Mounts {
			logHTTPRouteMount(ctx, "workspace_slug_check", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}

func mountInstanceWorkspaceSlugRoutes(ctx context.Context, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, mux loomhttp.Muxer) {
	if instanceWorkspaceEndpoints != nil {
		instanceWorkspaceServer := instanceworkspaceslugsvr.New(instanceWorkspaceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		instanceWorkspaceServer.Use(workspaceSlugCompatibilityMiddleware)
		instanceworkspaceslugsvr.Mount(mux, instanceWorkspaceServer)
		mountProtocolHandler(mux, "HEAD", "/api/instances/workspace-slug-check/", workspaceSlugHeadHandler(instanceWorkspaceServer.Check))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/workspace-slug-check/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range instanceWorkspaceServer.Mounts {
			logHTTPRouteMount(ctx, "instance_workspace_slug_check", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
