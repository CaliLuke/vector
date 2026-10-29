package main

import (
	"context"
	"fmt"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	csrftokensvr "github.com/CaliLuke/go-plane/gen/http/csrf_token/server"
	systemsvr "github.com/CaliLuke/go-plane/gen/http/system/server"
	timezonesvr "github.com/CaliLuke/go-plane/gen/http/timezones/server"
	system "github.com/CaliLuke/go-plane/gen/system"
	timezone "github.com/CaliLuke/go-plane/gen/timezones"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
)

// coreHTTPRoutes retains the built-in mounts until registration logging completes.
type coreHTTPRoutes struct {
	system     *systemsvr.Server
	csrf       *csrftokensvr.Server
	timezones  *timezonesvr.Server
	corsPolicy loomhttp.RuntimeCORSPolicy
}

func mountCoreHTTPRoutes(systemEndpoints *system.Endpoints, pending pendingReplacementEndpoints, mux loomhttp.Muxer) coreHTTPRoutes {
	systemServer := systemsvr.New(
		systemEndpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		errorHandler,
		nil,
	)
	csrfTokenEndpoints := pending.csrfToken
	if csrfTokenEndpoints == nil {
		csrfTokenEndpoints = csrftoken.NewEndpoints(systemapi.NewCSRFToken())
	}
	csrfTokenCORSPolicy, err := djangoCSRFTokenRuntimeCORSPolicy(pending.csrfTokenCORS)
	if err != nil {
		panic(fmt.Errorf("build CSRF token runtime CORS policy: %w", err))
	}
	readServicesCORSPolicy, err := djangoRuntimeCORSPolicy(pending.readServicesCORS)
	if err != nil {
		panic(fmt.Errorf("build read-services runtime CORS policy: %w", err))
	}
	csrfTokenServer := csrftokensvr.New(
		csrfTokenEndpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		errorHandler,
		nil,
		csrfTokenCORSPolicy,
	)
	csrfTokenHandler := csrfTokenServer.CsrfToken
	csrfTokenOptions := csrfTokenServer.CsrfTokenOptions
	csrfTokenHandler = djangoCSRFResponseCookiePolicy(pending.csrfTokenCookie).Handler(csrfTokenHandler)
	if pending.csrfTokenThrottle != nil {
		bodySizeLimit := pending.csrfTokenBodyLimit
		if !pending.csrfTokenBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityMiddleware := publicProjectMetadataReadsCompatibilityMiddleware(
			pending.csrfTokenThrottle,
			pending.csrfTokenCORS,
			pending.csrfTokenSessionCookie,
			nil,
			"",
			bodySizeLimit,
			false,
			false,
		)
		csrfTokenHandler = compatibilityMiddleware(csrfTokenHandler)
		csrfTokenOptions = compatibilityMiddleware(csrfTokenOptions)
	}
	headersMiddleware := csrfTokenHeadersCompatibilityMiddleware(csrfTokenCORSPolicy)
	csrfTokenServer.CsrfToken = csrfTokenResponseCompatibilityMiddleware(headersMiddleware(csrfTokenHandler))
	csrfTokenServer.CsrfTokenOptions = csrfTokenResponseCompatibilityMiddleware(headersMiddleware(csrfTokenOptions))
	systemsvr.Mount(mux, systemServer)
	csrftokensvr.Mount(mux, csrfTokenServer)
	csrfTokenGET := csrfTokenCORSPolicy.Handler(loomhttp.AsHandlerFunc(csrfTokenServer.CsrfToken))
	loomhttp.MountDerivedHead(mux, "/auth/get-csrf-token/", csrfTokenGET)
	timezoneEndpoints := timezone.NewEndpoints(systemapi.NewTimezones())
	timezoneEndpoints.Use(log.Endpoint)
	timezoneServer := timezonesvr.New(
		timezoneEndpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		errorHandler,
		nil,
		readServicesCORSPolicy,
	)
	timezoneServer.Use(timezoneCompatibilityMiddleware)
	timezonesvr.MountListHandler(mux, timezoneServer.List, readServicesCORSPolicy)
	timezoneHEAD := djangoProtocolOptionsJSONNegotiationMiddleware(loomhttp.AsHandlerFunc(timezoneServer.List))
	timezoneHEAD = timezoneProtocolHeadersMiddleware(timezoneHEAD)
	timezoneHEAD = readServicesCORSPolicy.Handler(timezoneHEAD.ServeHTTP)
	loomhttp.MountDerivedHead(mux, "/api/timezones/", timezoneHEAD)
	timezoneOptions := djangoProtocolOptionsWithoutActionsHandler(timezoneProtocolHeadersMiddleware, "Timezone Endpoint")
	timezoneOptions = djangoProtocolOptionsJSONNegotiationMiddleware(timezoneOptions)
	timezoneOptions = timezoneProtocolHeadersMiddleware(timezoneOptions)
	timezoneOptions = djangoProtocolCORSOptionsHandler(readServicesCORSPolicy, []string{http.MethodGet}, timezoneOptions)
	mountProtocolHandler(mux, http.MethodOptions, "/api/timezones/", timezoneOptions)
	return coreHTTPRoutes{system: systemServer, csrf: csrfTokenServer, timezones: timezoneServer, corsPolicy: readServicesCORSPolicy}
}

func (routes coreHTTPRoutes) logMounts(ctx context.Context) {
	for _, mount := range routes.system.Mounts {
		logHTTPRouteMount(ctx, "system", mount.Method, mount.Verb, mount.Pattern)
	}
	for _, mount := range routes.csrf.Mounts {
		logHTTPRouteMount(ctx, "csrf_token", mount.Method, mount.Verb, mount.Pattern)
	}
	for _, mount := range routes.timezones.Mounts {
		logHTTPRouteMount(ctx, "timezones", mount.Method, mount.Verb, mount.Pattern)
	}
}
