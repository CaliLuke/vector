package main

import (
	"net/http"

	analyticviews "github.com/CaliLuke/go-plane/gen/analytic_views"
	analyticviewssvr "github.com/CaliLuke/go-plane/gen/http/analytic_views/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func analyticViewsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mergeVary(response.Header(), "Cookie")
		workspaceContentCompatibilityMiddleware("analytic-view")(next).ServeHTTP(response, request)
	})
}

func mountAnalyticViewsProtocolRoutes(mux loomhttp.Muxer, list, retrieve http.Handler) {
	collection := "/api/workspaces/{slug}/analytic-view/"
	detail := collection + "{pk}/"
	mountProtocolHandler(mux, http.MethodHead, collection, analyticViewsHeadHandler(list))
	mountProtocolHandler(mux, http.MethodHead, detail, analyticViewsHeadHandler(retrieve))
	mountProtocolHandler(mux, http.MethodOptions, collection, analyticViewsCompatibilityMiddleware(workspaceContentOptionsHandler("analytic-view", "Analytic View List", true)))
	mountProtocolHandler(mux, http.MethodOptions, detail, analyticViewsCompatibilityMiddleware(workspaceContentOptionsHandler("analytic-view", "Analytic View", false)))
}

func analyticViewsHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		analyticViewsCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func mountAnalyticViewsRoutes(analyticViewEndpoints *analyticviews.Endpoints, mux loomhttp.Muxer) {
	if analyticViewEndpoints != nil {
		server := analyticviewssvr.New(analyticViewEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(analyticViewsCompatibilityMiddleware)
		analyticviewssvr.Mount(mux, server)
		mountAnalyticViewsProtocolRoutes(mux, server.List, server.Retrieve)
	}
}
