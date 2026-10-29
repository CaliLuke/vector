package main

import (
	"context"
	"net/http"

	analyticviews "github.com/CaliLuke/go-plane/gen/analytic_views"
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
	"github.com/CaliLuke/loom/security"
)

func newHTTPHandler(ctx context.Context, systemEndpoints *system.Endpoints) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, nil, nil, nil, nil)
}

func newHTTPHandlerWithStickies(ctx context.Context, systemEndpoints *system.Endpoints, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, stickyEndpoints, authenticate, nil, nil)
}

func newHTTPHandlerWithWorkspaceSlugCheck(ctx context.Context, systemEndpoints *system.Endpoints, workspaceEndpoints *workspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, nil, nil, workspaceEndpoints, nil)
}

func newHTTPHandlerWithInstanceWorkspaceSlugCheck(ctx context.Context, systemEndpoints *system.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, nil, nil, nil, instanceWorkspaceEndpoints)
}

func buildHTTPHandlerWithSlugChecks(ctx context.Context, systemEndpoints *system.Endpoints, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, nil, nil)
}

func newHTTPHandlerWithInstanceAdmin(ctx context.Context, systemEndpoints *system.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, nil, nil, nil, nil, userInstanceAdminEndpoints, nil)
}

func newHTTPHandlerWithProfileFlags(ctx context.Context, systemEndpoints *system.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, nil, nil, nil, nil, nil, userProfileFlagsEndpoints)
}

func newHTTPHandlerWithNotificationPreferences(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, nil, nil, nil, nil, nil)
}

func newHTTPHandlerWithUserAccounts(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, nil, nil, nil, nil)
}

func newHTTPHandlerWithAPITokensAndInstanceAdminSession(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, nil, nil)
}

func newHTTPHandlerWithWorkspaceContent(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints ...*workspacehomepreferences.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoints...)
}

func newHTTPHandlerWithWorkspaceReplacementsAndLastVisited(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints *workspacehomepreferences.Endpoints, memberMeEndpoints *workspacememberme.Endpoints, analyticViewEndpoints *analyticviews.Endpoints, recentVisitEndpoints *workspacerecentvisits.Endpoints, lastVisitedEndpoints *lastvisitedworkspace.Endpoints, favoriteEndpoints ...*workspacefavorites.Endpoints) http.Handler {
	return buildHTTPHandlerWithReplacementServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoints, memberMeEndpoints, analyticViewEndpoints, recentVisitEndpoints, lastVisitedEndpoints, favoriteEndpoints...)
}

func buildHTTPHandlerWithBackendReplacements(ctx context.Context, systemEndpoints *system.Endpoints, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, false, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, nil, nil, nil, nil, nil, nil)
}

func buildHTTPHandlerWithServices(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints ...*workspacehomepreferences.Endpoints) http.Handler {
	var homePreferenceEndpoint *workspacehomepreferences.Endpoints
	if len(homePreferenceEndpoints) > 0 {
		homePreferenceEndpoint = homePreferenceEndpoints[0]
	}
	return buildHTTPHandlerWithReplacementServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoint, nil, nil, nil, nil)
}

func buildHTTPHandlerWithReplacementServices(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints *workspacehomepreferences.Endpoints, memberMeEndpoints *workspacememberme.Endpoints, analyticViewEndpoints *analyticviews.Endpoints, recentVisitEndpoints *workspacerecentvisits.Endpoints, lastVisitedEndpoints *lastvisitedworkspace.Endpoints, favoriteEndpoints ...*workspacefavorites.Endpoints) http.Handler {
	return buildHTTPHandlerWithReplacementServicesAndPending(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoints, memberMeEndpoints, analyticViewEndpoints, recentVisitEndpoints, lastVisitedEndpoints, nil, pendingReplacementEndpoints{}, favoriteEndpoints...)
}
