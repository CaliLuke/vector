package main

import (
	"context"
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

func mountSessionServiceRoutes(ctx context.Context, dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, assetsMountedMethods map[assetsLifecycleMountedRoute]struct{}, stateProtocol moduleLifecycleSessionProtocolAuthorizer, sessionProtocolMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	if pending.accountAuthentication != nil || pending.invitationsMembership != nil || pending.assetsLifecycle != nil || pending.membershipLifecycle != nil || pending.notifications != nil || pending.notificationActions != nil || pending.projectLabelWrites != nil || pending.userAccountWrites != nil || pending.workspaceLifecycle != nil || pending.workspaceExports != nil || pending.personalDashboard != nil || pending.workspaceAnalytics != nil || pending.workspaceSearch != nil || pending.externalIntegrations != nil || pending.projectMemberPreferences != nil || pending.projectIntakeWrites != nil || pending.projectConfiguration != nil || pending.customViewCreation != nil || pending.workspaceWebhooks != nil || pending.cycleLifecycle != nil || pending.moduleLifecycle != nil || pending.projectLifecycle != nil || pending.projectPublication != nil || pending.workItemHistory != nil || pending.pageReadsMetadata != nil || stateProtocol != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		sessionStack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		stack := func(next http.Handler) http.Handler {
			return sessionStack(assetDuplicateThrottleMiddleware(pending.assetsDuplicateThrottle)(next))
		}
		mountNotificationsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountNotificationActionsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountProjectLabelWritesRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountWorkspaceExportsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountPersonalDashboardRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountWorkspaceSearchRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountWorkspaceAnalyticsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountExternalIntegrationsRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountAuthRedirectsRoutes(ctx, pending, mux, readServicesCORSPolicy)
		mountAccountAuthenticationRoutes(ctx, pending, mux, readServicesCORSPolicy)
		mountOAuthRedirectsRoutes(ctx, pending, mux, readServicesCORSPolicy)
		mountUserAccountWritesRoutes(pending, mux, readServicesCORSPolicy, stack)

		mountProjectMemberPreferencesRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountProjectIntakeWritesRoutes(pending, mux, readServicesCORSPolicy, stack)
		mountProjectConfigurationRoutes(pending, mux, readServicesCORSPolicy, options, stack)
		mountCustomViewCreationRoutes(pending, mux, readServicesCORSPolicy, options)
		mountWorkspaceWebhooksRoutes(pending, mux, readServicesCORSPolicy, stack)
		authorizer := workItemHistorySessionProtocolDispatch{projectPublicationSessionProtocolDispatch: projectPublicationSessionProtocolDispatch{labelSessionProtocolDispatch: labelSessionProtocolDispatch{exports: pending.workspaceExportsMetadata, personal: pending.personalDashboardMetadata, search: pending.workspaceSearchMetadata, analytics: pending.workspaceAnalyticsMetadata, userAccount: pending.userAccountWritesMetadata, workspaceLifecycle: pending.workspaceLifecycleMetadata, external: pending.externalIntegrationsMetadata, preferences: pending.projectMemberPreferencesMetadata, intakeWrites: pending.projectIntakeWritesMetadata, configuration: pending.projectConfigurationMetadata, webhooks: pending.workspaceWebhooksMetadata, notifications: pending.notificationsMetadata, labels: pending.projectLabelWritesMetadata, membership: pending.membershipLifecycleMetadata, assets: pending.assetsSessionMetadata, assetsMethods: assetsMountedMethods, invitations: pending.invitationsMetadata, cycles: pending.cycleLifecycleMetadata, modules: pending.moduleLifecycleMetadata, projects: pending.projectLifecycleMetadata, pages: pending.pageReadsMetadata, states: stateProtocol, userInsights: pending.workspaceUserInsightsMetadata, customViews: pending.customViewCreationMetadata}, publication: pending.projectPublicationMetadata}, history: pending.workItemHistoryMetadata}
		sessionProtocolMiddleware = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, accountAuthenticationSessionProtocolDispatch{account: pending.accountAuthenticationMetadata, fallback: authorizer}, readServicesCORSPolicy, stack, pending.readServicesCookie, dbg, options, pending.accountAuthenticationCSRFTrustedOrigins)
	}
	return sessionProtocolMiddleware
}
