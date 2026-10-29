package main

import (
	"context"
	"net/http"

	analyticviews "github.com/CaliLuke/go-plane/gen/analytic_views"
	entityfavorites "github.com/CaliLuke/go-plane/gen/entity_favorites"
	instanceadminsession "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instanceworkspaceslugcheck "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	lastvisitedworkspace "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	system "github.com/CaliLuke/go-plane/gen/system"
	useraccounts "github.com/CaliLuke/go-plane/gen/user_accounts"
	userapitokens "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	userinstanceadmin "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	usernotificationpreferences "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	userprofileflags "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	workspacefavorites "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	workspacehomepreferences "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	workspacememberme "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	workspacequicklinks "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisits "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemes "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/loom/clue/debug"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

func newHTTPHandlerWithPendingReplacements(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints *workspacehomepreferences.Endpoints, memberMeEndpoints *workspacememberme.Endpoints, analyticViewEndpoints *analyticviews.Endpoints, recentVisitEndpoints *workspacerecentvisits.Endpoints, lastVisitedEndpoints *lastvisitedworkspace.Endpoints, entityEndpoint *entityfavorites.Endpoints, pending pendingReplacementEndpoints, favoriteEndpoints ...*workspacefavorites.Endpoints) http.Handler {
	return buildHTTPHandlerWithReplacementServicesAndPending(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoints, memberMeEndpoints, analyticViewEndpoints, recentVisitEndpoints, lastVisitedEndpoints, entityEndpoint, pending, favoriteEndpoints...)
}

func buildHTTPHandlerWithReplacementServicesAndPending(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints *workspacehomepreferences.Endpoints, memberMeEndpoints *workspacememberme.Endpoints, analyticViewEndpoints *analyticviews.Endpoints, recentVisitEndpoints *workspacerecentvisits.Endpoints, lastVisitedEndpoints *lastvisitedworkspace.Endpoints, entityEndpoint *entityfavorites.Endpoints, pending pendingReplacementEndpoints, favoriteEndpoints ...*workspacefavorites.Endpoints) http.Handler {
	var favoriteEndpoint *workspacefavorites.Endpoints
	if len(favoriteEndpoints) > 0 {
		favoriteEndpoint = favoriteEndpoints[0]
	}
	mux := loomhttp.NewMuxer()
	if dbg {
		debug.MountPprofHandlers(debug.Adapt(mux))
		debug.MountDebugLogEnabler(debug.Adapt(mux))
	}

	core := mountCoreHTTPRoutes(systemEndpoints, pending, mux)
	readServicesCORSPolicy := core.corsPolicy
	var routes httpRouteMiddleware
	mountStickyRoutes(ctx, stickyEndpoints, authenticate, pending, mux, readServicesCORSPolicy)
	mountWorkspaceSlugRoutes(ctx, workspaceEndpoints, mux)
	mountInstanceWorkspaceSlugRoutes(ctx, instanceWorkspaceEndpoints, mux)
	mountUserInstanceAdminRoutes(ctx, userInstanceAdminEndpoints, mux)
	mountUserProfileFlagsRoutes(ctx, userProfileFlagsEndpoints, mux)
	mountUserNotificationPreferencesRoutes(ctx, notificationEndpoints, mux)
	mountUserAccountsRoutes(ctx, userAccountsEndpoints, mux)
	mountUserAPITokensRoutes(ctx, userAPITokensEndpoints, mux)
	mountInstanceAdminSessionRoutes(ctx, instanceAdminSessionEndpoints, mux)
	mountWorkspaceQuickLinksRoutes(ctx, quickLinkEndpoints, mux)
	mountWorkspaceThemesRoutes(ctx, themeEndpoints, mux)
	mountWorkspaceHomePreferencesRoutes(homePreferenceEndpoints, mux)
	mountWorkspaceMemberMeRoutes(memberMeEndpoints, mux)
	mountAnalyticViewsRoutes(analyticViewEndpoints, mux)
	mountWorkspaceFavoritesRoutes(mux, favoriteEndpoint)
	mountEntityFavoritesRoutes(ctx, entityEndpoint, mux)
	mountWorkspaceRecentVisitsRoutes(ctx, recentVisitEndpoints, mux)
	mountLastVisitedWorkspaceRoutes(lastVisitedEndpoints, mux)
	mountUserIdentityRoutes(pending, mux)
	mountUserPropertiesRoutes(pending, mux)
	mountWorkspaceMetadataRoutes(pending, mux)
	mountProjectIdentifiersRoutes(pending, mux)
	mountProjectMembershipReadsRoutes(pending, mux)
	mountProjectStateReadsRoutes(pending, mux)
	mountProjectLabelReadsRoutes(pending, mux)
	routes.publicMetadata = mountPublicProjectMetadataReadsRoutes(dbg, pending, mux)
	routes.publicFeedback = mountPublicFeedbackRoutes(dbg, pending, mux)
	mountStandardViewsRoutes(pending, mux)
	mountUserSettingsRoutes(pending, mux)
	mountWorkspaceSidebarPreferencesRoutes(pending, mux)
	assetsMountedMethods := make(map[assetsLifecycleMountedRoute]struct{})
	stateProtocol, _ := pending.projectStateMetadata.(moduleLifecycleSessionProtocolAuthorizer)
	routes.session = mountWorkspaceUserInsightsRoutes(dbg, pending, mux, readServicesCORSPolicy)
	mountWorkItemTypes(mux, pending, readServicesCORSPolicy, errorHandler)
	mountMCPOAuthLinks(mux, pending, readServicesCORSPolicy, errorHandler)
	routes.session = mountSessionServiceRoutes(ctx, dbg, pending, mux, readServicesCORSPolicy, assetsMountedMethods, stateProtocol, routes.session)
	routes.instance = mountInstanceManagementRoutes(dbg, pending, mux, readServicesCORSPolicy)
	mountMilestoneHTTP(mux, pending, errorHandler, readServicesCORSPolicy)
	routes.activityOptions, routes.activity = mountActivityReadsRoutes(dbg, pending, mux, readServicesCORSPolicy)
	mountWorkspaceProjectReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountCycleReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountCycleLifecycleRoutes(pending, mux, readServicesCORSPolicy, routes.activityOptions)

	mountProjectLifecycleRoutes(pending, mux, readServicesCORSPolicy, routes.activityOptions)
	mountWorkspaceLifecycleRoutes(pending, mux, readServicesCORSPolicy)
	mountProjectPublicationRoutes(pending, mux, readServicesCORSPolicy)
	mountModuleLifecycleRoutes(pending, mux, readServicesCORSPolicy, routes.activityOptions)
	mountModuleReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountEstimateReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountPageReadsRoutes(pending, mux, readServicesCORSPolicy, routes.activityOptions)
	mountIntakeInboxReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountWorkItemHistoryRoutes(pending, mux, readServicesCORSPolicy)
	mountDeployBoardReadsRoutes(pending, mux, readServicesCORSPolicy)
	mountInvitationsMembershipRoutes(pending, mux, readServicesCORSPolicy)
	mountAssetsLifecycleRoutes(ctx, pending, mux, assetsMountedMethods, routes.activityOptions)
	core.logMounts(ctx)

	return buildHTTPMiddleware(ctx, dbg, stickyEndpoints != nil, pending, mux, readServicesCORSPolicy, routes)
}
