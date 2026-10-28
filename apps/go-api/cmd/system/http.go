package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"math/big"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	milestoneapi "github.com/CaliLuke/go-plane/milestones/gen/milestones"

	workitemtypesapi "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"

	systemapi "github.com/CaliLuke/go-plane"
	accountauthapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	accountauthsvr "github.com/CaliLuke/go-plane/accountauthentication/gen/http/account_authentication/server"
	activityreadssvr "github.com/CaliLuke/go-plane/activityreads/gen/http/plane_internal_django_migration_api/server"
	activityreads "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	apicyclereads "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	apicyclereadssvr "github.com/CaliLuke/go-plane/apicyclereads/gen/http/api_cycle_reads/server"
	apilabelreads "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	apilabelreadssvr "github.com/CaliLuke/go-plane/apilabelreads/gen/http/api_label_reads/server"
	apimodulereads "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	apimodulereadssvr "github.com/CaliLuke/go-plane/apimodulereads/gen/http/api_module_reads/server"
	apistatereads "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	apistatereadssvr "github.com/CaliLuke/go-plane/apistatereads/gen/http/api_state_reads/server"
	apiuseridentity "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	apiuseridentitysvr "github.com/CaliLuke/go-plane/apiuseridentity/gen/http/api_user_identity/server"
	apiv1stickies "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	apiv1stickiessvr "github.com/CaliLuke/go-plane/apiv1stickies/gen/http/api_v1_stickies/server"
	assetslifecyclesvr "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/plane_internal_django_migration_api/server"
	assetslifecycle "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	authredirectsapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	authredirectssvr "github.com/CaliLuke/go-plane/authredirects/gen/http/auth_redirects/server"
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	cyclelifecycle "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	externalsvr "github.com/CaliLuke/go-plane/externalintegrations/gen/http/external_integrations/server"
	analyticviews "github.com/CaliLuke/go-plane/gen/analytic_views"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	cyclereads "github.com/CaliLuke/go-plane/gen/cycle_reads"
	deployboardreads "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	entityfavorites "github.com/CaliLuke/go-plane/gen/entity_favorites"
	estimatereads "github.com/CaliLuke/go-plane/gen/estimate_reads"
	analyticviewssvr "github.com/CaliLuke/go-plane/gen/http/analytic_views/server"
	csrftokensvr "github.com/CaliLuke/go-plane/gen/http/csrf_token/server"
	cyclereadssvr "github.com/CaliLuke/go-plane/gen/http/cycle_reads/server"
	deployboardreadssvr "github.com/CaliLuke/go-plane/gen/http/deploy_board_reads/server"
	entityfavoritessvr "github.com/CaliLuke/go-plane/gen/http/entity_favorites/server"
	estimatereadssvr "github.com/CaliLuke/go-plane/gen/http/estimate_reads/server"
	instanceadminsessionsvr "github.com/CaliLuke/go-plane/gen/http/instance_admin_session/server"
	instancemanagementsvr "github.com/CaliLuke/go-plane/gen/http/instance_management/server"
	instanceworkspaceslugsvr "github.com/CaliLuke/go-plane/gen/http/instance_workspace_slug_check/server"
	intakeinboxreadssvr "github.com/CaliLuke/go-plane/gen/http/intake_inbox_reads/server"
	lastvisitedworkspacesvr "github.com/CaliLuke/go-plane/gen/http/last_visited_workspace/server"
	modulereadssvr "github.com/CaliLuke/go-plane/gen/http/module_reads/server"
	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	projectidentifierssvr "github.com/CaliLuke/go-plane/gen/http/project_identifiers/server"
	projectlabelreadssvr "github.com/CaliLuke/go-plane/gen/http/project_label_reads/server"
	projectmembershipreadssvr "github.com/CaliLuke/go-plane/gen/http/project_membership_reads/server"
	projectstatereadssvr "github.com/CaliLuke/go-plane/gen/http/project_state_reads/server"
	publicfeedbacksvr "github.com/CaliLuke/go-plane/gen/http/public_feedback/server"
	publicprojectmetadatareadssvr "github.com/CaliLuke/go-plane/gen/http/public_project_metadata_reads/server"
	standardviewssvr "github.com/CaliLuke/go-plane/gen/http/standard_views/server"
	stickysvr "github.com/CaliLuke/go-plane/gen/http/stickies/server"
	systemsvr "github.com/CaliLuke/go-plane/gen/http/system/server"
	timezonesvr "github.com/CaliLuke/go-plane/gen/http/timezones/server"
	useraccountssvr "github.com/CaliLuke/go-plane/gen/http/user_accounts/server"
	userapitokenssvr "github.com/CaliLuke/go-plane/gen/http/user_api_tokens/server"
	useridentitysvr "github.com/CaliLuke/go-plane/gen/http/user_identity/server"
	userinstanceadminsvr "github.com/CaliLuke/go-plane/gen/http/user_instance_admin/server"
	usernotificationpreferencessvr "github.com/CaliLuke/go-plane/gen/http/user_notification_preferences/server"
	userprofileflagssvr "github.com/CaliLuke/go-plane/gen/http/user_profile_flags/server"
	userpropertiessvr "github.com/CaliLuke/go-plane/gen/http/user_properties/server"
	usersettingssvr "github.com/CaliLuke/go-plane/gen/http/user_settings/server"
	workspacefavoritessvr "github.com/CaliLuke/go-plane/gen/http/workspace_favorites/server"
	workspacehomepreferencessvr "github.com/CaliLuke/go-plane/gen/http/workspace_home_preferences/server"
	workspacemembermesvr "github.com/CaliLuke/go-plane/gen/http/workspace_member_me/server"
	workspacemetadatasvr "github.com/CaliLuke/go-plane/gen/http/workspace_metadata/server"
	workspaceprojectreadssvr "github.com/CaliLuke/go-plane/gen/http/workspace_project_reads/server"
	workspacequicklinkssvr "github.com/CaliLuke/go-plane/gen/http/workspace_quick_links/server"
	workspacerecentvisitssvr "github.com/CaliLuke/go-plane/gen/http/workspace_recent_visits/server"
	workspacesidebarpreferencessvr "github.com/CaliLuke/go-plane/gen/http/workspace_sidebar_preferences/server"
	workspaceslugsvr "github.com/CaliLuke/go-plane/gen/http/workspace_slug_check/server"
	workspacethemesvr "github.com/CaliLuke/go-plane/gen/http/workspace_themes/server"
	instanceadminsession "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instancemanagementapi "github.com/CaliLuke/go-plane/gen/instance_management"
	instanceworkspaceslugcheck "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	intakeinboxreads "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	lastvisitedworkspace "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	modulereads "github.com/CaliLuke/go-plane/gen/module_reads"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	projectidentifiers "github.com/CaliLuke/go-plane/gen/project_identifiers"
	projectlabelreads "github.com/CaliLuke/go-plane/gen/project_label_reads"
	projectmembershipreads "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	projectstatereads "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicfeedback "github.com/CaliLuke/go-plane/gen/public_feedback"
	publicprojectmetadatareads "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	standardviews "github.com/CaliLuke/go-plane/gen/standard_views"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	system "github.com/CaliLuke/go-plane/gen/system"
	timezone "github.com/CaliLuke/go-plane/gen/timezones"
	useraccounts "github.com/CaliLuke/go-plane/gen/user_accounts"
	userapitokens "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	useridentity "github.com/CaliLuke/go-plane/gen/user_identity"
	userinstanceadmin "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	usernotificationpreferences "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	userprofileflags "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	userproperties "github.com/CaliLuke/go-plane/gen/user_properties"
	usersettings "github.com/CaliLuke/go-plane/gen/user_settings"
	workspacefavorites "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	workspacehomepreferences "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	workspacememberme "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	workspacemetadata "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	workspaceprojectreads "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	workspacequicklinks "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisits "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspacesidebarpreferences "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemes "github.com/CaliLuke/go-plane/gen/workspace_themes"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/platform"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	invitationssvr "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	invitationsapi "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	mcpoauthlinksapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	modulelifecycle "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	notificationactionssvr "github.com/CaliLuke/go-plane/notificationactions/gen/http/notification_actions/server"
	notificationactions "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	notificationssvr "github.com/CaliLuke/go-plane/notificationreads/gen/http/notifications/server"
	notifications "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	oauthredirectssvr "github.com/CaliLuke/go-plane/oauthredirects/gen/http/oauth_redirects/server"
	oauthredirectsapi "github.com/CaliLuke/go-plane/oauthredirects/gen/oauth_redirects"
	labelwritessvr "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/server"
	labelwrites "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	projectconfigurationsvr "github.com/CaliLuke/go-plane/projectconfiguration/gen/http/project_configuration/server"
	projectconfigurationapi "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	intakewritessvr "github.com/CaliLuke/go-plane/projectintakewrites/gen/http/project_intake_writes/server"
	intakewritesapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	projectlifecycleapi "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	preferencessvr "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/http/project_member_preferences/server"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	membershiplifecyclesvr "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	membershiplifecycle "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	projectpublicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	collaborationsvr "github.com/CaliLuke/go-plane/ticket_collaboration/gen/http/ticket_collaboration/server"
	collaborationapi "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	useraccountwritessvr "github.com/CaliLuke/go-plane/user_account_writes/gen/http/user_account_writes/server"
	useraccountwritesapi "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
	workitemsvr "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/server"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	workspacewebhookssvr "github.com/CaliLuke/go-plane/workspace_webhooks/gen/http/workspace_webhooks/server"
	workspacewebhooksapi "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	analyticssvr "github.com/CaliLuke/go-plane/workspaceanalytics/gen/http/workspace_analytics/server"
	analyticsapi "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	exportssvr "github.com/CaliLuke/go-plane/workspaceexports/gen/http/workspace_exports/server"
	exportsapi "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	workspaceLifecycleAPI "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	personalsvr "github.com/CaliLuke/go-plane/workspacepersonal/gen/http/personal_dashboard/server"
	personalapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	searchsvr "github.com/CaliLuke/go-plane/workspacesearch/gen/http/workspace_search/server"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	workspaceuserinsightssvr "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/http/workspace_user_insights/server"
	workspaceuserinsightsapi "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
	"github.com/CaliLuke/loom/clue/debug"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	encodingunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var requestMetadataPolicy = func() loomhttp.RequestMetadataPolicy {
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie", "Content-Type"}, []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"),
	})
	if err != nil {
		panic(err)
	}
	return policy
}()

func mountProtocolHandler(mux loomhttp.Muxer, method, pattern string, handler http.Handler) {
	//loom:vet ignore route-outside-design -- Django protocol compatibility, not a separate application operation
	loomhttp.MountHandler(mux, method, pattern, handler)
}

type djangoPreflightResponseWriter struct {
	http.ResponseWriter
}

func (writer djangoPreflightResponseWriter) WriteHeader(status int) {
	if status == http.StatusNoContent {
		removeVary(writer.Header(), "Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers")
		mergeVary(writer.Header(), "origin")
		writer.Header().Set("Content-Length", "0")
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		status = http.StatusOK
	}
	writer.ResponseWriter.WriteHeader(status)
}

func djangoProtocolCORSOptionsHandler(policy loomhttp.RuntimeCORSPolicy, methods []string, next http.Handler) http.Handler {
	options := policy.OptionsHandler(loomhttp.AsHandlerFunc(next), methods)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0 {
			request = request.Clone(request.Context())
			request.Header = request.Header.Clone()
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
			options.ServeHTTP(djangoPreflightResponseWriter{response}, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func newHTTPHandler(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, dbg, nil, nil, nil, nil)
}

func newHTTPHandlerWithStickies(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, nil, nil)
}

func newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, nil)
}

func newHTTPHandlerWithSlugChecks(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithSlugChecks(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints)
}

func buildHTTPHandlerWithSlugChecks(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, nil, nil)
}

func newHTTPHandlerWithBackendReplacements(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, nil)
}

func newHTTPHandlerWithProfileFlags(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints) http.Handler {
	return buildHTTPHandlerWithBackendReplacements(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints)
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

type djangoSessionLifecycle interface {
	InspectSession(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error)
	RefreshSession(context.Context, *http.Request, djangosession.SessionRefresh) (djangosession.SessionRefresh, error)
}

type publicProjectMetadataThrottle interface {
	djangoSessionLifecycle
	AllowRequest(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error)
	AllowRequestWithoutTimezone(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error)
	FinalizeUnaccessedSession(context.Context, *http.Request, int) (bool, *djangosession.SessionRefresh, error)
}

type instanceManagementAnonymousThrottle interface {
	AllowAnonymousRequest(context.Context, *http.Request) (bool, time.Duration, error)
}

type userIdentitySessionLifecycle interface {
	publicProjectMetadataThrottle
	InspectSessionWithTimezone(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error)
}
type readServicesSessionLifecycle interface {
	userIdentitySessionLifecycle
	InspectReadServiceSession(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error)
}

type pendingReplacementEndpoints struct {
	mcpLinks                                *mcpoauthlinksapi.Endpoints
	mcpResourceURL                          string
	workItemTypes                           *workitemtypesapi.Endpoints
	workspaceLifecycle                      *workspaceLifecycleAPI.Endpoints
	accountAuthentication                   *accountauthapi.Endpoints
	accountAuthenticationMetadata           systemapi.SessionProtocolAuthorizer
	authRedirects                           *authredirectsapi.Endpoints
	oauthRedirects                          *oauthredirectsapi.Endpoints
	workspaceUserInsights                   *workspaceuserinsightsapi.Endpoints
	workspaceUserInsightsMetadata           sessionResourceProtocolAuthorizer
	readServicesAuthCookie                  djangoSessionCookieConfig
	readServicesAuthSession                 authSessionLifecycle
	workspaceExports                        *exportsapi.Endpoints
	workspaceExportsMetadata                sessionResourceProtocolAuthorizer
	personalDashboard                       *personalapi.Endpoints
	personalDashboardMetadata               sessionResourceProtocolAuthorizer
	userAccountWrites                       *useraccountwritesapi.Endpoints
	userAccountWritesMetadata               sessionResourceProtocolAuthorizer
	workspaceLifecycleMetadata              systemapi.SessionResourceProtocolAuthorizer
	workspaceSearch                         *searchapi.Endpoints
	workspaceSearchMetadata                 sessionResourceProtocolAuthorizer
	workspaceAnalytics                      *analyticsapi.Endpoints
	workspaceAnalyticsMetadata              sessionResourceProtocolAuthorizer
	externalIntegrations                    *externalapi.Endpoints
	externalIntegrationsMetadata            sessionResourceProtocolAuthorizer
	projectMemberPreferences                *preferencesapi.Endpoints
	projectMemberPreferencesMetadata        sessionResourceProtocolAuthorizer
	projectIntakeWrites                     *intakewritesapi.Endpoints
	projectIntakeWritesMetadata             systemapi.SessionResourceProtocolAuthorizer
	assetsLifecycle                         *assetslifecycle.Endpoints
	assetsSessionMetadata                   systemapi.SessionResourceProtocolAuthorizer
	assetsAPIMetadata                       systemapi.AssetsAPIProtocolAuthorizer
	assetsDuplicateThrottle                 *apikey.Throttle
	notifications                           *notifications.Endpoints
	notificationActions                     *notificationactions.Endpoints
	notificationsMetadata                   systemapi.SessionProtocolAuthorizer
	csrfToken                               *csrftoken.Endpoints
	csrfTokenCORS                           []string
	accountAuthenticationCSRFTrustedOrigins []string
	csrfTokenCookie                         djangoCSRFCookieConfig
	csrfTokenThrottle                       publicProjectMetadataThrottle
	csrfTokenSessionCookie                  djangoSessionCookieConfig
	csrfTokenBodyLimit                      int64
	csrfTokenBodySet                        bool
	userIdentity                            *useridentity.Endpoints
	userIdentityCORS                        []string
	userIdentityCookie                      djangoSessionCookieConfig
	userIdentitySession                     userIdentitySessionLifecycle
	userIdentityBodyLimit                   int64
	userIdentityBodySet                     bool
	userIdentityActivity                    publicprojectmetadatareadsstore.ActivityLogger
	userIdentitySecret                      string
	userProperties                          *userproperties.Endpoints
	workspaceMetadata                       *workspacemetadata.Endpoints
	projectIdentifiers                      *projectidentifiers.Endpoints
	projectMembershipReads                  *projectmembershipreads.Endpoints
	projectMembershipMetadata               systemapi.ProjectMembershipReadsMetadataAuthorizer
	projectStateReads                       *projectstatereads.Endpoints
	projectStateMetadata                    systemapi.ProjectStateReadsMetadataAuthorizer
	projectLabelReads                       *projectlabelreads.Endpoints
	projectLabelWrites                      *labelwrites.Endpoints
	projectConfiguration                    *projectconfigurationapi.Endpoints
	projectConfigurationMetadata            projectConfigurationProtocolAuthorizer
	customViewCreation                      *customviewcreationapi.Endpoints
	customViewCreationMetadata              systemapi.SessionResourceProtocolAuthorizer
	workspaceWebhooks                       *workspacewebhooksapi.Endpoints
	workspaceWebhooksMetadata               sessionResourceProtocolAuthorizer
	instanceManagement                      *instancemanagementapi.Endpoints
	instanceManagementMetadata              sessionResourceProtocolAuthorizer
	instanceManagementThrottle              instanceManagementAnonymousThrottle
	workItemCore                            *workitemapi.Endpoints
	ticketCollaboration                     *collaborationapi.Endpoints
	workItemProtocol                        systemapi.WorkItemProtocolAuthorizer
	projectLabelWritesMetadata              systemapi.SessionResourceProtocolAuthorizer
	publicProjectMetadataReads              *publicprojectmetadatareads.Endpoints
	publicFeedback                          *publicfeedback.Endpoints
	publicProjectMetadataCORS               []string
	publicProjectMetadataCookie             djangoSessionCookieConfig
	publicProjectMetadataAdminCookie        djangoSessionCookieConfig
	publicProjectMetadataThrottle           publicProjectMetadataThrottle
	publicProjectMetadataActivity           publicprojectmetadatareadsstore.ActivityLogger
	publicProjectMetadataSecret             string
	publicProjectMetadataBodyLimit          int64
	publicProjectMetadataBodySet            bool
	standardViews                           *standardviews.Endpoints
	userSettings                            *usersettings.Endpoints
	workspaceSidebarPreferences             *workspacesidebarpreferences.Endpoints
	stickyWorkspaceAuthorizer               systemapi.StickyWorkspaceAuthorizer
	readServicesCORS                        []string
	readServicesSession                     readServicesSessionLifecycle
	readServicesCookie                      djangoSessionCookieConfig
	workspaceProjectReads                   *workspaceprojectreads.Endpoints
	projectLifecycle                        *projectlifecycleapi.Endpoints
	projectLifecycleMetadata                projectLifecycleProtocolAuthorizer
	projectPublication                      *projectpublicationapi.Endpoints
	projectPublicationMetadata              sessionResourceProtocolAuthorizer
	workItemHistory                         *workitemhistoryapi.Endpoints
	workItemHistoryMetadata                 sessionResourceProtocolAuthorizer
	cycleReads                              *cyclereads.Endpoints
	cycleLifecycle                          *cyclelifecycle.Endpoints
	cycleLifecycleMetadata                  cycleLifecycleSessionProtocolAuthorizer
	moduleLifecycle                         *modulelifecycle.Endpoints
	moduleLifecycleMetadata                 moduleLifecycleSessionProtocolAuthorizer
	moduleReads                             *modulereads.Endpoints
	estimateReads                           *estimatereads.Endpoints
	pageReads                               *pagereads.Endpoints
	pageReadsMetadata                       systemapi.SessionResourceProtocolAuthorizer
	intakeInboxReads                        *intakeinboxreads.Endpoints
	deployBoardReads                        *deployboardreads.Endpoints
	activityReads                           *activityreads.Endpoints
	apiCycleReads                           *apicyclereads.Endpoints
	apiModuleReads                          *apimodulereads.Endpoints
	apiLabelReads                           *apilabelreads.Endpoints
	milestones                              *milestoneapi.Endpoints
	invitationsMembership                   *invitationsapi.Endpoints
	invitationsMetadata                     systemapi.InvitationsMembershipProtocolAuthorizer
	membershipLifecycle                     *membershiplifecycle.Endpoints
	membershipLifecycleMetadata             systemapi.MembershipLifecycleProtocolAuthorizer
	apiStateReads                           *apistatereads.Endpoints
	apiV1Stickies                           *apiv1stickies.Endpoints
	apiV1StickiesProtocol                   systemapi.APIV1StickyProtocolAuthorizer
	apiUserIdentity                         *apiuseridentity.Endpoints
	activityReadsMetadata                   systemapi.ActivityReadsProtocolAuthorizer
	activityReadsActivity                   publicprojectmetadatareadsstore.ActivityLogger
	activityReadsSecret                     string
	activityReadsBodyLimit                  int64
}

func (pending pendingReplacementEndpoints) hasReadServices() bool {
	if pending.workItemTypes != nil || pending.milestones != nil {
		return true
	}
	return pending.accountAuthentication != nil || pending.invitationsMembership != nil || pending.membershipLifecycle != nil || pending.workItemCore != nil || pending.projectLabelWrites != nil || pending.workspaceProjectReads != nil || pending.projectLifecycle != nil ||
		pending.cycleReads != nil || pending.cycleLifecycle != nil || pending.moduleLifecycle != nil || pending.workspaceUserInsights != nil || pending.projectPublication != nil || pending.workItemHistory != nil ||
		pending.moduleReads != nil ||
		pending.estimateReads != nil || pending.instanceManagement != nil || pending.userAccountWrites != nil || pending.workspaceLifecycle != nil || pending.workspaceExports != nil || pending.personalDashboard != nil || pending.workspaceAnalytics != nil || pending.workspaceSearch != nil || pending.externalIntegrations != nil || pending.projectMemberPreferences != nil || pending.projectIntakeWrites != nil || pending.projectConfiguration != nil || pending.workspaceWebhooks != nil ||
		pending.pageReads != nil || pending.customViewCreation != nil ||
		pending.intakeInboxReads != nil ||
		pending.deployBoardReads != nil ||
		pending.activityReads != nil || pending.apiUserIdentity != nil || pending.apiCycleReads != nil || pending.apiModuleReads != nil || pending.apiLabelReads != nil || pending.apiStateReads != nil || pending.apiV1Stickies != nil || pending.notifications != nil
}

func newHTTPHandlerWithPendingReplacements(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints, notificationEndpoints *usernotificationpreferences.Endpoints, userAccountsEndpoints *useraccounts.Endpoints, userAPITokensEndpoints *userapitokens.Endpoints, instanceAdminSessionEndpoints *instanceadminsession.Endpoints, quickLinkEndpoints *workspacequicklinks.Endpoints, themeEndpoints *workspacethemes.Endpoints, homePreferenceEndpoints *workspacehomepreferences.Endpoints, memberMeEndpoints *workspacememberme.Endpoints, analyticViewEndpoints *analyticviews.Endpoints, recentVisitEndpoints *workspacerecentvisits.Endpoints, lastVisitedEndpoints *lastvisitedworkspace.Endpoints, entityEndpoint *entityfavorites.Endpoints, pending pendingReplacementEndpoints, favoriteEndpoints ...*workspacefavorites.Endpoints) http.Handler {
	return buildHTTPHandlerWithReplacementServicesAndPending(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, notificationEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, quickLinkEndpoints, themeEndpoints, homePreferenceEndpoints, memberMeEndpoints, analyticViewEndpoints, recentVisitEndpoints, lastVisitedEndpoints, entityEndpoint, pending, favoriteEndpoints...)
}

func buildHTTPHandlerWithBackendReplacements(ctx context.Context, systemEndpoints *system.Endpoints, dbg bool, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, workspaceEndpoints *workspaceslugcheck.Endpoints, instanceWorkspaceEndpoints *instanceworkspaceslugcheck.Endpoints, userInstanceAdminEndpoints *userinstanceadmin.Endpoints, userProfileFlagsEndpoints *userprofileflags.Endpoints) http.Handler {
	return buildHTTPHandlerWithServices(ctx, systemEndpoints, dbg, stickyEndpoints, authenticate, workspaceEndpoints, instanceWorkspaceEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, nil, nil, nil, nil, nil, nil)
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
	if stickyEndpoints != nil {
		stickyServer := stickysvr.New(stickyEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		stickyServer.Use(stickyCompatibilityMiddleware)
		if authenticate != nil {
			stickyServer.Use(stickyAuthenticationMiddleware(authenticate))
		}
		stickyListHEAD := stickyHeadPaginationAuthorizationMiddleware(mux, authenticate, pending.stickyWorkspaceAuthorizer, loomhttp.AsHandlerFunc(stickyServer.List))
		stickyListHEAD = djangoProtocolOptionsJSONNegotiationMiddleware(stickyListHEAD)
		stickyListHEAD = stickyProtocolHeadersMiddleware(stickyListHEAD)
		stickyListHEAD = readServicesCORSPolicy.Handler(stickyListHEAD.ServeHTTP)
		stickyRetrieveHEAD := djangoProtocolOptionsJSONNegotiationMiddleware(loomhttp.AsHandlerFunc(stickyServer.Retrieve))
		stickyRetrieveHEAD = stickyProtocolHeadersMiddleware(stickyRetrieveHEAD)
		stickyRetrieveHEAD = readServicesCORSPolicy.Handler(stickyRetrieveHEAD.ServeHTTP)
		stickysvr.MountListHandler(mux, stickyServer.List, readServicesCORSPolicy)
		stickysvr.MountCreateHandler(mux, stickyServer.Create, readServicesCORSPolicy)
		stickysvr.MountRetrieveHandler(mux, stickyServer.Retrieve, readServicesCORSPolicy)
		stickysvr.MountUpdateHandler(mux, stickyServer.Update, readServicesCORSPolicy)
		stickysvr.MountDeleteHandler(mux, stickyServer.Delete, readServicesCORSPolicy)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/stickies/", stickyListHEAD)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/stickies/{pk}/", stickyRetrieveHEAD)
		listOptions := stickyListOptionsHandler(stickyCompatibilityMiddleware)
		detailOptions := djangoProtocolOptionsWithoutActionsHandler(stickyCompatibilityMiddleware, "Sticky Instance")
		if pending.stickyWorkspaceAuthorizer != nil {
			listOptions = stickyWorkspaceAuthorizationMiddleware(mux, pending.stickyWorkspaceAuthorizer, listOptions)
			detailOptions = stickyWorkspaceAuthorizationMiddleware(mux, pending.stickyWorkspaceAuthorizer, detailOptions)
		}
		if authenticate != nil {
			listOptions = stickyAuthenticationMiddleware(authenticate)(listOptions)
			detailOptions = stickyAuthenticationMiddleware(authenticate)(detailOptions)
		}
		listOptions = djangoProtocolOptionsJSONNegotiationMiddleware(listOptions)
		detailOptions = djangoProtocolOptionsJSONNegotiationMiddleware(detailOptions)
		detailOptions = stickyDetailUUIDMiddleware(mux, detailOptions)
		listOptions = stickyProtocolHeadersMiddleware(listOptions)
		detailOptions = stickyProtocolHeadersMiddleware(detailOptions)
		listOptions = djangoProtocolCORSOptionsHandler(readServicesCORSPolicy, []string{http.MethodGet, http.MethodPost}, listOptions)
		detailOptions = djangoProtocolCORSOptionsHandler(readServicesCORSPolicy, []string{http.MethodGet, http.MethodPatch, http.MethodDelete}, detailOptions)
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/stickies/", listOptions)
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/stickies/{pk}/", detailOptions)
		for _, mount := range stickyServer.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "stickies"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if workspaceEndpoints != nil {
		workspaceServer := workspaceslugsvr.New(workspaceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		workspaceServer.Use(workspaceSlugCompatibilityMiddleware)
		workspaceslugsvr.Mount(mux, workspaceServer)
		// Django exposes HEAD/OPTIONS as protocol-level companions to this GET
		// route. They are not separate application operations.
		mountProtocolHandler(mux, "HEAD", "/api/workspace-slug-check/", workspaceSlugHeadHandler(workspaceServer.Check))
		mountProtocolHandler(mux, "OPTIONS", "/api/workspace-slug-check/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range workspaceServer.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "workspace_slug_check"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if instanceWorkspaceEndpoints != nil {
		instanceWorkspaceServer := instanceworkspaceslugsvr.New(instanceWorkspaceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		instanceWorkspaceServer.Use(workspaceSlugCompatibilityMiddleware)
		instanceworkspaceslugsvr.Mount(mux, instanceWorkspaceServer)
		mountProtocolHandler(mux, "HEAD", "/api/instances/workspace-slug-check/", workspaceSlugHeadHandler(instanceWorkspaceServer.Check))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/workspace-slug-check/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range instanceWorkspaceServer.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "instance_workspace_slug_check"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if userInstanceAdminEndpoints != nil {
		userInstanceAdminServer := userinstanceadminsvr.New(userInstanceAdminEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		userInstanceAdminServer.Use(djangoJSONCompatibilityMiddleware)
		userinstanceadminsvr.Mount(mux, userInstanceAdminServer)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/instance-admin/", djangoJSONHeadHandler(userInstanceAdminServer.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/instance-admin/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range userInstanceAdminServer.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "user_instance_admin"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if userProfileFlagsEndpoints != nil {
		userProfileFlagsServer := userprofileflagssvr.New(userProfileFlagsEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		userProfileFlagsServer.Use(userProfileFlagsCompatibilityMiddleware)
		userprofileflagssvr.Mount(mux, userProfileFlagsServer)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/onboard/", http.HandlerFunc(userProfileFlagsHeadHandler))
		mountProtocolHandler(mux, "HEAD", "/api/users/me/tour-completed/", http.HandlerFunc(userProfileFlagsHeadHandler))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/onboard/", userProfileFlagsOptionsHandler("Update User On Boarded Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/tour-completed/", userProfileFlagsOptionsHandler("Update User Tour Completed Endpoint"))
		for _, mount := range userProfileFlagsServer.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "user_profile_flags"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if notificationEndpoints != nil {
		server := usernotificationpreferencessvr.New(notificationEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userNotificationPreferencesCompatibilityMiddleware)
		usernotificationpreferencessvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/notification-preferences/", userNotificationPreferencesHeadHandler(server.Get))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/notification-preferences/", http.HandlerFunc(userNotificationPreferencesOptionsHandler))
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "user_notification_preferences"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if userAccountsEndpoints != nil {
		server := useraccountssvr.New(userAccountsEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userAccountsCompatibilityMiddleware)
		useraccountssvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/accounts/", userAccountsHeadHandler(server.List))
		mountProtocolHandler(mux, "HEAD", "/api/users/me/accounts/{pk}/", userAccountsHeadHandler(server.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/accounts/", userAccountsOptionsHandler("Account Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/accounts/{pk}/", userAccountsOptionsHandler("Account Endpoint"))
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "user_accounts"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if userAPITokensEndpoints != nil {
		server := userapitokenssvr.New(userAPITokensEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userAPITokensCompatibilityMiddleware)
		userapitokenssvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/api-tokens/", userAPITokensHeadHandler(server.List))
		mountProtocolHandler(mux, "HEAD", "/api/users/api-tokens/{pk}/", userAPITokensHeadHandler(server.Retrieve))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/api-tokens/", userAPITokensOptionsHandler("API Token Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/api-tokens/{pk}/", userAPITokensOptionsHandler("API Token Endpoint"))
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "user_api_tokens"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if instanceAdminSessionEndpoints != nil {
		server := instanceadminsessionsvr.New(instanceAdminSessionEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(djangoJSONCompatibilityMiddleware)
		instanceadminsessionsvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/instances/admins/me/", djangoJSONHeadHandler(server.Me))
		mountProtocolHandler(mux, "HEAD", "/api/instances/admins/session/", djangoJSONHeadHandler(server.Session))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/admins/me/", http.HandlerFunc(workspaceSlugOptionsHandler))
		mountProtocolHandler(mux, "OPTIONS", "/api/instances/admins/session/", http.HandlerFunc(workspaceSlugOptionsHandler))
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "instance_admin_session"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if quickLinkEndpoints != nil {
		server := workspacequicklinkssvr.New(quickLinkEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceContentCompatibilityMiddleware("quick-links"))
		workspacequicklinkssvr.Mount(mux, server)
		mountWorkspaceContentProtocolRoutes(mux, "quick-links", "Workspace Quick Link", server.List, server.Retrieve)
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "workspace_quick_links"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if themeEndpoints != nil {
		server := workspacethemesvr.New(themeEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceContentCompatibilityMiddleware("workspace-themes"))
		workspacethemesvr.Mount(mux, server)
		mountWorkspaceContentProtocolRoutes(mux, "workspace-themes", "Workspace Theme", server.List, server.Retrieve)
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "workspace_themes"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if homePreferenceEndpoints != nil {
		server := workspacehomepreferencessvr.New(homePreferenceEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceHomePreferencesCompatibilityMiddleware)
		workspacehomepreferencessvr.Mount(mux, server)
		mountWorkspaceHomePreferencesProtocolRoutes(mux, server.List, server.Retrieve)
	}
	if memberMeEndpoints != nil {
		server := workspacemembermesvr.New(memberMeEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceMemberMeCompatibilityMiddleware)
		workspacemembermesvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/workspace-members/me/", workspaceMemberMeHeadHandler(server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/workspace-members/me/", workspaceMemberMeOptionsHandler())
	}
	if analyticViewEndpoints != nil {
		server := analyticviewssvr.New(analyticViewEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(analyticViewsCompatibilityMiddleware)
		analyticviewssvr.Mount(mux, server)
		mountAnalyticViewsProtocolRoutes(mux, server.List, server.Retrieve)
	}
	if favoriteEndpoint != nil {
		server := workspacefavoritessvr.New(favoriteEndpoint, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceFavoritesCompatibilityMiddleware)
		workspacefavoritessvr.Mount(mux, server)
		mountWorkspaceFavoritesProtocolRoutes(mux, server.List, server.Retrieve, server.Group)
	}
	if entityEndpoint != nil {
		server := entityfavoritessvr.New(entityEndpoint, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(entityFavoritesCompatibilityMiddleware)
		entityfavoritessvr.Mount(mux, server)
		mountEntityFavoritesProtocolRoutes(mux, server)
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "entity_favorites"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if recentVisitEndpoints != nil {
		server := workspacerecentvisitssvr.New(recentVisitEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceRecentVisitsCompatibilityMiddleware)
		workspacerecentvisitssvr.Mount(mux, server)
		mountWorkspaceRecentVisitsProtocolRoutes(mux, server.List)
		for _, mount := range server.Mounts {
			log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "workspace_recent_visits"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
		}
	}
	if lastVisitedEndpoints != nil {
		server := lastvisitedworkspacesvr.New(lastVisitedEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(lastVisitedWorkspaceCompatibilityMiddleware)
		lastvisitedworkspacesvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/last-visited-workspace/", lastVisitedWorkspaceHeadHandler(server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/last-visited-workspace/", lastVisitedWorkspaceOptionsHandler())
	}
	if pending.userIdentity != nil {
		bodySizeLimit := pending.userIdentityBodyLimit
		if !pending.userIdentityBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		server := useridentitysvr.New(pending.userIdentity, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userIdentityCompatibilityMiddleware)
		server.Me = userMeCacheControlMiddleware(server.Me)
		server.Me = userMeReadCompatibilityMiddleware(
			pending.userIdentitySession,
			pending.userIdentityCORS,
			pending.userIdentityCookie,
			bodySizeLimit,
			pending.userIdentityActivity,
			pending.userIdentitySecret,
		)(server.Me)
		useridentitysvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/session/", djangoProtocolHeadHandler(userIdentityCompatibilityMiddleware, server.Session))
		loomhttp.MountDerivedHead(mux, "/api/users/me/", userIdentityCompatibilityMiddleware(loomhttp.AsHandlerFunc(server.Me)))
		mountProtocolHandler(mux, http.MethodHead, "/api/users/me/profile/", djangoProtocolHeadHandler(userIdentityCompatibilityMiddleware, server.Profile))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/session/", userIdentityOptionsHandler("User Session Endpoint", false))
		if pending.userAccountWrites == nil {
			//loom:vet ignore route-outside-design -- Django OPTIONS metadata stays on the identity service only when the account-write service is absent.
			loomhttp.MountHandler(mux, http.MethodOptions, "/api/users/me/", djangoProtocolOptionsWithoutActionsHandler(userIdentityCompatibilityMiddleware, "User Endpoint"))
		}
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/me/profile/", userIdentityOptionsHandler("Profile Endpoint", true))
	}
	if pending.userProperties != nil {
		server := userpropertiessvr.New(pending.userProperties, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userPropertiesCompatibilityMiddleware)
		userpropertiessvr.Mount(mux, server)
		mountUserPropertiesProtocolRoutes(mux, server)
	}
	if pending.workspaceMetadata != nil {
		server := workspacemetadatasvr.New(pending.workspaceMetadata, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceMetadataCompatibilityMiddleware)
		workspacemetadatasvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/labels/", djangoProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.Labels))
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/states/", djangoProtocolHeadHandler(workspaceMetadataCompatibilityMiddleware, server.States))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/labels/", workspaceMetadataOptionsHandler("Workspace Labels Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/states/", workspaceMetadataOptionsHandler("Workspace States Endpoint"))
	}
	if pending.projectIdentifiers != nil {
		server := projectidentifierssvr.New(pending.projectIdentifiers, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectIdentifiersCompatibilityMiddleware)
		projectidentifierssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/project-identifiers/", djangoProtocolHeadHandler(projectIdentifiersCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/project-identifiers/", djangoProtocolOptionsHandler(projectIdentifiersCompatibilityMiddleware, "Project Identifier Endpoint", false))
	}
	if pending.projectMembershipReads != nil {
		server := projectmembershipreadssvr.New(pending.projectMembershipReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectMembershipReadsCompatibilityMiddleware)
		projectmembershipreadssvr.Mount(mux, server)
		mountProjectMembershipReadsProtocolRoutes(mux, server, pending.projectMembershipMetadata)
	}
	if pending.projectStateReads != nil {
		server := projectstatereadssvr.New(pending.projectStateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectStateReadsCompatibilityMiddleware)
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		writes := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server.Create = writes(server.Create)
		server.Update = writes(server.Update)
		server.Delete = writes(server.Delete)
		server.MarkDefault = writes(server.MarkDefault)
		projectstatereadssvr.Mount(mux, server)
		mountProjectStateReadsProtocolRoutes(mux, server, pending.projectStateMetadata)
	}
	if pending.projectLabelReads != nil {
		server := projectlabelreadssvr.New(pending.projectLabelReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectLabelReadsCompatibilityMiddleware)
		projectlabelreadssvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/issue-labels/", server.List)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/", server.Detail)
	}
	var publicProjectMetadataRouteCompatibility func(http.Handler) http.Handler
	var publicFeedbackRouteCompatibility func(http.Handler) http.Handler
	if pending.publicProjectMetadataReads != nil {
		if pending.publicProjectMetadataThrottle == nil {
			panic("public project metadata throttle is required")
		}
		bodySizeLimit := pending.publicProjectMetadataBodyLimit
		if !pending.publicProjectMetadataBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityMiddleware := publicProjectMetadataReadsCompatibilityMiddleware(
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			true,
			true,
		)
		server := publicprojectmetadatareadssvr.New(pending.publicProjectMetadataReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(compatibilityMiddleware)
		publicprojectmetadatareadssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/cycles/", djangoBufferedProtocolHeadHandler(server.Cycles))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/labels/", djangoBufferedProtocolHeadHandler(server.Labels))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/members/", djangoBufferedProtocolHeadHandler(server.Members))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/meta/", djangoBufferedProtocolHeadHandler(server.Meta))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/modules/", djangoBufferedProtocolHeadHandler(server.Modules))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/settings/", djangoBufferedProtocolHeadHandler(server.Settings))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/states/", djangoBufferedProtocolHeadHandler(server.States))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/workspaces/{slug}/projects/{project_id}/anchor/", djangoBufferedProtocolHeadHandler(server.ProjectAnchor))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/cycles/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Cycles Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/labels/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Labels Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/members/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Members Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/meta/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Meta Data Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/modules/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Modules Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/settings/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Deploy Board Public Settings Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/states/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project States Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/workspaces/{slug}/projects/{project_id}/anchor/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Workspace Project Anchor Endpoint"))
		publicProjectMetadataRouteCompatibility = publicProjectMetadataRouteCompatibilityMiddleware(
			compatibilityMiddleware,
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			dbg,
		)
	}
	if pending.publicFeedback != nil {
		if pending.publicProjectMetadataThrottle == nil {
			panic("public feedback session throttle is required")
		}
		bodySizeLimit := pending.publicProjectMetadataBodyLimit
		if !pending.publicProjectMetadataBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityMiddleware := publicProjectMetadataReadsCompatibilityMiddleware(
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			true,
			true,
			publicProjectMetadataRoutePolicy{
				allowMethods:    publicFeedbackAllowMethods,
				requiresSession: publicFeedbackRouteRequiresSession,
			},
		)
		publicFeedbackRouteCompatibility = publicFeedbackRouteCompatibilityMiddleware(
			compatibilityMiddleware,
			publicProjectMetadataNotFoundHandler(
				pending.publicProjectMetadataThrottle,
				pending.publicProjectMetadataCORS,
				pending.publicProjectMetadataCookie,
				pending.publicProjectMetadataActivity,
				pending.publicProjectMetadataSecret,
				bodySizeLimit,
				dbg,
				true,
			),
			publicProjectMetadataNotFoundHandler(
				pending.publicProjectMetadataThrottle,
				pending.publicProjectMetadataCORS,
				pending.publicProjectMetadataCookie,
				pending.publicProjectMetadataActivity,
				pending.publicProjectMetadataSecret,
				bodySizeLimit,
				dbg,
				false,
			),
		)
		server := publicfeedbacksvr.New(pending.publicFeedback, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(compatibilityMiddleware)
		publicfeedbacksvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/", server.IssueList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/{issue_id}/", server.IssueDetail)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", server.VoteList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", server.InboxList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", server.IntakeList)
		loomhttp.MountDerivedHead(mux, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", server.IntakeDetail)
		loomhttp.MountDerivedHead(mux, "/api/public/workspaces/{slug}/project-boards/", server.WorkspaceProjectBoards)
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/", false))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/{issue_id}/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/{issue_id}/", false))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/issues/{issue_id}/votes/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/", true))
		//loom:vet ignore route-outside-design -- Django public feedback OPTIONS metadata remains a protocol companion, not a business operation.
		loomhttp.MountHandler(mux, http.MethodOptions, "/api/public/workspaces/{slug}/project-boards/", publicFeedbackOptionsHandler(compatibilityMiddleware, "/api/public/workspaces/{slug}/project-boards/", false))
	}
	if pending.standardViews != nil {
		server := standardviewssvr.New(pending.standardViews, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(standardViewsCompatibilityMiddleware)
		standardviewssvr.Mount(mux, server)
		mountStandardViewsProtocolRoutes(mux, server)
	}
	if pending.userSettings != nil {
		server := usersettingssvr.New(pending.userSettings, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userSettingsCompatibilityMiddleware)
		usersettingssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/me/settings/", djangoProtocolHeadHandler(userSettingsCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/me/settings/", djangoProtocolOptionsHandler(userSettingsCompatibilityMiddleware, "User Settings Endpoint", false))
	}
	if pending.workspaceSidebarPreferences != nil {
		server := workspacesidebarpreferencessvr.New(pending.workspaceSidebarPreferences, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(workspaceSidebarPreferencesCompatibilityMiddleware)
		workspacesidebarpreferencessvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/workspaces/{slug}/sidebar-preferences/", djangoProtocolHeadHandler(workspaceSidebarPreferencesCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/sidebar-preferences/", djangoProtocolOptionsHandler(workspaceSidebarPreferencesCompatibilityMiddleware, "Workspace User Preference Endpoint", true))
	}
	assetsMountedMethods := make(map[assetsLifecycleMountedRoute]struct{})
	var sessionProtocolMiddleware func(http.Handler) http.Handler
	var instanceProtocolMiddleware func(http.Handler) http.Handler
	stateProtocol, _ := pending.projectStateMetadata.(moduleLifecycleSessionProtocolAuthorizer)
	if pending.workspaceUserInsights != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		stack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server := workspaceuserinsightssvr.New(pending.workspaceUserInsights, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		workspaceuserinsightssvr.Mount(mux, server)
		mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
			{"/api/workspaces/{slug}/user-activity/{user_id}/", server.Activity},
			{"/api/workspaces/{slug}/user-activity/{user_id}/export/", server.ExportActivity},
			{"/api/workspaces/{slug}/user-profile/{user_id}/", server.Profile},
			{"/api/workspaces/{slug}/user-stats/{user_id}/", server.Stats},
		}, readServicesCORSPolicy)
	}
	if pending.workspaceUserInsights != nil && sessionProtocolMiddleware == nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		stack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		authorizer := labelSessionProtocolDispatch{userInsights: pending.workspaceUserInsightsMetadata}
		sessionProtocolMiddleware = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, authorizer, readServicesCORSPolicy, stack, pending.readServicesCookie, dbg, options)
	}
	mountWorkItemTypes(mux, pending, readServicesCORSPolicy, errorHandler)
	mountMCPOAuthLinks(mux, pending, readServicesCORSPolicy, errorHandler)
	if pending.accountAuthentication != nil || pending.invitationsMembership != nil || pending.assetsLifecycle != nil || pending.membershipLifecycle != nil || pending.notifications != nil || pending.notificationActions != nil || pending.projectLabelWrites != nil || pending.userAccountWrites != nil || pending.workspaceLifecycle != nil || pending.workspaceExports != nil || pending.personalDashboard != nil || pending.workspaceAnalytics != nil || pending.workspaceSearch != nil || pending.externalIntegrations != nil || pending.projectMemberPreferences != nil || pending.projectIntakeWrites != nil || pending.projectConfiguration != nil || pending.customViewCreation != nil || pending.workspaceWebhooks != nil || pending.cycleLifecycle != nil || pending.moduleLifecycle != nil || pending.projectLifecycle != nil || pending.projectPublication != nil || pending.workItemHistory != nil || pending.pageReadsMetadata != nil || stateProtocol != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		sessionStack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		stack := func(next http.Handler) http.Handler {
			return sessionStack(assetDuplicateThrottleMiddleware(pending.assetsDuplicateThrottle)(next))
		}
		if pending.notifications != nil {
			server := notificationssvr.New(pending.notifications, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			notificationssvr.Mount(mux, server)
			mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
				{"/api/workspaces/{slug}/users/notifications/", server.List},
				{"/api/workspaces/{slug}/users/notifications/unread/", server.Counts},
				{"/api/workspaces/{slug}/users/notifications/{pk}/", server.Get},
			}, readServicesCORSPolicy)

		}
		if pending.notificationActions != nil {
			server := notificationactionssvr.New(pending.notificationActions, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			notificationactionssvr.MountMarkAllReadHandler(mux, server.MarkAllRead, readServicesCORSPolicy)
			notificationactionssvr.MountUpdateHandler(mux, server.Update, readServicesCORSPolicy)
			notificationactionssvr.MountDeleteHandler(mux, server.Delete, readServicesCORSPolicy)
			notificationactionssvr.MountUnarchiveHandler(mux, server.Unarchive, readServicesCORSPolicy)
			notificationactionssvr.MountArchiveHandler(mux, server.Archive, readServicesCORSPolicy)
			notificationactionssvr.MountMarkUnreadHandler(mux, server.MarkUnread, readServicesCORSPolicy)
			notificationactionssvr.MountMarkReadHandler(mux, server.MarkRead, readServicesCORSPolicy)
		}
		if pending.projectLabelWrites != nil {
			server := labelwritessvr.New(pending.projectLabelWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			labelwritessvr.Mount(mux, server)
		}
		if pending.workspaceExports != nil {
			server := exportssvr.New(pending.workspaceExports, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			exportssvr.Mount(mux, server)
			loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/export-issues/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.GetIssueExports)))
		}
		if pending.personalDashboard != nil {
			server := personalsvr.New(pending.personalDashboard, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			personalsvr.Mount(mux, server)
			mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
				{"/api/users/me/workspaces/{slug}/activity-graph/", server.ActivityGraph},
				{"/api/users/me/workspaces/{slug}/dashboard/", server.Dashboard},
				{"/api/users/me/workspaces/{slug}/issues-completed-graph/", server.IssuesCompletedGraph},
			}, readServicesCORSPolicy)
		}
		if pending.workspaceSearch != nil {
			server := searchsvr.New(pending.workspaceSearch, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			searchsvr.Mount(mux, server)
			mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
				{"/api/workspaces/{slug}/search/", server.GlobalSearch},
				{"/api/workspaces/{slug}/entity-search/", server.EntitySearch},
			}, readServicesCORSPolicy)
		}
		if pending.workspaceAnalytics != nil {
			server := analyticssvr.New(pending.workspaceAnalytics, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			analyticssvr.Mount(mux, server)
			mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
				{"/api/workspaces/{slug}/advance-analytics/", server.AdvanceAnalytics},
				{"/api/workspaces/{slug}/advance-analytics-stats/", server.AdvanceAnalyticsStats},
				{"/api/workspaces/{slug}/advance-analytics-charts/", server.AdvanceAnalyticsCharts},
				{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics/", server.ProjectAdvanceAnalytics},
				{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-stats/", server.ProjectAdvanceAnalyticsStats},
				{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-charts/", server.ProjectAdvanceAnalyticsCharts},
				{"/api/workspaces/{slug}/analytics/", server.Analytics},
				{"/api/workspaces/{slug}/default-analytics/", server.DefaultAnalytics},
				{"/api/workspaces/{slug}/saved-analytic-view/{analytic_id}/", server.SavedAnalyticView},
			}, readServicesCORSPolicy)
		}
		if pending.externalIntegrations != nil {
			server := externalsvr.New(pending.externalIntegrations, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			externalsvr.Mount(mux, server)
			loomhttp.MountDerivedHead(mux, "/api/unsplash/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Unsplash)))
		}
		if pending.authRedirects != nil {
			server := authredirectssvr.New(pending.authRedirects, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(authRedirectSecurityMiddleware)
			server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, authRedirectSessionPath))
			authredirectssvr.Mount(mux, server)
			for _, mount := range server.Mounts {
				log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "auth_redirects"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
			}
		}
		if pending.accountAuthentication != nil {
			server := accountauthsvr.New(pending.accountAuthentication, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			bodySizeLimit := pending.activityReadsBodyLimit
			if bodySizeLimit <= 0 {
				bodySizeLimit = defaultFileSizeLimit
			}
			options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: bodySizeLimit}
			server.Use(accountAuthenticationGzipMiddleware)
			server.Use(passwordLoginCSRFCookieMiddleware(pending.csrfTokenCookie))
			server.Use(accountAuthenticationProtocolMiddleware)
			server.Use(accountAuthenticationCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins))
			server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, accountAuthenticationSessionPath))
			server.Use(activityReadsAuditFallbackMiddleware(options))
			server.Use(activityReadsRequestSizeMiddleware(bodySizeLimit))
			accountauthsvr.Mount(mux, server)
			for _, mount := range server.Mounts {
				log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "account_authentication"}, log.KV{K: "method", V: mount.Method}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
			}
		}
		if pending.oauthRedirects != nil {
			server := oauthredirectssvr.New(pending.oauthRedirects, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(authRedirectSecurityMiddleware)
			server.Use(djangoAuthSessionMiddleware(pending.readServicesAuthSession, pending.readServicesAuthCookie, authRedirectSessionPath))
			oauthredirectssvr.Mount(mux, server)
			for _, mount := range server.Mounts {
				log.Print(ctx, log.KV{K: "msg", V: "HTTP route mounted"}, log.KV{K: "service", V: "oauth_redirects"}, log.KV{K: "http_method", V: mount.Verb}, log.KV{K: "http_path", V: mount.Pattern})
			}
		}
		if pending.userAccountWrites != nil {
			server := useraccountwritessvr.New(pending.userAccountWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			useraccountwritessvr.Mount(mux, server)
		}

		if pending.projectMemberPreferences != nil {
			server := preferencessvr.New(pending.projectMemberPreferences, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			preferencessvr.Mount(mux, server)
		}
		if pending.projectIntakeWrites != nil {
			server := intakewritessvr.New(pending.projectIntakeWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			intakewritessvr.Mount(mux, server)
		}
		if pending.projectConfiguration != nil {
			server := projectconfigurationsvr.New(pending.projectConfiguration, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			apiStack := activityReadsMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
			server.Use(func(next http.Handler) http.Handler {
				sessionHandler, apiHandler := stack(next), apiStack(next)
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					if strings.HasPrefix(request.URL.Path, "/api/v1/") {
						apiHandler.ServeHTTP(response, request)
						return
					}
					sessionHandler.ServeHTTP(response, request)
				})
			})
			projectconfigurationsvr.Mount(mux, server)
			loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/project-estimates/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.ListProjectEstimates)))
		}
		if pending.customViewCreation != nil {
			mountCustomViewCreation(mux, pending.customViewCreation, options, pending.readServicesSession, pending.readServicesCookie, readServicesCORSPolicy)
		}
		if pending.workspaceWebhooks != nil {
			server := workspacewebhookssvr.New(pending.workspaceWebhooks, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			server.Use(stack)
			workspacewebhookssvr.Mount(mux, server)
			loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhooks/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.List)))
			loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhooks/{pk}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Detail)))
			loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/webhook-logs/{webhook_id}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(server.Logs)))
		}
		authorizer := workItemHistorySessionProtocolDispatch{projectPublicationSessionProtocolDispatch: projectPublicationSessionProtocolDispatch{labelSessionProtocolDispatch: labelSessionProtocolDispatch{exports: pending.workspaceExportsMetadata, personal: pending.personalDashboardMetadata, search: pending.workspaceSearchMetadata, analytics: pending.workspaceAnalyticsMetadata, userAccount: pending.userAccountWritesMetadata, workspaceLifecycle: pending.workspaceLifecycleMetadata, external: pending.externalIntegrationsMetadata, preferences: pending.projectMemberPreferencesMetadata, intakeWrites: pending.projectIntakeWritesMetadata, configuration: pending.projectConfigurationMetadata, webhooks: pending.workspaceWebhooksMetadata, notifications: pending.notificationsMetadata, labels: pending.projectLabelWritesMetadata, membership: pending.membershipLifecycleMetadata, assets: pending.assetsSessionMetadata, assetsMethods: assetsMountedMethods, invitations: pending.invitationsMetadata, cycles: pending.cycleLifecycleMetadata, modules: pending.moduleLifecycleMetadata, projects: pending.projectLifecycleMetadata, pages: pending.pageReadsMetadata, states: stateProtocol, userInsights: pending.workspaceUserInsightsMetadata, customViews: pending.customViewCreationMetadata}, publication: pending.projectPublicationMetadata}, history: pending.workItemHistoryMetadata}
		sessionProtocolMiddleware = newSessionServiceProtocolMiddleware(notificationProtocolMetadata, accountAuthenticationSessionProtocolDispatch{account: pending.accountAuthenticationMetadata, fallback: authorizer}, readServicesCORSPolicy, stack, pending.readServicesCookie, dbg, options, pending.accountAuthenticationCSRFTrustedOrigins)
	}
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
	mountMilestoneHTTP(mux, pending, errorHandler, readServicesCORSPolicy)
	var activityOptions *apiV1CompatibilityOptions
	var activityProtocolMiddleware func(http.Handler) http.Handler
	if pending.activityReads != nil {
		activityOptions = &apiV1CompatibilityOptions{
			activityLogger: pending.activityReadsActivity,
			activitySecret: pending.activityReadsSecret,
			bodySizeLimit:  pending.activityReadsBodyLimit,
		}
		server := activityreadssvr.New(pending.activityReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		stack := activityReadsMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
		server.Use(stack)
		activityreadssvr.Mount(mux, server)
		mountActivityReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
		if pending.apiV1Stickies != nil {
			stickies := apiv1stickiessvr.New(pending.apiV1Stickies, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			stickyStack := apiV1StickyActivityMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
			stickies.Use(stickyStack)
			apiv1stickiessvr.Mount(mux, stickies)
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies.json", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies.json/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.List)))
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}.json", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
			loomhttp.MountDerivedHead(mux, "/api/v1/workspaces/{slug}/stickies/{pk}.json/", readServicesCORSPolicy.Handler(loomhttp.AsHandlerFunc(stickies.Retrieve)))
		}
		if pending.apiUserIdentity != nil {
			identity := apiuseridentitysvr.New(pending.apiUserIdentity, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			identity.Use(stack)
			apiuseridentitysvr.MountGetAPIV1UsersMeHandler(mux, identity.GetAPIV1UsersMe, readServicesCORSPolicy)
		}

		if pending.apiCycleReads != nil {
			cycles := apicyclereadssvr.New(pending.apiCycleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			cycles.Use(stack)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCycles, readServicesCORSPolicy)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLiteHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLite, readServicesCORSPolicy)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCycles, readServicesCORSPolicy)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues, readServicesCORSPolicy)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID, readServicesCORSPolicy)
			apicyclereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPkHandler(mux, cycles.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk, readServicesCORSPolicy)
		}
		if pending.apiModuleReads != nil {
			modules := apimodulereadssvr.New(pending.apiModuleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			modules.Use(stack)
			apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModules, readServicesCORSPolicy)
			apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLiteHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLite, readServicesCORSPolicy)
			apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModules, readServicesCORSPolicy)
			apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues, readServicesCORSPolicy)
			apimodulereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkHandler(mux, modules.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk, readServicesCORSPolicy)
		}
		if pending.apiLabelReads != nil {
			labels := apilabelreadssvr.New(pending.apiLabelReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			labels.Use(stack)
			apilabelreadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsHandler(mux, labels.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels, readServicesCORSPolicy)
			apilabelreadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPkHandler(mux, labels.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPk, readServicesCORSPolicy)
		}
		if pending.membershipLifecycle != nil {
			membership := membershiplifecyclesvr.New(pending.membershipLifecycle, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			apiStack := stack
			sessionStack := sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
			membership.Use(func(next http.Handler) http.Handler {
				api := apiStack(next)
				session := sessionStack(next)
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/api/v1/") {
						api.ServeHTTP(w, r)
					} else {
						session.ServeHTTP(w, r)
					}
				})
			})
			membershiplifecyclesvr.Mount(mux, membership)
		}
		if pending.apiStateReads != nil {
			states := apistatereadssvr.New(pending.apiStateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			states.Use(stack)
			apistatereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDStatesHandler(mux, states.GetAPIV1WorkspacesBySlugProjectsByProjectIDStates, readServicesCORSPolicy)
			apistatereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
			apistatereadssvr.MountCreateAPIV1WorkspacesBySlugProjectsByProjectIDStatesHandler(mux, states.CreateAPIV1WorkspacesBySlugProjectsByProjectIDStates, readServicesCORSPolicy)
			apistatereadssvr.MountUpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
			apistatereadssvr.MountDeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
		}
		if pending.workItemCore != nil {
			core := workitemsvr.New(pending.workItemCore, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			core.Use(sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
			workitemsvr.Mount(mux, core)
			mountWorkItemHeads(mux, core, readServicesCORSPolicy)
		}
		if pending.ticketCollaboration != nil {
			collaboration := collaborationsvr.New(pending.ticketCollaboration, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
			collaboration.Use(sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
			collaborationsvr.Mount(mux, collaboration)
			mountCollaborationHeads(mux, collaboration, readServicesCORSPolicy)
		}
		activityProtocolMiddleware = newActivityReadsProtocolMiddleware(membershipActivityProtocolDispatch{configuration: pending.projectConfigurationMetadata, ActivityReadsProtocolAuthorizer: pending.activityReadsMetadata, membership: pending.membershipLifecycleMetadata, assets: pending.assetsAPIMetadata, invitations: pending.invitationsMetadata, projects: pending.projectLifecycleMetadata, stickies: pending.apiV1StickiesProtocol}, readServicesCORSPolicy, stack, *activityOptions, pending.readServicesCookie, pending.activityReadsBodyLimit, dbg)
	}
	if pending.workspaceProjectReads != nil {
		server := workspaceprojectreadssvr.New(pending.workspaceProjectReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		workspaceprojectreadssvr.Mount(mux, server)
		mountWorkspaceProjectReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.cycleReads != nil {
		server := cyclereadssvr.New(pending.cycleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		cyclereadssvr.Mount(mux, server)
		mountCycleReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.cycleLifecycle != nil {
		options := apiV1CompatibilityOptions{}
		if activityOptions != nil {
			options = *activityOptions
		} else {
			options = apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		}
		mountCycleLifecycle(mux, pending.cycleLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}

	if pending.projectLifecycle != nil {
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		if activityOptions != nil {
			options = *activityOptions
		}
		mountProjectLifecycle(mux, pending.projectLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
	if pending.workspaceLifecycle != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountWorkspaceLifecycle(mux, pending.workspaceLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
	if pending.projectPublication != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountProjectPublication(mux, pending.projectPublication, pending.projectPublicationMetadata, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
	if pending.moduleLifecycle != nil {
		options := apiV1CompatibilityOptions{}
		if activityOptions != nil {
			options = *activityOptions
		} else {
			options = apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		}
		mountModuleLifecycle(mux, pending.moduleLifecycle, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
	if pending.moduleReads != nil {
		server := modulereadssvr.New(pending.moduleReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		modulereadssvr.Mount(mux, server)
		mountModuleReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.estimateReads != nil {
		server := estimatereadssvr.New(pending.estimateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		estimatereadssvr.Mount(mux, server)
		mountEstimateReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.pageReads != nil {
		pending.pageReads.List = markDjangoPageInternalError(pending.pageReads.List)
		pending.pageReads.Detail = markDjangoPageInternalError(pending.pageReads.Detail)
		server := pagereadssvr.New(pending.pageReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		if activityOptions != nil {
			options = *activityOptions
		}
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(sessionServicePrincipalLifecycle(pending.readServicesSession), pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		// Keep write transport boundaries outside middleware that can return
		// early, while leaving gzip outermost so audit records the plain response.
		server.Use(pageLifecycleWriteTransportMiddleware(options))
		server.Use(readServicesGzipMiddleware)
		pagereadssvr.Mount(mux, server)
		mountPageReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.intakeInboxReads != nil {
		server := intakeinboxreadssvr.New(pending.intakeInboxReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		intakeinboxreadssvr.Mount(mux, server)
		mountIntakeInboxReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.workItemHistory != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		mountWorkItemHistory(mux, pending.workItemHistory, readServicesCORSPolicy, options, pending.readServicesSession, pending.readServicesCookie)
	}
	if pending.deployBoardReads != nil {
		server := deployboardreadssvr.New(pending.deployBoardReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		deployboardreadssvr.Mount(mux, server)
		mountDeployBoardReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
	if pending.invitationsMembership != nil {
		options := apiV1CompatibilityOptions{activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret, bodySizeLimit: pending.activityReadsBodyLimit}
		server := invitationssvr.New(pending.invitationsMembership, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		apiStack := activityReadsMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		sessionStack := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server.Use(func(next http.Handler) http.Handler {
			api, session := apiStack(next), sessionStack(next)
			return readServicesCORSPolicy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if strings.HasPrefix(request.URL.Path, "/api/v1/") {
					api.ServeHTTP(response, request)
				} else {
					session.ServeHTTP(response, request)
				}
			}))
		})
		invitationssvr.Mount(mux, server)
		mountInvitationsMembershipHeads(mux, server, readServicesCORSPolicy)
	}
	if pending.assetsLifecycle != nil {
		server := assetslifecyclesvr.New(pending.assetsLifecycle, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(assetLifecycleProtocolHeaders(server.Mounts))
		server.Use(assetRequestMetadataMiddleware)
		if activityOptions != nil {
			apiStack := assetLifecycleAPIV1Middleware(activityReadsMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie))
			sessionStack := sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
			server.Use(func(next http.Handler) http.Handler {
				api := apiStack(next)
				session := sessionStack(assetDuplicateThrottleMiddleware(pending.assetsDuplicateThrottle)(assetsSessionPermissionMiddleware(pending.assetsSessionMetadata)(draftAssetCSRFMiddleware(pending.accountAuthenticationCSRFTrustedOrigins)(next))))
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					if strings.HasPrefix(request.URL.Path, "/api/v1/") {
						api.ServeHTTP(response, request)
					} else {
						session.ServeHTTP(response, request)
					}
				})
			})
		}
		assetslifecyclesvr.Mount(mux, server)
		mountAssetsLifecycleDerivedHeads(mux, server)
		for route := range assetsLifecycleMountedRoutes(server) {
			assetsMountedMethods[route] = struct{}{}
		}
		for _, mount := range server.Mounts {
			log.Print(ctx,
				log.KV{K: "msg", V: "HTTP route mounted"},
				log.KV{K: "service", V: "assets_lifecycle"},
				log.KV{K: "method", V: mount.Method},
				log.KV{K: "http_method", V: mount.Verb},
				log.KV{K: "http_path", V: mount.Pattern},
			)
		}
	}
	for _, mount := range systemServer.Mounts {
		log.Print(ctx,
			log.KV{K: "msg", V: "HTTP route mounted"},
			log.KV{K: "service", V: "system"},
			log.KV{K: "method", V: mount.Method},
			log.KV{K: "http_method", V: mount.Verb},
			log.KV{K: "http_path", V: mount.Pattern},
		)
	}
	for _, mount := range csrfTokenServer.Mounts {
		log.Print(ctx,
			log.KV{K: "msg", V: "HTTP route mounted"},
			log.KV{K: "service", V: "csrf_token"},
			log.KV{K: "method", V: mount.Method},
			log.KV{K: "http_method", V: mount.Verb},
			log.KV{K: "http_path", V: mount.Pattern},
		)
	}
	for _, mount := range timezoneServer.Mounts {
		log.Print(ctx,
			log.KV{K: "msg", V: "HTTP route mounted"},
			log.KV{K: "service", V: "timezones"},
			log.KV{K: "method", V: mount.Method},
			log.KV{K: "http_method", V: mount.Verb},
			log.KV{K: "http_path", V: mount.Pattern},
		)
	}

	var handler http.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request = captureReadServicesOriginalRawQuery(request)
		request = normalizeReadServiceScalarQueryValues(normalizeDjangoTolerantQuery(request))
		mux.ServeHTTP(response, request)
	})
	if activityProtocolMiddleware != nil {
		handler = activityProtocolMiddleware(handler)
	}
	if sessionProtocolMiddleware != nil {
		handler = sessionProtocolMiddleware(handler)
	}
	if instanceProtocolMiddleware != nil {
		handler = instanceManagementHandler(instanceProtocolMiddleware(handler), handler)
	}
	if pending.userIdentity != nil || pending.userAccountWrites != nil {
		handler = userAccountMeSlashlessCompatibilityMiddleware(handler, dbg, readServicesCORSPolicy)
	}
	if pending.workItemCore != nil {
		handler = workItemResponseContentTypeMiddleware(handler)
		if pending.workItemProtocol != nil && activityOptions != nil {
			stack := sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
			handler = workItemProtocolMiddleware(pending.workItemProtocol, readServicesCORSPolicy, stack, pending.readServicesCookie)(handler)
		}
	}
	handler = readServicesCompatibilityMiddleware(handler)
	if pending.hasReadServices() {
		handler = readServicesDjangoPathConverterMiddleware(readServicesCORSPolicy, activityOptions)(handler)
	}
	if activityOptions != nil || pending.notifications != nil || pending.notificationActions != nil || pending.projectLabelWrites != nil || pending.userAccountWrites != nil || pending.workspaceLifecycle != nil || pending.workspaceExports != nil || pending.personalDashboard != nil || pending.workItemHistory != nil || pending.workspaceAnalytics != nil || pending.workspaceSearch != nil || pending.externalIntegrations != nil || pending.projectMemberPreferences != nil || pending.projectIntakeWrites != nil || pending.projectConfiguration != nil || pending.customViewCreation != nil || pending.workspaceWebhooks != nil || pending.instanceManagement != nil {
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
		handler = workItemIdentifierResolver404Middleware(handler, activityOptions, readServicesCORSPolicy)
	}
	handler = csrfTokenCORSPreflightMiddleware(handler)
	handler = djangoCORSRequestCompatibilityMiddleware(handler)
	if stickyEndpoints != nil {
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
	if publicProjectMetadataRouteCompatibility != nil {
		// Keep compatibility responses outside the ordinary request logger. Exact
		// paths are filtered above, while cleaned malformed paths never enter it.
		handler = publicProjectMetadataRouteCompatibility(handler)
	}
	if publicFeedbackRouteCompatibility != nil {
		handler = publicFeedbackRouteCompatibility(handler)
	}
	// The public metadata compatibility layer logs the inner Django response at
	// RequestLoggerMiddleware's position. This outer layer installs its context.
	handler = telemetry.ClientAddressSpanMiddleware(handler)
	handler = log.HTTP(ctx, log.WithDisableRequestID(), log.WithDisableRequestLogging())(handler)
	handler = loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(handler)
	return platform.RequestIDMiddleware(handler)
}

var publicMetadataCompatibilityPathPattern = regexp.MustCompile(`^(?:` + strings.TrimPrefix(strings.TrimSuffix(publicProjectMetadataAnyPathPattern.String(), "$"), "^") + `|` + strings.TrimPrefix(strings.TrimSuffix(publicFeedbackPathPattern.String(), "$"), "^") + `)$`)

// assetLifecycleProtocolHeaders applies the Django REST framework response
// headers at the mounted assets boundary. Allow comes from source protocol
// observations and documented method repairs, including on application errors.
func assetLifecycleProtocolHeaders(mounts []*assetslifecyclesvr.MountPoint) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			allow := assetLifecycleAllowedMethods(mounts, request.URL.Path)
			if allow != "" {
				response.Header().Set("Allow", allow)
			}
			response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			response.Header().Set("Referrer-Policy", "same-origin")
			response.Header().Set("X-Content-Type-Options", "nosniff")
			response.Header().Set("X-Frame-Options", "DENY")
			mergeVary(response.Header(), "Cookie", "origin")
			next.ServeHTTP(response, request)
		})
	}
}

func assetLifecycleAllowedMethods(mounts []*assetslifecyclesvr.MountPoint, path string) string {
	for pattern, protocol := range assetsProtocolMetadata {
		if assetLifecyclePathMatches(pattern, path) {
			return protocol.Allow
		}
	}
	return ""
}

func assetLifecyclePathMatches(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if pathParts[index] == "" {
				return false
			}
			parameter := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			if strings.HasSuffix(parameter, "_id") || parameter == "id" || parameter == "pk" {
				if _, err := uuid.Parse(pathParts[index]); err != nil {
					return false
				}
			}
			continue
		}
		if part != pathParts[index] {
			return false
		}
	}
	return true
}

func assetRequestMetadataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		scheme := "http"
		if request.TLS != nil {
			scheme = "https"
		}
		if forwarded := request.Header.Get("X-Forwarded-Proto"); forwarded != "" {
			scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
		ctx := assetsstore.WithRequestOrigin(request.Context(), scheme, request.Host)
		ctx = assetsstore.WithRequestContentType(ctx, request.Header.Get("Content-Type"))
		if protocol, ok := resolveReadServiceProtocol(request.URL.Path); ok {
			ctx = systemapi.WithProtocolResource(ctx, sessionProtocolResource(request, protocol))
		}
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

type readServiceProtocol struct {
	workItem                              bool
	generatedMethods                      bool
	passthroughMethods                    []string
	protocolOnlyMethods                   []string
	throttleBeforeTimezone                bool
	optionsMetadata                       json.RawMessage
	optionsMetadataMissing                json.RawMessage
	optionsMetadataGuest                  json.RawMessage
	optionsStatus                         int
	pattern                               string
	allow                                 string
	name                                  string
	action                                string
	segments                              []string
	stringPathParameters                  []string
	derivedHead                           bool
	authenticateMethod                    bool
	authoredOptions                       bool
	permissionBeforeTimezone              bool
	permissionBeforeTimezoneExceptMethods []string
	csrfForAuthenticatedSession           bool
	pageLifecycle                         bool
	methodAllow                           map[string]string
	resolveMalformedSlashless             bool
	apiAuthorization                      string
}

type readServiceProtocolMount struct {
	pattern string
	get     http.Handler
}

func newReadServiceProtocol(pattern, allow, name, action string, derivedHead bool, apiAuthorization ...string) readServiceProtocol {
	policy := "project"
	if len(apiAuthorization) > 0 {
		policy = apiAuthorization[0]
	}
	return readServiceProtocol{
		apiAuthorization: policy,
		pattern:          pattern,
		allow:            allow,
		name:             name,
		action:           action,
		segments:         strings.Split(strings.Trim(pattern, "/"), "/"),
		derivedHead:      derivedHead,
	}
}

func newGeneratedAPIWriteProtocol(pattern, allow, name string) readServiceProtocol {
	protocol := newReadServiceProtocol(pattern, allow, name, "", false)
	protocol.generatedMethods = true
	return protocol
}

var readServiceProtocols = append(append(append(append(append(append(invitationsMembershipProtocols(), accountAuthenticationProtocols()...), assetsReadServiceProtocols()...), customViewCreationProtocols()...), workspaceAnalyticsProtocols()...), workspaceSearchProtocols()...), []readServiceProtocol{
	newSessionServiceProtocol("/api/workspaces/{slug}/export-issues/", "GET, POST, HEAD, OPTIONS", "Export Issues Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/export-analytics/", "POST, OPTIONS", "Export Analytics Endpoint", false),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/activity-graph/", "GET, HEAD, OPTIONS", "User Activity Graph Endpoint", true),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/dashboard/", "GET, HEAD, OPTIONS", "User Workspace Dashboard Endpoint", true),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/issues-completed-graph/", "GET, HEAD, OPTIONS", "User Issue Completed Graph Endpoint", true),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/{pk}/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/{pk}/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/property-history/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/{pk}/"),
	userAccountMeWriteProtocol(),
	newSessionServiceProtocol("/api/users/me/email/", "PATCH, OPTIONS", "User Endpoint", false),
	userAccountEmailCodeProtocol(),
	newSessionServiceProtocol("/api/unsplash/", "GET, HEAD, OPTIONS", "Unsplash Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/ai-assistant/", "POST, OPTIONS", "Workspace Gpt Integration Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/ai-assistant/", "POST, OPTIONS", "Gpt Integration Endpoint", false),
	projectMemberPreferencesProtocol(),
	newInstanceManagementProtocol("/api/instances/", "GET, PATCH, HEAD, OPTIONS", "Instance Endpoint", true),
	newInstanceManagementProtocol("/api/instances/admins/", "GET, POST, DELETE, HEAD, OPTIONS", "Instance Admin Endpoint", true),
	newInstanceManagementProtocol("/api/instances/admins/sign-in/", "POST, OPTIONS", "Instance Admin Sign In Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-up/", "POST, OPTIONS", "Instance Admin Sign Up Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-out/", "POST, OPTIONS", "Instance Admin Sign Out Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-up-screen-visited/", "POST, OPTIONS", "Sign Up Screen Visited Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/{pk}/", "GET, POST, DELETE, HEAD, OPTIONS", "Instance Admin Endpoint", true),
	newInstanceManagementProtocol("/api/instances/configurations/", "GET, PATCH, HEAD, OPTIONS", "Instance Configuration Endpoint", true),
	newInstanceManagementProtocol("/api/instances/telemetry/", "GET, HEAD, OPTIONS", "Instance Telemetry Endpoint", true),
	newInstanceManagementProtocol("/api/instances/configurations/disable-email-feature/", "DELETE, OPTIONS", "Disable Email Feature Endpoint", false),
	newInstanceManagementProtocol("/api/instances/email-credentials-check/", "POST, OPTIONS", "Email Credential Check Endpoint", false),
	newInstanceManagementProtocol("/api/instances/workspaces/", "GET, POST, HEAD, OPTIONS", "Instance Work Space Endpoint", true),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/members/", "GET, POST"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/", "GET, POST"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/", "PATCH, DELETE, GET"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/", "PATCH, DELETE, GET"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members-lite/", "GET"),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/", "POST, OPTIONS", true),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/{pk}/", "PATCH, DELETE, OPTIONS", true),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/leave/", "POST, OPTIONS", false),

	newLabelWriteProtocol("/api/workspaces/{slug}/projects/{project_id}/issue-labels/", "GET, POST, HEAD, OPTIONS"),
	newLabelWriteProtocol("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"),
	newProjectConfigurationAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/labels/", "POST", "GET, POST", "Label List Create API Endpoint"),
	newProjectConfigurationAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/", "PATCH, DELETE", "GET, PATCH, DELETE", "Label Detail API Endpoint"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/", "POST, OPTIONS", "GET, POST, HEAD, OPTIONS"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", "PATCH, DELETE, OPTIONS", "GET, PATCH, DELETE, HEAD, OPTIONS"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/mark-default/", "POST, OPTIONS", ""),
	newProjectStateAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/states/", "GET, POST", "State List Create API Endpoint"),
	newProjectStateAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/", "GET, PATCH, DELETE", "State Detail API Endpoint"),
	newReadServiceProtocol("/api/v1/users/me/", "GET", "User Endpoint", "", false, "identity"),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/read/", "POST, DELETE, OPTIONS", "Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/archive/", "POST, DELETE, OPTIONS", "Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/mark-all-read/", "POST, OPTIONS", "Mark All Read Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/", "GET, HEAD, OPTIONS", "Notification", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/unread/", "GET, HEAD, OPTIONS", "Unread Notification Endpoint", true),
	notificationRemainingDetailProtocol(),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/", "GET", "Cycle API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles-lite/", "GET", "Cycle API Endpoint", "", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/", "POST", true),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", "POST", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/{cycle_id}/unarchive/", "DELETE", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "GET, POST", "Cycle API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "GET, DELETE", "Cycle API Endpoint", "", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", "PATCH, DELETE", true),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/", "GET", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules-lite/", "GET", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/", "GET, POST", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/", "GET, POST", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "GET, PATCH, DELETE", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/users/me/activities/", "GET, HEAD, OPTIONS", "User Activity Endpoint", "", true),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/", "GET", "Issue Activity List API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/{pk}/", "GET", "Issue Activity Detail API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/", "GET", "Issue Activity List API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/{pk}/", "GET", "Issue Activity Detail API Endpoint", "", false),
	newReadServiceProtocol("/api/users/me/workspaces/", "GET, HEAD, OPTIONS", "User Work Spaces Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/", "GET, POST, HEAD, OPTIONS", "Work Space", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Work Space", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/project-stats/", "GET, HEAD, OPTIONS", "Project Stats Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/", "GET, POST, HEAD, OPTIONS", "Project", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/details/", "GET, HEAD, OPTIONS", "Project", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Project", http.MethodPut, true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-activity/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Activity Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-activity/{user_id}/export/", "POST, OPTIONS", "Export Workspace User Activity Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-profile/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Profile Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-stats/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Profile Stats Endpoint", true),

	newReadServiceProtocol("/api/workspaces/{slug}/cycles/", "GET, HEAD, OPTIONS", "Workspace Cycles Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/", "POST, OPTIONS", "Cycle", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/date-check/", "POST, OPTIONS", "Cycle Date Check Endpoint", false),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/", "GET, HEAD, OPTIONS", "Cycle Analytics Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "GET, POST, HEAD, OPTIONS", "Cycle Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Cycle Issue", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/", "GET, HEAD, OPTIONS", "Cycle Progress Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Cycle", true),

	newReadServiceProtocol("/api/workspaces/{slug}/modules/", "GET, HEAD, OPTIONS", "Workspace Modules Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/", "GET, POST, HEAD, OPTIONS", "Module", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/", "GET, POST, HEAD, OPTIONS", "Module Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module Issue", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/", "GET, POST, HEAD, OPTIONS", "Module Link", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module Link", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module", http.MethodPut, true),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/", "GET, POST, HEAD, OPTIONS", "Sticky List Create API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Sticky Detail API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies.json/", "GET, POST, HEAD, OPTIONS", "Sticky List Create API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/{pk}.json/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Sticky Detail API Endpoint"),

	newReadServiceProtocol("/api/workspaces/{slug}/estimates/", "GET, HEAD, OPTIONS", "Workspace Estimates Endpoint", "", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/", "POST, OPTIONS", "GET, POST, HEAD, OPTIONS", "Bulk Estimate Point Endpoint", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/", "PATCH, DELETE, OPTIONS", "GET, PATCH, DELETE, HEAD, OPTIONS", "Bulk Estimate Point Endpoint", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/", "POST, OPTIONS", "", "Estimate Point Endpoint", false),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/", "PATCH, DELETE, OPTIONS", "", "Estimate Point Endpoint", false),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/bulk-create-labels/", "POST, OPTIONS", "", "Bulk Create Issue Labels Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/project-estimates/", "GET, HEAD, OPTIONS", "Project Estimate Point Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhooks/", "GET, POST, PATCH, DELETE, HEAD, OPTIONS", "Webhook Endpoint", true),
	newWebhookDetailProtocol(),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhooks/{pk}/regenerate/", "POST, OPTIONS", "Webhook Secret Regenerate Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhook-logs/{webhook_id}/", "GET, HEAD, OPTIONS", "Webhook Logs Endpoint", true),

	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages-summary/", "GET, HEAD, OPTIONS", "Page", "", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/", "GET, POST, HEAD, OPTIONS", "Page", "collection", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Page", "detail", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", "GET, PATCH, HEAD, OPTIONS", "Pages Description", "description", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/", "POST, OPTIONS", "Page", "access", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/", "POST, DELETE, OPTIONS", "Page", "archive", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/", "POST, OPTIONS", "Page Duplicate Endpoint", "duplicate", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/", "POST, DELETE, OPTIONS", "Page", "lock", false),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/", "GET, HEAD, OPTIONS", "Page Version Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/", "GET, HEAD, OPTIONS", "Page Version Endpoint", "", true),

	newReadServiceProtocol("/api/workspaces/{slug}/draft-issues/", "GET, POST, HEAD, OPTIONS", "Workspace Draft Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/draft-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Workspace Draft Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", "GET, POST, HEAD, OPTIONS", "Intake Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake Issue", "", true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/inboxes/", "GET, POST, HEAD, OPTIONS", "Intake", http.MethodPost, true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/", "GET, POST, HEAD, OPTIONS", "Intake Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/", "GET, HEAD, OPTIONS", "Intake Work Item Description Version Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/", "GET, HEAD, OPTIONS", "Intake Work Item Description Version Endpoint", "", true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/intakes/", "GET, POST, HEAD, OPTIONS", "Intake", http.MethodPost, true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake", "", true),

	newProjectPublicationProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/", "GET, POST, HEAD, OPTIONS", "Deploy Board", http.MethodPost, true),
	newProjectPublicationProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Deploy Board", "", true),
}...)

func pageLifecycleWriteTransportMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		bounded := activityReadsRequestSizeMiddleware(options.bodySizeLimit)(activityReadsAuditFallbackMiddleware(options)(next))
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, ok := resolveReadServiceProtocol(request.URL.Path)
			if !ok || !protocol.pageLifecycle || protocol.methodAllow[request.Method] == "" || request.Method == http.MethodGet || request.Method == http.MethodHead {
				next.ServeHTTP(response, request)
				return
			}
			bounded.ServeHTTP(response, request)
		})
	}
}

func resolveReadServiceProtocol(requestPath string) (readServiceProtocol, bool) {
	// DRF format aliases resolve with or without a trailing slash. Resolve the
	// same policy at both the outer dispatcher and its authorization handler.
	if !strings.HasSuffix(requestPath, "/") {
		if format, _, valid := apiV1StickyFormatSuffix(requestPath); valid && format == "json" {
			requestPath += "/"
		}
	}
	if !strings.HasPrefix(requestPath, "/") || !strings.HasSuffix(requestPath, "/") {
		return readServiceProtocol{}, false
	}
	trimmed := strings.TrimSuffix(strings.TrimPrefix(requestPath, "/"), "/")
	if trimmed == "" || strings.HasSuffix(trimmed, "/") {
		return readServiceProtocol{}, false
	}
	segments := strings.Split(trimmed, "/")
	for _, protocol := range readServiceProtocols {
		if readServiceProtocolMatchesSegments(protocol, segments) {
			return protocol, true
		}
	}
	return readServiceProtocol{}, false
}

func readServiceProtocolMatchesPath(requestPath string, protocol readServiceProtocol) bool {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(requestPath, "/"), "/")
	if trimmed == "" || strings.HasSuffix(trimmed, "/") {
		return false
	}
	return readServiceProtocolMatchesSegments(protocol, strings.Split(trimmed, "/"))
}

func readServiceProtocolMatchesSegments(protocol readServiceProtocol, segments []string) bool {
	if len(segments) != len(protocol.segments) {
		return false
	}
	if protocol.pattern == "/api/instances/admins/{pk}/" && instanceAdminLiteralSegment(segments[len(segments)-1]) {
		return false
	}
	for index, patternSegment := range protocol.segments {
		if strings.HasPrefix(protocol.pattern, "/api/v1/workspaces/{slug}/stickies/") && patternSegment == "{pk}" && strings.Contains(segments[index], ".") {
			return false
		}
		if strings.HasPrefix(protocol.pattern, "/api/v1/workspaces/{slug}/stickies/") && patternSegment == "{pk}.json" {
			pk, suffix, found := strings.Cut(segments[index], ".")
			if !found || pk == "" || suffix != "json" {
				return false
			}
		}
		if strings.HasPrefix(patternSegment, "{") {
			if segments[index] == "" {
				return false
			}
			if strings.HasSuffix(patternSegment, ".json") && !strings.HasSuffix(segments[index], ".json") {
				return false
			}
			continue
		}
		if segments[index] != patternSegment {
			return false
		}
	}
	return true
}

func mustReadServiceProtocol(pattern string) readServiceProtocol {
	for _, protocol := range readServiceProtocols {
		if protocol.pattern == pattern {
			return protocol
		}
	}
	panic("missing read-service protocol for " + pattern)
}

func mountReadServiceProtocolRoutes(
	mux loomhttp.Muxer,
	routes []readServiceProtocolMount,
	corsPolicy loomhttp.RuntimeCORSPolicy,
) {
	for _, route := range routes {
		protocol := mustReadServiceProtocol(route.pattern)
		if protocol.derivedHead {
			loomhttp.MountDerivedHead(mux, protocol.pattern, corsPolicy.Handler(loomhttp.AsHandlerFunc(route.get)))
		}
	}
}

func mountActivityReadsProtocolRoutes(mux loomhttp.Muxer, server *activityreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	loomhttp.MountDerivedHead(mux, "/api/users/me/activities/", corsPolicy.Handler(loomhttp.AsHandlerFunc(server.GetAPIUsersMeActivities)))
}

type activityReadsMethodAuthorizer struct {
	readOnly systemapi.ActivityReadsProtocolAuthorizer
	primary  systemapi.ActivityReadsProtocolAuthorizer
	identity systemapi.APIIdentityProtocolAuthorizer
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeAPIIdentityProtocol(ctx context.Context, key string) (int, map[string]any) {
	if authorizer.identity == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return authorizer.identity.AuthorizeAPIIdentityProtocol(ctx, key)
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeAPIActivityProtocol(ctx context.Context, key, slug, projectID, method string) (int, map[string]any) {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return authorizer.readOnly.AuthorizeAPIActivityProtocol(ctx, key, slug, projectID, method)
	}
	return authorizer.primary.AuthorizeAPIActivityProtocol(ctx, key, slug, projectID, method)
}

func (authorizer activityReadsMethodAuthorizer) AuthorizeUserActivityProtocol(ctx context.Context, session string) (int, map[string]any) {
	return authorizer.primary.AuthorizeUserActivityProtocol(ctx, session)
}
func newActivityReadsProtocolMiddleware(
	authorizer systemapi.ActivityReadsProtocolAuthorizer,
	corsPolicy loomhttp.RuntimeCORSPolicy,
	stack func(http.Handler) http.Handler,
	compatibilityOptions apiV1CompatibilityOptions,
	cookieConfig djangoSessionCookieConfig,
	bodySizeLimit int64,
	debugMode bool,
) func(http.Handler) http.Handler {
	apiMethodNotAllowed := activityReadsAPIMethodNotAllowedHandler(authorizer)
	apiOrdinary := corsPolicy.Handler(loomhttp.AsHandlerFunc(stack(apiMethodNotAllowed)))
	apiPreflight := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, apiMethodNotAllowed)
	userOrdinary := corsPolicy.Handler(loomhttp.AsHandlerFunc(stack(activityReadsUserMethodNotAllowedHandler(authorizer, cookieConfig))))
	userPreflight := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, userOrdinary)
	return func(next http.Handler) http.Handler {
		slashRedirect := activityReadsSlashRedirectMiddleware(next, debugMode)
		apiSlashlessHandler := activityReadsSlashlessCompatibilityHandler(stack, slashRedirect)
		userSlashlessHandler := activityReadsSlashlessCompatibilityHandler(func(next http.Handler) http.Handler {
			next = activityReadsCompatibilityMiddleware(compatibilityOptions)(next)
			next = activityReadsRequestSizeMiddleware(bodySizeLimit)(next)
			return next
		}, slashRedirect)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
			slashless := false
			if !matched && !strings.HasSuffix(request.URL.Path, "/") {
				candidatePath := request.URL.Path + "/"
				protocol, matched = resolveReadServiceProtocolForMethod(candidatePath, request.Method)
				if matched && readServiceProtocolHasMalformedUUID(candidatePath, protocol) {
					matched = false
				}
				format, _, stickyFormatPath := apiV1StickyFormatSuffix(request.URL.Path)
				if stickyFormatPath && format == "json" {
					slashless = false
				} else {
					slashless = matched
				}
			}
			activityAPI := matched && activityReadsAPIProtocol(protocol.pattern)
			userActivity := matched && protocol.pattern == "/api/users/me/activities/"
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if activityAPI && preflight {
				request = withReadServicePathValues(request, protocol)
				apiPreflight.ServeHTTP(response, request)
				return
			}
			if userActivity && preflight {
				userPreflight.ServeHTTP(response, request)
				return
			}
			if slashless {
				if activityAPI {
					apiSlashlessHandler.ServeHTTP(response, request)
					return
				}
				if userActivity {
					userSlashlessHandler.ServeHTTP(response, request)
					return
				}
			}
			unsupportedUserMethod := userActivity && !readServiceMethodAllowed(protocol.allow, request.Method)
			if slices.Contains(protocol.passthroughMethods, request.Method) || (!activityAPI || request.Method == http.MethodGet && protocol.apiAuthorization != "assets" && !protocol.generatedMethods || (protocol.workItem || protocol.generatedMethods) && readServiceMethodAllowed(protocol.allow, request.Method) && !(protocol.apiAuthorization == "invitations" && request.Method == http.MethodOptions)) && !unsupportedUserMethod {
				next.ServeHTTP(response, request)
				return
			}
			request = withReadServicePathValues(request, protocol)
			if activityAPI {
				apiOrdinary.ServeHTTP(response, request)
				return
			}
			userOrdinary.ServeHTTP(response, request)
		})
	}
}
func withReadServicePathValues(request *http.Request, protocol readServiceProtocol) *http.Request {
	segments := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	clone := request.Clone(request.Context())
	for index, patternSegment := range protocol.segments {
		if strings.HasPrefix(patternSegment, "{") {
			end := strings.IndexByte(patternSegment, '}')
			if end > 1 {
				value := segments[index]
				suffix := patternSegment[end+1:]
				value = strings.TrimSuffix(value, suffix)
				clone.SetPathValue(patternSegment[1:end], value)
			}
		}
	}
	return clone
}

func activityReadsAPIMethodNotAllowedHandler(authorizer systemapi.ActivityReadsProtocolAuthorizer) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var status int
		var body map[string]any
		protocol, _ := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)

		if protocol.apiAuthorization == "stickies" {
			if stickies, ok := authorizer.(systemapi.APIV1StickyProtocolAuthorizer); ok {
				status, body = stickies.AuthorizeAPIV1StickyProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "project_configuration" {
			if configuration, ok := authorizer.(interface {
				AuthorizeAPIV1LabelProtocol(context.Context, string, string, string) (int, map[string]any)
			}); ok {
				status, body = configuration.AuthorizeAPIV1LabelProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"), request.PathValue("project_id"))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "project_lifecycle" {
			if project, ok := authorizer.(interface {
				AuthorizeProjectLifecycleAPIProtocol(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any)
			}); ok {
				status, body = project.AuthorizeProjectLifecycleAPIProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol))
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "assets" {
			if assets, ok := authorizer.(systemapi.AssetsAPIProtocolAuthorizer); ok {
				status, body = assets.AuthorizeAssetsAPIProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol))
			} else {
				status = 500
				body = map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "invitations" {
			if invitations, ok := authorizer.(systemapi.InvitationsMembershipProtocolAuthorizer); ok {
				status, body = invitations.AuthorizeInvitationsMembershipProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), sessionProtocolResource(request, protocol), true)
			} else {
				status, body = 500, map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "membership" {
			if membership, ok := authorizer.(systemapi.MembershipLifecycleProtocolAuthorizer); ok {
				status, body = membership.AuthorizeMembershipLifecycleProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), systemapi.SessionProtocolResource{Method: request.Method, Pattern: protocol.pattern, Slug: request.PathValue("slug"), ProjectID: request.PathValue("project_id")}, true)
			} else {
				status = 500
				body = map[string]any{"error": "Something went wrong please try again later"}
			}
		} else if protocol.apiAuthorization == "identity" {
			identity, ok := authorizer.(systemapi.APIIdentityProtocolAuthorizer)
			if !ok {
				writeReadServiceJSON(response, 500, map[string]string{"error": "Something went wrong please try again later"})
				return
			}
			status, body = identity.AuthorizeAPIIdentityProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"))
		} else {
			status, body = authorizer.AuthorizeAPIActivityProtocol(request.Context(), djangoJoinedHeader(request, "X-Api-Key"), request.PathValue("slug"), request.PathValue("project_id"), request.Method)
		}

		if status != 0 {
			writeReadServiceJSON(response, status, body)
			return
		}
		if protocol, ok := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method); ok {
			response.Header().Set("Allow", protocol.allow)
			if protocol.apiAuthorization == "invitations" && request.Method == http.MethodOptions && readServiceMethodAllowed(protocol.allow, http.MethodOptions) {
				status, metadata := notificationProtocolMetadata(protocol)
				writeReadServiceJSON(response, status, metadata)
				return
			}
		}
		writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

func activityReadsUserMethodNotAllowedHandler(authorizer systemapi.ActivityReadsProtocolAuthorizer, cookieConfig djangoSessionCookieConfig) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		sessionID := ""
		if cookie, err := request.Cookie(sessionCookieName(cookieConfig)); err == nil {
			sessionID = cookie.Value
		}
		status, body := authorizer.AuthorizeUserActivityProtocol(request.Context(), sessionID)
		if status != 0 {
			writeReadServiceJSON(response, status, body)
			return
		}
		writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}
func mountWorkspaceProjectReadsProtocolRoutes(mux loomhttp.Muxer, server *workspaceprojectreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/users/me/workspaces/", server.UserWorkspaces},
		{"/api/workspaces/", server.Workspaces},
		{"/api/workspaces/{slug}/", server.Workspace},
		{"/api/workspaces/{slug}/project-stats/", server.ProjectStats},
		{"/api/workspaces/{slug}/projects/", server.Projects},
		{"/api/workspaces/{slug}/projects/details/", server.ProjectDetails},
		{"/api/workspaces/{slug}/projects/{pk}/", server.Project},
	}, corsPolicy)
}

func mountCycleReadsProtocolRoutes(mux loomhttp.Muxer, server *cyclereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/cycles/", server.WorkspaceList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/", server.ArchivedList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/", server.ArchivedDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/", server.Analytics},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", server.ArchiveDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", server.Issues},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", server.IssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/", server.Progress},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", server.Detail},
	}, corsPolicy)
}

func mountModuleReadsProtocolRoutes(mux loomhttp.Muxer, server *modulereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/modules/", server.WorkspaceList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-modules/", server.ArchivedList},
		{"/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/", server.ArchivedDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/", server.ArchiveDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/", server.Issues},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/", server.IssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/", server.Links},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/", server.LinkDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/", server.Detail},
	}, corsPolicy)
}

func mountEstimateReadsProtocolRoutes(mux loomhttp.Muxer, server *estimatereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/estimates/", server.WorkspaceSelected},
		{"/api/workspaces/{slug}/projects/{project_id}/estimates/", server.ProjectList},
		{"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/", server.ProjectDetail},
	}, corsPolicy)
}

func mountPageReadsProtocolRoutes(mux loomhttp.Muxer, server *pagereadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/projects/{project_id}/pages-summary/", server.Summary},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/", server.List},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", server.Detail},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", server.Description},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/", server.VersionList},
		{"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/", server.VersionDetail},
	}, corsPolicy)
}

func mountIntakeInboxReadsProtocolRoutes(mux loomhttp.Muxer, server *intakeinboxreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/draft-issues/", server.DraftIssueList},
		{"/api/workspaces/{slug}/draft-issues/{pk}/", server.DraftIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", server.InboxIssueList},
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", server.InboxIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/", server.InboxList},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/", server.InboxDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-issues/", server.IntakeIssueList},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/", server.IntakeIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/", server.DescriptionVersionList},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/", server.DescriptionVersionDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intakes/", server.IntakeList},
		{"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/", server.IntakeDetail},
	}, corsPolicy)
}

func mountDeployBoardReadsProtocolRoutes(mux loomhttp.Muxer, server *deployboardreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/", server.Collection},
		{"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/", server.Detail},
	}, corsPolicy)
}

var (
	readServicesNotAcceptableBody  = []byte(`{"detail":"Could not satisfy the request Accept header."}`)
	readServicesFormatNotFoundBody = []byte(`{"detail":"Not found."}`)
)

type readServicesJSONIndentResponseWriter struct {
	http.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (writer *readServicesJSONIndentResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *readServicesJSONIndentResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *readServicesJSONIndentResponseWriter) finish(indent int) {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if len(body) > 0 && strings.HasPrefix(strings.ToLower(writer.Header().Get("Content-Type")), "application/json") {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", indent)); err == nil {
			body = indented.Bytes()
			body = bytes.ReplaceAll(body, []byte("\u2028"), []byte(`\u2028`))
			body = bytes.ReplaceAll(body, []byte("\u2029"), []byte(`\u2029`))
			writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
	}
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func readServicesContentNegotiationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		formats, formatErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
		if formatErr != nil {
			log.Error(request.Context(), formatErr, log.KV{K: "msg", V: "parse read-service renderer format"})
			writeReadServiceNegotiationResponse(response, request, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			return
		}
		if len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				writeReadServiceNegotiationResponse(response, request, http.StatusNotFound, "application/json", readServicesFormatNotFoundBody)
				return
			}
		}
		accept, present := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, err := projectMembershipReadsJSONNegotiation(accept, present)
		if err != nil {
			log.Error(request.Context(), err, log.KV{K: "msg", V: "negotiate read-service response"})
			writeReadServiceNegotiationResponse(response, request, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			return
		}
		if !acceptsJSON {
			writeReadServiceNegotiationResponse(response, request, http.StatusNotAcceptable, "application/json", readServicesNotAcceptableBody)
			return
		}
		if jsonIndent == 0 {
			next.ServeHTTP(response, request)
			return
		}
		writer := &readServicesJSONIndentResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		writer.finish(jsonIndent)
	})
}

func writeReadServiceNegotiationResponse(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	contentType string,
	body []byte,
) {
	// Django accesses the session while handling renderer rejection even before API authentication.
	if strings.HasPrefix(request.URL.Path, "/api/v1/") && (status == http.StatusNotFound || status == http.StatusNotAcceptable) {
		removeVary(response.Header(), "Cookie", "origin")
		mergeVary(response.Header(), "Cookie", "origin")
	}
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Content-Length", strconv.Itoa(len(body)))
	response.WriteHeader(status)
	if request.Method != http.MethodHead || apiV1AuditDeferred(request) {
		_, _ = response.Write(body)
	}
}

type readServicesGzipResponseWriter struct {
	http.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (writer *readServicesGzipResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *readServicesGzipResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *readServicesGzipResponseWriter) finish(request *http.Request) {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if protocol, matched := resolveReadServiceProtocol(request.URL.Path); matched &&
		protocol.pattern == "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/" &&
		(request.Method == http.MethodGet || request.Method == http.MethodHead) &&
		status == http.StatusOK && writer.Header().Get("Content-Encoding") == "" {
		prependVary(writer.Header(), "Accept-Encoding")
		if acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")) {
			if compressed, err := djangoGzip(body, nil); err == nil {
				writer.Header().Del("Content-Length")
				writer.Header().Set("Content-Encoding", "gzip")
				if request.Method == http.MethodHead {
					body = nil
				} else {
					body = compressed
				}
			}
		}
		writer.ResponseWriter.WriteHeader(status)
		if len(body) > 0 {
			_, _ = writer.ResponseWriter.Write(body)
		}
		return
	}
	representationLength := len(body)
	streamingHead := false
	if representationLength == 0 && request.Method == http.MethodHead {
		if length, err := strconv.Atoi(writer.Header().Get("Content-Length")); err == nil && length >= 200 {
			representationLength = length
			streamingHead = true
		}
	}
	if representationLength >= 200 && writer.Header().Get("Content-Encoding") == "" {
		prependVary(writer.Header(), "Accept-Encoding")
		if acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")) {
			if streamingHead {
				writer.Header().Del("Content-Length")
				writer.Header().Set("Content-Encoding", "gzip")
			} else if compressed, err := djangoGzip(body, nil); err == nil && len(compressed) < len(body) {
				body = compressed
				writer.Header().Set("Content-Encoding", "gzip")
				writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
			}
		}
	}
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func readServicesGzipMiddleware(next http.Handler) http.Handler {
	return readServicesGzipMiddlewareWithAuthBypass(next, true)
}

func readServicesGzipMiddlewareWithAuthBypass(next http.Handler, bypassAuthentication bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// Django's plain authentication views do not run DRF's response gzip
		// negotiation. Keep their headers and cookie responses on that contract.
		if bypassAuthentication && authResponseSessionPath(request) {
			next.ServeHTTP(response, request)
			return
		}
		writer := &readServicesGzipResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		writer.finish(request)
	})
}

type readServicesResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (writer *readServicesResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *readServicesResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

type readServicesSessionResponseWriter struct {
	http.ResponseWriter
	request       *http.Request
	sessions      readServicesSessionLifecycle
	sessionCookie djangoSessionCookieConfig
	refresh       *djangosession.SessionRefresh
	wroteHeader   bool
	failed        bool
}

func (writer *readServicesSessionResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	override, expire, explicit := djangosession.ResponseSessionActionFromContext(writer.request.Context())
	if expire || status < http.StatusInternalServerError && (explicit || writer.refresh != nil) {
		var err error
		switch {
		case expire:
			err = expireDjangoSessionCookie(writer.Header(), writer.sessionCookie)
		case explicit:
			err = refreshDjangoSessionCookie(writer.Header(), writer.sessionCookie, *override)
		default:
			var refresh djangosession.SessionRefresh
			refresh, err = writer.sessions.RefreshSession(writer.request.Context(), writer.request, *writer.refresh)
			if err == nil {
				err = refreshDjangoSessionCookie(writer.Header(), writer.sessionCookie, refresh)
			}
		}
		if err != nil {
			logPublicProjectMetadataCompatibilityError(writer.request.Context(), err)
			writer.failed = true
			writeReadServicesSessionError(writer.ResponseWriter, writer.request, err)
			return
		}
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *readServicesSessionResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

type activityReadsSessionAccess struct {
	accessed bool
}

type activityReadsAuditState struct {
	logged bool
}

type activityReadsAuditStateKey struct{}

type activityReadsSessionAccessKey struct{}

type activityReadsSessionFinalizationResponseWriter struct {
	http.ResponseWriter
	request       *http.Request
	sessions      readServicesSessionLifecycle
	sessionCookie djangoSessionCookieConfig
	access        *activityReadsSessionAccess
	wroteHeader   bool
	failed        bool
}

func (writer *activityReadsSessionFinalizationResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	if !writer.access.accessed {
		if err := finalizeUnaccessedDjangoSession(writer.Header(), writer.request, status, writer.sessionCookie, writer.sessions); err != nil {
			logPublicProjectMetadataCompatibilityError(writer.request.Context(), err)
			writer.failed = true
			writeReadServicesSessionError(writer.ResponseWriter, writer.request, err)
			return
		}
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *activityReadsSessionFinalizationResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

func activityReadsSessionFinalizationMiddleware(sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if authResponseSessionPath(request) {
				next.ServeHTTP(response, request)
				return
			}
			_, activityRoute := resolveActivityReadProtocol(request.URL.Path)
			if protocol, matched := resolveReadServiceProtocolForMethod(strings.TrimSuffix(request.URL.Path, "/")+"/", request.Method); matched && protocol.authenticateMethod {
				activityRoute = true
			}
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if !activityRoute || preflight || sessions == nil {
				next.ServeHTTP(response, request)
				return
			}
			access := &activityReadsSessionAccess{}
			request = normalizeDjangoCookie(request, sessionCookieName(cookie))
			request = request.WithContext(context.WithValue(request.Context(), activityReadsSessionAccessKey{}, access))
			next.ServeHTTP(&activityReadsSessionFinalizationResponseWriter{
				ResponseWriter: response,
				request:        request,
				sessions:       sessions,
				sessionCookie:  cookie,
				access:         access,
			}, request)
		})
	}
}

func resolveActivityReadProtocol(path string) (readServiceProtocol, bool) {
	protocol, matched := resolveReadServiceProtocol(path)
	if !matched && !strings.HasSuffix(path, "/") {
		protocol, matched = resolveReadServiceProtocol(path + "/")
	}
	if !matched {
		// Generated API-v1 lifecycle actions do not have a retained read
		// protocol entry, but their slashless requests still use the shared
		// Django append-slash behavior.
		protocol, matched = moduleLifecycleProtocolForPath(path)
		if !matched {
			protocol, matched = projectLifecycleProtocolForPath(path)
		}
	}
	if !matched || (!activityReadsAPIProtocol(protocol.pattern) && protocol.pattern != "/api/users/me/activities/") {
		return readServiceProtocol{}, false
	}
	return protocol, true
}
func activityReadsSlashlessProtocol(path string) bool {
	if strings.HasSuffix(path, "/") {
		return false
	}
	_, matched := resolveActivityReadProtocol(path + "/")
	return matched
}

func activityReadsSlashRedirectMiddleware(next http.Handler, debugMode bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if _, _, stickyFormatPath := apiV1StickyFormatSuffix(request.URL.Path); stickyFormatPath {
			next.ServeHTTP(response, request)
			return
		}
		if strings.HasSuffix(request.URL.Path, "/") {
			next.ServeHTTP(response, request)
			return
		}
		if _, matched := resolveActivityReadProtocol(request.URL.Path); !matched {
			next.ServeHTTP(response, request)
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		mergeVary(response.Header(), "Cookie", "origin")
		switch request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if debugMode {
				body, contentType := djangoDebugAppendSlashResponse(request)
				response.Header().Set("Content-Type", contentType)
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(body)
				return
			}
		}
		location := request.URL.Path + "/"
		if request.URL.RawQuery != "" {
			location += "?" + request.URL.RawQuery
		}
		response.Header().Set("Location", location)
		response.WriteHeader(http.StatusMovedPermanently)
	})
}
func activityReadsSlashlessCompatibilityHandler(stack func(http.Handler) http.Handler, redirect http.Handler) http.Handler {
	innerNotFound := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Content-Length", strconv.Itoa(len(djangoNotFoundBody)))
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write(djangoNotFoundBody)
	})
	inner := stack(innerNotFound)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		captured := newAPIV1BufferedResponse()
		inner.ServeHTTP(captured, request)
		if captured.status == 0 {
			captured.status = http.StatusOK
		}
		if captured.status == http.StatusNotFound {
			if values := captured.header.Values("Set-Cookie"); len(values) != 0 {
				response.Header()["Set-Cookie"] = append([]string(nil), values...)
			}
			redirect.ServeHTTP(response, request)
			return
		}
		for name, values := range captured.header {
			response.Header()[name] = append([]string(nil), values...)
		}
		response.WriteHeader(captured.status)
		if request.Method != http.MethodHead {
			_, _ = response.Write(captured.body.Bytes())
		}
	})
}

// assetLifecycleAPIV1Middleware gives the legacy V1 presign operations the
// same API activity, session, body-limit, and response-rate boundary as the
// other V1 replacements. V2 asset and public routes retain their own generated
// authentication shape and do not pass through the V1 compatibility policy.
func assetLifecycleAPIV1Middleware(apiV1 func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		apiHandler := apiV1(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				apiHandler.ServeHTTP(response, request)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func activityReadsMiddlewareStack(
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = readServicesQueryPresenceMiddleware(next)
		next = activityReadsCompatibilityMiddleware(options)(next)
		next = activityReadsSessionMiddleware(sessions, cookie)(next)
		next = readServicesContentNegotiationMiddleware(next)
		next = activityReadsAuditFallbackMiddleware(options)(next)
		next = activityReadsRequestSizeMiddleware(options.bodySizeLimit)(next)
		next = readServicesGzipMiddleware(next)
		next = activityReadsIntegerQueryMiddleware(next)
		return next
	}
}

// API v1 Sticky query values remain raw until the service validates them. The
// shared activity stack normalizes invalid integer filters for older APIs.
func apiV1StickyActivityMiddlewareStack(
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = apiV1StickyProtocolHeadersMiddleware(next)
		next = readServicesQueryPresenceMiddleware(next)
		next = activityReadsCompatibilityMiddleware(options)(next)
		next = activityReadsSessionMiddleware(sessions, cookie)(next)
		next = readServicesContentNegotiationMiddleware(next)
		next = activityReadsAuditFallbackMiddleware(options)(next)
		next = activityReadsRequestSizeMiddleware(options.bodySizeLimit)(next)
		next = readServicesGzipMiddleware(next)
		return next
	}
}

func apiV1StickyProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		segments := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
		allow := "GET, POST, HEAD, OPTIONS"
		if len(segments) == 6 {
			allow = "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
		}
		setDjangoProtocolCompatibilityHeaders(response.Header(), allow)
		next.ServeHTTP(response, request)
	})
}

func activityReadsIntegerQueryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		original := request.URL.Query()
		request = request.WithContext(systemapi.WithReadServiceQueryPresence(request.Context(), original))
		rawValues, present := original["per_page"]
		if !present || len(rawValues) == 0 {
			next.ServeHTTP(response, request)
			return
		}
		normalized := "1"
		if value, err := pythoncompat.DecimalInteger(rawValues[len(rawValues)-1]); err == nil && value.IsInt64() {
			normalized = value.String()
		}
		query := request.URL.Query()
		query.Set("per_page", normalized)
		clone := request.Clone(request.Context())
		clonedURL := *request.URL
		clonedURL.RawQuery = query.Encode()
		clone.URL = &clonedURL
		next.ServeHTTP(response, clone)
	})
}
func activityReadsSessionMiddleware(sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		session := readServicesSessionMiddleware(sessions, cookie)(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				next.ServeHTTP(response, request)
				return
			}
			if access, ok := request.Context().Value(activityReadsSessionAccessKey{}).(*activityReadsSessionAccess); ok {
				access.accessed = true
			}
			session.ServeHTTP(response, request)
		})
	}
}

func activityReadsRequestSizeMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			_, tooLarge, err := readPublicProjectMetadataRequestBody(request, limit)
			if tooLarge {
				// Django rejects the body before entering the view or accessing
				// the session. Remove headers installed by the outer dispatcher.
				response.Header().Del("Allow")
				removeVary(response.Header(), "Cookie")
				response.Header().Set("Content-Type", "application/json")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = response.Write(djangoRequestTooLargeBody)
				return
			}
			if err != nil {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				if request.Method != http.MethodHead {
					_, _ = response.Write(djangoServerErrorBody)
				}
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func activityReadsAuditFallbackMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if djangoJoinedHeader(request, "X-Api-Key") == "" {
				next.ServeHTTP(response, request)
				return
			}
			requestBody, _, err := readPublicProjectMetadataRequestBody(request, options.bodySizeLimit)
			if err != nil {
				next.ServeHTTP(response, request)
				return
			}
			state := &activityReadsAuditState{}
			request = request.WithContext(context.WithValue(request.Context(), activityReadsAuditStateKey{}, state))
			captured := newAPIV1BufferedResponse()
			next.ServeHTTP(captured, request)
			if captured.status == 0 {
				captured.status = http.StatusOK
			}
			if !state.logged {
				writeAPIActivityLog(request.Context(), options.activityLogger, options.activitySecret, request, requestBody, captured.status, captured.body.Bytes())
			}
			for name, values := range captured.header {
				response.Header()[name] = append([]string(nil), values...)
			}
			response.WriteHeader(captured.status)
			if request.Method != http.MethodHead {
				_, _ = response.Write(captured.body.Bytes())
			}
		})
	}
}
func activityReadsCompatibilityMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		apiV1 := apiV1CompatibilityMiddleware(options)(next)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if strings.HasPrefix(request.URL.Path, "/api/v1/") {
				apiV1.ServeHTTP(response, request)
				return
			}
			requestBody, tooLarge, err := readPublicProjectMetadataRequestBody(request, options.bodySizeLimit)
			if tooLarge {
				response.Header().Set("Content-Type", "application/json")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = response.Write(djangoRequestTooLargeBody)
				return
			}
			if err != nil {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				if request.Method != http.MethodHead {
					_, _ = response.Write(djangoServerErrorBody)
				}
				return
			}
			if djangoJoinedHeader(request, "X-Api-Key") == "" {
				next.ServeHTTP(response, request)
				return
			}
			captured := newAPIV1BufferedResponse()
			next.ServeHTTP(captured, request)
			if captured.status == 0 {
				captured.status = http.StatusOK
			}
			writeAPIActivityLog(request.Context(), options.activityLogger, options.activitySecret, request, requestBody, captured.status, captured.body.Bytes())
			if state, ok := request.Context().Value(activityReadsAuditStateKey{}).(*activityReadsAuditState); ok {
				state.logged = true
			}
			for name, values := range captured.header {
				response.Header()[name] = append([]string(nil), values...)
			}
			response.WriteHeader(captured.status)
			if request.Method != http.MethodHead {
				_, _ = response.Write(captured.body.Bytes())
			}
		})
	}
}
func readServicesQueryPresenceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if systemapi.ReadServiceQueryValues(request.Context()) == nil {
			request = request.WithContext(systemapi.WithReadServiceQueryPresence(request.Context(), request.URL.Query()))
		}
		next.ServeHTTP(response, request)
	})
}

func readServicesSessionMiddleware(
	sessions readServicesSessionLifecycle,
	sessionCookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return djangoSessionLifecycleMiddleware(sessions, sessionCookie, func(request *http.Request) bool {
		_, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		return matched
	})
}

// djangoSessionLifecycleMiddleware also serves auth routes that are outside the
// API protocol registry. Their mount supplies the exact route matcher and cookie.
func djangoSessionLifecycleMiddleware(
	sessions readServicesSessionLifecycle,
	sessionCookie djangoSessionCookieConfig,
	matches func(*http.Request) bool,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			matched := matches(request)
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if !matched || preflight || sessions == nil {
				next.ServeHTTP(response, request)
				return
			}
			request = normalizeDjangoCookie(request, sessionCookieName(sessionCookie))
			expireSession, refreshSession, prepared, principal, err := sessions.InspectReadServiceSession(request.Context(), request)
			if expireSession {
				if err := expireDjangoSessionCookie(response.Header(), sessionCookie); err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					writeReadServicesSessionError(response, request)
					return
				}
			}
			if prepared == nil && refreshSession {
				if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
					prepared = &djangosession.SessionRefresh{SessionKey: cookie.Value}
				}
			}
			if err != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), err)
				errorResponse := response
				if prepared != nil {
					errorResponse = &readServicesSessionResponseWriter{
						ResponseWriter: response,
						request:        request,
						sessions:       sessions,
						sessionCookie:  sessionCookie,
						refresh:        prepared,
					}
				}
				writeReadServicesSessionInspectionError(errorResponse, request, err)
				return
			}
			ctx, _ := djangosession.WithResponseSessionAction(request.Context())
			if prepared != nil {
				ctx = djangosession.WithCurrentSession(ctx, *prepared)
			} else if principal.SessionKey != "" {
				// Auth handlers need the loaded session even when automatic saving is disabled.
				ctx = djangosession.WithCurrentSession(ctx, principal.PrepareRefresh())
			}
			request = request.WithContext(systemapi.WithReadServiceSessionPrincipal(
				ctx, principal.UserID, principal.UserTimezone,
			))
			writer := &readServicesSessionResponseWriter{
				ResponseWriter: response,
				request:        request,
				sessions:       sessions,
				sessionCookie:  sessionCookie,
				refresh:        prepared,
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func writeReadServicesSessionInspectionError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, djangosession.ErrTimezoneNotFound):
		writeReadServiceJSON(response, http.StatusBadRequest, map[string]string{
			"error": "The required key does not exist.",
		})
	case errors.Is(err, djangosession.ErrInvalidTimezone):
		writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{
			"error": "Something went wrong please try again later",
		})
	default:
		writeReadServicesSessionError(response, request)
	}
}

func writeReadServicesSessionError(response http.ResponseWriter, request *http.Request, causes ...error) {
	status := http.StatusInternalServerError
	body := djangoServerErrorBody
	if len(causes) != 0 && (errors.Is(causes[0], djangosession.ErrSessionNotFound) || errors.Is(causes[0], djangosession.ErrSessionUpdate)) {
		status = http.StatusBadRequest
		body = djangoBadRequestBody
	}
	headers := response.Header()
	headers.Del("Allow")
	headers.Del("Content-Encoding")
	headers.Del("Cache-Control")
	headers.Del("X-Frame-Options")
	removeVary(headers, "Cookie", "Accept-Encoding")
	headers.Set("Content-Type", "text/html; charset=utf-8")
	if status == http.StatusInternalServerError {
		headers.Set("Content-Length", strconv.Itoa(len(body)))
	} else {
		headers.Del("Content-Length")
	}
	response.WriteHeader(status)
	if request.Method != http.MethodHead {
		_, _ = response.Write(body)
	}
}

func markDjangoPageInternalError(endpoint loom.Endpoint) loom.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		result, err := endpoint(ctx, request)
		if errors.Is(err, pagereadsstore.ErrInvalidOrder) || errors.Is(err, pagereadsstore.ErrDjangoPageNil) {
			systemapi.MarkReadServiceDjangoHTML500(ctx)
		}
		return result, err
	}
}

type djangoReadServiceHTML500ResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	wroteHeader bool
	failed      bool
	omitAllow   bool
}

func (writer *djangoReadServiceHTML500ResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	if status == http.StatusInternalServerError && systemapi.ReadServiceDjangoHTML500(writer.request.Context()) {
		writer.failed = true
		headers := writer.Header()
		if writer.omitAllow {
			headers.Del("Allow")
		}
		headers.Del("Content-Encoding")
		headers.Del("loom-error")
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
		writer.ResponseWriter.WriteHeader(status)
		if writer.request.Method != http.MethodHead {
			_, _ = writer.ResponseWriter.Write(djangoServerErrorBody)
		}
		return
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *djangoReadServiceHTML500ResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

type readServicesOriginalRawQueryKey struct{}

func readServicesOriginalRawQuery(request *http.Request) string {
	if rawQuery, ok := request.Context().Value(readServicesOriginalRawQueryKey{}).(string); ok {
		return rawQuery
	}
	return request.URL.RawQuery
}

// Capture once: outer activity compatibility may retain integer syntax before
// replacing it with transport-safe values, and audit/limit readers need that input.
func captureReadServicesOriginalRawQuery(request *http.Request) *http.Request {
	if _, captured := request.Context().Value(readServicesOriginalRawQueryKey{}).(string); captured {
		return request
	}
	return request.WithContext(context.WithValue(request.Context(), readServicesOriginalRawQueryKey{}, request.URL.RawQuery))
}

func readServicesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		if !matched {
			next.ServeHTTP(response, request)
			return
		}
		activityAPI := activityReadsAPIProtocol(protocol.pattern)
		activityRoute := activityAPI || protocol.pattern == "/api/users/me/activities/"
		preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
		if (activityRoute || protocol.authenticateMethod) && preflight {
			next.ServeHTTP(response, request)
			return
		}
		request = captureReadServicesOriginalRawQuery(request)
		request = normalizeDjangoTolerantQuery(request)
		request = normalizeReadServiceScalarQueryValues(request)
		setDjangoProtocolCompatibilityHeaders(response.Header(), protocol.allow)
		if isInstanceManagementPlainRedirect(protocol.pattern) && readServiceMethodAllowed(protocol.allow, request.Method) {
			// These three admin routes are plain Django Views. Their successful
			// form responses do not carry DRF's Allow header; unsupported-method
			// responses still retain the source-derived Allow value.
			response.Header().Del("Allow")
		}
		if activityAPI {
			removeVary(response.Header(), "Cookie")
		}
		if !activityRoute && !protocol.authenticateMethod && !readServiceMethodAllowed(protocol.allow, request.Method) {
			writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
				"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
			})
			return
		}
		compatibleResponse := http.ResponseWriter(&readServicesResponseWriter{ResponseWriter: response})
		if (request.Method == http.MethodGet || request.Method == http.MethodHead) &&
			readServiceUsesDjangoHTML500(protocol.pattern) {
			request = request.WithContext(systemapi.WithReadServiceDjangoHTML500(request.Context()))
			compatibleResponse = &djangoReadServiceHTML500ResponseWriter{
				ResponseWriter: compatibleResponse,
				request:        request,
			}
		}
		next.ServeHTTP(compatibleResponse, request)
	})
}

func activityReadsAPIProtocol(pattern string) bool {
	return strings.HasPrefix(pattern, "/api/v1/")
}
func readServiceMethodAllowed(allow, method string) bool {
	for _, allowed := range strings.Split(allow, ",") {
		if strings.TrimSpace(allowed) == method {
			return true
		}
	}
	return false
}
func writeReadServiceJSON(response http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("marshal read-service protocol response: %w", err))
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_, _ = response.Write(body)
}

func readServiceUsesDjangoHTML500(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/":
		return true
	default:
		return false
	}
}

func setDjangoProtocolCompatibilityHeaders(headers http.Header, allow string) {
	headers.Set("Allow", allow)
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func djangoProtocolCompatibilityMiddleware(allow func(string) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			setDjangoProtocolCompatibilityHeaders(response.Header(), allow(request.URL.Path))
			next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
		})
	}
}

func userIdentityCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(path string) string {
		if path == "/api/users/me/" {
			return "GET, PATCH, DELETE, HEAD, OPTIONS"
		}
		if path == "/api/users/me/profile/" {
			return "GET, PATCH, HEAD, OPTIONS"
		}
		return "GET, HEAD, OPTIONS"
	})(next)
}

type userMeCacheControlResponseWriter struct {
	http.ResponseWriter
}

func (writer userMeCacheControlResponseWriter) WriteHeader(status int) {
	if status == http.StatusOK {
		writer.Header().Set("Cache-Control", "private, max-age=12")
		writer.Header().Set("Vary", "Cookie, Accept-Encoding, origin")
	}
	if status == http.StatusUnauthorized {
		writer.Header().Set("Vary", "Cookie, origin")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func userMeCacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(userMeCacheControlResponseWriter{ResponseWriter: response}, request)
	})
}

func userMeReadCompatibilityMiddleware(
	sessions userIdentitySessionLifecycle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	bodySizeLimit int64,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			mergeVary(headers, "origin")
			publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
			if tooLarge {
				headers.Del("Allow")
				if sessions != nil {
					writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, sessions)
				} else {
					headers.Set("Content-Type", "application/json")
					headers.Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
					response.WriteHeader(http.StatusRequestEntityTooLarge)
					_, _ = response.Write(djangoRequestTooLargeBody)
				}
				return
			}
			if bodyErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
				headers.Del("Allow")
				if sessions != nil {
					if err := finalizeUnaccessedDjangoSession(headers, request, http.StatusInternalServerError, sessionCookie, sessions); err != nil {
						writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
						replacePublicProjectMetadataSessionError(writer, err)
						writer.finish()
						return
					}
				}
				headers.Set("Content-Type", "text/html; charset=utf-8")
				headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(djangoServerErrorBody)
				return
			}

			writer := &publicProjectMetadataReadsResponseWriter{
				ResponseWriter: response,
				acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
				appendGzipVary: true,
				skipCookieVary: true,
			}
			capture := &systemapi.UserIdentityResponseCapture{}
			writer.capturedResponse = func() json.RawMessage { return capture.Data }
			var preparedSessionRefresh *djangosession.SessionRefresh
			var sessionCookieErr error
			var userID string
			sessionInspected := false
			inspectSessionAfterResponse := false
			applySessionEffects := func(expireSession bool, refreshSession bool, rotatedSession *djangosession.SessionRefresh, resolvedUserID string) {
				if resolvedUserID != "" {
					userID = resolvedUserID
				}
				if expireSession {
					sessionCookieErr = expireDjangoSessionCookie(headers, sessionCookie)
				}
				if rotatedSession != nil {
					preparedSessionRefresh = rotatedSession
				} else if refreshSession {
					if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
						preparedSessionRefresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
					}
				}
			}
			defer func() {
				if inspectSessionAfterResponse && sessions != nil {
					writer.skipCookieVary = false
					mergeVary(headers, "Cookie")
					expireSession, refreshSession, rotatedSession, resolvedUserID, err := sessions.InspectSession(request.Context(), request)
					applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						writePublicProjectMetadataInternalError(writer, err, false)
					}
				}
				writeAPIActivityLog(
					request.Context(), activityLogger, activitySecret, request, requestBody,
					writer.status, writer.renderedBody(),
				)
				if !sessionInspected && !inspectSessionAfterResponse && sessions != nil {
					if err := finalizeUnaccessedDjangoSession(headers, request, writer.status, sessionCookie, sessions); err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					}
				} else if sessionCookieErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), sessionCookieErr)
					replacePublicProjectMetadataSessionError(writer, sessionCookieErr)
				} else if sessions != nil && preparedSessionRefresh != nil && writer.status < http.StatusInternalServerError {
					refresh, err := sessions.RefreshSession(request.Context(), request, *preparedSessionRefresh)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					} else if err := refreshDjangoSessionCookie(headers, sessionCookie, refresh); err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					}
				}
				writer.finish()
			}()

			formats, queryErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
			if queryErr != nil {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if len(formats) > 0 {
				format := formats[len(formats)-1]
				if format != "" && format != "json" {
					inspectSessionAfterResponse = true
					writeStickyJSON(writer, http.StatusNotFound, map[string]string{"detail": "Not found."})
					return
				}
			}
			accept, acceptPresent := djangoRequestAcceptHeader(request)
			acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
			if negotiationErr != nil {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if !acceptsJSON {
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusNotAcceptable, map[string]string{
					"detail": "Could not satisfy the request Accept header.",
				})
				return
			}
			writer.jsonIndent = jsonIndent
			writer.skipCookieVary = false
			mergeVary(headers, "Cookie")
			if sessions != nil {
				sessionInspected = true
				expireSession, refreshSession, rotatedSession, resolvedUserID, err := sessions.InspectSessionWithTimezone(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					writePublicProjectMetadataInternalError(writer, err, true)
					if writer.Header().Get("Allow") != "" {
						writer.Header().Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
					}
					return
				}
				resolvedContext, err := systemapi.WithUserIdentitySessionInspection(request.Context(), userID)
				if err != nil {
					writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
						"error": "Something went wrong please try again later",
					})
					return
				}
				request = request.WithContext(resolvedContext)
			}
			request = request.WithContext(systemapi.WithUserIdentityResponseCapture(request.Context(), capture))
			next.ServeHTTP(writer, request)
		})
	}
}

func finalizeUnaccessedDjangoSession(
	headers http.Header,
	request *http.Request,
	status int,
	sessionCookie djangoSessionCookieConfig,
	sessions publicProjectMetadataThrottle,
) error {
	expireSession, refreshSession, err := sessions.FinalizeUnaccessedSession(request.Context(), request, status)
	if expireSession {
		if cookieErr := expireDjangoSessionCookie(headers, sessionCookie); cookieErr != nil {
			return cookieErr
		}
	}
	if err != nil {
		return err
	}
	if refreshSession != nil && status < http.StatusInternalServerError {
		if cookieErr := refreshUnaccessedDjangoSessionCookie(headers, sessionCookie, *refreshSession); cookieErr != nil {
			return cookieErr
		}
	}
	return nil
}

func userPropertiesCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, PATCH, HEAD, OPTIONS" })(next)
}

func workspaceMetadataCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, HEAD, OPTIONS" })(next)
}

func projectIdentifiersCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, DELETE, HEAD, OPTIONS" })(next)
}

func userSettingsCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, HEAD, OPTIONS" })(next)
}

func workspaceSidebarPreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, PATCH, HEAD, OPTIONS" })(next)
}

func standardViewsCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(path string) string {
		if strings.HasSuffix(strings.TrimSuffix(path, "/"), "/views") {
			return "GET, POST, HEAD, OPTIONS"
		}
		return "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
	})(next)
}

func mountStandardViewsProtocolRoutes(mux loomhttp.Muxer, server *standardviewssvr.Server) {
	routes := []struct {
		path string
		name string
		get  http.Handler
	}{
		{"/api/workspaces/{slug}/projects/{project_id}/views/", "Issue View Endpoint", server.ListProject},
		{"/api/workspaces/{slug}/projects/{project_id}/views/{pk}/", "Issue View Endpoint", server.RetrieveProject},
		{"/api/workspaces/{slug}/views/", "Workspace View Endpoint", server.ListWorkspace},
		{"/api/workspaces/{slug}/views/{pk}/", "Workspace View Endpoint", server.RetrieveWorkspace},
	}
	for _, route := range routes {
		mountProtocolHandler(mux, http.MethodHead, route.path, djangoProtocolHeadHandler(standardViewsCompatibilityMiddleware, route.get))
		mountProtocolHandler(mux, http.MethodOptions, route.path, djangoProtocolOptionsHandler(standardViewsCompatibilityMiddleware, route.name, strings.Contains(route.path, "{pk}")))
	}
}

func djangoProtocolHeadHandler(middleware func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(workspaceSlugHeadWriter{response}, request)
	}))
}

func djangoBufferedProtocolHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func djangoProtocolOptionsHandler(middleware func(http.Handler) http.Handler, name string, patch bool) http.Handler {
	if patch {
		return djangoProtocolOptionsActionsHandler(middleware, name, http.MethodPatch)
	}
	return djangoProtocolOptionsActionsHandler(middleware, name)
}

func djangoProtocolOptionsActionsHandler(middleware func(http.Handler) http.Handler, name string, actionMethods ...string) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		actions := make(map[string]any, len(actionMethods))
		for _, method := range actionMethods {
			actions[method] = map[string]any{}
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": name, "description": "", "renders": []string{"application/json"},
			"parses":  []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions": actions,
		})
	}))
}

func djangoProtocolOptionsWithoutActionsHandler(middleware func(http.Handler) http.Handler, name string) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Renders     []string `json:"renders"`
			Parses      []string `json:"parses"`
		}{
			Name:        name,
			Description: "",
			Renders:     []string{"application/json"},
			Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func stickyListOptionsHandler(middleware func(http.Handler) http.Handler) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": "Sticky List", "description": "", "renders": []string{"application/json"},
			"parses":  []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions": map[string]any{http.MethodPost: stickyOptionsFields},
		})
	}))
}

var stickyOptionsFields = map[string]any{
	"id":                   map[string]any{"type": "field", "required": false, "read_only": true, "label": "Id"},
	"created_at":           map[string]any{"type": "datetime", "required": false, "read_only": true, "label": "Created At"},
	"updated_at":           map[string]any{"type": "datetime", "required": false, "read_only": true, "label": "Last Modified At"},
	"deleted_at":           map[string]any{"type": "datetime", "required": false, "read_only": false, "label": "Deleted At"},
	"name":                 map[string]any{"type": "string", "required": false, "read_only": false, "label": "Name"},
	"description":          map[string]any{"type": "field", "required": false, "read_only": false, "label": "Description"},
	"description_html":     map[string]any{"type": "string", "required": false, "read_only": false, "label": "Description html"},
	"description_stripped": map[string]any{"type": "string", "required": false, "read_only": false, "label": "Description stripped"},
	"description_binary":   map[string]any{"type": "field", "required": false, "read_only": true, "label": "Description binary"},
	"logo_props":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Logo props"},
	"color":                map[string]any{"type": "string", "required": false, "read_only": false, "label": "Color", "max_length": 255},
	"background_color":     map[string]any{"type": "string", "required": false, "read_only": false, "label": "Background color", "max_length": 255},
	"sort_order":           map[string]any{"type": "float", "required": false, "read_only": false, "label": "Sort order"},
	"created_by":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Created By"},
	"updated_by":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Last Modified By"},
	"workspace":            map[string]any{"type": "field", "required": false, "read_only": true, "label": "Workspace"},
	"owner":                map[string]any{"type": "field", "required": false, "read_only": true, "label": "Owner"},
}

func djangoProtocolOptionsJSONNegotiationMiddleware(next http.Handler) http.Handler {
	notAcceptable := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		mergeVary(response.Header(), "Accept")
		writeStickyJSON(response, http.StatusNotAcceptable, map[string]string{
			"detail": "Could not satisfy the request Accept header.",
		})
	})
	serverError := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		mergeVary(response.Header(), "Accept")
		writeStickyJSON(response, http.StatusInternalServerError, map[string]string{
			"error": "Something went wrong please try again later",
		})
	})
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		accept, present := djangoRequestAcceptHeader(request)
		acceptsJSON, _, err := projectMembershipReadsJSONNegotiation(accept, present)
		if err != nil {
			serverError.ServeHTTP(response, request)
			return
		}
		if !acceptsJSON {
			notAcceptable.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func stickyHeadPaginationAuthorizationMiddleware(mux loomhttp.Muxer, authenticate security.AuthAPIKeyFunc, authorizer systemapi.StickyWorkspaceAuthorizer, next http.Handler) http.Handler {
	invalidPagination := http.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setStickyProtocolHeaders(response.Header(), request.URL.Path)
		writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": validateStickyPagination(request)})
	}))
	if authorizer != nil {
		invalidPagination = stickyWorkspaceAuthorizationMiddleware(mux, authorizer, invalidPagination)
	}
	if authenticate != nil {
		invalidPagination = stickyAuthenticationMiddleware(authenticate)(invalidPagination)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if validateStickyPagination(request) == "" {
			next.ServeHTTP(response, request)
			return
		}
		invalidPagination.ServeHTTP(response, request)
	})
}

func stickyWorkspaceAuthorizationMiddleware(mux loomhttp.Muxer, authorizer systemapi.StickyWorkspaceAuthorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := authorizer.AuthorizeStickyWorkspace(request.Context(), mux.Vars(request)["slug"])
		if err == nil {
			next.ServeHTTP(response, request)
			return
		}
		var permission *sticky.StickyPermissionError
		if errors.As(err, &permission) {
			writeStickyJSON(response, http.StatusForbidden, map[string]string{"error": permission.Message})
			return
		}
		var unauthenticated *sticky.StickyUnauthenticatedError
		if errors.As(err, &unauthenticated) {
			writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": unauthenticated.Detail})
			return
		}
		writeStickyJSON(response, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
	})
}

func stickyDetailUUIDMiddleware(mux loomhttp.Muxer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if _, err := uuid.Parse(mux.Vars(request)["pk"]); err != nil {
			http.NotFound(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func publicProjectMetadataMethodNotAllowed() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

var (
	publicProjectMetadataPathPattern          = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)/$`)
	publicProjectMetadataSlashlessPathPattern = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)$`)
	publicProjectMetadataAnyPathPattern       = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)/?$`)
)

func publicProjectMetadataRouteCompatibilityMiddleware(
	compatibility func(http.Handler) http.Handler,
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	debugMode bool,
) func(http.Handler) http.Handler {
	fallback := compatibility(publicProjectMetadataMethodNotAllowed())
	slashless := publicProjectMetadataNotFoundHandler(
		throttle, allowedCORSOrigins, sessionCookie, activityLogger, activitySecret, bodySizeLimit, debugMode, true,
	)
	malformed := publicProjectMetadataNotFoundHandler(
		throttle, allowedCORSOrigins, sessionCookie, activityLogger, activitySecret, bodySizeLimit, debugMode, false,
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			trailingSlash := publicProjectMetadataPathPattern.MatchString(request.URL.Path)
			slashlessPath := publicProjectMetadataSlashlessPathPattern.MatchString(request.URL.Path)
			if !trailingSlash && !slashlessPath {
				cleanedPath := path.Clean(request.URL.Path)
				if publicProjectMetadataAnyPathPattern.MatchString(cleanedPath) {
					malformed.ServeHTTP(response, request)
					return
				}
				next.ServeHTTP(response, request)
				return
			}
			canonicalMethod := strings.ToUpper(request.Method)
			if canonicalMethod != request.Method {
				request = request.Clone(request.Context())
				request.Method = canonicalMethod
			}
			if slashlessPath {
				slashless.ServeHTTP(response, request)
				return
			}
			if canonicalMethod != http.MethodGet && canonicalMethod != http.MethodHead && canonicalMethod != http.MethodOptions {
				fallback.ServeHTTP(response, request)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func publicProjectMetadataNotFoundHandler(
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	debugMode bool,
	appendSlash bool,
) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		if publicProjectMetadataReadsCORSPreflight(headers, request, allowedCORSOrigins) {
			response.WriteHeader(http.StatusOK)
			return
		}
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
		requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
		if tooLarge {
			writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, throttle)
			return
		}
		if bodyErr != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
			cookieErr := expireUnaccessedDjangoSessionCookie(headers, request, sessionCookie)
			if cookieErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
				headers.Del("X-Frame-Options")
			}
			mergeVary(headers, "origin")
			headers.Set("Content-Type", "text/html; charset=utf-8")
			headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write(djangoServerErrorBody)
			return
		}
		mergeVary(headers, "Cookie", "origin")
		requestLogStarted := time.Now()
		writer := &publicProjectMetadataReadsResponseWriter{
			ResponseWriter: response,
			acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
		}
		expireSession, refreshSession, rotatedSession, userID, err := throttle.InspectSession(request.Context(), request)
		var sessionCookieErr error
		if expireSession {
			sessionCookieErr = expireDjangoSessionCookie(headers, sessionCookie)
		}
		if err != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), err)
			replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			writeAPIActivityLog(
				request.Context(), activityLogger, activitySecret, request, requestBody,
				http.StatusInternalServerError, djangoServerErrorBody,
			)
			writer.finish()
			return
		}
		logPublicProjectMetadataInnerResponse(request, http.StatusNotFound, requestLogStarted, userID)
		writeAPIActivityLog(
			request.Context(), activityLogger, activitySecret, request, requestBody,
			http.StatusNotFound, djangoNotFoundBody,
		)
		if sessionCookieErr != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), sessionCookieErr)
			replacePublicProjectMetadataSessionError(writer, sessionCookieErr)
			writer.finish()
			return
		}
		if appendSlash && debugMode && (request.Method == http.MethodPost || request.Method == http.MethodPut ||
			request.Method == http.MethodPatch || request.Method == http.MethodDelete) {
			debugBody, debugContentType := djangoDebugAppendSlashResponse(request)
			// CommonMiddleware raises after the inner GZip and XFrame middleware
			// have returned, so neither one processes Django's technical 500.
			headers.Del("X-Frame-Options")
			writer.skipGzip = true
			writer.omitLength = true
			replacePublicProjectMetadataResponse(
				writer, http.StatusInternalServerError, debugContentType, debugBody,
			)
			writer.finish()
			return
		}
		var preparedSessionRefresh *djangosession.SessionRefresh
		if rotatedSession != nil {
			preparedSessionRefresh = rotatedSession
		} else if refreshSession {
			if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
				preparedSessionRefresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
			}
		}
		if preparedSessionRefresh != nil {
			refresh, refreshErr := throttle.RefreshSession(request.Context(), request, *preparedSessionRefresh)
			if refreshErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), refreshErr)
				replacePublicProjectMetadataSessionError(writer, refreshErr)
				writer.finish()
				return
			}
			if cookieErr := refreshDjangoSessionCookie(headers, sessionCookie, refresh); cookieErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
				replacePublicProjectMetadataSessionError(writer, cookieErr)
				writer.finish()
				return
			}
		}
		if !appendSlash {
			headers.Set("Content-Type", "text/html; charset=utf-8")
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write(djangoNotFoundBody)
			writer.finish()
			return
		}
		location := djangoFullPath(request, true)
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Location", location)
		headers.Del("X-Frame-Options")
		writer.omitLength = true
		writer.WriteHeader(http.StatusMovedPermanently)
		writer.finish()
	})
}

func readPublicProjectMetadataRequestBody(request *http.Request, limit int64) ([]byte, bool, error) {
	// Django's body property enforces DATA_UPLOAD_MAX_MEMORY_SIZE solely from
	// the declared CONTENT_LENGTH, treating a missing value as zero. Unknown-
	// length and under-declared streams are read in full without a second check.
	declaredLength := request.ContentLength
	if declaredLength < 0 {
		declaredLength = 0
	}
	if declaredLength > limit {
		return nil, true, nil
	}
	if request.Body == nil {
		return nil, false, nil
	}
	body, err := io.ReadAll(request.Body)
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("read request body: %w", err)
	}
	return body, false, nil
}

func logPublicProjectMetadataInnerResponse(request *http.Request, status int, started time.Time, userID string) {
	originalRequest := request.Clone(request.Context())
	originalRequest.URL.RawQuery = readServicesOriginalRawQuery(request)
	log.Print(
		request.Context(),
		log.KV{K: "msg", V: fmt.Sprintf("%s %s %d", request.Method, djangoFullPath(originalRequest, false), status)},
		log.KV{K: "path", V: request.URL.Path},
		log.KV{K: "method", V: request.Method},
		log.KV{K: "status_code", V: status},
		log.KV{K: "duration_ms", V: time.Since(started).Milliseconds()},
		log.KV{K: "remote_addr", V: apiActivityIPAddress(request)},
		log.KV{K: "user_agent", V: djangoJoinedHeader(request, "User-Agent")},
		log.KV{K: "user_id", V: nullableString(userID)},
	)
}

func logPublicProjectMetadataCompatibilityError(ctx context.Context, err error) {
	log.Error(ctx, err, log.KV{K: "msg", V: "public project metadata compatibility failure"})
}

func writeAPIActivityLog(
	ctx context.Context,
	logger publicprojectmetadatareadsstore.ActivityLogger,
	secret string,
	request *http.Request,
	requestBody []byte,
	responseCode int,
	responseBody []byte,
) {
	apiKey := djangoJoinedHeader(request, "X-Api-Key")
	if logger == nil || apiKey == "" {
		return
	}
	record := publicprojectmetadatareadsstore.APIActivityLog{
		TokenIdentifier: apiActivityTokenIdentifier(secret, apiKey),
		Path:            request.URL.Path,
		Method:          request.Method,
		QueryParams:     readServicesOriginalRawQuery(request),
		Headers:         djangoRequestHeaders(request),
		Body:            safeDecodeAPIActivityBody(requestBody),
		ResponseCode:    responseCode,
		ResponseBody:    safeDecodeAPIActivityBody(responseBody),
		IPAddress:       nullableString(apiActivityIPAddress(request)),
		UserAgent:       nullableString(djangoJoinedHeader(request, "User-Agent")),
	}
	// Wait for the durable publisher confirmation. Ignore client cancellation as
	// Django does while submitting the task, but retain the request log values.
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := logger.Log(logCtx, record); err != nil {
		log.Error(ctx, err, log.KV{K: "msg", V: "publish API activity log"})
	}
}

func apiActivityTokenIdentifier(secret string, apiKey string) string {
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write([]byte(apiKey))
	return hex.EncodeToString(digest.Sum(nil))
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func safeDecodeAPIActivityBody(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	if bytes.HasPrefix(value, []byte("\x89PNG")) || bytes.HasPrefix(value, []byte("\xff\xd8\xff")) || bytes.HasPrefix(value, []byte("%PDF")) {
		return "[Binary Content]"
	}
	if !utf8.Valid(value) {
		return "[Could not decode content]"
	}
	return string(value)
}

func apiActivityIPAddress(request *http.Request) string {
	return loomhttp.EffectiveClientAddress(request)
}

func djangoRequestHeaders(request *http.Request) string {
	values := make(map[string]string, len(request.Header)+4)
	for name, headerValues := range request.Header {
		if strings.ContainsRune(name, '_') {
			continue
		}
		value := djangoLatin1String([]byte(strings.Join(headerValues, ",")))
		if strings.EqualFold(name, "X-Api-Key") || strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Cookie") {
			value = "[REDACTED]"
		}
		values[http.CanonicalHeaderKey(name)] = value
	}
	if request.Host != "" {
		values["Host"] = djangoLatin1String([]byte(request.Host))
	}
	if request.ContentLength > 0 {
		values["Content-Length"] = strconv.FormatInt(request.ContentLength, 10)
	}
	if len(request.TransferEncoding) > 0 {
		values["Transfer-Encoding"] = strings.Join(request.TransferEncoding, ",")
	}
	if request.Close {
		values["Connection"] = "close"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, pythonStringRepr(key)+": "+pythonStringRepr(values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func pythonStringRepr(value string) string {
	quote := rune('\'')
	if strings.ContainsRune(value, '\'') && !strings.ContainsRune(value, '"') {
		quote = '"'
	}
	var result strings.Builder
	result.WriteRune(quote)
	for _, character := range value {
		switch character {
		case '\\':
			result.WriteString(`\\`)
		case '\t':
			result.WriteString(`\t`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		default:
			switch {
			case character == quote:
				result.WriteRune('\\')
				result.WriteRune(character)
			case unicode.IsPrint(character):
				result.WriteRune(character)
			case character <= 0xff:
				_, _ = fmt.Fprintf(&result, `\x%02x`, character)
			case character <= 0xffff:
				_, _ = fmt.Fprintf(&result, `\u%04x`, character)
			default:
				_, _ = fmt.Fprintf(&result, `\U%08x`, character)
			}
		}
	}
	result.WriteRune(quote)
	return result.String()
}

func djangoReadOnlyOptionsHandler(
	mux loomhttp.Muxer,
	middleware func(http.Handler) http.Handler,
	name string,
	authorizer systemapi.ProjectMembershipReadsMetadataAuthorizer,
	requireWorkspaceMembership bool,
) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var sessionID string
		if cookie, err := request.Cookie("session-id"); err == nil {
			sessionID = cookie.Value
		}
		err := authorizer.AuthorizeMetadata(
			request.Context(),
			sessionID,
			mux.Vars(request)["slug"],
			requireWorkspaceMembership,
		)
		if err != nil {
			var unauthenticated *projectmembershipreads.ProjectMembershipReadsUnauthenticated
			var forbidden *projectmembershipreads.ProjectMembershipReadsForbidden
			var internalError *projectmembershipreads.ProjectMembershipReadsInternalError
			switch {
			case errors.As(err, &unauthenticated):
				writeStickyJSON(response, http.StatusUnauthorized, unauthenticated)
			case errors.As(err, &forbidden):
				writeStickyJSON(response, http.StatusForbidden, forbidden)
			case errors.As(err, &internalError):
				writeStickyJSON(response, http.StatusInternalServerError, internalError)
			default:
				writeStickyJSON(response, http.StatusInternalServerError, map[string]string{
					"message": "Something went wrong please try again later",
				})
			}
			return
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": name, "description": "", "renders": []string{"application/json"},
			"parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func userIdentityOptionsHandler(name string, patch bool) http.Handler {
	return djangoProtocolOptionsHandler(userIdentityCompatibilityMiddleware, name, patch)
}

func workspaceMetadataOptionsHandler(name string) http.Handler {
	return djangoProtocolOptionsHandler(workspaceMetadataCompatibilityMiddleware, name, false)
}

func mountProjectMembershipReadsProtocolRoutes(
	mux loomhttp.Muxer,
	server *projectmembershipreadssvr.Server,
	authorizer systemapi.ProjectMembershipReadsMetadataAuthorizer,
) {
	routes := []struct {
		path                       string
		name                       string
		get                        http.Handler
		mountOptions               bool
		requireWorkspaceMembership bool
	}{
		{
			path:         "/api/workspaces/{slug}/projects/{project_id}/project-members/me/",
			name:         "Project Member User Endpoint",
			get:          server.Membership,
			mountOptions: true,
		},
		{
			path:                       "/api/users/me/workspaces/{slug}/project-roles/",
			name:                       "User Project Roles Endpoint",
			get:                        server.Roles,
			mountOptions:               true,
			requireWorkspaceMembership: true,
		},
		{
			path:         "/api/workspaces/{slug}/members/",
			name:         "Work Space Member",
			get:          server.WorkspaceMembers,
			mountOptions: true,
		},
		{
			path: "/api/workspaces/{slug}/members/{pk}/",
			name: "Workspace Member Endpoint",
			get:  server.WorkspaceMember,
		},
		{
			path:                       "/api/workspaces/{slug}/project-members/",
			name:                       "Workspace Project Member Endpoint",
			get:                        server.WorkspaceProjectMembers,
			mountOptions:               true,
			requireWorkspaceMembership: true,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/members/",
			name: "Project Member Endpoint",
			get:  server.ProjectMembers,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/members/{pk}/",
			name: "Project Member Endpoint",
			get:  server.ProjectMember,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/",
			name: "Project Member Preference Endpoint",
			get:  server.Preference,
		},
	}
	for _, route := range routes {
		mountProtocolHandler(
			mux,
			http.MethodHead,
			route.path,
			djangoProtocolHeadHandler(projectMembershipReadsCompatibilityMiddleware, route.get),
		)
		if route.mountOptions {
			mountProtocolHandler(
				mux,
				http.MethodOptions,
				route.path,
				djangoReadOnlyOptionsHandler(
					mux,
					projectMembershipReadsCompatibilityMiddleware,
					route.name,
					authorizer,
					route.requireWorkspaceMembership,
				),
			)
		}
	}
}

type projectMembershipReadsResponseWriter struct {
	http.ResponseWriter
	requestPath   string
	requestMethod string
	rawResponse   *systemapi.ProjectMembershipReadsResponseCapture
	jsonIndent    int
	status        int
	wroteHeader   bool
}

func (writer *projectMembershipReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	mergeVary(writer.Header(), "Cookie")
	if status == http.StatusOK && writer.requestMethod != http.MethodOptions &&
		!strings.Contains(writer.requestPath, "/project-roles/") {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectMembershipReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	inputLength := len(body)
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 && writer.requestMethod != http.MethodOptions {
		body = writer.rawResponse.Data
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, djangoJSONColonSpacing(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	if len(body) != inputLength {
		written, err := writer.ResponseWriter.Write(body)
		if err != nil {
			return 0, err
		}
		if written != len(body) {
			return 0, io.ErrShortWrite
		}
		return inputLength, nil
	}
	return writer.ResponseWriter.Write(body)
}

func projectMembershipReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", projectMembershipReadsAllow(request.URL.Path))
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectMembershipReadsJSON(
					response,
					request,
					http.StatusNotFound,
					map[string]string{"detail": "Not found."},
				)
				return
			}
		}
		accept, acceptPresent := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
		if negotiationErr != nil {
			writeDjangoAPIViewServerError(response, request)
			return
		}
		if !acceptsJSON {
			headers.Set("Content-Type", "application/json")
			writeProjectMembershipReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		if values := request.URL.Query()["search"]; len(values) > 1 {
			query := request.URL.Query()
			query.Set("search", values[len(values)-1])
			request = request.Clone(request.Context())
			clonedURL := *request.URL
			request.URL = &clonedURL
			request.URL.RawQuery = query.Encode()
		}
		rawResponse := &systemapi.ProjectMembershipReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectMembershipReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectMembershipReadsResponseWriter{
			ResponseWriter: response,
			requestPath:    request.URL.Path,
			requestMethod:  request.Method,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectMembershipReadsJSON(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	value any,
) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func writeDjangoAPIViewServerError(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	writeProjectMembershipReadsJSON(response, request, http.StatusInternalServerError, map[string]string{
		"error": "Something went wrong please try again later",
	})
}

func djangoRequestAcceptHeader(request *http.Request) (string, bool) {
	values, present := request.Header[http.CanonicalHeaderKey("Accept")]
	return strings.Join(values, ","), present
}

func projectMembershipReadsJSONNegotiation(accept string, present bool) (bool, int, error) {
	if !present {
		accept = "*/*"
	}
	bestPrecedence := -1
	bestIndent := 0
	for _, value := range strings.Split(accept, ",") {
		mediaType, parameters, err := djangoParseHeaderParameters(strings.TrimSpace(value))
		if err != nil {
			return false, 0, err
		}
		precedence := -1
		switch mediaType {
		case "*/*":
			precedence = 0
		case "application/*":
			precedence = 1
		case "application/json":
			precedence = 2
			for name := range parameters {
				if name != "q" {
					precedence = 3
					break
				}
			}
		}
		if precedence <= bestPrecedence {
			continue
		}
		bestPrecedence = precedence
		bestIndent = 0
		bestIndent = djangoJSONIndent(parameters["indent"])
	}
	return bestPrecedence >= 0, bestIndent, nil
}

func djangoParseHeaderParameters(value string) (string, map[string]string, error) {
	parts := make([]string, 0, 2)
	start := 0
	inQuotes := false
	escaped := false
	for index := 0; index < len(value); index++ {
		switch {
		case escaped:
			escaped = false
		case inQuotes && value[index] == '\\':
			escaped = true
		case value[index] == '"':
			inQuotes = !inQuotes
		case value[index] == ';' && !inQuotes:
			parts = append(parts, strings.TrimSpace(value[start:index]))
			start = index + 1
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	mediaType := strings.ToLower(parts[0])
	parameters := make(map[string]string, len(parts)-1)
	for _, parameter := range parts[1:] {
		name, parameterValue, found := strings.Cut(parameter, "=")
		if !found {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		parameterValue = strings.TrimSpace(parameterValue)
		if len(parameterValue) >= 2 && parameterValue[0] == '"' && parameterValue[len(parameterValue)-1] == '"' {
			parameterValue = parameterValue[1 : len(parameterValue)-1]
			parameterValue = strings.ReplaceAll(parameterValue, `\\`, `\`)
			parameterValue = strings.ReplaceAll(parameterValue, `\"`, `"`)
		}
		if strings.HasSuffix(name, "*") {
			name = strings.TrimSuffix(name, "*")
			if strings.Count(parameterValue, "'") == 2 {
				extended := strings.Split(parameterValue, "'")
				decodedValue, err := djangoUnquoteHeaderParameter(extended[2], extended[0])
				if err != nil {
					return "", nil, err
				}
				parameterValue = decodedValue
			}
		}
		parameters[name] = parameterValue
	}
	return mediaType, parameters, nil
}

func djangoUnquoteHeaderParameter(value string, encodingName string) (string, error) {
	if !strings.Contains(value, "%") {
		return value, nil
	}
	decoded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == '%' && index+2 < len(value) {
			high, highOK := queryHexDigit(value[index+1])
			low, lowOK := queryHexDigit(value[index+2])
			if highOK && lowOK {
				decoded = append(decoded, high<<4|low)
				index += 2
				continue
			}
		}
		decoded = append(decoded, value[index])
	}
	normalizedEncoding := strings.ToLower(strings.ReplaceAll(encodingName, "_", "-"))
	stripUTF8BOM := false
	switch normalizedEncoding {
	case "u8", "utf8", "cp65001":
		normalizedEncoding = "utf-8"
	case "utf-8-sig":
		normalizedEncoding = "utf-8"
		stripUTF8BOM = true
	}
	if stripUTF8BOM && bytes.HasPrefix(decoded, []byte{0xEF, 0xBB, 0xBF}) {
		decoded = decoded[3:]
	}
	if normalizedEncoding == "unicode-escape" || normalizedEncoding == "raw-unicode-escape" {
		return decodePythonUnicodeEscape(decoded, normalizedEncoding == "raw-unicode-escape"), nil
	}
	decoder, err := ianaindex.IANA.Encoding(normalizedEncoding)
	if err != nil {
		decoder, err = htmlindex.Get(normalizedEncoding)
		if err != nil {
			return "", fmt.Errorf("decode Django header parameter with encoding %q: %w", encodingName, err)
		}
	}
	if decoder == nil {
		return "", fmt.Errorf("decode Django header parameter with unsupported encoding %q", encodingName)
	}
	decodedValue, _, err := transform.String(decoder.NewDecoder(), string(decoded))
	if err != nil {
		return "", fmt.Errorf("decode Django header parameter with encoding %q: %w", encodingName, err)
	}
	return decodedValue, nil
}

func decodePythonUnicodeEscape(value []byte, raw bool) string {
	var decoded strings.Builder
	for index := 0; index < len(value); {
		if value[index] != '\\' {
			decoded.WriteRune(rune(value[index]))
			index++
			continue
		}
		if raw {
			runEnd := index
			for runEnd < len(value) && value[runEnd] == '\\' {
				runEnd++
			}
			count := runEnd - index
			if runEnd >= len(value) || (value[runEnd] != 'u' && value[runEnd] != 'U') || count%2 == 0 {
				decoded.WriteString(strings.Repeat("\\", count))
				index = runEnd
				continue
			}
			decoded.WriteString(strings.Repeat("\\", count-1))
			index = runEnd - 1
		}
		if index+1 >= len(value) {
			decoded.WriteRune(utf8.RuneError)
			break
		}
		escape := value[index+1]
		if escape == 'u' || escape == 'U' {
			digits := 4
			if escape == 'U' {
				digits = 8
			}
			if character, ok := pythonHexRune(value, index+2, digits); ok {
				decoded.WriteRune(character)
				index += 2 + digits
				continue
			}
			decoded.WriteRune(utf8.RuneError)
			index += min(2+digits, len(value)-index)
			continue
		}
		if escape == 'x' {
			if character, ok := pythonHexRune(value, index+2, 2); ok {
				decoded.WriteRune(character)
				index += 4
				continue
			}
			decoded.WriteRune(utf8.RuneError)
			index += min(4, len(value)-index)
			continue
		}
		if escape == 'N' {
			if character, consumed, ok := pythonNamedDigit(value[index+2:]); ok {
				decoded.WriteRune(character)
				index += 2 + consumed
				continue
			}
		}
		if escape >= '0' && escape <= '7' {
			end := index + 2
			for end < len(value) && end < index+4 && value[end] >= '0' && value[end] <= '7' {
				end++
			}
			character, _ := strconv.ParseUint(string(value[index+1:end]), 8, 8)
			decoded.WriteRune(rune(character))
			index = end
			continue
		}
		if character, ok := map[byte]byte{
			'\\': '\\', '\'': '\'', '"': '"', 'a': '\a', 'b': '\b', 'f': '\f',
			'n': '\n', 'r': '\r', 't': '\t', 'v': '\v',
		}[escape]; ok {
			decoded.WriteByte(character)
			index += 2
			continue
		}
		decoded.WriteByte('\\')
		decoded.WriteByte(escape)
		index += 2
	}
	return decoded.String()
}

func pythonHexRune(value []byte, start int, digits int) (rune, bool) {
	if start+digits > len(value) {
		return 0, false
	}
	parsed, err := strconv.ParseUint(string(value[start:start+digits]), 16, 32)
	if err != nil || parsed > utf8.MaxRune || parsed >= 0xD800 && parsed <= 0xDFFF {
		return 0, false
	}
	return rune(parsed), true
}

func pythonNamedDigit(value []byte) (rune, int, bool) {
	if len(value) < 3 || value[0] != '{' {
		return 0, 0, false
	}
	end := bytes.IndexByte(value, '}')
	if end < 0 {
		return 0, 0, false
	}
	names := [...]string{"ZERO", "ONE", "TWO", "THREE", "FOUR", "FIVE", "SIX", "SEVEN", "EIGHT", "NINE"}
	name := string(value[1:end])
	for digit, word := range names {
		if name == "DIGIT "+word {
			return rune('0' + digit), end + 1, true
		}
	}
	return 0, 0, false
}

func djangoJSONIndent(value string) int {
	integer, err := pythoncompat.DecimalInteger(value)
	if err != nil || integer.Sign() <= 0 {
		return 0
	}
	maximum := big.NewInt(8)
	if integer.Cmp(maximum) >= 0 {
		return 8
	}
	return int(integer.Int64())
}

func projectMembershipReadsAllow(path string) string {
	switch {
	case strings.Contains(path, "/preferences/member/"):
		return "GET, PATCH, HEAD, OPTIONS"
	case strings.Contains(path, "/projects/") && strings.Contains(path, "/members/"):
		trimmed := strings.TrimSuffix(path, "/")
		if strings.HasSuffix(trimmed, "/members") {
			return "GET, POST, HEAD, OPTIONS"
		}
		return "GET, PATCH, DELETE, HEAD, OPTIONS"
	case strings.Contains(path, "/members/") && !strings.Contains(path, "/project-members/"):
		trimmed := strings.TrimSuffix(path, "/")
		if strings.HasSuffix(trimmed, "/members") {
			return "GET, HEAD, OPTIONS"
		}
		return "GET, PATCH, DELETE, HEAD, OPTIONS"
	default:
		return "GET, HEAD, OPTIONS"
	}
}

func mountProjectStateReadsProtocolRoutes(
	mux loomhttp.Muxer,
	server *projectstatereadssvr.Server,
	authorizer systemapi.ProjectStateReadsMetadataAuthorizer,
) {
	routes := []struct {
		path string
		get  http.Handler
	}{
		{path: "/api/workspaces/{slug}/projects/{project_id}/states/", get: server.List},
		{path: "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", get: server.Detail},
		{path: "/api/workspaces/{slug}/projects/{project_id}/intake-state/", get: server.Intake},
	}
	for _, route := range routes {
		mountProtocolHandler(
			mux,
			http.MethodHead,
			route.path,
			djangoProtocolHeadHandler(projectStateReadsCompatibilityMiddleware, route.get),
		)
	}
	mountProtocolHandler(
		mux,
		http.MethodOptions,
		"/api/workspaces/{slug}/projects/{project_id}/intake-state/",
		projectStateReadsOptionsHandler(authorizer),
	)
}

func projectStateReadsOptionsHandler(
	authorizer systemapi.ProjectStateReadsMetadataAuthorizer,
) http.Handler {
	return projectStateReadsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var sessionID string
		if cookie, err := request.Cookie("session-id"); err == nil {
			sessionID = cookie.Value
		}
		err := authorizer.AuthorizeMetadata(request.Context(), sessionID)
		if err != nil {
			var unauthenticated *projectstatereads.ProjectStateReadsUnauthenticated
			var permissionDenied *projectstatereads.ProjectStateReadsPermissionDenied
			var internalError *projectstatereads.ProjectStateReadsInternalError
			switch {
			case errors.As(err, &unauthenticated):
				writeStickyJSON(response, http.StatusUnauthorized, unauthenticated)
			case errors.As(err, &permissionDenied):
				writeStickyJSON(response, http.StatusForbidden, permissionDenied)
			case errors.As(err, &internalError):
				writeStickyJSON(response, http.StatusInternalServerError, internalError)
			default:
				writeStickyJSON(response, http.StatusInternalServerError, map[string]string{
					"error": projectStateReadsInternalErrorDetail,
				})
			}
			return
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": "Intake State Endpoint", "description": "", "renders": []string{"application/json"},
			"parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

const projectStateReadsInternalErrorDetail = "Something went wrong please try again later"

type projectStateReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse *systemapi.ProjectStateReadsResponseCapture
	jsonIndent  int
	status      int
	wroteHeader bool
}

func (writer *projectStateReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	removeVary(writer.Header(), "Cookie", "origin")
	mergeVary(writer.Header(), "Cookie", "origin")
	if status == http.StatusOK {
		prependVary(writer.Header(), "Accept-Encoding")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectStateReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	inputLength := len(body)
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 {
		body = writer.rawResponse.Data
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	if len(body) != inputLength {
		written, err := writer.ResponseWriter.Write(body)
		if err != nil {
			return 0, err
		}
		if written != len(body) {
			return 0, io.ErrShortWrite
		}
		return inputLength, nil
	}
	return writer.ResponseWriter.Write(body)
}

func projectStateReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		allow := projectStateReadsAllow(request.URL.Path)
		switch request.Method {
		case http.MethodPost:
			allow = "POST, OPTIONS"
		case http.MethodPatch, http.MethodDelete:
			allow = "PATCH, DELETE, OPTIONS"
		}
		headers.Set("Allow", allow)
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectStateReadsJSON(response, request, http.StatusNotFound, map[string]string{"detail": "Not found."})
				return
			}
		}
		accept, acceptPresent := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
		if negotiationErr != nil {
			writeDjangoAPIViewServerError(response, request)
			return
		}
		if !acceptsJSON {
			headers.Set("Content-Type", "application/json")
			writeProjectStateReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		if values := request.URL.Query()["grouped"]; len(values) > 1 {
			query := request.URL.Query()
			query.Set("grouped", values[len(values)-1])
			request = request.Clone(request.Context())
			clonedURL := *request.URL
			request.URL = &clonedURL
			request.URL.RawQuery = query.Encode()
		}
		rawResponse := &systemapi.ProjectStateReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectStateReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectStateReadsResponseWriter{
			ResponseWriter: response,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectStateReadsJSON(response http.ResponseWriter, request *http.Request, status int, value any) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func projectStateReadsAllow(path string) string {
	if strings.Contains(path, "/intake-state/") {
		return "GET, HEAD, OPTIONS"
	}
	trimmed := strings.TrimSuffix(path, "/")
	if strings.HasSuffix(trimmed, "/states") {
		return "GET, POST, HEAD, OPTIONS"
	}
	return "GET, PATCH, DELETE, HEAD, OPTIONS"
}

type projectLabelReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse *systemapi.ProjectLabelReadsResponseCapture
	jsonIndent  int
	status      int
	wroteHeader bool
}

func (writer *projectLabelReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	mergeVary(writer.Header(), "Cookie")
	if status == http.StatusOK {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectLabelReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	inputLength := len(body)
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 {
		body = writer.rawResponse.Data
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	if len(body) != inputLength {
		written, err := writer.ResponseWriter.Write(body)
		if err != nil {
			return 0, err
		}
		if written != len(body) {
			return 0, io.ErrShortWrite
		}
		return inputLength, nil
	}
	return writer.ResponseWriter.Write(body)
}

func projectLabelReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", projectLabelReadsAllow(request.URL.Path))
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectLabelReadsJSON(response, request, http.StatusNotFound, map[string]string{"detail": "Not found."})
				return
			}
		}
		accept, acceptPresent := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
		if negotiationErr != nil {
			writeDjangoAPIViewServerError(response, request)
			return
		}
		if !acceptsJSON {
			headers.Set("Content-Type", "application/json")
			writeProjectLabelReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		rawResponse := &systemapi.ProjectLabelReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectLabelReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectLabelReadsResponseWriter{
			ResponseWriter: response,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectLabelReadsJSON(response http.ResponseWriter, request *http.Request, status int, value any) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func projectLabelReadsAllow(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	if strings.HasSuffix(trimmed, "/issue-labels") {
		return "GET, POST, HEAD, OPTIONS"
	}
	return "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
}

type publicProjectMetadataReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse      *systemapi.PublicProjectMetadataReadsResponseCapture
	capturedResponse func() json.RawMessage
	jsonIndent       int
	status           int
	wroteHeader      bool
	body             bytes.Buffer
	acceptsGzip      bool
	appendGzipVary   bool
	skipCookieVary   bool
	skipGzip         bool
	omitLength       bool
	gzipPadding      func() (int, error)
}

func (writer *publicProjectMetadataReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	if !writer.skipCookieVary {
		mergeVary(writer.Header(), "Cookie")
	}
}

func (writer *publicProjectMetadataReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *publicProjectMetadataReadsResponseWriter) renderedBody() []byte {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.rawResponse != nil && writer.rawResponse.RenderError {
		writer.status = http.StatusInternalServerError
		writer.Header().Del("Allow")
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		return djangoServerErrorBody
	}
	body := writer.body.Bytes()
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 {
		body = writer.rawResponse.Data
	}
	if writer.status == http.StatusOK && writer.capturedResponse != nil {
		if captured := writer.capturedResponse(); len(captured) > 0 {
			body = captured
		}
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, bytes.TrimSpace(body)); err == nil {
		body = compacted.Bytes()
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	// DRF's JSONRenderer escapes JavaScript line and paragraph separators even
	// with ensure_ascii disabled.
	body = bytes.ReplaceAll(body, []byte("\u2028"), []byte(`\u2028`))
	body = bytes.ReplaceAll(body, []byte("\u2029"), []byte(`\u2029`))
	return body
}

func (writer *publicProjectMetadataReadsResponseWriter) finish() {
	body := writer.renderedBody()
	if !writer.skipGzip && len(body) >= 200 {
		if writer.appendGzipVary {
			mergeVary(writer.Header(), "Accept-Encoding")
		} else {
			prependVary(writer.Header(), "Accept-Encoding")
		}
		if writer.acceptsGzip && writer.Header().Get("Content-Encoding") == "" {
			compressed, err := djangoGzip(body, writer.gzipPadding)
			if err == nil && len(compressed) < len(body) {
				body = compressed
				writer.Header().Set("Content-Encoding", "gzip")
			}
		}
	}
	if writer.omitLength {
		writer.Header().Del("Content-Length")
	} else {
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	writer.ResponseWriter.WriteHeader(writer.status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

type djangoSessionCookieConfig struct {
	Name       string
	Domain     string
	Secure     bool
	Configured bool
}

const defaultCSRFCookieName = "csrftoken"

type djangoCSRFCookieConfig struct {
	Domain string
	Secure bool
}

type djangoCORSOriginalOriginContextKey struct{}

type djangoCORSOriginResponseWriter struct {
	http.ResponseWriter
	originalOrigin string
}

func (writer *djangoCORSOriginResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *djangoCORSOriginResponseWriter) restoreOrigin() {
	if writer.originalOrigin != "" && writer.Header().Get("Access-Control-Allow-Origin") != "" {
		writer.Header().Set("Access-Control-Allow-Origin", writer.originalOrigin)
	}
}

func (writer *djangoCORSOriginResponseWriter) WriteHeader(status int) {
	writer.restoreOrigin()
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *djangoCORSOriginResponseWriter) Write(body []byte) (int, error) {
	writer.restoreOrigin()
	return writer.ResponseWriter.Write(body)
}

type csrfTokenCORSHeaderWriter struct {
	headers http.Header
}

func (writer *csrfTokenCORSHeaderWriter) Header() http.Header {
	return writer.headers
}

func (writer *csrfTokenCORSHeaderWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (writer *csrfTokenCORSHeaderWriter) WriteHeader(int) {}

func djangoCORSRequestCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		normalized := normalizeDjangoCORSOrigin(request)
		if normalized == request {
			next.ServeHTTP(response, request)
			return
		}
		writer := &djangoCORSOriginResponseWriter{
			ResponseWriter: response,
			originalOrigin: djangoCORSOriginalOrigin(normalized),
		}
		next.ServeHTTP(writer, normalized)
	})
}

func normalizeDjangoCORSOrigin(request *http.Request) *http.Request {
	if _, normalized := request.Context().Value(djangoCORSOriginalOriginContextKey{}).(string); normalized {
		return request
	}
	if len(request.Header.Values("Origin")) == 0 {
		return request
	}
	original := djangoJoinedHeader(request, "Origin")
	clone := request.Clone(context.WithValue(request.Context(), djangoCORSOriginalOriginContextKey{}, original))
	clone.Header = request.Header.Clone()
	clone.Header.Del("Origin")
	if len(request.Header.Values("Origin")) > 1 {
		return clone
	}
	parsed, valid := djangoURLSplit(original)
	if original == "" || !valid {
		return clone
	}
	normalized := original
	if original != "null" && parsed.scheme != "" && parsed.netloc != "" {
		normalized = parsed.scheme + "://" + parsed.netloc
	}
	clone.Header.Set("Origin", normalized)
	return clone
}

func djangoCORSOriginalOrigin(request *http.Request) string {
	if original, ok := request.Context().Value(djangoCORSOriginalOriginContextKey{}).(string); ok {
		return original
	}
	return djangoJoinedHeader(request, "Origin")
}

func csrfTokenHeadersCompatibilityMiddleware(corsPolicy loomhttp.RuntimeCORSPolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/auth/get-csrf-token/" {
				next.ServeHTTP(response, request)
				return
			}
			setCSRFTokenCompatibilityHeaders(response.Header(), request, corsPolicy)
			next.ServeHTTP(response, request)
		})
	}
}

func csrfTokenCORSPreflightMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/auth/get-csrf-token/" && request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0 {
			writer := &csrfTokenResponseWriter{ResponseWriter: response}
			// Django accepts any requested method. Normalize it before forwarding to
			// Loom's generated OPTIONS dispatcher, then adapt Loom's preflight response.
			policyRequest := request.Clone(request.Context())
			policyRequest.Header = request.Header.Clone()
			policyRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
			next.ServeHTTP(writer, policyRequest)
			mergeVary(writer.Header(), "Origin")
			origin := djangoCORSOriginalOrigin(request)
			if origin == "" || !djangoCORSOriginValid(origin) {
				removeCORSAllowHeaders(writer.Header())
			} else if writer.Header().Get("Access-Control-Allow-Origin") != "" {
				writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			writer.status = http.StatusOK
			writer.wroteHeader = true
			writer.body.Reset()
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			removeVary(writer.Header(), "Access-Control-Request-Method", "Access-Control-Request-Headers")
			writer.finish(http.MethodOptions)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func removeCORSAllowHeaders(headers http.Header) {
	for _, name := range []string{
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Access-Control-Max-Age",
	} {
		headers.Del(name)
	}
}

func csrfTokenResponseCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/auth/get-csrf-token/" {
			next.ServeHTTP(response, request)
			return
		}
		if request.Method == http.MethodGet || request.Method == http.MethodHead {
			request = normalizeDjangoCSRFCookie(request)
		}
		writer := &csrfTokenResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		if writer.status == http.StatusOK && (request.Method == http.MethodGet || request.Method == http.MethodHead) {
			mergeVary(writer.Header(), "Cookie")
		} else if request.Method == http.MethodGet || request.Method == http.MethodHead {
			removeResponseCookie(writer.Header(), defaultCSRFCookieName)
		}
		if writer.status == http.StatusRequestEntityTooLarge {
			writer.Header().Del("Allow")
			if len(writer.Header().Values("Set-Cookie")) == 0 {
				removeVary(writer.Header(), "Cookie")
			}
		}
		writer.finish(request.Method)
	})
}

func djangoCSRFResponseCookiePolicy(config djangoCSRFCookieConfig) loomhttp.ResponseCookiePolicy {
	return func(_ context.Context, cookie *http.Cookie) error {
		if cookie.Name != defaultCSRFCookieName {
			return nil
		}
		if cookie.MaxAge <= 0 {
			return errors.New("generated CSRF response cookie has no positive Max-Age")
		}
		cookie.Domain = config.Domain
		cookie.Secure = config.Secure
		cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
		return nil
	}
}

func djangoCSRFTokenRuntimeCORSPolicy(allowedOrigins []string) (loomhttp.RuntimeCORSPolicy, error) {
	return djangoRuntimeCORSPolicy(allowedOrigins)
}

func djangoRuntimeCORSPolicy(allowedOrigins []string) (loomhttp.RuntimeCORSPolicy, error) {
	originPolicy := loomhttp.CORSOrigin{
		Methods:     []string{"DELETE", "GET", "OPTIONS", "PATCH", "POST", "PUT"},
		Headers:     []string{"accept", "authorization", "content-type", "user-agent", "x-csrftoken", "x-requested-with", "X-API-Key"},
		MaxAge:      86_400,
		Credentials: true,
	}
	origins := make([]loomhttp.CORSOrigin, 0, max(1, len(allowedOrigins)))
	if len(allowedOrigins) == 0 {
		origin := originPolicy
		origin.Pattern = ".+"
		origin.Regex = true
		origins = append(origins, origin)
	} else {
		seen := make(map[string]struct{}, len(allowedOrigins))
		for _, configured := range allowedOrigins {
			pattern := configured
			if configured != "null" {
				if parsed, valid := djangoURLSplit(configured); valid && parsed.scheme != "" && parsed.netloc != "" {
					pattern = parsed.scheme + "://" + parsed.netloc
				}
			}
			if pattern == "" {
				continue
			}
			if _, exists := seen[pattern]; exists {
				continue
			}
			seen[pattern] = struct{}{}
			origin := originPolicy
			origin.Pattern = pattern
			origins = append(origins, origin)
		}
	}
	return loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: origins})
}

// Go's request-cookie helper returns the first duplicate, while Django's cookie
// parser keeps the last. Prepending Django's effective value gives Loom's
// generated decoder the same input without replacing the generated decoder.
func normalizeDjangoCSRFCookie(request *http.Request) *http.Request {
	return normalizeDjangoCookie(request, defaultCSRFCookieName)
}

func normalizeDjangoCookie(request *http.Request, name string) *http.Request {
	cookies := request.CookiesNamed(name)
	if len(cookies) < 2 {
		return request
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	original := clone.Header.Values("Cookie")
	clone.Header.Del("Cookie")
	clone.Header.Add("Cookie", cookies[len(cookies)-1].String())
	for _, value := range original {
		clone.Header.Add("Cookie", value)
	}
	return clone
}

func removeResponseCookie(headers http.Header, name string) {
	values := headers.Values("Set-Cookie")
	headers.Del("Set-Cookie")
	for _, value := range values {
		cookie, err := http.ParseSetCookie(value)
		if err == nil && cookie.Name == name {
			continue
		}
		headers.Add("Set-Cookie", value)
	}
}

type csrfTokenResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func (writer *csrfTokenResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *csrfTokenResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *csrfTokenResponseWriter) finish(method string) {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if status == http.StatusOK {
		body = bytes.TrimSuffix(body, []byte{'\n'})
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func setCSRFTokenCompatibilityHeaders(headers http.Header, request *http.Request, corsPolicy loomhttp.RuntimeCORSPolicy) {
	headers.Set("Allow", "GET, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	removeVary(headers, "Origin")
	mergeVary(headers, "Cookie", "origin")
	origin := djangoCORSOriginalOrigin(request)
	if headers.Get("Access-Control-Allow-Origin") != "" {
		headers.Set("Access-Control-Allow-Origin", origin)
		if request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) == 0 {
			policyRequest := request.Clone(request.Context())
			policyRequest.Header = request.Header.Clone()
			policyRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
			policyResponse := &csrfTokenCORSHeaderWriter{headers: make(http.Header)}
			corsPolicy.HandlePreflight(policyResponse, policyRequest, []string{http.MethodGet, http.MethodOptions})
			for _, name := range []string{
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Max-Age",
			} {
				if value := policyResponse.Header().Get(name); value != "" {
					headers.Set(name, value)
				}
			}
		}
	}
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func djangoJoinedHeader(request *http.Request, name string) string {
	return djangoLatin1String([]byte(strings.Join(request.Header.Values(name), ",")))
}

func publicProjectMetadataReadsCompatibilityMiddleware(
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	manageCORS bool,
	activateTimezone bool,
	routePolicies ...publicProjectMetadataRoutePolicy,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			if manageCORS && publicProjectMetadataReadsCORSPreflight(headers, request, allowedCORSOrigins) {
				response.WriteHeader(http.StatusOK)
				return
			}
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			if manageCORS {
				publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
			}
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
			if tooLarge {
				writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, throttle)
				return
			}
			if bodyErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
				cookieErr := expireUnaccessedDjangoSessionCookie(headers, request, sessionCookie)
				if cookieErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
					headers.Del("X-Frame-Options")
				}
				mergeVary(headers, "origin")
				headers.Set("Content-Type", "text/html; charset=utf-8")
				headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(djangoServerErrorBody)
				return
			}
			mergeVary(headers, "Cookie", "origin")
			requestLogStarted := time.Now()
			allow := "GET, HEAD, OPTIONS"
			var routePolicy *publicProjectMetadataRoutePolicy
			if len(routePolicies) > 0 {
				routePolicy = &routePolicies[0]
				if routePolicy.allowMethods != nil {
					allow = routePolicy.allowMethods(request)
				}
			}
			headers.Set("Allow", allow)
			rawResponse := &systemapi.PublicProjectMetadataReadsResponseCapture{}
			writer := &publicProjectMetadataReadsResponseWriter{
				ResponseWriter: response,
				rawResponse:    rawResponse,
				acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
			}
			var preparedSessionRefresh *djangosession.SessionRefresh
			var userID string
			inspectSessionAfterResponse := false
			var sessionCookieErr error
			skipInnerRequestLog := false
			applySessionEffects := func(expireSession bool, refreshSession bool, rotatedSession *djangosession.SessionRefresh, resolvedUserID string) {
				if resolvedUserID != "" {
					userID = resolvedUserID
				}
				if expireSession {
					sessionCookieErr = expireDjangoSessionCookie(headers, sessionCookie)
				}
				if rotatedSession != nil {
					preparedSessionRefresh = rotatedSession
				} else if refreshSession {
					if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
						preparedSessionRefresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
					}
				}
			}
			defer func() {
				if inspectSessionAfterResponse {
					expireSession, refreshSession, rotatedSession, resolvedUserID, err := throttle.InspectSession(request.Context(), request)
					applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						skipInnerRequestLog = true
						writePublicProjectMetadataInternalError(writer, err, false, allow)
					}
				}
				innerBody := writer.renderedBody()
				if !skipInnerRequestLog {
					logPublicProjectMetadataInnerResponse(request, writer.status, requestLogStarted, userID)
				}
				writeAPIActivityLog(
					request.Context(), activityLogger, activitySecret, request, requestBody,
					writer.status, innerBody,
				)
				if sessionCookieErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), sessionCookieErr)
					replacePublicProjectMetadataSessionError(writer, sessionCookieErr)
				} else if preparedSessionRefresh != nil && writer.status < http.StatusInternalServerError {
					refresh, err := throttle.RefreshSession(request.Context(), request, *preparedSessionRefresh)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						replacePublicProjectMetadataSessionError(writer, err)
					} else if cookieErr := refreshDjangoSessionCookie(headers, sessionCookie, refresh); cookieErr != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
						replacePublicProjectMetadataSessionError(writer, cookieErr)
					}
				}
				writer.finish()
			}()
			formats, queryErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
			if queryErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), queryErr)
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if len(formats) > 0 {
				format := formats[len(formats)-1]
				if format != "" && format != "json" {
					inspectSessionAfterResponse = true
					headers.Set("Content-Type", "application/json")
					writeStickyJSON(writer, http.StatusNotFound, map[string]string{"detail": "Not found."})
					return
				}
			}
			accept, acceptPresent := djangoRequestAcceptHeader(request)
			acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
			if negotiationErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), negotiationErr)
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if !acceptsJSON {
				inspectSessionAfterResponse = true
				headers.Set("Content-Type", "application/json")
				writeStickyJSON(writer, http.StatusNotAcceptable, map[string]string{
					"detail": "Could not satisfy the request Accept header.",
				})
				return
			}
			writer.jsonIndent = jsonIndent
			if routePolicy != nil && routePolicy.requiresSession != nil && routePolicy.requiresSession(request) {
				lifecycle, ok := throttle.(userIdentitySessionLifecycle)
				if !ok {
					err := errors.New("public feedback required-session route needs session inspection")
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				expireSession, refreshSession, rotatedSession, resolvedUserID, err := lifecycle.InspectSessionWithTimezone(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					skipInnerRequestLog = publicProjectMetadataLazyAuthenticationError(err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				if resolvedUserID == "" {
					writeStickyJSON(writer, http.StatusUnauthorized, map[string]string{
						"detail": "Authentication credentials were not provided.",
					})
					return
				}
				request = request.WithContext(publicprojectmetadatareadsstore.WithUserID(request.Context(), resolvedUserID))
			} else {
				allowRequest := throttle.AllowRequestWithoutTimezone
				if activateTimezone {
					allowRequest = throttle.AllowRequest
				}
				allowed, wait, expireSession, refreshSession, rotatedSession, resolvedUserID, err := allowRequest(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					skipInnerRequestLog = publicProjectMetadataLazyAuthenticationError(err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				if !allowed {
					seconds := int(math.Ceil(wait.Seconds()))
					writer.Header().Set("Retry-After", strconv.Itoa(seconds))
					writeStickyJSON(writer, http.StatusTooManyRequests, struct {
						ErrorCode    int    `json:"error_code"`
						ErrorMessage string `json:"error_message"`
					}{
						ErrorCode: 5900, ErrorMessage: "RATE_LIMIT_EXCEEDED",
					})
					return
				}
				request = request.WithContext(publicprojectmetadatareadsstore.WithUserID(request.Context(), resolvedUserID))
			}
			request = request.WithContext(systemapi.WithPublicProjectMetadataReadsResponseCapture(request.Context(), rawResponse))
			next.ServeHTTP(writer, request)
		})
	}
}

type publicProjectMetadataRoutePolicy struct {
	allowMethods    func(*http.Request) string
	requiresSession func(*http.Request) bool
}

func publicFeedbackAllowMethods(request *http.Request) string {
	route, matched := classifyPublicFeedbackPath(request.URL.Path)
	if !matched {
		return "GET, HEAD, OPTIONS"
	}
	return route.allowMethods
}

var publicFeedbackPathPattern = regexp.MustCompile(`^/api/public/(?:anchor/[^/]+/(?:issues(?:/[^/]+(?:/votes)?)?|intakes/[^/]+/(?:inbox-issues|intake-issues(?:/[^/]+)?))|workspaces/[^/]+/project-boards)/?$`)

type publicFeedbackRoute struct {
	allowMethods    string
	requiresSession bool
	malformedUUID   bool
}

func classifyPublicFeedbackPath(requestPath string) (publicFeedbackRoute, bool) {
	path := strings.TrimPrefix(requestPath, "/")
	path = strings.TrimSuffix(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 5 || segments[0] != "api" || segments[1] != "public" {
		return publicFeedbackRoute{}, false
	}
	switch {
	case len(segments) == 5 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS"}, true
	case len(segments) == 6 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS", malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 7 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "issues" && segments[6] == "votes":
		return publicFeedbackRoute{allowMethods: "GET, POST, DELETE, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 7 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "intakes" && segments[5] != "" && (segments[6] == "inbox-issues" || segments[6] == "intake-issues"):
		return publicFeedbackRoute{allowMethods: "GET, POST, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5])}, true
	case len(segments) == 8 && segments[2] == "anchor" && segments[3] != "" && segments[4] == "intakes" && segments[6] == "intake-issues":
		return publicFeedbackRoute{allowMethods: "GET, PATCH, DELETE, HEAD, OPTIONS", requiresSession: true, malformedUUID: !djangoUUIDPathSegment(segments[5]) || !djangoUUIDPathSegment(segments[7])}, true
	case len(segments) == 5 && segments[2] == "workspaces" && segments[3] != "" && segments[4] == "project-boards":
		return publicFeedbackRoute{allowMethods: "GET, HEAD, OPTIONS"}, true
	default:
		return publicFeedbackRoute{}, false
	}
}

func publicFeedbackRouteRequiresSession(request *http.Request) bool {
	route, matched := classifyPublicFeedbackPath(request.URL.Path)
	return matched && route.requiresSession
}

func publicFeedbackRouteCompatibilityMiddleware(compatibility func(http.Handler) http.Handler, slashless http.Handler, malformed http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		methodNotAllowed := compatibility(publicFeedbackMethodNotAllowedHandler())
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			route, matched := classifyPublicFeedbackPath(request.URL.Path)
			if !matched {
				next.ServeHTTP(response, request)
				return
			}
			if route.malformedUUID {
				malformed.ServeHTTP(response, request)
				return
			}

			canonicalPath := strings.HasSuffix(request.URL.Path, "/")
			if !canonicalPath {
				slashless.ServeHTTP(response, request)
				return
			}

			method := strings.ToUpper(request.Method)
			if method != request.Method {
				request = request.Clone(request.Context())
				request.Method = method
			}
			if readServiceMethodAllowed(publicFeedbackAllowMethods(request), method) {
				next.ServeHTTP(response, request)
				return
			}
			methodNotAllowed.ServeHTTP(response, request)
		})
	}
}

func publicFeedbackMethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if publicFeedbackRouteRequiresSession(request) && publicprojectmetadatareadsstore.UserID(request.Context()) == "" {
			writeStickyJSON(response, http.StatusUnauthorized, map[string]string{
				"detail": "Authentication credentials were not provided.",
			})
			return
		}
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

func lastDjangoSessionCookie(request *http.Request, name string) *http.Cookie {
	return djangosession.CookieFromHeaderValues(request.Header.Values("Cookie"), name)
}

func djangoLatin1String(value []byte) string {
	characters := make([]rune, len(value))
	for index, octet := range value {
		characters[index] = rune(octet)
	}
	return string(characters)
}

func expireUnaccessedDjangoSessionCookie(headers http.Header, request *http.Request, config djangoSessionCookieConfig) error {
	cookie := lastDjangoSessionCookie(request, sessionCookieName(config))
	// SessionBase retains an unaccessed, valid-shaped key. Its lazy cache is
	// empty only when the supplied key fails Django's eight-character minimum.
	if cookie != nil && utf8.RuneCountInString(cookie.Value) < 8 {
		return expireDjangoSessionCookie(headers, config)
	}
	return nil
}

func writePublicProjectMetadataInternalError(writer *publicProjectMetadataReadsResponseWriter, err error, viewHandled bool, allowMethods ...string) {
	if !viewHandled || publicProjectMetadataLazyAuthenticationError(err) {
		replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
		return
	}
	if errors.Is(err, djangosession.ErrTimezoneNotFound) {
		replacePublicProjectMetadataResponse(writer, http.StatusBadRequest, "application/json", []byte(`{"error":"The required key does not exist."}`))
		writer.Header().Set("Allow", publicProjectMetadataErrorAllow(allowMethods))
		return
	}
	replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "application/json", []byte(`{"error":"Something went wrong please try again later"}`))
	writer.Header().Set("Allow", publicProjectMetadataErrorAllow(allowMethods))
}

func publicProjectMetadataErrorAllow(allowMethods []string) string {
	if len(allowMethods) > 0 && allowMethods[0] != "" {
		return allowMethods[0]
	}
	return "GET, HEAD, OPTIONS"
}

func publicProjectMetadataLazyAuthenticationError(err error) bool {
	return errors.Is(err, djangosession.ErrStore) ||
		errors.Is(err, djangosession.ErrSessionStore) ||
		errors.Is(err, djangosession.ErrUserStore) ||
		errors.Is(err, djangosession.ErrInvalidSessionShape) ||
		errors.Is(err, djangosession.ErrInvalidUserID)
}

func djangoQueryValues(rawQuery string, name string) ([]string, error) {
	if strings.Count(rawQuery, "&") >= 1000 {
		return nil, fmt.Errorf("query exceeds Django's 1000-field limit")
	}
	values := make([]string, 0, 1)
	for _, field := range strings.Split(rawQuery, "&") {
		key, value, found := strings.Cut(field, "=")
		if !found {
			value = ""
		}
		if djangoQueryUnescape(key) == name {
			values = append(values, djangoQueryUnescape(value))
		}
	}
	return values, nil
}

func normalizeDjangoTolerantQuery(request *http.Request) *http.Request {
	if request.URL.RawQuery == "" {
		return request
	}
	values := make(url.Values)
	for _, field := range strings.Split(request.URL.RawQuery, "&") {
		if field == "" {
			continue
		}
		key, value, found := strings.Cut(field, "=")
		if !found {
			value = ""
		}
		key = djangoQueryUnescape(key)
		values[key] = append(values[key], djangoQueryUnescape(value))
	}
	clone := request.Clone(request.Context())
	clonedURL := *request.URL
	clonedURL.RawQuery = values.Encode()
	clone.URL = &clonedURL
	return clone
}

func djangoQueryUnescape(value string) string {
	decoded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		switch {
		case value[index] == '+':
			decoded = append(decoded, ' ')
		case value[index] == '%' && index+2 < len(value):
			high, highOK := queryHexDigit(value[index+1])
			low, lowOK := queryHexDigit(value[index+2])
			if highOK && lowOK {
				decoded = append(decoded, high<<4|low)
				index += 2
				continue
			}
			decoded = append(decoded, value[index])
		default:
			decoded = append(decoded, value[index])
		}
	}
	// The UTF-8 decoder replaces maximal invalid subparts, as Python does.
	// ToValidUTF8 coalesces adjacent errors and loses that source behavior.
	text, _ := encodingunicode.UTF8.NewDecoder().String(string(decoded))
	return text
}

func queryHexDigit(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

var djangoBadRequestBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Bad Request (400)</title>\n</head>\n<body>\n  <h1>Bad Request (400)</h1><p></p>\n</body>\n</html>\n")

var djangoNotFoundBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Not Found</title>\n</head>\n<body>\n  <h1>Not Found</h1><p>The requested resource was not found on this server.</p>\n</body>\n</html>\n")

var djangoServerErrorBody = []byte("\n<!doctype html>\n<html lang=\"en\">\n<head>\n  <title>Server Error (500)</title>\n</head>\n<body>\n  <h1>Server Error (500)</h1><p></p>\n</body>\n</html>\n")

var djangoRequestTooLargeBody = []byte(`{"error": "REQUEST_BODY_TOO_LARGE", "detail": "The size of the request body exceeds the maximum allowed size."}`)

func djangoDebugAppendSlashResponse(request *http.Request) ([]byte, string) {
	pathWithSlash := djangoFullPath(request, true)
	message := fmt.Sprintf(
		"You called this URL via %s, but the URL doesn't end in a slash and you have APPEND_SLASH set. Django can't redirect to the slash URL while maintaining %s data. Change your form to point to %s%s (note the trailing slash), or set APPEND_SLASH=False in your Django settings.",
		request.Method, request.Method, request.Host, pathWithSlash,
	)
	if !djangoRequestAcceptsHTML(request) {
		return []byte(fmt.Sprintf("RuntimeError at %s\n\n%s\n", request.URL.Path, message)), "text/plain; charset=utf-8"
	}
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>RuntimeError at %s</title></head>
<body><h1>RuntimeError at %s</h1><pre class="exception_value">%s</pre></body>
</html>`, html.EscapeString(request.URL.Path), html.EscapeString(request.URL.Path), html.EscapeString(message))), "text/html; charset=utf-8"
}

func djangoDebugAppendSlashBody(request *http.Request) []byte {
	body, _ := djangoDebugAppendSlashResponse(request)
	return body
}

func djangoRequestAcceptsHTML(request *http.Request) bool {
	accept, present := djangoRequestAcceptHeader(request)
	if !present {
		return true
	}
	for _, value := range strings.Split(accept, ",") {
		mediaType, parameters, err := djangoParseHeaderParameters(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		quality := 1.0
		if rawQuality, exists := parameters["q"]; exists {
			if parsed, parseErr := strconv.ParseFloat(rawQuality, 64); parseErr == nil && parsed >= 0 && parsed <= 1 {
				quality = math.Round(parsed*1000) / 1000
			}
		}
		if quality == 0 {
			continue
		}
		parameterMismatch := false
		for name := range parameters {
			if name != "q" {
				parameterMismatch = true
				break
			}
		}
		if !parameterMismatch && (mediaType == "*/*" || mediaType == "text/*" || mediaType == "text/html") {
			return true
		}
	}
	return false
}

func djangoEscapeURIPath(path string) string {
	return djangoPercentEncode(path, "/:@&+$,-.!~*'()")
}

func djangoFullPath(request *http.Request, appendSlash bool) string {
	path := request.URL.Path
	if appendSlash {
		path += "/"
	}
	fullPath := djangoEscapeURIPath(path)
	if request.URL.RawQuery != "" {
		fullPath += "?" + djangoPercentEncode(request.URL.RawQuery, "/#%[]=:;$&()+,!?*@'~")
	}
	return fullPath
}

func djangoPercentEncode(value string, safe string) string {
	const hexadecimal = "0123456789ABCDEF"
	var escaped strings.Builder
	escaped.Grow(len(value))
	for index := 0; index < len(value); index++ {
		octet := value[index]
		if (octet >= 'a' && octet <= 'z') || (octet >= 'A' && octet <= 'Z') ||
			(octet >= '0' && octet <= '9') || strings.ContainsRune("_-."+safe, rune(octet)) {
			escaped.WriteByte(octet)
			continue
		}
		escaped.WriteByte('%')
		escaped.WriteByte(hexadecimal[octet>>4])
		escaped.WriteByte(hexadecimal[octet&0x0f])
	}
	return escaped.String()
}

func replacePublicProjectMetadataResponse(writer *publicProjectMetadataReadsResponseWriter, status int, contentType string, body []byte) {
	writer.body.Reset()
	writer.rawResponse = nil
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("Allow")
	writer.Header().Del("Content-Encoding")
	writer.Header().Del("Cache-Control")
	writer.Header().Set("Content-Type", contentType)
	_, _ = writer.body.Write(body)
}

func replacePublicProjectMetadataSessionError(writer *publicProjectMetadataReadsResponseWriter, err error) {
	status := http.StatusInternalServerError
	body := djangoServerErrorBody
	if errors.Is(err, djangosession.ErrSessionNotFound) || errors.Is(err, djangosession.ErrSessionUpdate) {
		status = http.StatusBadRequest
		body = djangoBadRequestBody
	}
	replacePublicProjectMetadataResponse(writer, status, "text/html; charset=utf-8", body)
	// SessionMiddleware raises outside CommonMiddleware, XFrameOptionsMiddleware,
	// and GZipMiddleware. Their headers and body processing do not apply to the
	// replacement error response.
	writer.Header().Del("X-Frame-Options")
	writer.Header().Del("Location")
	writer.Header().Del("Retry-After")
	removeVary(writer.Header(), "Cookie", "Accept-Encoding")
	writer.skipGzip = true
	writer.omitLength = true
}

func writePublicProjectMetadataRequestTooLarge(
	response http.ResponseWriter,
	request *http.Request,
	sessionCookie djangoSessionCookieConfig,
	throttle publicProjectMetadataThrottle,
) {
	headers := response.Header()
	expireSession, refreshSession, err := throttle.FinalizeUnaccessedSession(
		request.Context(), request, http.StatusRequestEntityTooLarge,
	)
	if expireSession {
		if cookieErr := expireDjangoSessionCookie(headers, sessionCookie); cookieErr != nil {
			mergeVary(headers, "origin")
			logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
			writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
			replacePublicProjectMetadataSessionError(writer, cookieErr)
			writer.finish()
			return
		}
	}
	if err != nil {
		mergeVary(headers, "origin")
		logPublicProjectMetadataCompatibilityError(request.Context(), err)
		writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
		replacePublicProjectMetadataSessionError(writer, err)
		writer.finish()
		return
	}
	if refreshSession != nil {
		if cookieErr := refreshUnaccessedDjangoSessionCookie(headers, sessionCookie, *refreshSession); cookieErr != nil {
			mergeVary(headers, "origin")
			logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
			writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
			replacePublicProjectMetadataSessionError(writer, cookieErr)
			writer.finish()
			return
		}
	}
	mergeVary(headers, "origin")
	headers.Set("Content-Type", "application/json")
	headers.Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
	response.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = response.Write(djangoRequestTooLargeBody)
}

func expireDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig) error {
	config.Name = sessionCookieName(config)
	cookie := &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     "/",
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("delete Django session cookie: %w", err)
	}
	parts := []string{config.Name + `=""`}
	if config.Domain != "" {
		parts = append(parts, "Domain="+config.Domain)
	}
	parts = append(parts, "expires=Thu, 01 Jan 1970 00:00:00 GMT", "Max-Age=0", "Path=/", "SameSite=Lax")
	if strings.HasPrefix(config.Name, "__Secure-") || strings.HasPrefix(config.Name, "__Host-") {
		parts = append(parts, "Secure")
	}
	headers.Add("Set-Cookie", strings.Join(parts, "; "))
	mergeVary(headers, "Cookie")
	return nil
}

func refreshDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh) error {
	return writeDjangoSessionCookie(headers, config, refresh, true)
}

func refreshUnaccessedDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh) error {
	return writeDjangoSessionCookie(headers, config, refresh, false)
}

func writeDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh, varyCookie bool) error {
	config.Name = sessionCookieName(config)
	cookie := &http.Cookie{Name: config.Name, Value: refresh.SessionKey, Path: "/", Domain: config.Domain}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("set Django session cookie: %w", err)
	}
	parts := []string{config.Name + "=" + refresh.SessionKey}
	if config.Domain != "" {
		parts = append(parts, "Domain="+config.Domain)
	}
	if !refresh.BrowserClose {
		expiresAt := refresh.CookieExpiresAt
		if expiresAt.IsZero() {
			expiresAt = refresh.ExpiresAt
		}
		parts = append(parts, "expires="+expiresAt.UTC().Format(http.TimeFormat))
	}
	parts = append(parts, "HttpOnly")
	if !refresh.BrowserClose {
		parts = append(parts, "Max-Age="+strconv.FormatInt(int64(refresh.MaxAgeSeconds), 10))
	}
	parts = append(parts, "Path=/", "SameSite=Lax")
	if config.Secure {
		parts = append(parts, "Secure")
	}
	headers.Add("Set-Cookie", strings.Join(parts, "; "))
	if varyCookie {
		mergeVary(headers, "Cookie")
	}
	return nil
}

func sessionCookieName(config djangoSessionCookieConfig) string {
	if config.Configured || config.Name != "" {
		return config.Name
	}
	return djangosession.DefaultCookieName
}

func publicProjectMetadataReadsCORSPreflight(headers http.Header, request *http.Request, allowedOrigins []string) bool {
	if request.Method != http.MethodOptions || len(request.Header.Values("Access-Control-Request-Method")) == 0 {
		return false
	}
	headers.Set("Content-Length", "0")
	headers.Set("Content-Type", "text/html; charset=utf-8")
	mergeVary(headers, "origin")
	if publicProjectMetadataReadsCORSHeaders(headers, request, allowedOrigins) {
		headers.Set("Access-Control-Allow-Headers", "accept, authorization, content-type, user-agent, x-csrftoken, x-requested-with, X-API-Key")
		headers.Set("Access-Control-Allow-Methods", "DELETE, GET, OPTIONS, PATCH, POST, PUT")
		headers.Set("Access-Control-Max-Age", "86400")
	}
	return true
}

func publicProjectMetadataReadsCORSHeaders(headers http.Header, request *http.Request, allowedOrigins []string) bool {
	origin := djangoJoinedHeader(request, "Origin")
	originURL, valid := djangoURLSplit(origin)
	if origin == "" || !valid {
		return false
	}
	allowed := len(allowedOrigins) == 0
	for _, candidate := range allowedOrigins {
		candidateURL, candidateValid := djangoURLSplit(candidate)
		if (origin == "null" && candidate == "null") ||
			(candidateValid && originURL.scheme == candidateURL.scheme && originURL.netloc == candidateURL.netloc) {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Credentials", "true")
	return true
}

func djangoCORSOriginValid(origin string) bool {
	_, valid := djangoURLSplit(origin)
	return valid
}

type djangoURLSplitResult struct {
	scheme string
	netloc string
}

func djangoURLSplit(origin string) (djangoURLSplitResult, bool) {
	// Python 3.12's urlsplit() strips these characters before examining the
	// authority and rejects malformed bracketed hosts and NFKC delimiters.
	origin = strings.TrimLeftFunc(strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(origin), func(value rune) bool {
		return value <= ' '
	})
	result := djangoURLSplitResult{}
	rest := origin
	if separator := strings.IndexByte(rest, ':'); separator > 0 {
		firstDelimiter := strings.IndexAny(rest, "/?#")
		if (firstDelimiter < 0 || separator < firstDelimiter) && djangoURLSchemeValid(rest[:separator]) {
			result.scheme = strings.ToLower(rest[:separator])
			rest = rest[separator+1:]
		}
	}
	if !strings.HasPrefix(rest, "//") {
		return result, true
	}
	authority := rest[2:]
	if end := strings.IndexAny(authority, "/?#"); end >= 0 {
		authority = authority[:end]
	}
	result.netloc = authority
	if strings.Contains(authority, "[") != strings.Contains(authority, "]") {
		return djangoURLSplitResult{}, false
	}
	if strings.Contains(authority, "[") {
		// Python 3.12 validates only the contents between the first '[' and
		// the first following ']'; it intentionally does not validate where
		// that pair appears in the authority or what follows it.
		host := strings.SplitN(strings.SplitN(authority, "[", 2)[1], "]", 2)[0]
		if strings.HasPrefix(host, "v") {
			version, suffix, found := strings.Cut(host[1:], ".")
			if !found || version == "" || suffix == "" || strings.IndexFunc(version, func(value rune) bool {
				return !((value >= '0' && value <= '9') || (value >= 'a' && value <= 'f') || (value >= 'A' && value <= 'F'))
			}) >= 0 {
				return djangoURLSplitResult{}, false
			}
		} else {
			address := host
			if base, zone, found := strings.Cut(host, "%"); found {
				if zone == "" || strings.Contains(zone, "%") {
					return djangoURLSplitResult{}, false
				}
				address = base
			}
			if net.ParseIP(address) == nil || !strings.Contains(address, ":") {
				return djangoURLSplitResult{}, false
			}
		}
	}
	check := strings.NewReplacer("@", "", ":", "", "#", "", "?", "").Replace(authority)
	normalized := norm.NFKC.String(check)
	if check != normalized && strings.ContainsAny(normalized, "/?#@:") {
		return djangoURLSplitResult{}, false
	}
	return result, true
}

func djangoURLSchemeValid(value string) bool {
	for index, character := range value {
		if index == 0 && !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')) {
			return false
		}
		if index > 0 && !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '+' || character == '-' || character == '.') {
			return false
		}
	}
	return value != ""
}

func djangoJSONColonSpacing(body []byte) []byte {
	spaced := make([]byte, 0, len(body)+bytes.Count(body, []byte(":")))
	inString := false
	escaped := false
	for _, value := range body {
		spaced = append(spaced, value)
		if inString {
			if escaped {
				escaped = false
			} else if value == '\\' {
				escaped = true
			} else if value == '"' {
				inString = false
			}
			continue
		}
		if value == '"' {
			inString = true
		} else if value == ':' {
			spaced = append(spaced, ' ')
		}
	}
	return spaced
}

func prependVary(header http.Header, value string) {
	values := []string{value}
	for _, existing := range strings.Split(header.Get("Vary"), ",") {
		existing = strings.TrimSpace(existing)
		if existing != "" && !strings.EqualFold(existing, value) {
			values = append(values, existing)
		}
	}
	header.Set("Vary", strings.Join(values, ", "))
}

func acceptsGzipEncoding(header string) bool {
	for offset := 0; offset < len(header); {
		match := strings.Index(header[offset:], "gzip")
		if match < 0 {
			return false
		}
		start := offset + match
		end := start + len("gzip")
		leftWord := false
		if start > 0 {
			character, _ := utf8.DecodeLastRuneInString(header[:start])
			leftWord = djangoRegexWordCharacter(character)
		}
		rightWord := false
		if end < len(header) {
			character, _ := utf8.DecodeRuneInString(header[end:])
			rightWord = djangoRegexWordCharacter(character)
		}
		if !leftWord && !rightWord {
			return true
		}
		offset = start + 1
	}
	return false
}

func djangoRegexWordCharacter(character rune) bool {
	return character == '_' || unicode.IsLetter(character) || unicode.IsNumber(character)
}

func djangoGzip(body []byte, paddingLength func() (int, error)) ([]byte, error) {
	var compressed bytes.Buffer
	compressor := gzip.NewWriter(&compressed)
	if _, err := compressor.Write(body); err != nil {
		return nil, err
	}
	if err := compressor.Close(); err != nil {
		return nil, err
	}
	if paddingLength == nil {
		paddingLength = func() (int, error) {
			length, err := rand.Int(rand.Reader, big.NewInt(100))
			if err != nil {
				return 0, err
			}
			return int(length.Int64()), nil
		}
	}
	length, err := paddingLength()
	if err != nil {
		return nil, err
	}
	if length < 0 || length >= 100 {
		return nil, fmt.Errorf("invalid gzip padding length %d", length)
	}
	data := compressed.Bytes()
	padded := make([]byte, 0, len(data)+length+1)
	padded = append(padded, data[:3]...)
	padded = append(padded, data[3]|0x08)
	padded = append(padded, data[4:10]...)
	padded = append(padded, bytes.Repeat([]byte{'a'}, length)...)
	padded = append(padded, 0)
	padded = append(padded, data[10:]...)
	return padded, nil
}

func mountUserPropertiesProtocolRoutes(mux loomhttp.Muxer, server *userpropertiessvr.Server) {
	routes := []struct {
		path string
		name string
		get  http.Handler
	}{
		{"/api/workspaces/{slug}/user-properties/", "Workspace User Properties Endpoint", server.WorkspaceGet},
		{"/api/workspaces/{slug}/projects/{project_id}/user-properties/", "Project User Display Property Endpoint", server.ProjectGet},
		{"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/user-properties/", "Cycle User Properties Endpoint", server.CycleGet},
		{"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/user-properties/", "Module User Properties Endpoint", server.ModuleGet},
	}
	for _, route := range routes {
		mountProtocolHandler(mux, http.MethodHead, route.path, djangoProtocolHeadHandler(userPropertiesCompatibilityMiddleware, route.get))
		mountProtocolHandler(mux, http.MethodOptions, route.path, djangoProtocolOptionsHandler(userPropertiesCompatibilityMiddleware, route.name, true))
	}
}

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

func djangoJSONCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "Cookie", "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

type workspaceContentResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (writer *workspaceContentResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.Header().Del("loom-error")
	if status == http.StatusOK || status == http.StatusCreated {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	if status == http.StatusUnauthorized {
		mergeVary(writer.Header(), "Cookie")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *workspaceContentResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func workspaceContentCompatibilityMiddleware(resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			if strings.HasSuffix(strings.TrimSuffix(request.URL.Path, "/"), "/"+resource) {
				headers.Set("Allow", "GET, POST, HEAD, OPTIONS")
			} else {
				headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
			}
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			mergeVary(headers, "origin")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
		})
	}
}

func mountWorkspaceContentProtocolRoutes(mux loomhttp.Muxer, resource, name string, list, retrieve http.Handler) {
	collection := "/api/workspaces/{slug}/" + resource + "/"
	detail := collection + "{pk}/"
	mountProtocolHandler(mux, "HEAD", collection, workspaceContentHeadHandler(resource, list))
	mountProtocolHandler(mux, "HEAD", detail, workspaceContentHeadHandler(resource, retrieve))
	mountProtocolHandler(mux, "OPTIONS", collection, workspaceContentOptionsHandler(resource, name+" List", true))
	mountProtocolHandler(mux, "OPTIONS", detail, workspaceContentOptionsHandler(resource, name, false))
}

func workspaceContentHeadHandler(resource string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceContentCompatibilityMiddleware(resource)(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceContentOptionsHandler(resource, name string, collection bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceContentCompatibilityMiddleware(resource)(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			method := "PATCH"
			if collection {
				method = "POST"
			}
			writeStickyJSON(response, http.StatusOK, map[string]any{
				"name":        name,
				"description": "",
				"renders":     []string{"application/json"},
				"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
				"actions":     map[string]any{method: map[string]any{}},
			})
		})).ServeHTTP(response, request)
	})
}

func workspaceHomePreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, PATCH, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func mountWorkspaceHomePreferencesProtocolRoutes(mux loomhttp.Muxer, list, retrieve http.Handler) {
	collection := "/api/workspaces/{slug}/home-preferences/"
	detail := collection + "{key}/"
	mountProtocolHandler(mux, "HEAD", collection, workspaceHomePreferencesHeadHandler(list))
	mountProtocolHandler(mux, "HEAD", detail, workspaceHomePreferencesHeadHandler(retrieve))
	mountProtocolHandler(mux, "OPTIONS", collection, workspaceHomePreferencesOptionsHandler("Workspace Home Preference List"))
	mountProtocolHandler(mux, "OPTIONS", detail, workspaceHomePreferencesOptionsHandler("Workspace Home Preference"))
}

func workspaceHomePreferencesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceHomePreferencesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceHomePreferencesOptionsHandler(name string) http.Handler {
	return workspaceHomePreferencesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions":     map[string]any{"PATCH": map[string]any{}},
		})
	}))
}

func workspaceMemberMeCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func workspaceMemberMeHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceMemberMeCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceMemberMeOptionsHandler() http.Handler {
	return workspaceMemberMeCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "Workspace Member User Endpoint",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func workspaceRecentVisitsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func mountWorkspaceRecentVisitsProtocolRoutes(mux loomhttp.Muxer, list http.Handler) {
	collection := "/api/workspaces/{slug}/recent-visits/"
	mountProtocolHandler(mux, http.MethodHead, collection, workspaceRecentVisitsHeadHandler(list))
	mountProtocolHandler(mux, http.MethodOptions, collection, workspaceRecentVisitsOptionsHandler())
	mountProtocolHandler(mux, http.MethodPost, collection, workspaceRecentVisitsMethodNotAllowed())
}

func workspaceRecentVisitsHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceRecentVisitsCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceRecentVisitsOptionsHandler() http.Handler {
	return workspaceRecentVisitsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "Workspace Recent Visit List",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func workspaceRecentVisitsMethodNotAllowed() http.Handler {
	return workspaceRecentVisitsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{"detail": fmt.Sprintf("Method %q not allowed.", request.Method)})
	}))
}

func lastVisitedWorkspaceCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func lastVisitedWorkspaceHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		lastVisitedWorkspaceCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func lastVisitedWorkspaceOptionsHandler() http.Handler {
	return lastVisitedWorkspaceCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        "User Last Project With Workspace Endpoint",
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

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

func workspaceFavoritesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		if strings.HasSuffix(request.URL.Path, "/group/") {
			headers.Set("Allow", "GET, HEAD, OPTIONS")
		} else {
			headers.Set("Allow", "GET, POST, PATCH, DELETE, HEAD, OPTIONS")
		}
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
	})
}

func entityFavoritesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Allow", entityFavoritesAllow(r.URL.Path))
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Referrer-Policy", "same-origin")
		mergeVary(h, "Cookie", "origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: w}, r)
	})
}

func entityFavoritesAllow(path string) string {
	if strings.HasSuffix(path, "user-favorite-projects/") || strings.HasSuffix(path, "user-favorite-cycles/") || strings.HasSuffix(path, "user-favorite-modules/") || strings.HasSuffix(path, "user-favorite-views/") {
		return "GET, POST, HEAD, OPTIONS"
	}
	if strings.Contains(path, "favorite-pages/") {
		return "POST, DELETE, OPTIONS"
	}
	return "DELETE, OPTIONS"
}

func mountEntityFavoritesProtocolRoutes(mux loomhttp.Muxer, server *entityfavoritessvr.Server) {
	collections := []struct {
		path string
		list http.Handler
	}{
		{"/api/workspaces/{slug}/user-favorite-projects/", server.ProjectList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-cycles/", server.CycleList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-modules/", server.ModuleList},
		{"/api/workspaces/{slug}/projects/{project_id}/user-favorite-views/", server.ViewList},
	}
	for _, route := range collections {
		mountProtocolHandler(mux, http.MethodHead, route.path, entityFavoritesHeadHandler(route.list))
		mountProtocolHandler(mux, http.MethodOptions, route.path, entityFavoritesOptionsHandler())
		mountProtocolHandler(mux, http.MethodOptions, route.path+"{id}/", entityFavoritesOptionsHandler())
	}
	page := "/api/workspaces/{slug}/projects/{project_id}/favorite-pages/{page_id}/"
	mountProtocolHandler(mux, http.MethodOptions, page, entityFavoritesOptionsHandler())
}

func entityFavoritesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityFavoritesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{w}, r)
	})
}

func entityFavoritesOptionsHandler() http.Handler {
	return entityFavoritesCompatibilityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(w, http.StatusOK, map[string]any{"name": "Entity Favorite Endpoint", "description": "", "renders": []string{"application/json"}, "parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}})
	}))
}

func mountWorkspaceFavoritesProtocolRoutes(mux loomhttp.Muxer, list, retrieve, group http.Handler) {
	collection := "/api/workspaces/{slug}/user-favorites/"
	detail := collection + "{favorite_id}/"
	groupPath := detail + "group/"
	mountProtocolHandler(mux, http.MethodHead, collection, workspaceFavoritesHeadHandler(list))
	mountProtocolHandler(mux, http.MethodHead, detail, workspaceFavoritesHeadHandler(retrieve))
	mountProtocolHandler(mux, http.MethodHead, groupPath, workspaceFavoritesHeadHandler(group))
	mountProtocolHandler(mux, http.MethodOptions, collection, workspaceFavoritesOptionsHandler("Workspace Favorite List", "POST"))
	mountProtocolHandler(mux, http.MethodOptions, detail, workspaceFavoritesOptionsHandler("Workspace Favorite", "PATCH"))
	mountProtocolHandler(mux, http.MethodOptions, groupPath, workspaceFavoritesOptionsHandler("Workspace Favorite Group", ""))
}

func workspaceFavoritesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		workspaceFavoritesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func workspaceFavoritesOptionsHandler(name, action string) http.Handler {
	return workspaceFavoritesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		body := map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		}
		if action != "" {
			body["actions"] = map[string]any{action: map[string]any{}}
		}
		writeStickyJSON(response, http.StatusOK, body)
	}))
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

func djangoJSONHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		djangoJSONCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userNotificationPreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserNotificationPreferencesHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userNotificationPreferencesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userNotificationPreferencesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userNotificationPreferencesOptionsHandler(response http.ResponseWriter, _ *http.Request) {
	setUserNotificationPreferencesHeaders(response.Header())
	writeStickyJSON(response, http.StatusOK, map[string]any{
		"name":        "User Notification Preference Endpoint",
		"description": "",
		"renders":     []string{"application/json"},
		"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	})
}

func setUserNotificationPreferencesHeaders(headers http.Header) {
	headers.Set("Allow", "GET, PATCH, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func userAccountsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserAccountsHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userAccountsHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userAccountsCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userAccountsOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserAccountsHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	})
}

func setUserAccountsHeaders(headers http.Header) {
	headers.Set("Allow", "GET, DELETE, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func userAPITokensCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserAPITokensHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userAPITokensHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userAPITokensCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userAPITokensOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserAPITokensHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	})
}

func setUserAPITokensHeaders(headers http.Header) {
	headers.Set("Allow", "GET, POST, PATCH, DELETE, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func userProfileFlagsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserProfileFlagsHeaders(response.Header())
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
			return
		}
		body, status, validation := normalizeUserProfileFlagsBody(request.URL.Path, body)
		if validation != nil {
			writeStickyJSON(response, status, validation)
			return
		}
		request.Header.Set("Content-Type", "application/json")
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func normalizeUserProfileFlagsBody(path string, body []byte) ([]byte, int, map[string]string) {
	if len(bytes.TrimSpace(body)) == 0 {
		return []byte("{}"), 0, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		if !json.Valid(body) {
			return nil, http.StatusBadRequest, map[string]string{"detail": "JSON parse error - " + err.Error()}
		}
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	if fields == nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	field := "is_onboarded"
	if path == "/api/users/me/tour-completed/" {
		field = "is_tour_completed"
	}
	raw, exists := fields[field]
	if !exists {
		return body, 0, nil
	}
	value, status, validation := djangoBooleanValue(raw)
	if validation != nil {
		return nil, status, validation
	}
	fields[field], _ = json.Marshal(value)
	normalized, err := json.Marshal(fields)
	if err != nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	return normalized, 0, nil
}

func djangoBooleanValue(raw json.RawMessage) (bool, int, map[string]string) {
	switch string(bytes.TrimSpace(raw)) {
	case "true", "1":
		return true, 0, nil
	case "false", "0":
		return false, 0, nil
	case "null":
		return false, http.StatusBadRequest, map[string]string{"error": "The payload is not valid"}
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		switch value {
		case "t", "True", "1":
			return true, 0, nil
		case "f", "False", "0":
			return false, 0, nil
		}
	}
	return false, http.StatusBadRequest, map[string]string{"error": "Please provide valid detail"}
}

func setUserProfileFlagsHeaders(headers http.Header) {
	headers.Set("Allow", "PATCH, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func userProfileFlagsHeadHandler(response http.ResponseWriter, _ *http.Request) {
	setUserProfileFlagsHeaders(response.Header())
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusMethodNotAllowed)
}

func userProfileFlagsOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserProfileFlagsHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
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

const timezoneCacheDuration = 2 * time.Hour

func timezoneCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		setTimezoneProtocolHeaders(headers)
		headers.Set("Cache-Control", "max-age=7200")
		headers.Set("Expires", time.Now().UTC().Add(timezoneCacheDuration).Format(http.TimeFormat))
		next.ServeHTTP(response, request)
	})
}

func timezoneProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setTimezoneProtocolHeaders(response.Header())
		next.ServeHTTP(response, request)
	})
}

func setTimezoneProtocolHeaders(headers http.Header) {
	headers.Set("Allow", "GET, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	headers.Set("Vary", "Accept-Encoding, Cookie, origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func stickyFrontendTrailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if stickyDetailPathNeedsTrailingSlash(request.Method, request.URL.Path) {
			request = request.Clone(request.Context())
			requestURL := *request.URL
			requestURL.Path += "/"
			request.URL = &requestURL
		}
		next.ServeHTTP(response, request)
	})
}

func stickyDetailPathNeedsTrailingSlash(method, path string) bool {
	if method != http.MethodGet && method != http.MethodDelete {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 5 && parts[0] == "api" && parts[1] == "workspaces" && parts[2] != "" && parts[3] == "stickies" && parts[4] != "" && !strings.HasSuffix(path, "/")
}
func stickyAuthenticationMiddleware(authenticate security.AuthAPIKeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			setStickyHeaders(response.Header())
			cookie, err := request.Cookie("session-id")
			if err != nil || cookie.Value == "" {
				writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			ctx, err := authenticate(request.Context(), cookie.Value, &security.APIKeyScheme{Name: "plane_session_cookie"})
			if err != nil {
				writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

const stickyRequestLimit = 10 * 1024 * 1024

type stickyResponseWriter struct{ http.ResponseWriter }

func (writer stickyResponseWriter) WriteHeader(status int) {
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
}

func (writer stickyResponseWriter) Write(body []byte) (int, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if decoder.Decode(&value) == nil {
		addStickyNulls(value)
		encoded, err := json.Marshal(value)
		if err == nil {
			encoded = append(encoded, '\n')
			_, writeErr := writer.ResponseWriter.Write(encoded)
			return len(body), writeErr
		}
	}
	return writer.ResponseWriter.Write(body)
}

func addStickyNulls(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	if results, ok := object["results"].([]any); ok {
		for _, field := range []string{"grouped_by", "sub_grouped_by", "extra_stats"} {
			if _, exists := object[field]; !exists {
				object[field] = nil
			}
		}
		for _, result := range results {
			addStickyNulls(result)
		}
	}
	if _, isSticky := object["id"]; !isSticky {
		return
	}
	for _, field := range []string{"created_by", "updated_by", "deleted_at", "name", "color", "background_color", "description_stripped", "description_binary"} {
		if _, exists := object[field]; !exists {
			object[field] = nil
		}
	}
}

func stickyCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		setStickyProtocolHeaders(headers, request.URL.Path)
		if (request.Method == http.MethodGet || request.Method == http.MethodHead) && strings.HasSuffix(request.URL.Path, "/stickies/") {
			if detail := validateStickyPagination(request); detail != "" {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": detail})
				return
			}
		}
		if request.Method == http.MethodPost || request.Method == http.MethodPatch {
			body, err := io.ReadAll(io.LimitReader(request.Body, stickyRequestLimit+1))
			if err != nil {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
				return
			}
			if len(body) > stickyRequestLimit {
				headers.Set("Content-Type", "application/json")
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			body, err = normalizeStickyRequestBody(request.Header.Get("Content-Type"), body)
			if err != nil {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Body = io.NopCloser(bytes.NewReader(body))
			if validation := validateStickyRequest(body); validation != nil {
				writeStickyJSON(response, http.StatusBadRequest, validation)
				return
			}
		}
		next.ServeHTTP(stickyResponseWriter{response}, request)
	})
}

func stickyProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setStickyProtocolHeaders(response.Header(), request.URL.Path)
		next.ServeHTTP(response, request)
	})
}

func setStickyProtocolHeaders(headers http.Header, path string) {
	setStickyHeaders(headers)
	if strings.HasSuffix(path, "/stickies/") {
		headers.Set("Allow", "GET, POST, HEAD, OPTIONS")
		return
	}
	headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
}

func normalizeStickyRequestBody(contentType string, body []byte) ([]byte, error) {
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		return nil, err
	}
	switch mediaType {
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		return encodeStickyForm(values)
	case "multipart/form-data":
		boundary := parameters["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart boundary is required")
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		values := url.Values{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			value, err := io.ReadAll(io.LimitReader(part, stickyRequestLimit+1))
			_ = part.Close()
			if err != nil || len(value) > stickyRequestLimit {
				return nil, fmt.Errorf("read multipart field")
			}
			if name := part.FormName(); name != "" && part.FileName() == "" {
				values.Add(name, string(value))
			}
		}
		return encodeStickyForm(values)
	default:
		return body, nil
	}
}

func encodeStickyForm(values url.Values) ([]byte, error) {
	result := make(map[string]any, len(values))
	for name, items := range values {
		if len(items) == 0 {
			continue
		}
		value := items[len(items)-1]
		switch name {
		case "description", "logo_props":
			if json.Valid([]byte(value)) {
				result[name] = json.RawMessage(value)
			} else {
				result[name] = value
			}
		case "sort_order":
			if decoded, err := strconv.ParseFloat(value, 64); err == nil {
				result[name] = decoded
			} else {
				result[name] = value
			}
		default:
			result[name] = value
		}
	}
	return json.Marshal(result)
}

func setStickyHeaders(headers http.Header) {
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func mergeVary(headers http.Header, required ...string) {
	seen := make(map[string]bool)
	values := make([]string, 0, len(required)+len(headers.Values("Vary")))
	for _, line := range append(headers.Values("Vary"), strings.Join(required, ", ")) {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	headers.Set("Vary", strings.Join(values, ", "))
}

func removeVary(headers http.Header, removed ...string) {
	blocked := make(map[string]bool, len(removed))
	for _, value := range removed {
		blocked[strings.ToLower(value)] = true
	}
	values := make([]string, 0, len(headers.Values("Vary")))
	seen := make(map[string]bool)
	for _, line := range headers.Values("Vary") {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || blocked[key] || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		headers.Del("Vary")
		return
	}
	headers.Set("Vary", strings.Join(values, ", "))
}

func validateStickyPagination(request *http.Request) string {
	if raw := request.URL.Query().Get("per_page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return "Invalid per_page parameter."
		}
		if value > 1000 {
			return "Invalid per_page value. Cannot exceed 1000."
		}
	}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		parts := strings.Split(raw, ":")
		if len(parts) != 3 {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.ParseFloat(parts[0], 64); err != nil {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return "Invalid cursor parameter."
		}
	}
	return ""
}

func validateStickyRequest(body []byte) map[string][]string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil
	}
	if raw, ok := fields["sort_order"]; ok && string(raw) != "null" {
		var value float64
		if json.Unmarshal(raw, &value) != nil {
			return map[string][]string{"sort_order": {"A valid number is required."}}
		}
	}
	return nil
}

func writeStickyJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func listenAndServeHTTP(
	ctx context.Context,
	address string,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	return serveHTTP(ctx, listener, handler, shutdownTimeout)
}

func serveHTTP(
	ctx context.Context,
	listener net.Listener,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	server := &http.Server{
		Handler:             handler,
		ReadHeaderTimeout:   5 * time.Second,
		MaxHeaderValueCount: http.DefaultMaxHeaderValueCount,
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Print(ctx,
			log.KV{K: "msg", V: "HTTP server listening"},
			log.KV{K: "address", V: listener.Addr().String()},
		)
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		log.Print(ctx, log.KV{K: "msg", V: "HTTP server shutting down"})
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP during shutdown: %w", err)
		}
		log.Print(ctx, log.KV{K: "msg", V: "HTTP server stopped"})
		return nil
	}
}

func errorHandler(ctx context.Context, _ http.ResponseWriter, err error) {
	log.Error(ctx, err, log.KV{K: "msg", V: "encode HTTP response"})
}
