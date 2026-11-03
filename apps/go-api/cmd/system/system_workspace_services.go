package main

import (
	systemapi "github.com/CaliLuke/go-plane"
	externalapi "github.com/CaliLuke/go-plane/externalintegrations/gen/external_integrations"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	externalstore "github.com/CaliLuke/go-plane/internal/externalintegrations"
	personalstore "github.com/CaliLuke/go-plane/internal/personaldashboard"
	analyticsstore "github.com/CaliLuke/go-plane/internal/workspaceanalytics"
	exportsstore "github.com/CaliLuke/go-plane/internal/workspaceexports"
	searchstore "github.com/CaliLuke/go-plane/internal/workspacesearch"
	insightsstore "github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	analyticsapi "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	exportsapi "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	personalapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	searchapi "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	insightsapi "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
	"github.com/CaliLuke/loom/clue/log"
)

func (assembly *systemAssembly) workspaceReports() error {
	var err error
	searchRepository, err := searchstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	searchService, err := systemapi.NewWorkspaceSearch(searchRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	searchEndpoints := searchapi.NewEndpoints(searchService)
	searchEndpoints.Use(log.Endpoint)
	analyticsRepository, err := analyticsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	analyticsService, err := systemapi.NewWorkspaceAnalytics(analyticsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	analyticsEndpoints := analyticsapi.NewEndpoints(analyticsService)
	analyticsEndpoints.Use(log.Endpoint)
	personalRepository, err := personalstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	personalService, err := systemapi.NewPersonalDashboard(personalRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	personalEndpoints := personalapi.NewEndpoints(personalService)
	personalEndpoints.Use(log.Endpoint)
	assembly.routes.personalDashboard = personalEndpoints
	assembly.routes.personalDashboardMetadata = personalService.(sessionResourceProtocolAuthorizer)
	assembly.routes.workspaceSearch = searchEndpoints
	assembly.routes.workspaceSearchMetadata = searchService.(sessionResourceProtocolAuthorizer)
	assembly.routes.workspaceAnalytics = analyticsEndpoints
	assembly.routes.workspaceAnalyticsMetadata = analyticsService.(sessionResourceProtocolAuthorizer)
	return nil
}

func (assembly *systemAssembly) exports() error {
	var err error
	exportsRepository, err := exportsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	exportsTaskPublisher := assembly.activityTaskPublisher
	exportsTaskPublisher, err = backgroundtasks.NewIssueExportTaskPublisher(exportsTaskPublisher, assembly.issueExportAdmission)
	if err != nil {
		return err
	}
	if assembly.analyticsExportDispatcher != nil {
		exportsTaskPublisher, err = backgroundtasks.NewAnalyticsExportTaskPublisher(exportsTaskPublisher, assembly.analyticsExportDispatcher)
		if err != nil {
			return err
		}
	}
	exportsService, err := systemapi.NewWorkspaceExports(exportsRepository, assembly.sessionValidator, exportsTaskPublisher)
	if err != nil {
		return err
	}
	exportsEndpoints := exportsapi.NewEndpoints(exportsService)
	exportsEndpoints.Use(log.Endpoint)
	assembly.routes.workspaceExports = exportsEndpoints
	assembly.routes.workspaceExportsMetadata = exportsService.(sessionResourceProtocolAuthorizer)
	return nil
}

func (assembly *systemAssembly) externalIntegrations() error {
	var err error
	insightsRepository, err := insightsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	insightsService, err := systemapi.NewWorkspaceUserInsights(insightsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	insightsEndpoints := insightsapi.NewEndpoints(insightsService)
	insightsEndpoints.Use(log.Endpoint)
	externalRepository, err := externalstore.NewRepository(assembly.pool, assembly.instanceCipher)
	if err != nil {
		return err
	}
	externalService, err := systemapi.NewExternalIntegrations(externalRepository, assembly.sessionValidator, nil)
	if err != nil {
		return err
	}
	externalEndpoints := externalapi.NewEndpoints(externalService)
	externalEndpoints.Use(log.Endpoint)

	assembly.routes.workspaceUserInsights = insightsEndpoints
	assembly.routes.workspaceUserInsightsMetadata = insightsService.(sessionResourceProtocolAuthorizer)
	assembly.routes.externalIntegrations = externalEndpoints
	assembly.routes.externalIntegrationsMetadata = externalService.(sessionResourceProtocolAuthorizer)
	return nil
}
