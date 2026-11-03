package main

import (
	"os"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/features"
	"github.com/CaliLuke/go-plane/internal/initiatives"
	"github.com/CaliLuke/go-plane/internal/releases"
	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/workitemrelations"
	"github.com/CaliLuke/go-plane/internal/worklogs"

	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	cyclereadsstore "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	projectmembershipreadsstore "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	useridentitystore "github.com/CaliLuke/go-plane/internal/useridentity"
)

func (assembly *systemAssembly) serve() error {
	var err error
	assembly.routes.mcpResourceURL = assembly.cfg.mcp.ResourceURL
	assembly.routes.readServicesAuthCookie = browserSessionCookieConfig{Name: assembly.cfg.sessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.csrfTokenCORS = assembly.cfg.corsOrigins
	assembly.routes.accountAuthenticationCSRFTrustedOrigins = assembly.cfg.csrfTrustedOrigins
	assembly.routes.csrfTokenCookie = nativeCSRFCookieConfig{Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure}
	assembly.routes.csrfTokenSessionCookie = browserSessionCookieConfig{Name: assembly.cfg.sessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.csrfTokenBodyLimit = assembly.cfg.fileSizeLimit
	assembly.routes.csrfTokenBodySet = true
	assembly.routes.userIdentityCORS = assembly.cfg.corsOrigins
	assembly.routes.userIdentityCookie = browserSessionCookieConfig{Name: assembly.cfg.sessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.userIdentityBodyLimit = assembly.cfg.fileSizeLimit
	assembly.routes.userIdentityBodySet = true
	assembly.routes.userIdentitySecret = assembly.cfg.secretKey
	assembly.routes.publicProjectMetadataCORS = assembly.cfg.corsOrigins
	assembly.routes.publicProjectMetadataCookie = browserSessionCookieConfig{Name: assembly.cfg.sessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.publicProjectMetadataAdminCookie = browserSessionCookieConfig{Name: assembly.cfg.adminSessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.publicProjectMetadataSecret = assembly.cfg.secretKey
	assembly.routes.publicProjectMetadataBodyLimit = assembly.cfg.fileSizeLimit
	assembly.routes.publicProjectMetadataBodySet = true
	assembly.routes.readServicesCORS = assembly.cfg.corsOrigins
	assembly.routes.readServicesCookie = browserSessionCookieConfig{Name: assembly.cfg.sessionCookie, Domain: assembly.cfg.cookieDomain, Secure: assembly.cfg.sessionCookieSecure, Configured: true}
	assembly.routes.activityReadsSecret = assembly.cfg.secretKey
	assembly.routes.activityReadsBodyLimit = assembly.cfg.fileSizeLimit
	handler := newHTTPHandlerWithPendingReplacements(assembly.logCtx, assembly.systemEndpoints, assembly.cfg.debug, assembly.stickyEndpoints, assembly.stickyService.(sticky.Authorizer).APIKeyAuth, assembly.workspaceSlugEndpoints, assembly.instanceWorkspaceSlugEndpoints, assembly.userInstanceAdminEndpoints, assembly.userProfileFlagsEndpoints, assembly.userNotificationPreferencesEndpoints, assembly.userAccountsEndpoints, assembly.userAPITokensEndpoints, assembly.instanceAdminSessionEndpoints, assembly.workspaceQuickLinksEndpoints, assembly.workspaceThemesEndpoints, assembly.workspaceHomePreferencesEndpoints, assembly.workspaceMemberMeEndpoints, assembly.analyticViewsEndpoints, assembly.workspaceRecentVisitsEndpoints, assembly.lastVisitedWorkspaceEndpoints, assembly.entityFavoritesEndpoints, assembly.routes, assembly.workspaceFavoritesEndpoints)
	mcpIdentity, err := useridentitystore.NewIdentityReader(assembly.userIdentityRepository)
	if err != nil {
		return err
	}
	mcpMembers, err := projectmembershipreadsstore.NewMemberReader(assembly.membershipRepository)
	if err != nil {
		return err
	}
	mcpDirectory, err := projectmembershipreadsstore.NewDirectoryReader(assembly.pool)
	if err != nil {
		return err
	}
	mcpCycleReads, err := cyclereadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	mcpAttachments, err := assetsstore.NewAttachmentApplication(assembly.assetRepository, assembly.assetStorage, backgroundtasks.FetchPublicFile, os.Getenv("WEB_URL"), assembly.cfg.fileSizeLimit)
	if err != nil {
		return err
	}
	assembly.mcpOptions.Service, err = mcpactions.New(mcpactions.Applications{Identity: mcpIdentity, ProjectMembers: mcpMembers, MemberDirectory: mcpDirectory, Projects: assembly.workspaceProjectReadsRepository, ProjectWrites: assembly.projectLifecycleRepository, Cycles: mcpCycleReads, CycleWrites: assembly.cycleLifecycleRepository, Modules: assembly.moduleReadsRepository, ModuleWrites: assembly.moduleLifecycleRepository, Labels: assembly.projectLabelReadsRepository, LabelWrites: assembly.configurationRepository, Collaboration: assembly.workItemRepository, Activities: assembly.activityReadsPrimaryAPIRepository, Milestones: assembly.milestoneRepository, States: assembly.projectStateReadsRepository, Estimates: assembly.configurationRepository, Types: assembly.typeRepository, Attachments: mcpAttachments, WorkLogs: worklogs.NewRepository(assembly.pool), Pages: pagereadsstore.NewPageAPIRepository(assembly.pool), Customers: customers.NewRepository(assembly.pool), Templates: templates.NewRepository(assembly.pool), Features: features.NewRepository(assembly.pool), Initiatives: initiatives.NewRepository(assembly.pool), Releases: releases.NewRepository(assembly.pool), Relations: workitemrelations.NewRepository(assembly.pool), Properties: assembly.typeRepository}, mcpactions.WithLogContext(assembly.logCtx), mcpactions.WithOrigin(os.Getenv("WEB_URL")))
	if err != nil {
		return err
	}
	handler, err = mcptransport.New(assembly.cfg.mcp, assembly.mcpOptions, handler)
	if err != nil {
		return err
	}
	handler = mcptransport.RedactRejectedQueries(assembly.wrapHTTP(handler))
	serveErr := listenAndServeHTTP(assembly.logCtx, assembly.cfg.httpAddress, handler, assembly.cfg.shutdownTimeout)
	return serveErr
}
