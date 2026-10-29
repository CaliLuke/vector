package main

import (
	"context"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/platform"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	"github.com/CaliLuke/loom/clue/debug"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
)

// httpRouteMiddleware contains protocol wrappers produced during registration.
// Assembly applies them in request order after every route has been mounted.
type httpRouteMiddleware struct {
	activity, session, instance, publicMetadata, publicFeedback func(http.Handler) http.Handler
	activityOptions                                             *apiV1CompatibilityOptions
}

func buildHTTPMiddleware(ctx context.Context, dbg, stickyEnabled bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, routes httpRouteMiddleware) http.Handler {
	var handler http.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request = captureReadServicesOriginalRawQuery(request)
		request = normalizeReadServiceScalarQueryValues(normalizeDjangoTolerantQuery(request))
		mux.ServeHTTP(response, request)
	})
	if routes.activity != nil {
		handler = routes.activity(handler)
	}
	if routes.session != nil {
		handler = routes.session(handler)
	}
	if routes.instance != nil {
		handler = instanceManagementHandler(routes.instance(handler), handler)
	}
	if pending.userIdentity != nil || pending.userAccountWrites != nil {
		handler = userAccountMeSlashlessCompatibilityMiddleware(handler, dbg, readServicesCORSPolicy)
	}
	if pending.workItemCore != nil {
		handler = workItemResponseContentTypeMiddleware(handler)
		if pending.workItemProtocol != nil && routes.activityOptions != nil {
			stack := sessionServiceMiddlewareStack(*routes.activityOptions, pending.readServicesSession, pending.readServicesCookie)
			handler = workItemProtocolMiddleware(pending.workItemProtocol, readServicesCORSPolicy, stack, pending.readServicesCookie)(handler)
		}
	}
	handler = readServicesCompatibilityMiddleware(handler)
	if pending.hasReadServices() {
		handler = readServicesDjangoPathConverterMiddleware(readServicesCORSPolicy, routes.activityOptions)(handler)
	}
	if routes.activityOptions != nil || pending.notifications != nil || pending.notificationActions != nil || pending.projectLabelWrites != nil || pending.userAccountWrites != nil || pending.workspaceLifecycle != nil || pending.workspaceExports != nil || pending.personalDashboard != nil || pending.workItemHistory != nil || pending.workspaceAnalytics != nil || pending.workspaceSearch != nil || pending.externalIntegrations != nil || pending.projectMemberPreferences != nil || pending.projectIntakeWrites != nil || pending.projectConfiguration != nil || pending.customViewCreation != nil || pending.workspaceWebhooks != nil || pending.instanceManagement != nil {
		ordinary := activityReadsSessionFinalizationMiddleware(pending.readServicesSession, pending.readServicesCookie)(handler)
		if pending.instanceManagement != nil {
			cookie := pending.publicProjectMetadataAdminCookie
			if cookie.Name == "" {
				cookie.Name = "admin-session-id"
				cookie.Configured = true
			}
			admin := activityReadsSessionFinalizationMiddleware(pending.readServicesSession, cookie)(handler)
			handler = instanceManagementHandler(admin, ordinary)
		} else {
			handler = ordinary
		}
	}
	if pending.workItemCore != nil {
		handler = workItemIdentifierResolver404Middleware(handler, routes.activityOptions, readServicesCORSPolicy)
	}
	handler = csrfTokenCORSPreflightMiddleware(handler)
	handler = djangoCORSRequestCompatibilityMiddleware(handler)
	if stickyEnabled {
		handler = stickyFrontendTrailingSlashMiddleware(handler)
	}
	if dbg {
		handler = debug.HTTP()(handler)
	}
	handler = logHTTPServerErrors(handler)
	handler = log.HTTP(
		ctx, log.WithDisableRequestID(), log.WithPathFilter(publicMetadataCompatibilityPathPattern),
	)(handler)
	if pending.authRedirects != nil || pending.oauthRedirects != nil || pending.instanceManagement != nil {
		handler = djangoPlainViewCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins, pending.activityReadsBodyLimit)(handler)
	}
	if routes.publicMetadata != nil {
		// Keep compatibility responses outside the ordinary request logger. Exact
		// paths are filtered above, while cleaned malformed paths never enter it.
		handler = routes.publicMetadata(handler)
	}
	if routes.publicFeedback != nil {
		handler = routes.publicFeedback(handler)
	}
	// The public metadata compatibility layer logs the inner Django response at
	// RequestLoggerMiddleware's position. This outer layer installs its context.
	handler = telemetry.ClientAddressSpanMiddleware(handler)
	handler = log.HTTP(ctx, log.WithDisableRequestID(), log.WithDisableRequestLogging())(handler)
	handler = loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(handler)
	return platform.RequestIDMiddleware(handler)
}
