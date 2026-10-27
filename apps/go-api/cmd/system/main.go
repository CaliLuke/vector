package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	workitemtypesstore "github.com/CaliLuke/go-plane/internal/workitemtypes"
	workitemtypesapi "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"

	systemapi "github.com/CaliLuke/go-plane"
	accountauthenticationapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	activityreadsapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	apicyclereadsapi "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	apilabelreadsapi "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	apimodulereadsapi "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	apistatereadsapi "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	apiuseridentityapi "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	apiv1stickiesapi "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	assetslifecycleapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	authredirectsapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	cyclelifecycleapi "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	analyticviewsapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	csrftokenapi "github.com/CaliLuke/go-plane/gen/csrf_token"
	cyclereadsapi "github.com/CaliLuke/go-plane/gen/cycle_reads"
	deployboardreadsapi "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	entityfavoritesapi "github.com/CaliLuke/go-plane/gen/entity_favorites"
	estimatereadsapi "github.com/CaliLuke/go-plane/gen/estimate_reads"
	instanceadminsessionapi "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instancemanagementapi "github.com/CaliLuke/go-plane/gen/instance_management"
	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	intakeinboxreadsapi "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	lastvisitedworkspaceapi "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	modulereadsapi "github.com/CaliLuke/go-plane/gen/module_reads"
	pagereadsapi "github.com/CaliLuke/go-plane/gen/page_reads"
	projectidentifiersapi "github.com/CaliLuke/go-plane/gen/project_identifiers"
	projectlabelreadsapi "github.com/CaliLuke/go-plane/gen/project_label_reads"
	projectmembershipreadsapi "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	projectstatereadsapi "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicfeedbackapi "github.com/CaliLuke/go-plane/gen/public_feedback"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	standardviewsapi "github.com/CaliLuke/go-plane/gen/standard_views"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	system "github.com/CaliLuke/go-plane/gen/system"
	useraccountsapi "github.com/CaliLuke/go-plane/gen/user_accounts"
	userapitokensapi "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	useridentityapi "github.com/CaliLuke/go-plane/gen/user_identity"
	userinstanceadminapi "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	usernotificationpreferencesapi "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	userprofileflagsapi "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	userpropertiesapi "github.com/CaliLuke/go-plane/gen/user_properties"
	usersettingsapi "github.com/CaliLuke/go-plane/gen/user_settings"
	workspacefavoritesapi "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	workspacehomepreferencesapi "github.com/CaliLuke/go-plane/gen/workspace_home_preferences"
	workspacemembermeapi "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	workspacemetadataapi "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	workspaceprojectreadsapi "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	workspacequicklinksapi "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisitsapi "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspacesidebarpreferencesapi "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemesapi "github.com/CaliLuke/go-plane/gen/workspace_themes"
	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	activityreadsstore "github.com/CaliLuke/go-plane/internal/activityreads"
	analyticviewsstore "github.com/CaliLuke/go-plane/internal/analyticviews"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/authworkflow"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authentication/passwordstrength"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/celery"
	customviewcreationstore "github.com/CaliLuke/go-plane/internal/customviewcreation"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/database"
	deployboardreadsstore "github.com/CaliLuke/go-plane/internal/deployboardreads"
	entityfavoritesstore "github.com/CaliLuke/go-plane/internal/entityfavorites"
	estimatereadsstore "github.com/CaliLuke/go-plane/internal/estimatereads"
	externalstore "github.com/CaliLuke/go-plane/internal/externalintegrations"
	instanceadminsessionstore "github.com/CaliLuke/go-plane/internal/instanceadminsession"
	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	instancemanagementstore "github.com/CaliLuke/go-plane/internal/instancemanagement"
	instanceworkspaceslugstore "github.com/CaliLuke/go-plane/internal/instanceworkspaceslugcheck"
	intakeinboxreadsstore "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	lastvisitedworkspacestore "github.com/CaliLuke/go-plane/internal/lastvisitedworkspace"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	notificationsstore "github.com/CaliLuke/go-plane/internal/notifications"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	personalstore "github.com/CaliLuke/go-plane/internal/personaldashboard"
	"github.com/CaliLuke/go-plane/internal/platform"
	projectconfigurationstore "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	projectidentifiersstore "github.com/CaliLuke/go-plane/internal/projectidentifiers"
	intakewritesstore "github.com/CaliLuke/go-plane/internal/projectintakewrites"
	projectlabelreadsstore "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	preferencesstore "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	projectmembershipreadsstore "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	projectpublicationstore "github.com/CaliLuke/go-plane/internal/projectpublication"
	projectstatereadsstore "github.com/CaliLuke/go-plane/internal/projectstatereads"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	standardviewsstore "github.com/CaliLuke/go-plane/internal/standardviews"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	useraccountsstore "github.com/CaliLuke/go-plane/internal/useraccounts"
	useraccountwritesstore "github.com/CaliLuke/go-plane/internal/useraccountwrites"
	userapitokensstore "github.com/CaliLuke/go-plane/internal/userapitokens"
	useridentitystore "github.com/CaliLuke/go-plane/internal/useridentity"
	userinstanceadminstore "github.com/CaliLuke/go-plane/internal/userinstanceadmin"
	usernotificationpreferencesstore "github.com/CaliLuke/go-plane/internal/usernotificationpreferences"
	userprofileflagsstore "github.com/CaliLuke/go-plane/internal/userprofileflags"
	userpropertiesstore "github.com/CaliLuke/go-plane/internal/userproperties"
	usersettingsstore "github.com/CaliLuke/go-plane/internal/usersettings"
	workitemhistorystore "github.com/CaliLuke/go-plane/internal/workitemhistory"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	analyticsstore "github.com/CaliLuke/go-plane/internal/workspaceanalytics"
	exportsstore "github.com/CaliLuke/go-plane/internal/workspaceexports"
	workspacefavoritesstore "github.com/CaliLuke/go-plane/internal/workspacefavorites"
	workspacehomepreferencesstore "github.com/CaliLuke/go-plane/internal/workspacehomepreferences"
	workspacemembermestore "github.com/CaliLuke/go-plane/internal/workspacememberme"
	workspacemetadatastore "github.com/CaliLuke/go-plane/internal/workspacemetadata"
	workspaceprojectreadsstore "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	workspacequicklinksstore "github.com/CaliLuke/go-plane/internal/workspacequicklinks"
	workspacerecentvisitsstore "github.com/CaliLuke/go-plane/internal/workspacerecentvisits"
	searchstore "github.com/CaliLuke/go-plane/internal/workspacesearch"
	workspacesidebarpreferencesstore "github.com/CaliLuke/go-plane/internal/workspacesidebarpreferences"
	workspaceslugstore "github.com/CaliLuke/go-plane/internal/workspaceslugcheck"
	workspacethemesstore "github.com/CaliLuke/go-plane/internal/workspacethemes"
	insightsstore "github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	workspacewebhooksstore "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	invitationsapi "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	mcpoauthlinksapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	modulelifecycleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	notificationactionsapi "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	notificationsapi "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	oauthredirectsapi "github.com/CaliLuke/go-plane/oauthredirects/gen/oauth_redirects"
	labelwritesapi "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	projectconfigurationapi "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	intakewritesapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	projectlifecycleapi "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	membershiplifecycleapi "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	projectpublicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	collaborationapi "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	useraccountwritesapi "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	workspacewebhooksapi "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	analyticsapi "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	exportsapi "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	workspacelifecycleapi "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	personalapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	insightsapi "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/selvedge"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	defaultHTTPAddress     = "0.0.0.0:8080"
	defaultShutdownTimeout = 30 * time.Second
	defaultFileSizeLimit   = int64(5 * 1024 * 1024)
)

type config struct {
	mcp                        mcptransport.Config
	mcpOAuth                   mcpoauth.Config
	goBackgroundTasks          bool
	goBackgroundTasksNoWorkers bool
	httpAddress                string
	shutdownTimeout            time.Duration
	debug                      bool
	databaseURL                string
	readReplicaDatabaseURL     string
	redisURL                   string
	apiKeyRate                 string
	amqpURL                    string
	corsOrigins                []string
	csrfTrustedOrigins         []string
	sessionCookie              string
	adminSessionCookie         string
	cookieDomain               string
	sessionCookieAge           int64
	adminSessionCookieAge      int64
	sessionCookieSecure        bool
	saveEveryRequest           bool
	secretKey                  string
	fallbackKeys               []string
	fileSizeLimit              int64
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) (runErr error) {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if err = validateMCPAvailability(cfg.mcp, cfg.mcpOAuth); err != nil {
		return err
	}

	var mcpOAuthProvider *mcpoauth.Provider
	if cfg.mcp.Enabled && (cfg.mcp.Mode == "oauth" || cfg.mcp.Mode == "both") {
		mcpOAuthProvider, err = mcpoauth.NewProvider(ctx, cfg.mcpOAuth, nil)
		if err != nil {
			return err
		}
	}

	wrapHTTP, otelLogOutput, shutdownTelemetry, err := telemetry.ConfigureHTTPObservability(ctx, "plane-go-api")
	if err != nil {
		return fmt.Errorf("configure HTTP observability: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		runErr = errors.Join(runErr, shutdownTelemetry(shutdownCtx))
	}()
	logCtx := newLogContext(ctx, output, cfg.debug, otelLogOutput)
	defer func() {
		if runErr != nil {
			log.Error(logCtx, runErr, log.KV{K: log.MessageKey, V: "Go API exited with an error"})
		}
	}()
	if cfg.databaseURL == "" {
		return fmt.Errorf("database-url is required")
	}
	if !cfg.goBackgroundTasks {
		return fmt.Errorf("native background tasks are required for transactional work item effects; start with --go-background-tasks")
	}
	poolConfig, err := pgxpool.ParseConfig(cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("configure database: %w", err)
	}
	if err := database.ConfigurePool(poolConfig); err != nil {
		return fmt.Errorf("configure database adapters: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	var issueDescriptionVersionAdmission workitemstore.IssueDescriptionVersionAdmitter
	var recentVisitAdmission workitemstore.RecentVisitAdmitter
	var issueActivityAdmission *backgroundtasks.IssueActivityAdmission
	var emailTaskAdmission *backgroundtasks.TemplatedEmailTaskAdmission
	var webhookTaskAdmission *backgroundtasks.WebhookTaskAdmission
	var issueExportAdmission backgroundtasks.IssueExportTaskAdmitter
	var analyticsExportDispatcher *backgroundtasks.AnalyticsExportDispatcher
	var eventTrackingAdmission backgroundtasks.EventTrackingTaskAdmitter
	var assetMetadataDispatcher *backgroundtasks.FileAssetMetadataDispatcher
	var workspaceSeedDispatcher *backgroundtasks.WorkspaceSeedDispatcher
	var linkPreviewDispatcher *backgroundtasks.LinkPreviewDispatcher
	var deletionDispatcher *backgroundtasks.DeletionDispatcher
	var pageTaskAdmission pagereadsstore.NativePageTaskAdmitter
	var publicProjectMetadataActivityLogger publicprojectmetadatareadsstore.ActivityLogger
	var backgroundTaskRuntime *backgroundtasks.Runtime
	if cfg.goBackgroundTasks {
		var runtimeOptions []selvedge.Option
		if cfg.goBackgroundTasksNoWorkers {
			runtimeOptions = append(runtimeOptions, selvedge.WithActivityQueues())
		}
		backgroundTaskRuntime, err = backgroundtasks.NewRuntime(ctx, cfg.databaseURL, runtimeOptions...)
		if err != nil {
			return err
		}
		defer func() {
			shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
			defer cancel()
			if err := backgroundTaskRuntime.Shutdown(shutdownContext); err != nil {
				log.Error(ctx, err)
			}
		}()
		if err := backgroundTaskRuntime.Start(ctx); err != nil {
			return err
		}
		pageTaskAdmission, err = backgroundtasks.NewPageTaskAdmission(backgroundTaskRuntime.PageTransactions, backgroundTaskRuntime.Versions, backgroundTaskRuntime.DescriptionAssets)
		if err != nil {
			return err
		}
		issueDescriptionVersionAdmission, err = backgroundtasks.NewIssueDescriptionVersionAdmission(backgroundTaskRuntime.Versions)
		if err != nil {
			return err
		}
		recentVisitAdmission, err = backgroundtasks.NewRecentVisitAdmission(backgroundTaskRuntime.RecentVisits)
		if err != nil {
			return err
		}
		issueActivityAdmission, err = backgroundtasks.NewIssueActivityAdmission(backgroundTaskRuntime.IssueActivities)
		if err != nil {
			return err
		}
		emailTaskAdmission = backgroundTaskRuntime.EmailTaskAdmission
		webhookTaskAdmission, err = backgroundtasks.NewWebhookTaskAdmission(backgroundTaskRuntime.Webhooks)
		if err != nil {
			return err
		}
		issueExportAdmission = backgroundTaskRuntime.IssueExports
		analyticsExportDispatcher = backgroundTaskRuntime.AnalyticsExports
		eventTrackingAdmission = backgroundTaskRuntime.EventTracking
		assetMetadataDispatcher = backgroundTaskRuntime.Assets
		workspaceSeedDispatcher = backgroundTaskRuntime.WorkspaceSeed
		linkPreviewDispatcher = backgroundTaskRuntime.LinkPreviews
		deletionDispatcher = backgroundTaskRuntime.Deletion
		publicProjectMetadataActivityLogger = backgroundTaskRuntime.Logger
	}
	activityReadsPool := pool
	if cfg.readReplicaDatabaseURL != "" {
		readReplicaConfig, err := pgxpool.ParseConfig(cfg.readReplicaDatabaseURL)
		if err != nil {
			return fmt.Errorf("configure read replica database: %w", err)
		}
		if err := database.ConfigurePool(readReplicaConfig); err != nil {
			return fmt.Errorf("configure read replica database adapters: %w", err)
		}
		activityReadsPool, err = pgxpool.NewWithConfig(ctx, readReplicaConfig)
		if err != nil {
			return fmt.Errorf("create read replica database pool: %w", err)
		}
		defer activityReadsPool.Close()
		if err := activityReadsPool.Ping(ctx); err != nil {
			return fmt.Errorf("connect read replica database: %w", err)
		}
	}
	var taskFallback backgroundtasks.TaskFallback
	var closeActivityPublisher func()
	if cfg.goBackgroundTasks {
		taskFallback = rejectingTaskPublisher{}
	} else {
		activityPublisher, err := celery.NewPublisherPool(cfg.amqpURL)
		if err != nil {
			return err
		}
		closeActivityPublisher = func() { _ = activityPublisher.Close() }
		taskFallback = activityPublisher
		publicProjectMetadataActivityLogger = activityPublisher
	}
	if closeActivityPublisher != nil {
		defer closeActivityPublisher()
	}
	activityTaskPublisher, err := issueActivityTaskPublisher(taskFallback, issueActivityAdmission)
	if err != nil {
		return err
	}
	if emailTaskAdmission != nil {
		activityTaskPublisher, err = backgroundtasks.NewTemplatedEmailTaskPublisher(activityTaskPublisher, emailTaskAdmission)
		if err != nil {
			return err
		}
	}
	if webhookTaskAdmission != nil {
		activityTaskPublisher, err = backgroundtasks.NewWebhookTaskPublisher(activityTaskPublisher, webhookTaskAdmission)
		if err != nil {
			return err
		}
	}
	if deletionDispatcher != nil {
		activityTaskPublisher, err = backgroundtasks.NewDeletionTaskPublisher(activityTaskPublisher, deletionDispatcher)
		if err != nil {
			return err
		}
	}
	if eventTrackingAdmission != nil {
		activityTaskPublisher, err = backgroundtasks.NewEventTrackingTaskPublisher(activityTaskPublisher, eventTrackingAdmission)
		if err != nil {
			return err
		}
	}
	if assetMetadataDispatcher != nil {
		activityTaskPublisher, err = backgroundtasks.NewFileAssetMetadataTaskPublisher(activityTaskPublisher, assetMetadataDispatcher)
		if err != nil {
			return err
		}
	}
	if workspaceSeedDispatcher != nil {
		activityTaskPublisher, err = backgroundtasks.NewWorkspaceSeedTaskPublisher(activityTaskPublisher, workspaceSeedDispatcher)
		if err != nil {
			return err
		}
	}
	if linkPreviewDispatcher != nil {
		activityTaskPublisher, err = backgroundtasks.NewLinkPreviewTaskPublisher(activityTaskPublisher, linkPreviewDispatcher)
		if err != nil {
			return err
		}
	}
	if backgroundTaskRuntime != nil {
		activityTaskPublisher, err = backgroundtasks.NewVersionTaskPublisher(
			activityTaskPublisher,
			backgroundTaskRuntime.IssueVersions,
			backgroundTaskRuntime.IssueVersionSync,
			backgroundTaskRuntime.IssueDescriptionSync,
			backgroundTaskRuntime.Logger,
		)
		if err != nil {
			return err
		}
	}
	systemEndpoints := system.NewEndpoints(systemapi.NewSystemWithDatabase(pool))
	systemEndpoints.Use(log.Endpoint)
	csrfTokenEndpoints := csrftokenapi.NewEndpoints(systemapi.NewCSRFToken())
	csrfTokenEndpoints.Use(log.Endpoint)
	repository, err := stickystore.NewRepository(pool)
	if err != nil {
		return err
	}
	sessionValidator, err := djangosession.New(djangosession.NewPostgreSQLReader(pool), djangosession.Config{
		CookieName: cfg.sessionCookie, CookieNameConfigured: true, SecretKey: cfg.secretKey, FallbackKeys: cfg.fallbackKeys,
		AllowedBackends: []string{"django.contrib.auth.backends.ModelBackend"},
	})
	if err != nil {
		return fmt.Errorf("configure Django session validation: %w", err)
	}
	stickyService, err := systemapi.NewStickies(repository, sessionValidator)
	if err != nil {
		return err
	}
	stickyEndpoints := sticky.NewEndpoints(stickyService)
	stickyEndpoints.Use(log.Endpoint)
	workspaceSlugRepository, err := workspaceslugstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceSlugService, err := systemapi.NewWorkspaceSlugCheck(workspaceSlugRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceSlugEndpoints := workspaceslugcheck.NewEndpoints(workspaceSlugService)
	workspaceSlugEndpoints.Use(log.Endpoint)
	instanceWorkspaceSlugRepository, err := instanceworkspaceslugstore.NewRepository(pool)
	if err != nil {
		return err
	}
	instanceWorkspaceSlugService, err := systemapi.NewInstanceWorkspaceSlugCheck(instanceWorkspaceSlugRepository, sessionValidator)
	if err != nil {
		return err
	}
	instanceWorkspaceSlugEndpoints := instanceworkspaceapi.NewEndpoints(instanceWorkspaceSlugService)
	instanceWorkspaceSlugEndpoints.Use(log.Endpoint)
	userInstanceAdminRepository, err := userinstanceadminstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userInstanceAdminService, err := systemapi.NewUserInstanceAdmin(userInstanceAdminRepository, sessionValidator)
	if err != nil {
		return err
	}
	userInstanceAdminEndpoints := userinstanceadminapi.NewEndpoints(userInstanceAdminService)
	userInstanceAdminEndpoints.Use(log.Endpoint)
	userProfileFlagsRepository, err := userprofileflagsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userProfileFlagsService, err := systemapi.NewUserProfileFlags(userProfileFlagsRepository, sessionValidator)
	if err != nil {
		return err
	}
	userProfileFlagsEndpoints := userprofileflagsapi.NewEndpoints(userProfileFlagsService)
	userProfileFlagsEndpoints.Use(log.Endpoint)
	userNotificationPreferencesRepository, err := usernotificationpreferencesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userNotificationPreferencesService, err := systemapi.NewUserNotificationPreferences(userNotificationPreferencesRepository, sessionValidator)
	if err != nil {
		return err
	}
	userNotificationPreferencesEndpoints := usernotificationpreferencesapi.NewEndpoints(userNotificationPreferencesService)
	userNotificationPreferencesEndpoints.Use(log.Endpoint)
	userAccountsRepository, err := useraccountsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userAccountsService, err := systemapi.NewUserAccounts(userAccountsRepository, sessionValidator)
	if err != nil {
		return err
	}
	userAccountsEndpoints := useraccountsapi.NewEndpoints(userAccountsService)
	userAccountsEndpoints.Use(log.Endpoint)
	userAPITokensRepository, err := userapitokensstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userAPITokensService, err := systemapi.NewUserAPITokens(userAPITokensRepository, sessionValidator)
	if err != nil {
		return err
	}
	userAPITokensEndpoints := userapitokensapi.NewEndpoints(userAPITokensService)
	userAPITokensEndpoints.Use(log.Endpoint)
	instanceAdminSessionRepository, err := instanceadminsessionstore.NewRepository(pool)
	if err != nil {
		return err
	}
	instanceAdminSessionService, err := systemapi.NewInstanceAdminSession(instanceAdminSessionRepository, sessionValidator)
	if err != nil {
		return err
	}
	instanceAdminSessionEndpoints := instanceadminsessionapi.NewEndpoints(instanceAdminSessionService)
	instanceAdminSessionEndpoints.Use(log.Endpoint)
	workspaceQuickLinksRepository, err := workspacequicklinksstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceQuickLinksService, err := systemapi.NewWorkspaceQuickLinks(workspaceQuickLinksRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceQuickLinksEndpoints := workspacequicklinksapi.NewEndpoints(workspaceQuickLinksService)
	workspaceQuickLinksEndpoints.Use(log.Endpoint)
	workspaceThemesRepository, err := workspacethemesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceThemesService, err := systemapi.NewWorkspaceThemes(workspaceThemesRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceThemesEndpoints := workspacethemesapi.NewEndpoints(workspaceThemesService)
	workspaceThemesEndpoints.Use(log.Endpoint)
	workspaceHomePreferencesRepository, err := workspacehomepreferencesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceHomePreferencesService, err := systemapi.NewWorkspaceHomePreferences(workspaceHomePreferencesRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceHomePreferencesEndpoints := workspacehomepreferencesapi.NewEndpoints(workspaceHomePreferencesService)
	workspaceHomePreferencesEndpoints.Use(log.Endpoint)
	workspaceMemberMeRepository, err := workspacemembermestore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceMemberMeService, err := systemapi.NewWorkspaceMemberMe(workspaceMemberMeRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceMemberMeEndpoints := workspacemembermeapi.NewEndpoints(workspaceMemberMeService)
	workspaceMemberMeEndpoints.Use(log.Endpoint)
	analyticViewsRepository, err := analyticviewsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	analyticViewsService, err := systemapi.NewAnalyticViews(analyticViewsRepository, sessionValidator)
	if err != nil {
		return err
	}
	analyticViewsEndpoints := analyticviewsapi.NewEndpoints(analyticViewsService)
	analyticViewsEndpoints.Use(log.Endpoint)
	workspaceFavoritesRepository, err := workspacefavoritesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceFavoritesService, err := systemapi.NewWorkspaceFavorites(workspaceFavoritesRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceFavoritesEndpoints := workspacefavoritesapi.NewEndpoints(workspaceFavoritesService)
	workspaceFavoritesEndpoints.Use(log.Endpoint)
	entityFavoritesRepository, err := entityfavoritesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	entityFavoritesService, err := systemapi.NewEntityFavorites(entityFavoritesRepository, sessionValidator)
	if err != nil {
		return err
	}
	entityFavoritesEndpoints := entityfavoritesapi.NewEndpoints(entityFavoritesService)
	entityFavoritesEndpoints.Use(log.Endpoint)
	workspaceRecentVisitsRepository, err := workspacerecentvisitsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceRecentVisitsService, err := systemapi.NewWorkspaceRecentVisits(workspaceRecentVisitsRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceRecentVisitsEndpoints := workspacerecentvisitsapi.NewEndpoints(workspaceRecentVisitsService)
	workspaceRecentVisitsEndpoints.Use(log.Endpoint)
	workspaceProjectReadsRepository, err := workspaceprojectreadsstore.NewRepositoryWithRecentVisits(pool, activityTaskPublisher, recentVisitAdmission)
	if err != nil {
		return err
	}
	workspaceProjectReadsService, err := systemapi.NewWorkspaceProjectReads(workspaceProjectReadsRepository, sessionValidator, workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	workspaceProjectReadsEndpoints := workspaceprojectreadsapi.NewEndpoints(workspaceProjectReadsService)
	workspaceProjectReadsEndpoints.Use(log.Endpoint)

	cycleReadsRepository, err := cyclereadsstore.NewRepositoryWithRecentVisits(pool, activityTaskPublisher, recentVisitAdmission)
	if err != nil {
		return err
	}
	cycleReadsService, err := systemapi.NewCycleReads(cycleReadsRepository, sessionValidator, workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	cycleReadsEndpoints := cyclereadsapi.NewEndpoints(cycleReadsService)
	cycleReadsEndpoints.Use(log.Endpoint)
	moduleReadsRepository, err := modulereadsstore.NewRepositoryWithRecentVisits(pool, activityTaskPublisher, recentVisitAdmission)
	if err != nil {
		return err
	}
	moduleReadsService, err := systemapi.NewModuleReads(moduleReadsRepository, sessionValidator, workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	moduleReadsEndpoints := modulereadsapi.NewEndpoints(moduleReadsService)
	moduleReadsEndpoints.Use(log.Endpoint)
	estimateReadsRepository, err := estimatereadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	estimateReadsService, err := systemapi.NewEstimateReads(estimateReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	estimateReadsEndpoints := estimatereadsapi.NewEndpoints(estimateReadsService)
	estimateReadsEndpoints.Use(log.Endpoint)
	pageReadsRepository, err := pagereadsstore.NewRepositoryWithNativeTasks(pool, activityTaskPublisher, pageTaskAdmission, recentVisitAdmission)
	if err != nil {
		return err
	}
	pageReadsService, err := systemapi.NewPageReads(pageReadsRepository, sessionValidator, workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	pageReadsEndpoints := pagereadsapi.NewEndpoints(pageReadsService)
	pageReadsEndpoints.Use(log.Endpoint)
	intakeInboxReadsRepository, err := intakeinboxreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	intakeInboxReadsService, err := systemapi.NewIntakeInboxReads(intakeInboxReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	intakeInboxReadsEndpoints := intakeinboxreadsapi.NewEndpoints(intakeInboxReadsService)
	intakeInboxReadsEndpoints.Use(log.Endpoint)
	deployBoardReadsRepository, err := deployboardreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	deployBoardReadsService, err := systemapi.NewDeployBoardReads(deployBoardReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	deployBoardReadsEndpoints := deployboardreadsapi.NewEndpoints(deployBoardReadsService)
	deployBoardReadsEndpoints.Use(log.Endpoint)
	lastVisitedWorkspaceRepository, err := lastvisitedworkspacestore.NewRepository(pool)
	if err != nil {
		return err
	}
	lastVisitedWorkspaceService, err := systemapi.NewLastVisitedWorkspace(lastVisitedWorkspaceRepository, sessionValidator)
	if err != nil {
		return err
	}
	lastVisitedWorkspaceEndpoints := lastvisitedworkspaceapi.NewEndpoints(lastVisitedWorkspaceService)
	lastVisitedWorkspaceEndpoints.Use(log.Endpoint)
	userIdentityRepository, err := useridentitystore.NewRepository(pool)
	if err != nil {
		return err
	}
	userIdentityService, err := systemapi.NewUserIdentity(userIdentityRepository, sessionValidator)
	if err != nil {
		return err
	}
	userIdentityEndpoints := useridentityapi.NewEndpoints(userIdentityService)
	userIdentityEndpoints.Use(log.Endpoint)
	userPropertiesRepository, err := userpropertiesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userPropertiesService, err := systemapi.NewUserProperties(userPropertiesRepository, sessionValidator)
	if err != nil {
		return err
	}
	userPropertiesEndpoints := userpropertiesapi.NewEndpoints(userPropertiesService)
	userPropertiesEndpoints.Use(log.Endpoint)
	workspaceMetadataRepository, err := workspacemetadatastore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceMetadataService, err := systemapi.NewWorkspaceMetadata(workspaceMetadataRepository, sessionValidator)
	if err != nil {
		return err
	}
	workspaceMetadataEndpoints := workspacemetadataapi.NewEndpoints(workspaceMetadataService)
	workspaceMetadataEndpoints.Use(log.Endpoint)
	projectIdentifiersRepository, err := projectidentifiersstore.NewRepository(pool)
	if err != nil {
		return err
	}
	projectIdentifiersService, err := systemapi.NewProjectIdentifiers(projectIdentifiersRepository, sessionValidator)
	if err != nil {
		return err
	}
	projectIdentifiersEndpoints := projectidentifiersapi.NewEndpoints(projectIdentifiersService)
	projectIdentifiersEndpoints.Use(log.Endpoint)
	projectMembershipReadsRepository, err := projectmembershipreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	projectMembershipReadsService, err := systemapi.NewProjectMembershipReads(projectMembershipReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	projectMembershipReadsEndpoints := projectmembershipreadsapi.NewEndpoints(projectMembershipReadsService)
	projectMembershipReadsEndpoints.Use(log.Endpoint)
	projectStateWriteCache, err := projectstatereadsstore.NewStateWriteCache(cfg.redisURL)
	if err != nil {
		return err
	}
	defer func() { _ = projectStateWriteCache.Close() }()
	projectStateReadsRepository, err := projectstatereadsstore.NewWriteRepository(pool, activityTaskPublisher)
	if err != nil {
		return err
	}
	projectStateReadsRepository.SetCache(projectStateWriteCache)
	projectStateReadsService, err := systemapi.NewProjectStateReads(projectStateReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	projectStateReadsEndpoints := projectstatereadsapi.NewEndpoints(projectStateReadsService)
	projectStateReadsEndpoints.Use(log.Endpoint)
	projectLabelReadsRepository, err := projectlabelreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	projectLabelReadsService, err := systemapi.NewProjectLabelReads(projectLabelReadsRepository, sessionValidator)
	if err != nil {
		return err
	}
	projectLabelReadsEndpoints := projectlabelreadsapi.NewEndpoints(projectLabelReadsService)
	projectLabelReadsEndpoints.Use(log.Endpoint)
	publicProjectMetadataReadsRepository, err := publicprojectmetadatareadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsService, err := systemapi.NewPublicProjectMetadataReads(publicProjectMetadataReadsRepository)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsEndpoints := publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsService)
	publicProjectMetadataReadsEndpoints.Use(log.Endpoint)
	publicProjectMetadataReadsCache, err := publicprojectmetadatareadsstore.NewRedisThrottleCache(cfg.redisURL)
	if err != nil {
		return err
	}
	defer publicProjectMetadataReadsCache.Close()
	apiKeyThrottle, err := apikey.NewThrottle(publicProjectMetadataReadsCache, cfg.apiKeyRate)
	if err != nil {
		return fmt.Errorf("configure API-key throttle: %w", err)
	}
	apiKeyValidator, err := apikey.NewWithReaderWriter(activityReadsPool, pool)
	if err != nil {
		return err
	}
	apiV1StickiesService, err := systemapi.NewAPIV1Stickies(repository, apiKeyValidator, apiKeyThrottle, activityTaskPublisher)
	if err != nil {
		return err
	}
	apiV1StickiesEndpoints := apiv1stickiesapi.NewEndpoints(apiV1StickiesService)
	apiV1StickiesEndpoints.Use(log.Endpoint)
	activityReadsPrimaryAPIKeyValidator, err := apikey.NewWithReaderWriter(pool, pool)
	if err != nil {
		return err
	}
	mcpOptions := mcptransport.Options{Observer: mcptransport.LogObserver(logCtx)}
	var mcpLinks *mcpoauth.Links
	var mcpLinkEndpoints *mcpoauthlinksapi.Endpoints
	if cfg.mcp.Enabled {
		memberships, membershipErr := mcpauth.NewWorkspaceReader(pool)
		if membershipErr != nil {
			return membershipErr
		}
		headerAuth, authErr := mcpauth.NewPersonalTokens(activityReadsPrimaryAPIKeyValidator, memberships, apiKeyThrottle)
		if authErr != nil {
			return authErr
		}
		mcpOptions.HeaderAuth = headerAuth.Authenticate
		if mcpOAuthProvider != nil {
			mcpLinks, err = mcpoauth.NewLinks(pool, cfg.mcpOAuth.Issuer)
			if err != nil {
				return err
			}
			oauthAuth, oauthErr := mcpauth.NewOAuth(mcpOAuthProvider, mcpLinks, memberships, apiKeyThrottle)
			if oauthErr != nil {
				return oauthErr
			}
			mcpOptions.OAuthAuth, mcpOptions.OAuthIssuer = oauthAuth.Authenticate, cfg.mcpOAuth.Issuer
		}
	}
	var assetsLifecycleEndpoints *assetslifecycleapi.Endpoints
	var assetsLifecycleMetadata interface {
		systemapi.SessionResourceProtocolAuthorizer
		systemapi.AssetsAPIProtocolAuthorizer
	}
	assetDuplicateThrottle, err := apikey.NewThrottle(publicProjectMetadataReadsCache, "5/minute")
	if err != nil {
		return fmt.Errorf("configure duplicate asset throttle: %w", err)
	}
	assetRepository, err := assetsstore.NewPostgresRepository(pool)
	if err != nil {
		return err
	}
	storageConfig, err := assetsstore.StorageConfigFromEnv()
	if err != nil {
		return err
	}
	assetStorage, err := assetsstore.NewStorage(storageConfig)
	if err != nil {
		return err
	}
	assetCache, err := assetsstore.NewRedisCacheInvalidator(cfg.redisURL)
	if err != nil {
		return err
	}
	defer func() { _ = assetCache.Close() }()
	assetAppBaseURL := os.Getenv("APP_BASE_URL")
	assetAppURL, assetAppURLErr := url.Parse(assetAppBaseURL)
	if assetAppURLErr != nil || assetAppURL.Scheme == "" || assetAppURL.Host == "" {
		assetAppBaseURL = os.Getenv("WEB_URL")
	}
	assetService, err := assetsstore.NewService(assetRepository, assetStorage, apiKeyValidator, apiKeyThrottle, cfg.fileSizeLimit, assetsstore.AssetEffects{
		Tasks:      activityTaskPublisher,
		Cache:      assetCache,
		AppBaseURL: assetAppBaseURL,
	})
	if err != nil {
		return err
	}
	lifecycleService, err := assetsstore.NewLifecycleService(assetService)
	if err != nil {
		return err
	}
	assetsLifecycleMetadata = lifecycleService
	assetsLifecycleEndpoints = assetslifecycleapi.NewEndpoints(lifecycleService)
	assetsLifecycleEndpoints.Use(log.Endpoint)
	activityReadsUserRepository, err := activityreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	activityReadsAPIRepository, err := activityreadsstore.NewRepository(activityReadsPool)
	if err != nil {
		return err
	}
	activityReadsPrimaryAPIRepository, err := activityreadsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	activityReadsService, err := systemapi.NewActivityReads(activityReadsUserRepository, activityReadsAPIRepository, apiKeyValidator, sessionValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	activityReadsPrimaryService, err := systemapi.NewActivityReads(activityReadsUserRepository, activityReadsPrimaryAPIRepository, activityReadsPrimaryAPIKeyValidator, sessionValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiUserIdentityService, err := systemapi.NewAPIUserIdentity(userIdentityRepository, activityReadsPrimaryAPIKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiUserIdentityEndpoints := apiuseridentityapi.NewEndpoints(apiUserIdentityService)
	apiUserIdentityEndpoints.Use(log.Endpoint)

	activityReadsMetadata := activityReadsMethodAuthorizer{
		identity: apiUserIdentityService.(systemapi.APIIdentityProtocolAuthorizer),
		readOnly: activityReadsService.(systemapi.ActivityReadsProtocolAuthorizer),
		primary:  activityReadsPrimaryService.(systemapi.ActivityReadsProtocolAuthorizer),
	}
	apiCycleRepository, err := cyclereadsstore.NewRepository(activityReadsPool)
	if err != nil {
		return err
	}
	apiCycleService, err := systemapi.NewAPICycleReads(apiCycleRepository, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiCycleEndpoints := apicyclereadsapi.NewEndpoints(apiCycleService)
	apiCycleEndpoints.Use(log.Endpoint)
	apiModuleRepository, err := modulereadsstore.NewRepository(activityReadsPool)
	if err != nil {
		return err
	}
	apiModuleService, err := systemapi.NewAPIModuleReads(apiModuleRepository, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiModuleEndpoints := apimodulereadsapi.NewEndpoints(apiModuleService)
	apiModuleEndpoints.Use(log.Endpoint)
	apiLabelRepository, err := projectlabelreadsstore.NewRepository(activityReadsPool)
	if err != nil {
		return err
	}
	apiLabelService, err := systemapi.NewAPILabelReads(apiLabelRepository, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiLabelEndpoints := apilabelreadsapi.NewEndpoints(apiLabelService)
	apiLabelEndpoints.Use(log.Endpoint)
	preferencesRepository, err := preferencesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	preferencesService, err := systemapi.NewProjectMemberPreferences(preferencesRepository, sessionValidator)
	if err != nil {
		return err
	}
	preferencesEndpoints := preferencesapi.NewEndpoints(preferencesService)
	preferencesEndpoints.Use(log.Endpoint)
	intakeWritesRepository, err := intakewritesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	intakeWritesService, err := systemapi.NewProjectIntakeWrites(intakeWritesRepository, sessionValidator)
	if err != nil {
		return err
	}
	intakeWritesEndpoints := intakewritesapi.NewEndpoints(intakeWritesService)
	intakeWritesEndpoints.Use(log.Endpoint)
	webhookRepository, err := workspacewebhooksstore.NewRepository(pool)
	if err != nil {
		return err
	}
	webhookService, err := systemapi.NewWorkspaceWebhooks(webhookRepository, sessionValidator, activityTaskPublisher)
	if err != nil {
		return err
	}
	webhookEndpoints := workspacewebhooksapi.NewEndpoints(webhookService)
	webhookEndpoints.Use(log.Endpoint)
	configurationRepository, err := projectconfigurationstore.NewRepository(pool)
	if err != nil {
		return err
	}
	configurationService, err := systemapi.NewProjectConfiguration(configurationRepository, sessionValidator, apiKeyValidator, apiKeyThrottle, activityTaskPublisher)
	if err != nil {
		return err
	}
	configurationEndpoints := projectconfigurationapi.NewEndpoints(configurationService)
	configurationEndpoints.Use(log.Endpoint)
	customViewCreationRepository, err := customviewcreationstore.NewRepository(pool)
	if err != nil {
		return err
	}
	customViewCreationService, err := systemapi.NewCustomViewCreation(customViewCreationRepository, sessionValidator)
	if err != nil {
		return err
	}
	customViewCreationEndpoints := customviewcreationapi.NewEndpoints(customViewCreationService)
	customViewCreationEndpoints.Use(log.Endpoint)
	apiStateRepository, err := projectstatereadsstore.NewWriteRepository(pool, activityTaskPublisher)
	if err != nil {
		return err
	}
	apiStateService, err := systemapi.NewAPIStateReads(apiStateRepository, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	apiStateEndpoints := apistatereadsapi.NewEndpoints(apiStateService)
	apiStateEndpoints.Use(log.Endpoint)
	notificationRepository, err := notificationsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	notificationCounts, err := notificationsstore.NewRepository(activityReadsPool)
	if err != nil {
		return err
	}
	notificationsService, err := systemapi.NewNotificationReads(notificationRepository, notificationCounts, sessionValidator)
	if err != nil {
		return err
	}
	notificationsEndpoints := notificationsapi.NewEndpoints(notificationsService)
	notificationsEndpoints.Use(log.Endpoint)
	var notificationTaskFallback backgroundtasks.TaskFallback = taskFallback
	var closeNotificationPublisher func()
	if !cfg.goBackgroundTasks {
		notificationPublisher, err := celery.NewPublisher(cfg.amqpURL)
		if err != nil {
			return err
		}
		closeNotificationPublisher = func() { _ = notificationPublisher.Close() }
		notificationTaskFallback = notificationPublisher
	}
	if closeNotificationPublisher != nil {
		defer closeNotificationPublisher()
	}
	notificationTaskPublisher, err := issueActivityTaskPublisher(notificationTaskFallback, issueActivityAdmission)
	if err != nil {
		return err
	}
	if emailTaskAdmission != nil {
		notificationTaskPublisher, err = backgroundtasks.NewTemplatedEmailTaskPublisher(notificationTaskPublisher, emailTaskAdmission)
		if err != nil {
			return err
		}
	}
	if webhookTaskAdmission != nil {
		notificationTaskPublisher, err = backgroundtasks.NewWebhookTaskPublisher(notificationTaskPublisher, webhookTaskAdmission)
		if err != nil {
			return err
		}
	}
	if deletionDispatcher != nil {
		notificationTaskPublisher, err = backgroundtasks.NewDeletionTaskPublisher(notificationTaskPublisher, deletionDispatcher)
		if err != nil {
			return err
		}
	}
	if eventTrackingAdmission != nil {
		notificationTaskPublisher, err = backgroundtasks.NewEventTrackingTaskPublisher(notificationTaskPublisher, eventTrackingAdmission)
		if err != nil {
			return err
		}
	}
	if backgroundTaskRuntime != nil {
		notificationTaskPublisher, err = backgroundtasks.NewVersionTaskPublisher(
			notificationTaskPublisher,
			backgroundTaskRuntime.IssueVersions,
			backgroundTaskRuntime.IssueVersionSync,
			backgroundTaskRuntime.IssueDescriptionSync,
			backgroundTaskRuntime.Logger,
		)
		if err != nil {
			return err
		}
	}
	projectPublicationRepository, err := projectpublicationstore.NewRepository(pool, notificationTaskPublisher)
	if err != nil {
		return err
	}
	projectPublicationService, err := systemapi.NewProjectPublication(projectPublicationRepository, sessionValidator)
	if err != nil {
		return err
	}
	projectPublicationEndpoints := projectpublicationapi.NewEndpoints(projectPublicationService)
	projectPublicationEndpoints.Use(log.Endpoint)
	workItemHistoryRepository, err := workitemhistorystore.NewRepository(pool)
	if err != nil {
		return err
	}
	workItemHistoryService, err := systemapi.NewWorkItemHistory(workItemHistoryRepository, sessionValidator)
	if err != nil {
		return err
	}
	workItemHistoryEndpoints := workitemhistoryapi.NewEndpoints(workItemHistoryService)
	workItemHistoryEndpoints.Use(log.Endpoint)
	notificationActionsRepository, err := notificationsstore.NewActionRepository(pool, notificationTaskPublisher)
	if err != nil {
		return err
	}
	notificationActionsService, err := systemapi.NewNotificationActions(notificationActionsRepository, sessionValidator)
	if err != nil {
		return err
	}
	notificationActionsEndpoints := notificationactionsapi.NewEndpoints(notificationActionsService)
	notificationActionsEndpoints.Use(log.Endpoint)
	labelWriteCache, err := projectlabelreadsstore.NewWriteCache(cfg.redisURL)
	if err != nil {
		return err
	}
	defer func() { _ = labelWriteCache.Close() }()
	membershipBaseHost := os.Getenv("APP_BASE_URL")
	membershipAppURL, membershipAppURLErr := url.Parse(membershipBaseHost)
	if membershipAppURLErr != nil || membershipAppURL.Scheme == "" || membershipAppURL.Host == "" {
		membershipBaseHost = os.Getenv("WEB_URL")
	}
	cycleLifecycleRepository, err := cyclereadsstore.NewLifecycleRepository(pool, notificationTaskPublisher, membershipBaseHost)
	if err != nil {
		return err
	}
	cycleLifecycleService, err := systemapi.NewCycleLifecycle(cycleLifecycleRepository, sessionValidator, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	cycleLifecycleEndpoints := cyclelifecycleapi.NewEndpoints(cycleLifecycleService)
	cycleLifecycleEndpoints.Use(log.Endpoint)
	projectLifecycleRepository, err := workspaceprojectreadsstore.NewLifecycleRepository(pool, notificationTaskPublisher, membershipBaseHost, projectLifecycleCoverValidator(assetRepository))
	if err != nil {
		return err
	}
	projectLifecycleService, err := systemapi.NewProjectLifecycle(projectLifecycleRepository, sessionValidator, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	projectLifecycleEndpoints := projectlifecycleapi.NewEndpoints(projectLifecycleService)
	projectLifecycleEndpoints.Use(log.Endpoint)
	moduleLifecycleRepository, err := modulereadsstore.NewLifecycleRepository(pool, notificationTaskPublisher, membershipBaseHost)
	if err != nil {
		return err
	}
	moduleLifecycleService, err := systemapi.NewModuleLifecycle(moduleLifecycleRepository, sessionValidator, apiKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	moduleLifecycleEndpoints := modulelifecycleapi.NewEndpoints(moduleLifecycleService)
	moduleLifecycleEndpoints.Use(log.Endpoint)
	membershipRepository, err := projectmembershipreadsstore.NewLifecycleRepositoryWithReaderWriter(activityReadsPool, pool, notificationTaskPublisher, membershipBaseHost)
	if err != nil {
		return err
	}
	membershipWriteKeys, err := apikey.New(pool)
	if err != nil {
		return err
	}
	membershipKeys := membershipAPIKeyRouter{reader: apiKeyValidator, writer: membershipWriteKeys}
	membershipService, err := systemapi.NewProjectMembershipLifecycle(membershipRepository, membershipKeys, apiKeyThrottle, sessionValidator)
	if err != nil {
		return err
	}
	membershipEndpoints := membershiplifecycleapi.NewEndpoints(membershipService)
	membershipEndpoints.Use(log.Endpoint)
	invitationsRepository, err := projectmembershipreadsstore.NewInvitationsRepositoryWithReaderWriter(activityReadsPool, pool, notificationTaskPublisher, membershipBaseHost, cfg.secretKey)
	if err != nil {
		return err
	}
	invitationsCache, err := projectmembershipreadsstore.NewInvitationCache(cfg.redisURL)
	if err != nil {
		return err
	}
	defer func() { _ = invitationsCache.Close() }()
	invitationsService, err := systemapi.NewInvitationsMembership(invitationsRepository, membershipKeys, sessionValidator, apiKeyThrottle, invitationsCache)
	if err != nil {
		return err
	}
	invitationsEndpoints := invitationsapi.NewEndpoints(invitationsService)
	invitationsEndpoints.Use(log.Endpoint)
	labelWriteRepository, err := projectlabelreadsstore.NewWriteRepository(pool)
	if err != nil {
		return err
	}
	labelWriteService, err := systemapi.NewProjectLabelWrites(labelWriteRepository, sessionValidator, labelWriteCache)
	if err != nil {
		return err
	}
	labelWriteEndpoints := labelwritesapi.NewEndpoints(labelWriteService)
	labelWriteEndpoints.Use(log.Endpoint)
	workItemRepository, err := workitemstore.NewRepositoryWithNativeTasks(pool, notificationTaskPublisher, os.Getenv("WEB_URL"), issueDescriptionVersionAdmission, recentVisitAdmission, issueActivityAdmission)
	if err != nil {
		return err
	}
	typeService, err := systemapi.NewWorkItemTypes(workitemtypesstore.NewRepository(pool), sessionValidator, workItemRepository)
	if err != nil {
		return err
	}
	typeEndpoints := workitemtypesapi.NewEndpoints(typeService)
	typeEndpoints.Use(log.Endpoint)
	publicFeedbackService, err := systemapi.NewPublicFeedback(publicProjectMetadataReadsRepository, workItemRepository)
	if err != nil {
		return err
	}
	publicFeedbackEndpoints := publicfeedbackapi.NewEndpoints(publicFeedbackService)
	publicFeedbackEndpoints.Use(log.Endpoint)
	workItemService, err := systemapi.NewWorkItemCore(workItemRepository, sessionValidator, activityReadsPrimaryAPIKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	workItemEndpoints := workitemapi.NewEndpoints(workItemService)
	workItemEndpoints.Use(log.Endpoint)
	collaborationService, err := systemapi.NewTicketCollaboration(workItemRepository, sessionValidator, activityReadsPrimaryAPIKeyValidator, apiKeyThrottle)
	if err != nil {
		return err
	}
	collaborationEndpoints := collaborationapi.NewEndpoints(collaborationService)
	collaborationEndpoints.Use(log.Endpoint)

	activityReadsEndpoints := activityreadsapi.NewEndpoints(activityReadsService)
	activityReadsEndpoints.Use(log.Endpoint)
	publicProjectMetadataSessionInvalidator, err := publicprojectmetadatareadsstore.NewPostgreSQLSessionInvalidator(pool)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsThrottle, err := publicprojectmetadatareadsstore.NewAnonymousThrottle(
		publicProjectMetadataReadsCache, sessionValidator, publicProjectMetadataSessionInvalidator,
		cfg.sessionCookie, cfg.saveEveryRequest, cfg.sessionCookieAge,
		publicprojectmetadatareadsstore.AdminSessionConfig{CookieName: cfg.adminSessionCookie, CookieAge: cfg.adminSessionCookieAge},
	)
	if err != nil {
		return err
	}
	standardViewsRepository, err := standardviewsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	standardViewsService, err := systemapi.NewStandardViews(standardViewsRepository, sessionValidator, recentVisitAdmission)
	if err != nil {
		return err
	}
	standardViewsEndpoints := standardviewsapi.NewEndpoints(standardViewsService)
	standardViewsEndpoints.Use(log.Endpoint)
	userSettingsRepository, err := usersettingsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	userSettingsService, err := systemapi.NewUserSettings(userSettingsRepository, sessionValidator)
	if err != nil {
		return err
	}
	userSettingsEndpoints := usersettingsapi.NewEndpoints(userSettingsService)
	userSettingsEndpoints.Use(log.Endpoint)
	workspaceSidebarPreferencesRepository, err := workspacesidebarpreferencesstore.NewRepository(pool)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesBackend, err := workspacesidebarpreferencesstore.NewService(workspaceSidebarPreferencesRepository)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesService, err := systemapi.NewWorkspaceSidebarPreferences(workspaceSidebarPreferencesBackend, sessionValidator)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesEndpoints := workspacesidebarpreferencesapi.NewEndpoints(workspaceSidebarPreferencesService)
	workspaceSidebarPreferencesEndpoints.Use(log.Endpoint)

	instanceCipher, err := instancecipher.New(cfg.secretKey)
	if err != nil {
		return err
	}
	if mcpLinks != nil {
		linking, linkErr := mcpoauth.NewLinking(mcpLinks, mcpOAuthProvider, instanceCipher)
		if linkErr != nil {
			return linkErr
		}
		linkService, linkErr := systemapi.NewMCPOAuthLinks(linking, sessionValidator, cfg.mcpOAuth.Issuer, cfg.mcp.ResourceURL)
		if linkErr != nil {
			return linkErr
		}
		mcpLinkEndpoints = mcpoauthlinksapi.NewEndpoints(linkService)
		mcpLinkEndpoints.Use(log.Endpoint)
		stopCleanup := startMCPLinkCleanup(logCtx, mcpLinks)
		defer stopCleanup()
	}
	useDatabaseConfiguration := os.Getenv("SKIP_ENV_VAR") == "1" || strings.EqualFold(os.Getenv("SKIP_ENV_VAR"), "true")
	accountRepository, err := accountauthentication.NewPoolRepository(pool, instanceCipher, useDatabaseConfiguration)
	if err != nil {
		return err
	}
	postAuthWorkflow, err := authworkflow.NewPostUserAuthWorkflow(pool, invitationsCache, activityTaskPublisher)
	if err != nil {
		return err
	}
	passwordHasher := password.Default()
	authRedirectService, err := authredirects.NewService(accountRepository, passwordHasher, passwordstrength.Scorer{}, sessionValidator, postAuthWorkflow, activityTaskPublisher)
	if err != nil {
		return err
	}
	redisOptions, err := redis.ParseURL(cfg.redisURL)
	if err != nil {
		return fmt.Errorf("configure authentication Redis client: %w", err)
	}
	authRedis := redis.NewClient(redisOptions)
	defer func() { _ = authRedis.Close() }()
	magicCodeVerifier, err := authredirects.NewRedisMagicCodeVerifier(authRedis)
	if err != nil {
		return err
	}
	authThrottle, authThrottleCache, err := authredirects.NewAuthenticationDRFThrottle(cfg.redisURL, "")
	if err != nil {
		return err
	}
	defer func() { _ = authThrottleCache.Close() }()
	if err := authRedirectService.SetMagicCodeDependencies(magicCodeVerifier, authThrottle); err != nil {
		return err
	}
	magicIssuer, err := newRedisMagicCodeIssuer(authRedis, func(ctx context.Context, email string) (bool, error) {
		_, err := accountRepository.UserByEmail(ctx, email)
		if errors.Is(err, accountauthentication.ErrUserNotFound) {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return err
	}
	resetTokenIssuer, err := newDjangoPasswordResetTokenIssuer(cfg.secretKey)
	if err != nil {
		return err
	}
	if err := authRedirectService.SetPasswordResetTokenVerifier(resetTokenIssuer); err != nil {
		return err
	}
	accountBaseURL := os.Getenv("APP_BASE_URL")
	if accountBaseURL == "" {
		accountBaseURL = os.Getenv("WEB_URL")
	}
	spaceBaseURL := os.Getenv("SPACE_BASE_URL")
	if spaceBaseURL == "" {
		spacePath := os.Getenv("SPACE_BASE_PATH")
		if spacePath == "" {
			spacePath = "/spaces/"
		}
		spaceBaseURL = strings.TrimRight(os.Getenv("WEB_URL"), "/") + "/" + strings.Trim(spacePath, "/") + "/"
	}
	accountAuthenticationService, err := systemapi.NewAccountAuthentication(accountRepository, authThrottle, invitationsCache, activityTaskPublisher, magicIssuer, resetTokenIssuer, passwordHasher, sessionValidator, accountBaseURL, spaceBaseURL)
	if err != nil {
		return err
	}
	accountAuthenticationEndpoints := accountauthenticationapi.NewEndpoints(accountAuthenticationService)
	accountAuthenticationEndpoints.Use(log.Endpoint)
	authRedirectAdapter, err := systemapi.NewAuthRedirects(authRedirectService)
	if err != nil {
		return err
	}
	authRedirectEndpoints := authredirectsapi.NewEndpoints(authRedirectAdapter)
	authRedirectEndpoints.Use(log.Endpoint)
	oauthRepository, err := systemapi.NewOAuthRedirectPostgresRepository(pool, instanceCipher, useDatabaseConfiguration)
	if err != nil {
		return err
	}
	oauthRedirectService, err := systemapi.NewOAuthRedirects(oauthRepository, sessionValidator, postAuthWorkflow, &http.Client{Timeout: 15 * time.Second})
	if err != nil {
		return err
	}
	oauthRedirectEndpoints := oauthredirectsapi.NewEndpoints(oauthRedirectService)
	oauthRedirectEndpoints.Use(log.Endpoint)
	instanceRepository, err := instancemanagementstore.NewRepository(pool, instanceCipher)
	if err != nil {
		return err
	}
	workspaceLifecycleService, err := systemapi.NewWorkspaceLifecycle(instanceRepository, sessionValidator, activityTaskPublisher)
	if err != nil {
		return err
	}
	workspaceLifecycleEndpoints := workspacelifecycleapi.NewEndpoints(workspaceLifecycleService)
	workspaceLifecycleEndpoints.Use(log.Endpoint)
	instanceService, err := systemapi.NewInstanceManagement(instanceRepository, sessionValidator, instanceCipher, instancemail.Sender{}, invitationsCache, accountRepository, authRedirectService)
	if err != nil {
		return err
	}
	userAccountWritesRepository, err := useraccountwritesstore.NewRepository(pool, userIdentityRepository)
	if err != nil {
		return err
	}
	userAccountWritesCache, err := useraccountwritesstore.NewCache(cfg.redisURL)
	if err != nil {
		return err
	}
	defer func() { _ = userAccountWritesCache.Close() }()
	userAccountWritesService, err := systemapi.NewUserAccountWrites(userAccountWritesRepository, sessionValidator, userAccountWritesCache, activityTaskPublisher, assetAppBaseURL)
	if err != nil {
		return err
	}
	userAccountWritesEndpoints := useraccountwritesapi.NewEndpoints(userAccountWritesService)
	userAccountWritesEndpoints.Use(log.Endpoint)
	instanceEndpoints := instancemanagementapi.NewEndpoints(instanceService)
	instanceEndpoints.Use(log.Endpoint)
	searchRepository, err := searchstore.NewRepository(pool)
	if err != nil {
		return err
	}
	searchService, err := systemapi.NewWorkspaceSearch(searchRepository, sessionValidator)
	if err != nil {
		return err
	}
	searchEndpoints := searchapi.NewEndpoints(searchService)
	searchEndpoints.Use(log.Endpoint)
	analyticsRepository, err := analyticsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	analyticsService, err := systemapi.NewWorkspaceAnalytics(analyticsRepository, sessionValidator)
	if err != nil {
		return err
	}
	analyticsEndpoints := analyticsapi.NewEndpoints(analyticsService)
	analyticsEndpoints.Use(log.Endpoint)
	personalRepository, err := personalstore.NewRepository(pool)
	if err != nil {
		return err
	}
	personalService, err := systemapi.NewPersonalDashboard(personalRepository, sessionValidator)
	if err != nil {
		return err
	}
	personalEndpoints := personalapi.NewEndpoints(personalService)
	personalEndpoints.Use(log.Endpoint)
	exportsRepository, err := exportsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	exportsTaskPublisher := activityTaskPublisher
	if issueExportAdmission != nil {
		exportsTaskPublisher, err = backgroundtasks.NewIssueExportTaskPublisher(exportsTaskPublisher, issueExportAdmission)
		if err != nil {
			return err
		}
	}
	if analyticsExportDispatcher != nil {
		exportsTaskPublisher, err = backgroundtasks.NewAnalyticsExportTaskPublisher(exportsTaskPublisher, analyticsExportDispatcher)
		if err != nil {
			return err
		}
	}
	exportsService, err := systemapi.NewWorkspaceExports(exportsRepository, sessionValidator, exportsTaskPublisher)
	if err != nil {
		return err
	}
	exportsEndpoints := exportsapi.NewEndpoints(exportsService)
	exportsEndpoints.Use(log.Endpoint)
	insightsRepository, err := insightsstore.NewRepository(pool)
	if err != nil {
		return err
	}
	insightsService, err := systemapi.NewWorkspaceUserInsights(insightsRepository, sessionValidator)
	if err != nil {
		return err
	}
	insightsEndpoints := insightsapi.NewEndpoints(insightsService)
	insightsEndpoints.Use(log.Endpoint)
	externalRepository, err := externalstore.NewRepository(pool, instanceCipher)
	if err != nil {
		return err
	}
	externalService, err := systemapi.NewExternalIntegrations(externalRepository, sessionValidator, nil)
	if err != nil {
		return err
	}
	externalEndpoints := externalapi.NewEndpoints(externalService)
	externalEndpoints.Use(log.Endpoint)

	handler := newHTTPHandlerWithPendingReplacements(logCtx, systemEndpoints, cfg.debug, stickyEndpoints, stickyService.(sticky.Authorizer).APIKeyAuth, workspaceSlugEndpoints, instanceWorkspaceSlugEndpoints, userInstanceAdminEndpoints, userProfileFlagsEndpoints, userNotificationPreferencesEndpoints, userAccountsEndpoints, userAPITokensEndpoints, instanceAdminSessionEndpoints, workspaceQuickLinksEndpoints, workspaceThemesEndpoints, workspaceHomePreferencesEndpoints, workspaceMemberMeEndpoints, analyticViewsEndpoints, workspaceRecentVisitsEndpoints, lastVisitedWorkspaceEndpoints, entityFavoritesEndpoints, pendingReplacementEndpoints{
		workItemTypes:                           typeEndpoints,
		mcpLinks:                                mcpLinkEndpoints,
		mcpResourceURL:                          cfg.mcp.ResourceURL,
		accountAuthentication:                   accountAuthenticationEndpoints,
		accountAuthenticationMetadata:           accountAuthenticationService.(systemapi.SessionProtocolAuthorizer),
		authRedirects:                           authRedirectEndpoints,
		oauthRedirects:                          oauthRedirectEndpoints,
		readServicesAuthSession:                 publicProjectMetadataReadsThrottle,
		readServicesAuthCookie:                  djangoSessionCookieConfig{Name: cfg.sessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		userAccountWrites:                       userAccountWritesEndpoints,
		userAccountWritesMetadata:               userAccountWritesService.(sessionResourceProtocolAuthorizer),
		instanceManagement:                      instanceEndpoints,
		workspaceUserInsights:                   insightsEndpoints,
		workspaceUserInsightsMetadata:           insightsService.(sessionResourceProtocolAuthorizer),
		workspaceExports:                        exportsEndpoints,
		workspaceExportsMetadata:                exportsService.(sessionResourceProtocolAuthorizer),
		personalDashboard:                       personalEndpoints,
		personalDashboardMetadata:               personalService.(sessionResourceProtocolAuthorizer),
		workspaceSearch:                         searchEndpoints,
		workspaceSearchMetadata:                 searchService.(sessionResourceProtocolAuthorizer),
		workspaceAnalytics:                      analyticsEndpoints,
		workspaceAnalyticsMetadata:              analyticsService.(sessionResourceProtocolAuthorizer),
		externalIntegrations:                    externalEndpoints,
		externalIntegrationsMetadata:            externalService.(sessionResourceProtocolAuthorizer),
		instanceManagementMetadata:              instanceService.(sessionResourceProtocolAuthorizer),
		instanceManagementThrottle:              publicProjectMetadataReadsThrottle,
		csrfToken:                               csrfTokenEndpoints,
		csrfTokenCORS:                           cfg.corsOrigins,
		accountAuthenticationCSRFTrustedOrigins: cfg.csrfTrustedOrigins,
		csrfTokenCookie:                         djangoCSRFCookieConfig{Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure},
		csrfTokenThrottle:                       publicProjectMetadataReadsThrottle,
		csrfTokenSessionCookie:                  djangoSessionCookieConfig{Name: cfg.sessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		csrfTokenBodyLimit:                      cfg.fileSizeLimit,
		csrfTokenBodySet:                        true,
		userIdentity:                            userIdentityEndpoints,
		userIdentitySession:                     publicProjectMetadataReadsThrottle,
		userIdentityCORS:                        cfg.corsOrigins,
		userIdentityCookie:                      djangoSessionCookieConfig{Name: cfg.sessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		userIdentityBodyLimit:                   cfg.fileSizeLimit,
		userIdentityBodySet:                     true,
		userIdentityActivity:                    publicProjectMetadataActivityLogger,
		userIdentitySecret:                      cfg.secretKey,
		assetsLifecycle:                         assetsLifecycleEndpoints,
		assetsSessionMetadata:                   assetsLifecycleMetadata,
		assetsAPIMetadata:                       assetsLifecycleMetadata,
		assetsDuplicateThrottle:                 assetDuplicateThrottle,
		userProperties:                          userPropertiesEndpoints,
		workspaceMetadata:                       workspaceMetadataEndpoints,
		projectIdentifiers:                      projectIdentifiersEndpoints,
		projectMembershipReads:                  projectMembershipReadsEndpoints,
		projectMembershipMetadata:               projectMembershipReadsService.(systemapi.ProjectMembershipReadsMetadataAuthorizer),
		projectStateReads:                       projectStateReadsEndpoints,
		projectStateMetadata:                    projectStateReadsService.(systemapi.ProjectStateReadsMetadataAuthorizer),
		projectLabelReads:                       projectLabelReadsEndpoints,
		projectLabelWrites:                      labelWriteEndpoints,
		workItemCore:                            workItemEndpoints,
		ticketCollaboration:                     collaborationEndpoints,
		workItemProtocol:                        collaborationProtocolAuthorizer{core: workItemService.(systemapi.WorkItemProtocolAuthorizer), collaboration: collaborationService.(systemapi.WorkItemProtocolAuthorizer)},
		projectLabelWritesMetadata:              labelWriteService.(systemapi.SessionResourceProtocolAuthorizer),
		publicProjectMetadataReads:              publicProjectMetadataReadsEndpoints,
		publicFeedback:                          publicFeedbackEndpoints,
		publicProjectMetadataThrottle:           publicProjectMetadataReadsThrottle,
		publicProjectMetadataCORS:               cfg.corsOrigins,
		publicProjectMetadataCookie:             djangoSessionCookieConfig{Name: cfg.sessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		publicProjectMetadataAdminCookie:        djangoSessionCookieConfig{Name: cfg.adminSessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		publicProjectMetadataActivity:           publicProjectMetadataActivityLogger,
		publicProjectMetadataSecret:             cfg.secretKey,
		publicProjectMetadataBodyLimit:          cfg.fileSizeLimit,
		publicProjectMetadataBodySet:            true,
		standardViews:                           standardViewsEndpoints,
		userSettings:                            userSettingsEndpoints,
		workspaceSidebarPreferences:             workspaceSidebarPreferencesEndpoints,
		readServicesCORS:                        cfg.corsOrigins,
		stickyWorkspaceAuthorizer:               stickyService.(systemapi.StickyWorkspaceAuthorizer),
		readServicesSession:                     publicProjectMetadataReadsThrottle,
		readServicesCookie:                      djangoSessionCookieConfig{Name: cfg.sessionCookie, Domain: cfg.cookieDomain, Secure: cfg.sessionCookieSecure, Configured: true},
		workspaceProjectReads:                   workspaceProjectReadsEndpoints,
		workspaceLifecycle:                      workspaceLifecycleEndpoints,
		workspaceLifecycleMetadata:              workspaceLifecycleService.(systemapi.SessionResourceProtocolAuthorizer),
		projectLifecycle:                        projectLifecycleEndpoints,
		projectLifecycleMetadata:                projectLifecycleService,
		projectPublication:                      projectPublicationEndpoints,
		projectPublicationMetadata:              projectPublicationService.(sessionResourceProtocolAuthorizer),
		workItemHistory:                         workItemHistoryEndpoints,
		workItemHistoryMetadata:                 workItemHistoryService.(sessionResourceProtocolAuthorizer),
		cycleReads:                              cycleReadsEndpoints,
		cycleLifecycle:                          cycleLifecycleEndpoints,
		cycleLifecycleMetadata:                  cycleLifecycleService,
		moduleLifecycle:                         moduleLifecycleEndpoints,
		moduleLifecycleMetadata:                 moduleLifecycleService,
		moduleReads:                             moduleReadsEndpoints,
		activityReads:                           activityReadsEndpoints,
		notifications:                           notificationsEndpoints,
		notificationActions:                     notificationActionsEndpoints,
		notificationsMetadata:                   notificationsService.(systemapi.SessionProtocolAuthorizer),
		activityReadsMetadata:                   activityReadsMetadata,
		apiCycleReads:                           apiCycleEndpoints,
		apiModuleReads:                          apiModuleEndpoints,
		apiLabelReads:                           apiLabelEndpoints,
		projectConfiguration:                    configurationEndpoints,
		customViewCreation:                      customViewCreationEndpoints,
		customViewCreationMetadata:              customViewCreationService.(systemapi.SessionResourceProtocolAuthorizer),
		projectMemberPreferences:                preferencesEndpoints,
		projectMemberPreferencesMetadata:        preferencesService.(sessionResourceProtocolAuthorizer),
		projectIntakeWrites:                     intakeWritesEndpoints,
		projectIntakeWritesMetadata:             intakeWritesService.(systemapi.SessionResourceProtocolAuthorizer),
		projectConfigurationMetadata:            configurationService.(projectConfigurationProtocolAuthorizer),
		workspaceWebhooks:                       webhookEndpoints,
		workspaceWebhooksMetadata:               webhookService.(sessionResourceProtocolAuthorizer),
		invitationsMembership:                   invitationsEndpoints,
		invitationsMetadata:                     invitationsService.(systemapi.InvitationsMembershipProtocolAuthorizer),
		membershipLifecycle:                     membershipEndpoints,
		membershipLifecycleMetadata:             membershipService.(systemapi.MembershipLifecycleProtocolAuthorizer),
		apiStateReads:                           apiStateEndpoints,
		apiV1Stickies:                           apiV1StickiesEndpoints,
		apiV1StickiesProtocol:                   apiV1StickiesService.(systemapi.APIV1StickyProtocolAuthorizer),
		apiUserIdentity:                         apiUserIdentityEndpoints,
		activityReadsActivity:                   publicProjectMetadataActivityLogger,
		activityReadsSecret:                     cfg.secretKey,
		activityReadsBodyLimit:                  cfg.fileSizeLimit,
		estimateReads:                           estimateReadsEndpoints,
		pageReads:                               pageReadsEndpoints,
		pageReadsMetadata:                       pageReadsService.(systemapi.SessionResourceProtocolAuthorizer),
		intakeInboxReads:                        intakeInboxReadsEndpoints,
		deployBoardReads:                        deployBoardReadsEndpoints,
	}, workspaceFavoritesEndpoints)
	mcpIdentity, err := useridentitystore.NewIdentityReader(userIdentityRepository)
	if err != nil {
		return err
	}
	mcpMembers, err := projectmembershipreadsstore.NewMemberReader(membershipRepository)
	if err != nil {
		return err
	}
	mcpDirectory, err := projectmembershipreadsstore.NewDirectoryReader(pool)
	if err != nil {
		return err
	}
	mcpOptions.Service, err = mcpactions.New(mcpIdentity, mcpMembers, mcpDirectory)
	if err != nil {
		return err
	}
	handler, err = mcptransport.New(cfg.mcp, mcpOptions, handler)
	if err != nil {
		return err
	}
	handler = mcptransport.RedactRejectedQueries(wrapHTTP(handler))
	serveErr := listenAndServeHTTP(logCtx, cfg.httpAddress, handler, cfg.shutdownTimeout)
	return serveErr
}

func issueActivityTaskPublisher(fallback backgroundtasks.TaskFallback, admission backgroundtasks.IssueActivityTaskAdmitter) (workitemstore.Publisher, error) {
	if fallback == nil {
		return nil, errors.New("task publisher is required")
	}
	if admission == nil {
		return fallback, nil
	}
	return backgroundtasks.NewIssueActivityTaskPublisher(fallback, admission)
}

type rejectingTaskPublisher struct{}

func (rejectingTaskPublisher) Task(_ context.Context, name string, _ []any, _ map[string]any) error {
	return fmt.Errorf("native background task %q has no registered Go handler", name)
}

func (rejectingTaskPublisher) SoftDelete(_ context.Context, table, id, field string) error {
	return fmt.Errorf("native soft delete %s.%s by %s has no registered Go handler", table, field, id)
}

func readReplicaDatabaseURL() (string, error) {
	if os.Getenv("ENABLE_READ_REPLICA") != "1" {
		return "", nil
	}
	if configured := os.Getenv("DATABASE_READ_REPLICA_URL"); configured != "" {
		return configured, nil
	}
	databaseName := os.Getenv("POSTGRES_READ_REPLICA_DB")
	username := os.Getenv("POSTGRES_READ_REPLICA_USER")
	host := os.Getenv("POSTGRES_READ_REPLICA_HOST")
	if databaseName == "" || username == "" || host == "" {
		return "", fmt.Errorf("read replica requires DATABASE_READ_REPLICA_URL or POSTGRES_READ_REPLICA_DB, POSTGRES_READ_REPLICA_USER, and POSTGRES_READ_REPLICA_HOST")
	}
	password := os.Getenv("POSTGRES_READ_REPLICA_PASSWORD")
	user := url.User(username)
	if password != "" {
		user = url.UserPassword(username, password)
	}
	return (&url.URL{
		Scheme: "postgresql",
		User:   user,
		Host:   net.JoinHostPort(host, envValueOrDefault("POSTGRES_READ_REPLICA_PORT", "5432")),
		Path:   "/" + databaseName,
	}).String(), nil
}
func parseConfig(args []string) (config, error) {
	mcpConfig, err := mcpConfigFromEnvironment()
	if err != nil {
		return config{}, err
	}
	sessionCookieAge, err := envSeconds("SESSION_COOKIE_AGE", 604800)
	if err != nil {
		return config{}, err
	}
	adminSessionCookieAge, err := envSeconds("ADMIN_SESSION_COOKIE_AGE", 3600)
	if err != nil {
		return config{}, err
	}
	fileSizeLimit, err := envInt64("FILE_SIZE_LIMIT", defaultFileSizeLimit)
	if err != nil {
		return config{}, err
	}
	debugMode, err := envDjangoDebug()
	if err != nil {
		return config{}, err
	}

	flags := flag.NewFlagSet("go-plane", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	brokerURL := os.Getenv("AMQP_URL")
	if brokerURL == "" {
		brokerURL = fmt.Sprintf(
			"amqp://%s:%s@%s:%s/%s",
			envValueOrDefault("RABBITMQ_USER", "guest"),
			envValueOrDefault("RABBITMQ_PASSWORD", "guest"),
			envValueOrDefault("RABBITMQ_HOST", "localhost"),
			envValueOrDefault("RABBITMQ_PORT", "5672"),
			envValueOrDefault("RABBITMQ_VHOST", "/"),
		)
	}

	cfg := config{sessionCookieAge: sessionCookieAge, adminSessionCookieAge: adminSessionCookieAge, mcp: mcpConfig, mcpOAuth: mcpOAuthConfigFromEnvironment(mcpConfig.ResourceURL)}
	flags.StringVar(&cfg.httpAddress, "http-address", defaultHTTPAddress, "HTTP listen address")
	flags.DurationVar(&cfg.shutdownTimeout, "shutdown-timeout", defaultShutdownTimeout, "graceful shutdown timeout")
	flags.BoolVar(&cfg.debug, "debug", debugMode, "enable Django-compatible debug behavior and debug logs")
	flags.BoolVar(&cfg.goBackgroundTasks, "go-background-tasks", false, "run native durable tasks and admit migrated task groups")
	flags.BoolVar(&cfg.goBackgroundTasksNoWorkers, "go-background-tasks-no-workers", false, "initialize native durable admission without starting activity workers")
	flags.StringVar(&cfg.databaseURL, "database-url", os.Getenv("DATABASE_URL"), "PostgreSQL URL")
	flags.StringVar(&cfg.readReplicaDatabaseURL, "read-replica-database-url", "", "PostgreSQL read replica URL")
	flags.StringVar(&cfg.redisURL, "redis-url", os.Getenv("REDIS_URL"), "Redis URL")
	flags.StringVar(&cfg.apiKeyRate, "api-key-rate", envValueOrDefault("API_KEY_RATE_LIMIT", "60/minute"), "Django API-key throttle rate")
	flags.StringVar(&cfg.amqpURL, "celery-broker-url", brokerURL, "Celery AMQP broker URL")
	var corsOrigins string
	flags.StringVar(&corsOrigins, "cors-allowed-origins", os.Getenv("CORS_ALLOWED_ORIGINS"), "comma-separated CORS allowed origins")
	csrfTrustedOriginsRaw, csrfTrustedOriginsSet := os.LookupEnv("CSRF_TRUSTED_ORIGINS")
	if !csrfTrustedOriginsSet {
		csrfTrustedOriginsRaw = os.Getenv("CORS_ALLOWED_ORIGINS")
	}
	flags.StringVar(&csrfTrustedOriginsRaw, "csrf-trusted-origins", csrfTrustedOriginsRaw, "comma-separated trusted origins for account-authentication CSRF checks")
	flags.StringVar(&cfg.sessionCookie, "django-session-cookie-name", envValueOrDefault("SESSION_COOKIE_NAME", djangosession.DefaultCookieName), "Django session cookie name")
	flags.StringVar(&cfg.adminSessionCookie, "django-admin-session-cookie-name", "admin-session-id", "Django admin session cookie name")
	flags.StringVar(&cfg.cookieDomain, "django-session-cookie-domain", os.Getenv("COOKIE_DOMAIN"), "Django session cookie domain")
	flags.Func("django-session-cookie-age", "Django session cookie age", func(value string) error {
		age, err := sessionAgeFlag(value)
		if err == nil {
			cfg.sessionCookieAge = age
		}
		return err
	})
	flags.Func("django-admin-session-cookie-age", "Django admin session cookie age", func(value string) error {
		age, err := sessionAgeFlag(value)
		if err == nil {
			cfg.adminSessionCookieAge = age
		}
		return err
	})
	flags.BoolVar(&cfg.saveEveryRequest, "django-session-save-every-request", os.Getenv("SESSION_SAVE_EVERY_REQUEST") == "1", "refresh Django sessions after every successful request")
	flags.StringVar(&cfg.secretKey, "django-secret-key", os.Getenv("SECRET_KEY"), "Django session signing key")
	var fallbackKeys string
	flags.StringVar(&fallbackKeys, "django-fallback-secret-keys", os.Getenv("SECRET_KEY_FALLBACKS"), "comma-separated Django fallback signing keys")
	flags.Int64Var(&cfg.fileSizeLimit, "file-size-limit", fileSizeLimit, "Django request body size limit")

	if err := flags.Parse(args); err != nil {
		return config{}, fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if cfg.goBackgroundTasksNoWorkers && !cfg.goBackgroundTasks {
		return config{}, fmt.Errorf("go-background-tasks-no-workers requires go-background-tasks")
	}
	readReplicaFlagSet := false
	flags.Visit(func(parsed *flag.Flag) {
		readReplicaFlagSet = readReplicaFlagSet || parsed.Name == "read-replica-database-url"
	})
	if !readReplicaFlagSet {
		cfg.readReplicaDatabaseURL, err = readReplicaDatabaseURL()
		if err != nil {
			return config{}, err
		}
	}
	if cfg.httpAddress == "" {
		return config{}, fmt.Errorf("http-address must not be empty")
	}
	if cfg.shutdownTimeout <= 0 {
		return config{}, fmt.Errorf("shutdown-timeout must be positive")
	}
	if cfg.secretKey == "" {
		return config{}, fmt.Errorf("django-secret-key is required")
	}
	if cfg.adminSessionCookie == "" {
		return config{}, fmt.Errorf("django-admin-session-cookie-name must not be empty")
	}
	for _, key := range strings.Split(fallbackKeys, ",") {
		if key = strings.TrimSpace(key); key != "" {
			cfg.fallbackKeys = append(cfg.fallbackKeys, key)
		}
	}
	for _, origin := range strings.Split(corsOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.corsOrigins = append(cfg.corsOrigins, origin)
		}
	}
	flags.Visit(func(parsed *flag.Flag) {
		if parsed.Name == "csrf-trusted-origins" {
			csrfTrustedOriginsSet = true
		}
	})
	if csrfTrustedOriginsSet {
		for _, origin := range strings.Split(csrfTrustedOriginsRaw, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				cfg.csrfTrustedOrigins = append(cfg.csrfTrustedOrigins, origin)
			}
		}
	} else {
		cfg.csrfTrustedOrigins = append(cfg.csrfTrustedOrigins, cfg.corsOrigins...)
	}
	cfg.sessionCookieSecure = len(cfg.corsOrigins) > 0
	for _, origin := range cfg.corsOrigins {
		if strings.Contains(origin, "http:") {
			cfg.sessionCookieSecure = false
			break
		}
	}
	return cfg, nil
}

func envSeconds(name string, fallback int64) (int64, error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return fallback, nil
	}
	value, err := boundedInt64(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer number of seconds: %w", name, err)
	}
	return value, nil
}

func sessionAgeFlag(value string) (int64, error) {
	if duration, err := time.ParseDuration(value); err == nil {
		return int64(duration / time.Second), nil
	}
	age, err := boundedInt64(value)
	if err != nil {
		return 0, fmt.Errorf("session cookie age must be a duration or integer seconds: %w", err)
	}
	return age, nil
}

func boundedInt64(value string) (int64, error) {
	integer, err := pythoncompat.DecimalInteger(value)
	if err != nil {
		return 0, err
	}
	if integer.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		// Python accepts values wider than int64 within its decimal conversion
		// limit. Saturation keeps startup compatible; the session save rejects
		// the resulting date as outside datetime's year 1..9999 range.
		return math.MaxInt64, nil
	}
	if integer.Cmp(big.NewInt(math.MinInt64)) < 0 {
		return math.MinInt64, nil
	}
	return integer.Int64(), nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envValueOrDefault(name, fallback string) string {
	if value, found := os.LookupEnv(name); found {
		return value
	}
	return fallback
}

func envInt64(name string, fallback int64) (int64, error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return fallback, nil
	}
	value, err := pythoncompat.DecimalInteger(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	if value.Cmp(big.NewInt(1<<63-1)) > 0 {
		return 1<<63 - 1, nil
	}
	if value.Cmp(big.NewInt(-1<<63)) < 0 {
		return -1 << 63, nil
	}
	return value.Int64(), nil
}

func envDjangoDebug() (bool, error) {
	raw, found := os.LookupEnv("DEBUG")
	if !found {
		return false, nil
	}
	value, err := pythoncompat.DecimalInteger(raw)
	if err != nil {
		return false, fmt.Errorf("DEBUG must be an integer: %w", err)
	}
	// The API process loads plane.settings.production, which normalizes DEBUG
	// with exactly this comparison after importing the shared settings.
	return value.IsInt64() && value.Int64() == 1, nil
}

func newLogContext(ctx context.Context, output io.Writer, debug bool, additionalOutputs ...log.Output) context.Context {
	outputs := []log.Output{{Writer: output, Format: log.FormatJSON}}
	outputs = append(outputs, additionalOutputs...)
	options := []log.LogOption{
		log.WithOutputs(outputs...),
		log.WithDisableBuffering(func(context.Context) bool { return true }),
		log.WithFunc(log.Span),
		log.WithFunc(func(ctx context.Context) []log.KV {
			var fields []log.KV
			requestID, ok := platform.RequestID(ctx)
			if ok {
				fields = append(fields, log.KV{K: log.RequestIDKey, V: requestID})
			}
			switch ctx.Err() {
			case context.Canceled:
				fields = append(fields, log.KV{K: "request_context_error", V: "context_canceled"})
			case context.DeadlineExceeded:
				fields = append(fields, log.KV{K: "request_context_error", V: "deadline_exceeded"})
			}
			return fields
		}),
	}
	if debug {
		options = append(options, log.WithDebug())
	}
	return log.Context(ctx, options...)
}
