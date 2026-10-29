package main

import (
	"context"
	"net/http"

	instanceadminsessionsvr "github.com/CaliLuke/go-plane/gen/http/instance_admin_session/server"
	instanceadminsession "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountInstanceAdminSessionRoutes(ctx context.Context, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, mux loomhttp.Muxer) {
	if instanceAdminSessionEndpoints != nil {
		server := instanceadminsessionsvr.New(instanceAdminSessionEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(djangoJSONCompatibilityMiddleware)
		instanceadminsessionsvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/instances/admins/me/", djangoJSONHeadHandler(server.Me))
		mountProtocolHandler(mux, "HEAD", "/api/instances/admins/session/", djangoJSONHeadHandler(server.Session))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/admins/me/", http.HandlerFunc(workspaceSlugOptionsHandler))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/admins/session/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "instance_admin_session", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
