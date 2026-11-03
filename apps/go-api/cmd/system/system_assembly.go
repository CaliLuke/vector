package main

import (
	"context"
	"net/http"

	milestonestore "github.com/CaliLuke/go-plane/internal/milestones"

	workitemtypesstore "github.com/CaliLuke/go-plane/internal/workitemtypes"

	activityreadsapi "github.com/CaliLuke/go-plane/activityreads/gen/activity_reads"
	analyticviewsapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	entityfavoritesapi "github.com/CaliLuke/go-plane/gen/entity_favorites"
	instanceadminsessionapi "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	lastvisitedworkspaceapi "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	system "github.com/CaliLuke/go-plane/gen/system"
	useraccountsapi "github.com/CaliLuke/go-plane/gen/user_accounts"
	userapitokensapi "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	userinstanceadminapi "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	usernotificationpreferencesapi "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	userprofileflagsapi "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	workspacefavoritesapi "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	workspacehomepreferencesapi "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	workspacemembermeapi "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	workspacequicklinksapi "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisitsapi "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemesapi "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	activityreadsstore "github.com/CaliLuke/go-plane/internal/activityreads"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/authworkflow"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	projectconfigurationstore "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	projectlabelreadsstore "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	projectmembershipreadsstore "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	projectstatereadsstore "github.com/CaliLuke/go-plane/internal/projectstatereads"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/ratelimit"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	useridentitystore "github.com/CaliLuke/go-plane/internal/useridentity"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	workspaceprojectreadsstore "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	workspacerecentvisitsstore "github.com/CaliLuke/go-plane/internal/workspacerecentvisits"
	mcpoauthlinksapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	"github.com/jackc/pgx/v5/pgxpool"
)

// systemAssembly owns startup wiring and resource lifetimes. Domain services stay
// local to their constructor; only dependencies shared across domains remain here.
// Configuration runs in order and stops at the first failure. Every acquired
// resource is closed in reverse order, including on partial startup failure.
type systemAssembly struct {
	pool                                 *pgxpool.Pool
	issueDescriptionVersionAdmission     workitemstore.IssueDescriptionVersionAdmitter
	recentVisitAdmission                 workitemstore.RecentVisitAdmitter
	issueActivityAdmission               *backgroundtasks.IssueActivityAdmission
	emailTaskAdmission                   *backgroundtasks.TemplatedEmailTaskAdmission
	webhookTaskAdmission                 *backgroundtasks.WebhookTaskAdmission
	issueExportAdmission                 backgroundtasks.IssueExportTaskAdmitter
	analyticsExportDispatcher            *backgroundtasks.AnalyticsExportDispatcher
	eventTrackingAdmission               backgroundtasks.EventTrackingTaskAdmitter
	assetMetadataDispatcher              *backgroundtasks.FileAssetMetadataDispatcher
	workspaceSeedDispatcher              *backgroundtasks.WorkspaceSeedDispatcher
	linkPreviewDispatcher                *backgroundtasks.LinkPreviewDispatcher
	deletionDispatcher                   *backgroundtasks.DeletionDispatcher
	pageTaskAdmission                    pagereadsstore.NativePageTaskAdmitter
	backgroundTaskRuntime                *backgroundtasks.Runtime
	activityReadsPool                    *pgxpool.Pool
	activityTaskPublisher                workitemstore.Publisher
	systemEndpoints                      *system.Endpoints
	repository                           *stickystore.Repository
	sessionValidator                     *browsersession.Validator
	stickyService                        sticky.Service
	stickyEndpoints                      *sticky.Endpoints
	workspaceSlugEndpoints               *workspaceslugcheck.Endpoints
	instanceWorkspaceSlugEndpoints       *instanceworkspaceapi.Endpoints
	userInstanceAdminEndpoints           *userinstanceadminapi.Endpoints
	userProfileFlagsEndpoints            *userprofileflagsapi.Endpoints
	userNotificationPreferencesEndpoints *usernotificationpreferencesapi.Endpoints
	userAccountsEndpoints                *useraccountsapi.Endpoints
	userAPITokensEndpoints               *userapitokensapi.Endpoints
	instanceAdminSessionEndpoints        *instanceadminsessionapi.Endpoints
	workspaceQuickLinksEndpoints         *workspacequicklinksapi.Endpoints
	workspaceThemesEndpoints             *workspacethemesapi.Endpoints
	workspaceHomePreferencesEndpoints    *workspacehomepreferencesapi.Endpoints
	workspaceMemberMeEndpoints           *workspacemembermeapi.Endpoints
	analyticViewsEndpoints               *analyticviewsapi.Endpoints
	workspaceFavoritesEndpoints          *workspacefavoritesapi.Endpoints
	entityFavoritesEndpoints             *entityfavoritesapi.Endpoints
	workspaceRecentVisitsRepository      *workspacerecentvisitsstore.Repository
	workspaceRecentVisitsEndpoints       *workspacerecentvisitsapi.Endpoints
	workspaceProjectReadsRepository      *workspaceprojectreadsstore.Repository
	moduleReadsRepository                *modulereadsstore.Repository
	lastVisitedWorkspaceEndpoints        *lastvisitedworkspaceapi.Endpoints
	userIdentityRepository               *useridentitystore.Repository
	projectStateReadsRepository          *projectstatereadsstore.WriteRepository
	projectLabelReadsRepository          *projectlabelreadsstore.Repository
	publicProjectMetadataReadsRepository *publicprojectmetadatareadsstore.Repository
	publicProjectMetadataReadsCache      *ratelimit.RedisStore
	apiKeyThrottle                       *apikey.Throttle
	apiKeyValidator                      *apikey.Validator
	activityReadsPrimaryAPIKeyValidator  *apikey.Validator
	milestoneRepository                  *milestonestore.Repository
	mcpOptions                           mcptransport.Options
	mcpLinks                             *mcpoauth.Links
	mcpLinkEndpoints                     *mcpoauthlinksapi.Endpoints
	assetRepository                      *assetsstore.PostgresRepository
	assetStorage                         *assetsstore.Storage
	assetAppBaseURL                      string
	activityReadsPrimaryAPIRepository    *activityreadsstore.Repository
	activityReadsService                 activityreadsapi.Service
	configurationRepository              *projectconfigurationstore.Repository
	notificationTaskPublisher            workitemstore.Publisher
	labelWriteCache                      *projectlabelreadsstore.WriteCache
	membershipBaseHost                   string
	cycleLifecycleRepository             *cyclereadsstore.LifecycleRepository
	projectLifecycleRepository           *workspaceprojectreadsstore.LifecycleRepository
	moduleLifecycleRepository            *modulereadsstore.LifecycleRepository
	membershipRepository                 *projectmembershipreadsstore.LifecycleRepository
	invitationsRepository                *projectmembershipreadsstore.InvitationsRepository
	invitationsCache                     *projectmembershipreadsstore.InvitationCache
	workItemRepository                   *workitemstore.Repository
	typeRepository                       *workitemtypesstore.Repository
	instanceCipher                       *instancecipher.Cipher
	useDatabaseConfiguration             bool
	accountRepository                    *accountauthentication.Repository
	postAuthWorkflow                     *authworkflow.PostUserAuthWorkflow
	passwordHasher                       *password.Hasher
	authRedirectService                  *authredirects.Service
	authThrottle                         *authredirects.RedisAuthenticationThrottle
	magicIssuer                          *redisMagicCodeIssuer
	resetTokenIssuer                     *passwordResetTokenIssuer
	ctx                                  context.Context
	cfg                                  config
	logCtx                               context.Context
	wrapHTTP                             func(http.Handler) http.Handler
	mcpOAuthProvider                     *mcpoauth.Provider
	routes                               pendingReplacementEndpoints
	cleanup                              []func(*error)
}

func (assembly *systemAssembly) onClose(close func(*error)) {
	assembly.cleanup = append(assembly.cleanup, close)
}

func (assembly *systemAssembly) close(result *error) {
	for i := len(assembly.cleanup) - 1; i >= 0; i-- {
		assembly.cleanup[i](result)
	}
}

func (assembly *systemAssembly) configure() error {
	for _, configure := range []func() error{
		assembly.database,
		assembly.backgroundRuntime,
		assembly.readReplica,
		assembly.activityPublisher,
		assembly.browserSessions,
		assembly.accountReads,
		assembly.workspacePreferences,
		assembly.favorites,
		assembly.planningReads,
		assembly.pageReads,
		assembly.identityReads,
		assembly.projectReads,
		assembly.publicMetadata,
		assembly.apiAuthentication,
		assembly.mcpAuthentication,
		assembly.assets,
		assembly.activityReads,
		assembly.apiPlanningReads,
		assembly.projectConfiguration,
		assembly.notifications,
		assembly.notificationPublisher,
		assembly.historyAndNotificationActions,
		assembly.projectLifecycle,
		assembly.membership,
		assembly.workItems,
		assembly.browserMetadata,
		assembly.accountAuthentication,
		assembly.accountEndpoints,
		assembly.accountManagement,
		assembly.workspaceReports,
		assembly.exports,
		assembly.externalIntegrations,
	} {
		if err := configure(); err != nil {
			return err
		}
	}
	return nil
}
