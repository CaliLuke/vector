package main

import (
	systemapi "github.com/CaliLuke/go-plane"
	apicyclereadsapi "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	apilabelreadsapi "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	apimodulereadsapi "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	apistatereadsapi "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	apiuseridentityapi "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	activityreadsstore "github.com/CaliLuke/go-plane/internal/activityreads"
	customviewcreationstore "github.com/CaliLuke/go-plane/internal/customviewcreation"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	modulereadsstore "github.com/CaliLuke/go-plane/internal/modulereads"
	notificationsstore "github.com/CaliLuke/go-plane/internal/notifications"
	projectconfigurationstore "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	intakewritesstore "github.com/CaliLuke/go-plane/internal/projectintakewrites"
	projectlabelreadsstore "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	preferencesstore "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	projectpublicationstore "github.com/CaliLuke/go-plane/internal/projectpublication"
	projectstatereadsstore "github.com/CaliLuke/go-plane/internal/projectstatereads"
	workitemhistorystore "github.com/CaliLuke/go-plane/internal/workitemhistory"
	workspacewebhooksstore "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	notificationactionsapi "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	notificationsapi "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	projectconfigurationapi "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	intakewritesapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	projectpublicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	workspacewebhooksapi "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	"github.com/CaliLuke/loom/clue/log"
)

func (assembly *systemAssembly) activityReads() error {
	var err error
	activityReadsUserRepository, err := activityreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	activityReadsAPIRepository, err := activityreadsstore.NewRepository(assembly.activityReadsPool)
	if err != nil {
		return err
	}
	assembly.activityReadsPrimaryAPIRepository, err = activityreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	assembly.activityReadsService, err = systemapi.NewActivityReads(activityReadsUserRepository, activityReadsAPIRepository, assembly.apiKeyValidator, assembly.sessionValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	activityReadsPrimaryService, err := systemapi.NewActivityReads(activityReadsUserRepository, assembly.activityReadsPrimaryAPIRepository, assembly.activityReadsPrimaryAPIKeyValidator, assembly.sessionValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiUserIdentityService, err := systemapi.NewAPIUserIdentity(assembly.userIdentityRepository, assembly.activityReadsPrimaryAPIKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiUserIdentityEndpoints := apiuseridentityapi.NewEndpoints(apiUserIdentityService)
	apiUserIdentityEndpoints.Use(log.Endpoint)

	activityReadsMetadata := activityReadsMethodAuthorizer{
		identity: apiUserIdentityService.(systemapi.APIIdentityProtocolAuthorizer),
		readOnly: assembly.activityReadsService.(systemapi.ActivityReadsProtocolAuthorizer),
		primary:  activityReadsPrimaryService.(systemapi.ActivityReadsProtocolAuthorizer),
	}
	assembly.routes.activityReadsMetadata = activityReadsMetadata
	assembly.routes.apiUserIdentity = apiUserIdentityEndpoints
	return nil
}

func (assembly *systemAssembly) apiPlanningReads() error {
	var err error
	apiCycleRepository, err := cyclereadsstore.NewRepository(assembly.activityReadsPool)
	if err != nil {
		return err
	}
	apiCycleService, err := systemapi.NewAPICycleReads(apiCycleRepository, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiCycleEndpoints := apicyclereadsapi.NewEndpoints(apiCycleService)
	apiCycleEndpoints.Use(log.Endpoint)
	apiModuleRepository, err := modulereadsstore.NewRepository(assembly.activityReadsPool)
	if err != nil {
		return err
	}
	apiModuleService, err := systemapi.NewAPIModuleReads(apiModuleRepository, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiModuleEndpoints := apimodulereadsapi.NewEndpoints(apiModuleService)
	apiModuleEndpoints.Use(log.Endpoint)
	apiLabelRepository, err := projectlabelreadsstore.NewRepository(assembly.activityReadsPool)
	if err != nil {
		return err
	}
	apiLabelService, err := systemapi.NewAPILabelReads(apiLabelRepository, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiLabelEndpoints := apilabelreadsapi.NewEndpoints(apiLabelService)
	apiLabelEndpoints.Use(log.Endpoint)
	assembly.routes.apiCycleReads = apiCycleEndpoints
	assembly.routes.apiModuleReads = apiModuleEndpoints
	assembly.routes.apiLabelReads = apiLabelEndpoints
	return nil
}

func (assembly *systemAssembly) projectConfiguration() error {
	var err error
	preferencesRepository, err := preferencesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	preferencesService, err := systemapi.NewProjectMemberPreferences(preferencesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	preferencesEndpoints := preferencesapi.NewEndpoints(preferencesService)
	preferencesEndpoints.Use(log.Endpoint)
	intakeWritesRepository, err := intakewritesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	intakeWritesService, err := systemapi.NewProjectIntakeWrites(intakeWritesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	intakeWritesEndpoints := intakewritesapi.NewEndpoints(intakeWritesService)
	intakeWritesEndpoints.Use(log.Endpoint)
	webhookRepository, err := workspacewebhooksstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	webhookService, err := systemapi.NewWorkspaceWebhooks(webhookRepository, assembly.sessionValidator, assembly.activityTaskPublisher)
	if err != nil {
		return err
	}
	webhookEndpoints := workspacewebhooksapi.NewEndpoints(webhookService)
	webhookEndpoints.Use(log.Endpoint)
	assembly.configurationRepository, err = projectconfigurationstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	configurationService, err := systemapi.NewProjectConfiguration(assembly.configurationRepository, assembly.sessionValidator, assembly.apiKeyValidator, assembly.apiKeyThrottle, assembly.activityTaskPublisher)
	if err != nil {
		return err
	}
	configurationEndpoints := projectconfigurationapi.NewEndpoints(configurationService)
	configurationEndpoints.Use(log.Endpoint)
	customViewCreationRepository, err := customviewcreationstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	customViewCreationService, err := systemapi.NewCustomViewCreation(customViewCreationRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	customViewCreationEndpoints := customviewcreationapi.NewEndpoints(customViewCreationService)
	customViewCreationEndpoints.Use(log.Endpoint)
	apiStateRepository, err := projectstatereadsstore.NewWriteRepository(assembly.pool)
	if err != nil {
		return err
	}
	apiStateService, err := systemapi.NewAPIStateReads(apiStateRepository, assembly.apiKeyValidator, assembly.apiKeyThrottle)
	if err != nil {
		return err
	}
	apiStateEndpoints := apistatereadsapi.NewEndpoints(apiStateService)
	apiStateEndpoints.Use(log.Endpoint)
	assembly.routes.projectConfiguration = configurationEndpoints
	assembly.routes.customViewCreation = customViewCreationEndpoints
	assembly.routes.customViewCreationMetadata = customViewCreationService.(systemapi.SessionResourceProtocolAuthorizer)
	assembly.routes.projectMemberPreferences = preferencesEndpoints
	assembly.routes.projectMemberPreferencesMetadata = preferencesService.(sessionResourceProtocolAuthorizer)
	assembly.routes.projectIntakeWrites = intakeWritesEndpoints
	assembly.routes.projectIntakeWritesMetadata = intakeWritesService.(systemapi.SessionResourceProtocolAuthorizer)
	assembly.routes.projectConfigurationMetadata = configurationService.(projectConfigurationProtocolAuthorizer)
	assembly.routes.workspaceWebhooks = webhookEndpoints
	assembly.routes.workspaceWebhooksMetadata = webhookService.(sessionResourceProtocolAuthorizer)
	assembly.routes.apiStateReads = apiStateEndpoints
	return nil
}

func (assembly *systemAssembly) notifications() error {
	var err error
	notificationRepository, err := notificationsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	notificationCounts, err := notificationsstore.NewRepository(assembly.activityReadsPool)
	if err != nil {
		return err
	}
	notificationsService, err := systemapi.NewNotificationReads(notificationRepository, notificationCounts, assembly.sessionValidator)
	if err != nil {
		return err
	}
	notificationsEndpoints := notificationsapi.NewEndpoints(notificationsService)
	notificationsEndpoints.Use(log.Endpoint)
	assembly.routes.notifications = notificationsEndpoints
	assembly.routes.notificationsMetadata = notificationsService.(systemapi.SessionProtocolAuthorizer)
	return nil
}

func (assembly *systemAssembly) notificationPublisher() error {
	publisher, err := assembly.commonTaskPublisher()
	if err != nil {
		return err
	}
	assembly.notificationTaskPublisher, err = assembly.withVersionTasks(publisher)
	return err
}

func (assembly *systemAssembly) historyAndNotificationActions() error {
	var err error
	projectPublicationRepository, err := projectpublicationstore.NewRepository(assembly.pool, assembly.notificationTaskPublisher)
	if err != nil {
		return err
	}
	projectPublicationService, err := systemapi.NewProjectPublication(projectPublicationRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	projectPublicationEndpoints := projectpublicationapi.NewEndpoints(projectPublicationService)
	projectPublicationEndpoints.Use(log.Endpoint)
	workItemHistoryRepository, err := workitemhistorystore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workItemHistoryService, err := systemapi.NewWorkItemHistory(workItemHistoryRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	workItemHistoryEndpoints := workitemhistoryapi.NewEndpoints(workItemHistoryService)
	workItemHistoryEndpoints.Use(log.Endpoint)
	notificationActionsRepository, err := notificationsstore.NewActionRepository(assembly.pool, assembly.notificationTaskPublisher)
	if err != nil {
		return err
	}
	notificationActionsService, err := systemapi.NewNotificationActions(notificationActionsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	notificationActionsEndpoints := notificationactionsapi.NewEndpoints(notificationActionsService)
	notificationActionsEndpoints.Use(log.Endpoint)
	assembly.routes.projectPublication = projectPublicationEndpoints
	assembly.routes.projectPublicationMetadata = projectPublicationService.(sessionResourceProtocolAuthorizer)
	assembly.routes.workItemHistory = workItemHistoryEndpoints
	assembly.routes.workItemHistoryMetadata = workItemHistoryService.(sessionResourceProtocolAuthorizer)
	assembly.routes.notificationActions = notificationActionsEndpoints
	return nil
}
