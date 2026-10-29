package main

import (
	"context"
	"net/http"
	"time"

	milestoneapi "github.com/CaliLuke/go-plane/milestones/gen/milestones"

	workitemtypesapi "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"

	systemapi "github.com/CaliLuke/go-plane"
	accountauthapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	activityreads "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	apicyclereads "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	apilabelreads "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	apimodulereads "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	apistatereads "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	apiuseridentity "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	apiv1stickies "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	assetslifecycle "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	authredirectsapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	cyclelifecycle "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	cyclereads "github.com/CaliLuke/go-plane/gen/cycle_reads"
	deployboardreads "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	estimatereads "github.com/CaliLuke/go-plane/gen/estimate_reads"
	instancemanagementapi "github.com/CaliLuke/go-plane/gen/instance_management"
	intakeinboxreads "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	modulereads "github.com/CaliLuke/go-plane/gen/module_reads"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	projectidentifiers "github.com/CaliLuke/go-plane/gen/project_identifiers"
	projectlabelreads "github.com/CaliLuke/go-plane/gen/project_label_reads"
	projectmembershipreads "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	projectstatereads "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicfeedback "github.com/CaliLuke/go-plane/gen/public_feedback"
	publicprojectmetadatareads "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	standardviews "github.com/CaliLuke/go-plane/gen/standard_views"
	useridentity "github.com/CaliLuke/go-plane/gen/user_identity"
	userproperties "github.com/CaliLuke/go-plane/gen/user_properties"
	usersettings "github.com/CaliLuke/go-plane/gen/user_settings"
	workspacemetadata "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	workspaceprojectreads "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	workspacesidebarpreferences "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	invitationsapi "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	mcpoauthlinksapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	modulelifecycle "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	notificationactions "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	notifications "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	oauthredirectsapi "github.com/CaliLuke/go-plane/oauthredirects/gen/oauth_redirects"
	labelwrites "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	projectconfigurationapi "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	intakewritesapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	projectlifecycleapi "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	preferencesapi "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	membershiplifecycle "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	projectpublicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	collaborationapi "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	useraccountwritesapi "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	workspacewebhooksapi "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	analyticsapi "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	exportsapi "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	workspaceLifecycleAPI "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	personalapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	workspaceuserinsightsapi "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
)

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
