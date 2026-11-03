package main

import (
	"net/url"
	"os"

	workitemtypesstore "github.com/CaliLuke/go-plane/internal/workitemtypes"
	workitemtypesapi "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"

	systemapi "github.com/CaliLuke/go-plane"
	activityreadsapi "github.com/CaliLuke/go-plane/activityreads/gen/activity_reads"
	cyclelifecycleapi "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	publicfeedbackapi "github.com/CaliLuke/go-plane/gen/public_feedback"
	standardviewsapi "github.com/CaliLuke/go-plane/gen/standard_views"
	usersettingsapi "github.com/CaliLuke/go-plane/gen/user_settings"
	workspacesidebarpreferencesapi "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	projectlabelreadsstore "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	projectmembershipreadsstore "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	standardviewsstore "github.com/CaliLuke/go-plane/internal/standardviews"
	usersettingsstore "github.com/CaliLuke/go-plane/internal/usersettings"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	workspaceprojectreadsstore "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	workspacesidebarpreferencesstore "github.com/CaliLuke/go-plane/internal/workspacesidebarpreferences"
	invitationsapi "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	modulelifecycleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	labelwritesapi "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	projectlifecycleapi "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	membershiplifecycleapi "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	collaborationapi "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	"github.com/CaliLuke/loom/clue/log"
)

func (assembly *systemAssembly) projectLifecycle() error {
	var err error
	assembly.labelWriteCache, err = projectlabelreadsstore.NewWriteCache(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "label write cache", assembly.labelWriteCache) })
	assembly.membershipBaseHost = os.Getenv("APP_BASE_URL")
	membershipAppURL, membershipAppURLErr := url.Parse(assembly.membershipBaseHost)
	if membershipAppURLErr != nil || membershipAppURL.Scheme == "" || membershipAppURL.Host == "" {
		assembly.membershipBaseHost = os.Getenv("WEB_URL")
	}
	assembly.cycleLifecycleRepository, err = cyclereadsstore.NewLifecycleRepository(assembly.pool, assembly.notificationTaskPublisher, assembly.membershipBaseHost)
	if err != nil {
		return err
	}
	cycleLifecycleService, err := systemapi.NewCycleLifecycle(assembly.cycleLifecycleRepository, assembly.sessionValidator, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	cycleLifecycleEndpoints := cyclelifecycleapi.NewEndpoints(cycleLifecycleService)
	cycleLifecycleEndpoints.Use(log.Endpoint)
	assembly.projectLifecycleRepository, err = workspaceprojectreadsstore.NewLifecycleRepository(assembly.pool, assembly.notificationTaskPublisher, assembly.membershipBaseHost, projectLifecycleCoverValidator(assembly.assetRepository))
	if err != nil {
		return err
	}
	projectLifecycleService, err := systemapi.NewProjectLifecycle(assembly.projectLifecycleRepository, assembly.sessionValidator, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	projectLifecycleEndpoints := projectlifecycleapi.NewEndpoints(projectLifecycleService)
	projectLifecycleEndpoints.Use(log.Endpoint)
	assembly.moduleLifecycleRepository, err = modulereadsstore.NewLifecycleRepository(assembly.pool, assembly.notificationTaskPublisher, assembly.membershipBaseHost)
	if err != nil {
		return err
	}
	moduleLifecycleService, err := systemapi.NewModuleLifecycle(assembly.moduleLifecycleRepository, assembly.sessionValidator, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	moduleLifecycleEndpoints := modulelifecycleapi.NewEndpoints(moduleLifecycleService)
	moduleLifecycleEndpoints.Use(log.Endpoint)
	assembly.routes.projectLifecycle = projectLifecycleEndpoints
	assembly.routes.projectLifecycleMetadata = projectLifecycleService
	assembly.routes.cycleLifecycle = cycleLifecycleEndpoints
	assembly.routes.cycleLifecycleMetadata = cycleLifecycleService
	assembly.routes.moduleLifecycle = moduleLifecycleEndpoints
	assembly.routes.moduleLifecycleMetadata = moduleLifecycleService
	return nil
}

func (assembly *systemAssembly) membership() error {
	var err error
	assembly.membershipRepository, err = projectmembershipreadsstore.NewLifecycleRepositoryWithReaderWriter(assembly.activityReadsPool, assembly.pool, assembly.membershipBaseHost)
	if err != nil {
		return err
	}
	membershipWriteKeys, err := apikey.New(assembly.pool)
	if err != nil {
		return err
	}
	membershipKeys := membershipAPIKeyRouter{reader: assembly.apiKeyValidator, writer: membershipWriteKeys}
	membershipService, err := systemapi.NewProjectMembershipLifecycle(assembly.membershipRepository, membershipKeys, assembly.apiKeyThrottle, assembly.sessionValidator)
	if err != nil {
		return err
	}
	membershipEndpoints := membershiplifecycleapi.NewEndpoints(membershipService)
	membershipEndpoints.Use(log.Endpoint)
	assembly.invitationsRepository, err = projectmembershipreadsstore.NewInvitationsRepositoryWithReaderWriter(assembly.activityReadsPool, assembly.pool, assembly.notificationTaskPublisher, assembly.membershipBaseHost, assembly.cfg.secretKey)
	if err != nil {
		return err
	}
	assembly.invitationsCache, err = projectmembershipreadsstore.NewInvitationCache(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "invitation cache", assembly.invitationsCache) })
	invitationsService, err := systemapi.NewInvitationsMembership(assembly.invitationsRepository, membershipKeys, assembly.sessionValidator, assembly.apiKeyThrottle, assembly.invitationsCache)
	if err != nil {
		return err
	}
	invitationsEndpoints := invitationsapi.NewEndpoints(invitationsService)
	invitationsEndpoints.Use(log.Endpoint)
	assembly.routes.invitationsMembership = invitationsEndpoints
	assembly.routes.invitationsMetadata = invitationsService.(systemapi.InvitationsMembershipProtocolAuthorizer)
	assembly.routes.membershipLifecycle = membershipEndpoints
	assembly.routes.membershipLifecycleMetadata = membershipService.(systemapi.MembershipLifecycleProtocolAuthorizer)
	return nil
}

func (assembly *systemAssembly) workItems() error {
	var err error
	labelWriteRepository, err := projectlabelreadsstore.NewWriteRepository(assembly.pool)
	if err != nil {
		return err
	}
	labelWriteService, err := systemapi.NewProjectLabelWrites(labelWriteRepository, assembly.sessionValidator, assembly.labelWriteCache)
	if err != nil {
		return err
	}
	labelWriteEndpoints := labelwritesapi.NewEndpoints(labelWriteService)
	labelWriteEndpoints.Use(log.Endpoint)
	assembly.workItemRepository, err = workitemstore.NewRepositoryWithNativeTasks(assembly.pool, assembly.notificationTaskPublisher, os.Getenv("WEB_URL"), assembly.issueDescriptionVersionAdmission, assembly.recentVisitAdmission, assembly.issueActivityAdmission)
	if err != nil {
		return err
	}
	assembly.typeRepository = workitemtypesstore.NewRepository(assembly.pool)
	typeService, err := systemapi.NewWorkItemTypes(assembly.typeRepository, assembly.sessionValidator, assembly.workItemRepository)
	if err != nil {
		return err
	}
	typeEndpoints := workitemtypesapi.NewEndpoints(typeService)
	typeEndpoints.Use(log.Endpoint)
	publicFeedbackService, err := systemapi.NewPublicFeedback(assembly.publicProjectMetadataReadsRepository, assembly.workItemRepository)
	if err != nil {
		return err
	}
	publicFeedbackEndpoints := publicfeedbackapi.NewEndpoints(publicFeedbackService)
	publicFeedbackEndpoints.Use(log.Endpoint)
	workItemService, err := systemapi.NewWorkItemCore(assembly.workItemRepository, assembly.sessionValidator, assembly.activityReadsPrimaryAPIKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	workItemEndpoints := workitemapi.NewEndpoints(workItemService)
	workItemEndpoints.Use(log.Endpoint)
	collaborationService, err := systemapi.NewTicketCollaboration(assembly.workItemRepository, assembly.sessionValidator, assembly.activityReadsPrimaryAPIKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	collaborationEndpoints := collaborationapi.NewEndpoints(collaborationService)
	collaborationEndpoints.Use(log.Endpoint)

	assembly.routes.workItemTypes = typeEndpoints
	assembly.routes.projectLabelWrites = labelWriteEndpoints
	assembly.routes.workItemCore = workItemEndpoints
	assembly.routes.ticketCollaboration = collaborationEndpoints
	assembly.routes.workItemProtocol = collaborationProtocolAuthorizer{core: workItemService.(systemapi.WorkItemProtocolAuthorizer), collaboration: collaborationService.(systemapi.WorkItemProtocolAuthorizer)}
	assembly.routes.projectLabelWritesMetadata = labelWriteService.(systemapi.SessionResourceProtocolAuthorizer)
	assembly.routes.publicFeedback = publicFeedbackEndpoints
	return nil
}

func (assembly *systemAssembly) browserMetadata() error {
	var err error
	activityReadsEndpoints := activityreadsapi.NewEndpoints(assembly.activityReadsService)
	activityReadsEndpoints.Use(log.Endpoint)
	publicProjectMetadataSessionInvalidator, err := publicprojectmetadatareadsstore.NewPostgreSQLSessionInvalidator(assembly.pool)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsThrottle, err := publicprojectmetadatareadsstore.NewAnonymousThrottle(
		assembly.publicProjectMetadataReadsCache, assembly.sessionValidator, publicProjectMetadataSessionInvalidator,
		assembly.cfg.sessionCookie, assembly.cfg.saveEveryRequest, assembly.cfg.sessionCookieAge,
		publicprojectmetadatareadsstore.AdminSessionConfig{CookieName: assembly.cfg.adminSessionCookie, CookieAge: assembly.cfg.adminSessionCookieAge},
	)
	if err != nil {
		return err
	}
	standardViewsRepository, err := standardviewsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	standardViewsService, err := systemapi.NewStandardViews(standardViewsRepository, assembly.sessionValidator, assembly.recentVisitAdmission)
	if err != nil {
		return err
	}
	standardViewsEndpoints := standardviewsapi.NewEndpoints(standardViewsService)
	standardViewsEndpoints.Use(log.Endpoint)
	userSettingsRepository, err := usersettingsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userSettingsService, err := systemapi.NewUserSettings(userSettingsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	userSettingsEndpoints := usersettingsapi.NewEndpoints(userSettingsService)
	userSettingsEndpoints.Use(log.Endpoint)
	workspaceSidebarPreferencesRepository, err := workspacesidebarpreferencesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesBackend, err := workspacesidebarpreferencesstore.NewService(workspaceSidebarPreferencesRepository)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesService, err := systemapi.NewWorkspaceSidebarPreferences(workspaceSidebarPreferencesBackend, assembly.sessionValidator)
	if err != nil {
		return err
	}
	workspaceSidebarPreferencesEndpoints := workspacesidebarpreferencesapi.NewEndpoints(workspaceSidebarPreferencesService)
	workspaceSidebarPreferencesEndpoints.Use(log.Endpoint)

	assembly.routes.readServicesAuthSession = publicProjectMetadataReadsThrottle
	assembly.routes.instanceManagementThrottle = publicProjectMetadataReadsThrottle
	assembly.routes.csrfTokenThrottle = publicProjectMetadataReadsThrottle
	assembly.routes.userIdentitySession = publicProjectMetadataReadsThrottle
	assembly.routes.publicProjectMetadataThrottle = publicProjectMetadataReadsThrottle
	assembly.routes.standardViews = standardViewsEndpoints
	assembly.routes.userSettings = userSettingsEndpoints
	assembly.routes.workspaceSidebarPreferences = workspaceSidebarPreferencesEndpoints
	assembly.routes.readServicesSession = publicProjectMetadataReadsThrottle
	assembly.routes.activityReads = activityReadsEndpoints
	return nil
}
