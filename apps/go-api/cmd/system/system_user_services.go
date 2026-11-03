package main

import (
	"fmt"

	systemapi "github.com/CaliLuke/go-plane"
	analyticviewsapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	csrftokenapi "github.com/CaliLuke/go-plane/gen/csrf_token"
	cyclereadsapi "github.com/CaliLuke/go-plane/gen/cycle_reads"
	deployboardreadsapi "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	entityfavoritesapi "github.com/CaliLuke/go-plane/gen/entity_favorites"
	estimatereadsapi "github.com/CaliLuke/go-plane/gen/estimate_reads"
	instanceadminsessionapi "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	intakeinboxreadsapi "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	lastvisitedworkspaceapi "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	modulereadsapi "github.com/CaliLuke/go-plane/gen/module_reads"
	pagereadsapi "github.com/CaliLuke/go-plane/gen/page_reads"
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
	workspaceprojectreadsapi "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	workspacequicklinksapi "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	workspacerecentvisitsapi "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspaceslugcheck "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	workspacethemesapi "github.com/CaliLuke/go-plane/gen/workspace_themes"
	analyticviewsstore "github.com/CaliLuke/go-plane/internal/analyticviews"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	deployboardreadsstore "github.com/CaliLuke/go-plane/internal/deployboardreads"
	entityfavoritesstore "github.com/CaliLuke/go-plane/internal/entityfavorites"
	estimatereadsstore "github.com/CaliLuke/go-plane/internal/estimatereads"
	instanceadminsessionstore "github.com/CaliLuke/go-plane/internal/instanceadminsession"
	instanceworkspaceslugstore "github.com/CaliLuke/go-plane/internal/instanceworkspaceslugcheck"
	intakeinboxreadsstore "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	lastvisitedworkspacestore "github.com/CaliLuke/go-plane/internal/lastvisitedworkspace"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	useraccountsstore "github.com/CaliLuke/go-plane/internal/useraccounts"
	userapitokensstore "github.com/CaliLuke/go-plane/internal/userapitokens"
	userinstanceadminstore "github.com/CaliLuke/go-plane/internal/userinstanceadmin"
	usernotificationpreferencesstore "github.com/CaliLuke/go-plane/internal/usernotificationpreferences"
	userprofileflagsstore "github.com/CaliLuke/go-plane/internal/userprofileflags"
	workspacefavoritesstore "github.com/CaliLuke/go-plane/internal/workspacefavorites"
	workspacehomepreferencesstore "github.com/CaliLuke/go-plane/internal/workspacehomepreferences"
	workspacemembermestore "github.com/CaliLuke/go-plane/internal/workspacememberme"
	workspaceprojectreadsstore "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	workspacequicklinksstore "github.com/CaliLuke/go-plane/internal/workspacequicklinks"
	workspacerecentvisitsstore "github.com/CaliLuke/go-plane/internal/workspacerecentvisits"
	workspaceslugstore "github.com/CaliLuke/go-plane/internal/workspaceslugcheck"
	workspacethemesstore "github.com/CaliLuke/go-plane/internal/workspacethemes"
	"github.com/CaliLuke/loom/clue/log"
)

func (assembly *systemAssembly) browserSessions() error {
	var err error
	assembly.systemEndpoints = system.NewEndpoints(systemapi.NewSystemWithDatabase(assembly.pool))
	assembly.systemEndpoints.Use(log.Endpoint)
	csrfTokenEndpoints := csrftokenapi.NewEndpoints(systemapi.NewCSRFToken())
	csrfTokenEndpoints.Use(log.Endpoint)
	assembly.repository, err = stickystore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	assembly.sessionValidator, err = browsersession.New(browsersession.NewPostgreSQLReader(assembly.pool), browsersession.Config{
		CookieName: assembly.cfg.sessionCookie, CookieNameConfigured: true, SecretKey: assembly.cfg.secretKey, FallbackKeys: assembly.cfg.fallbackKeys,
	})
	if err != nil {
		return fmt.Errorf("configure browser session validation: %w", err)
	}
	assembly.stickyService, err = systemapi.NewStickies(assembly.repository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.stickyEndpoints = sticky.NewEndpoints(assembly.stickyService)
	assembly.stickyEndpoints.Use(log.Endpoint)
	workspaceSlugRepository, err := workspaceslugstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceSlugService, err := systemapi.NewWorkspaceSlugCheck(workspaceSlugRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceSlugEndpoints = workspaceslugcheck.NewEndpoints(workspaceSlugService)
	assembly.workspaceSlugEndpoints.Use(log.Endpoint)
	instanceWorkspaceSlugRepository, err := instanceworkspaceslugstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	instanceWorkspaceSlugService, err := systemapi.NewInstanceWorkspaceSlugCheck(instanceWorkspaceSlugRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.instanceWorkspaceSlugEndpoints = instanceworkspaceapi.NewEndpoints(instanceWorkspaceSlugService)
	assembly.instanceWorkspaceSlugEndpoints.Use(log.Endpoint)
	assembly.routes.csrfToken = csrfTokenEndpoints
	assembly.routes.stickyWorkspaceAuthorizer = assembly.stickyService.(systemapi.StickyWorkspaceAuthorizer)
	return nil
}

func (assembly *systemAssembly) accountReads() error {
	var err error
	userInstanceAdminRepository, err := userinstanceadminstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userInstanceAdminService, err := systemapi.NewUserInstanceAdmin(userInstanceAdminRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.userInstanceAdminEndpoints = userinstanceadminapi.NewEndpoints(userInstanceAdminService)
	assembly.userInstanceAdminEndpoints.Use(log.Endpoint)
	userProfileFlagsRepository, err := userprofileflagsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userProfileFlagsService, err := systemapi.NewUserProfileFlags(userProfileFlagsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.userProfileFlagsEndpoints = userprofileflagsapi.NewEndpoints(userProfileFlagsService)
	assembly.userProfileFlagsEndpoints.Use(log.Endpoint)
	userNotificationPreferencesRepository, err := usernotificationpreferencesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userNotificationPreferencesService, err := systemapi.NewUserNotificationPreferences(userNotificationPreferencesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.userNotificationPreferencesEndpoints = usernotificationpreferencesapi.NewEndpoints(userNotificationPreferencesService)
	assembly.userNotificationPreferencesEndpoints.Use(log.Endpoint)
	userAccountsRepository, err := useraccountsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userAccountsService, err := systemapi.NewUserAccounts(userAccountsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.userAccountsEndpoints = useraccountsapi.NewEndpoints(userAccountsService)
	assembly.userAccountsEndpoints.Use(log.Endpoint)
	userAPITokensRepository, err := userapitokensstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userAPITokensService, err := systemapi.NewUserAPITokens(userAPITokensRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.userAPITokensEndpoints = userapitokensapi.NewEndpoints(userAPITokensService)
	assembly.userAPITokensEndpoints.Use(log.Endpoint)
	instanceAdminSessionRepository, err := instanceadminsessionstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	instanceAdminSessionService, err := systemapi.NewInstanceAdminSession(instanceAdminSessionRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.instanceAdminSessionEndpoints = instanceadminsessionapi.NewEndpoints(instanceAdminSessionService)
	assembly.instanceAdminSessionEndpoints.Use(log.Endpoint)

	return nil
}

func (assembly *systemAssembly) workspacePreferences() error {
	var err error
	workspaceQuickLinksRepository, err := workspacequicklinksstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceQuickLinksService, err := systemapi.NewWorkspaceQuickLinks(workspaceQuickLinksRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceQuickLinksEndpoints = workspacequicklinksapi.NewEndpoints(workspaceQuickLinksService)
	assembly.workspaceQuickLinksEndpoints.Use(log.Endpoint)
	workspaceThemesRepository, err := workspacethemesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceThemesService, err := systemapi.NewWorkspaceThemes(workspaceThemesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceThemesEndpoints = workspacethemesapi.NewEndpoints(workspaceThemesService)
	assembly.workspaceThemesEndpoints.Use(log.Endpoint)
	workspaceHomePreferencesRepository, err := workspacehomepreferencesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceHomePreferencesService, err := systemapi.NewWorkspaceHomePreferences(workspaceHomePreferencesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceHomePreferencesEndpoints = workspacehomepreferencesapi.NewEndpoints(workspaceHomePreferencesService)
	assembly.workspaceHomePreferencesEndpoints.Use(log.Endpoint)
	workspaceMemberMeRepository, err := workspacemembermestore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceMemberMeService, err := systemapi.NewWorkspaceMemberMe(workspaceMemberMeRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceMemberMeEndpoints = workspacemembermeapi.NewEndpoints(workspaceMemberMeService)
	assembly.workspaceMemberMeEndpoints.Use(log.Endpoint)
	analyticViewsRepository, err := analyticviewsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	analyticViewsService, err := systemapi.NewAnalyticViews(analyticViewsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.analyticViewsEndpoints = analyticviewsapi.NewEndpoints(analyticViewsService)
	assembly.analyticViewsEndpoints.Use(log.Endpoint)

	return nil
}

func (assembly *systemAssembly) favorites() error {
	var err error
	workspaceFavoritesRepository, err := workspacefavoritesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceFavoritesService, err := systemapi.NewWorkspaceFavorites(workspaceFavoritesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceFavoritesEndpoints = workspacefavoritesapi.NewEndpoints(workspaceFavoritesService)
	assembly.workspaceFavoritesEndpoints.Use(log.Endpoint)
	entityFavoritesRepository, err := entityfavoritesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	entityFavoritesService, err := systemapi.NewEntityFavorites(entityFavoritesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.entityFavoritesEndpoints = entityfavoritesapi.NewEndpoints(entityFavoritesService)
	assembly.entityFavoritesEndpoints.Use(log.Endpoint)

	return nil
}

func (assembly *systemAssembly) planningReads() error {
	var err error
	assembly.workspaceRecentVisitsRepository, err = workspacerecentvisitsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceRecentVisitsService, err := systemapi.NewWorkspaceRecentVisits(assembly.workspaceRecentVisitsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.workspaceRecentVisitsEndpoints = workspacerecentvisitsapi.NewEndpoints(workspaceRecentVisitsService)
	assembly.workspaceRecentVisitsEndpoints.Use(log.Endpoint)
	assembly.workspaceProjectReadsRepository, err = workspaceprojectreadsstore.NewRepositoryWithRecentVisits(assembly.pool, assembly.activityTaskPublisher, assembly.recentVisitAdmission)
	if err != nil {
		return err
	}
	workspaceProjectReadsService, err := systemapi.NewWorkspaceProjectReads(assembly.workspaceProjectReadsRepository, assembly.sessionValidator, assembly.workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	workspaceProjectReadsEndpoints := workspaceprojectreadsapi.NewEndpoints(workspaceProjectReadsService)
	workspaceProjectReadsEndpoints.Use(log.Endpoint)

	cycleReadsRepository, err := cyclereadsstore.NewRepositoryWithRecentVisits(assembly.pool, assembly.activityTaskPublisher, assembly.recentVisitAdmission)
	if err != nil {
		return err
	}
	cycleReadsService, err := systemapi.NewCycleReads(cycleReadsRepository, assembly.sessionValidator, assembly.workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	cycleReadsEndpoints := cyclereadsapi.NewEndpoints(cycleReadsService)
	cycleReadsEndpoints.Use(log.Endpoint)
	assembly.moduleReadsRepository, err = modulereadsstore.NewRepositoryWithRecentVisits(assembly.pool, assembly.activityTaskPublisher, assembly.recentVisitAdmission)
	if err != nil {
		return err
	}
	moduleReadsService, err := systemapi.NewModuleReads(assembly.moduleReadsRepository, assembly.sessionValidator, assembly.workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	moduleReadsEndpoints := modulereadsapi.NewEndpoints(moduleReadsService)
	moduleReadsEndpoints.Use(log.Endpoint)
	estimateReadsRepository, err := estimatereadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	estimateReadsService, err := systemapi.NewEstimateReads(estimateReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	estimateReadsEndpoints := estimatereadsapi.NewEndpoints(estimateReadsService)
	estimateReadsEndpoints.Use(log.Endpoint)
	assembly.routes.workspaceProjectReads = workspaceProjectReadsEndpoints
	assembly.routes.cycleReads = cycleReadsEndpoints
	assembly.routes.moduleReads = moduleReadsEndpoints
	assembly.routes.estimateReads = estimateReadsEndpoints
	return nil
}

func (assembly *systemAssembly) pageReads() error {
	var err error
	pageReadsRepository, err := pagereadsstore.NewRepositoryWithNativeTasks(assembly.pool, assembly.activityTaskPublisher, assembly.pageTaskAdmission, assembly.recentVisitAdmission)
	if err != nil {
		return err
	}
	pageReadsService, err := systemapi.NewPageReads(pageReadsRepository, assembly.sessionValidator, assembly.workspaceRecentVisitsRepository)
	if err != nil {
		return err
	}
	pageReadsEndpoints := pagereadsapi.NewEndpoints(pageReadsService)
	pageReadsEndpoints.Use(log.Endpoint)
	intakeInboxReadsRepository, err := intakeinboxreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	intakeInboxReadsService, err := systemapi.NewIntakeInboxReads(intakeInboxReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	intakeInboxReadsEndpoints := intakeinboxreadsapi.NewEndpoints(intakeInboxReadsService)
	intakeInboxReadsEndpoints.Use(log.Endpoint)
	deployBoardReadsRepository, err := deployboardreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	deployBoardReadsService, err := systemapi.NewDeployBoardReads(deployBoardReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	deployBoardReadsEndpoints := deployboardreadsapi.NewEndpoints(deployBoardReadsService)
	deployBoardReadsEndpoints.Use(log.Endpoint)
	lastVisitedWorkspaceRepository, err := lastvisitedworkspacestore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	lastVisitedWorkspaceService, err := systemapi.NewLastVisitedWorkspace(lastVisitedWorkspaceRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	assembly.lastVisitedWorkspaceEndpoints = lastvisitedworkspaceapi.NewEndpoints(lastVisitedWorkspaceService)
	assembly.lastVisitedWorkspaceEndpoints.Use(log.Endpoint)
	assembly.routes.pageReads = pageReadsEndpoints
	assembly.routes.pageReadsMetadata = pageReadsService.(systemapi.SessionResourceProtocolAuthorizer)
	assembly.routes.intakeInboxReads = intakeInboxReadsEndpoints
	assembly.routes.deployBoardReads = deployBoardReadsEndpoints
	return nil
}
