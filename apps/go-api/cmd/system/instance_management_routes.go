package main

import (
	"net/http"

	instancemanagementsvr "github.com/CaliLuke/go-plane/gen/http/instance_management/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountInstanceManagementRoutes(dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) func(http.Handler) http.Handler {
	var instanceProtocolMiddleware func(http.Handler) http.Handler
	if pending.instanceManagement != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		cookie := pending.publicProjectMetadataAdminCookie
		if cookie.Name == "" {
			cookie.Name = "admin-session-id"
			cookie.Configured = true
		}
		sessionStack := sessionServiceMiddlewareStack(options, pending.readServicesSession, cookie)
		stack := func(next http.Handler) http.Handler {
			return sessionStack(instanceManagementAnonymousThrottleMiddleware(pending.instanceManagementThrottle)(next))
		}
		server := instancemanagementsvr.New(pending.instanceManagement, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		instancemanagementsvr.Mount(mux, server)
		for _, path := range []string{
			"/api/instances/admins/sign-in/",
			"/api/instances/admins/sign-up/",
			"/api/instances/admins/sign-out/",
		} {
			mountProtocolHandler(mux, http.MethodOptions, path, instanceManagementPlainRedirectOptionsHandler())
		}
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/instances/admins/sign-in/", server.AdminSignIn},
			{"/api/instances/admins/sign-up/", server.AdminSignUp},
			{"/api/instances/admins/sign-out/", server.AdminSignOut},
			{"/api/instances/", server.RootGet},
			{"/api/instances/admins/", server.AdminsGet},
			{"/api/instances/admins/{pk}/", server.AdminsDetailGet},
			{"/api/instances/configurations/", server.ConfigGet},
			{"/api/instances/telemetry/", server.TelemetryGet},
			{"/api/instances/workspaces/", server.WorkspacesGet},
		}, readServicesCORSPolicy)
		instanceProtocolMiddleware = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, pending.instanceManagementMetadata, readServicesCORSPolicy, stack, cookie, dbg, options)
	}
	return instanceProtocolMiddleware
}
