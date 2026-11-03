package main

import (
	"fmt"
	"net/url"
	"os"

	milestonestore "github.com/CaliLuke/go-plane/internal/milestones"
	milestoneapi "github.com/CaliLuke/go-plane/milestones/gen/milestones"

	systemapi "github.com/CaliLuke/go-plane"
	apiv1stickiesapi "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	assetslifecycleapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/assets_lifecycle"
	projectidentifiersapi "github.com/CaliLuke/go-plane/gen/project_identifiers"
	projectlabelreadsapi "github.com/CaliLuke/go-plane/gen/project_label_reads"
	projectmembershipreadsapi "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	projectstatereadsapi "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	useridentityapi "github.com/CaliLuke/go-plane/gen/user_identity"
	userpropertiesapi "github.com/CaliLuke/go-plane/gen/user_properties"
	workspacemetadataapi "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	assetsstore "github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	projectidentifiersstore "github.com/CaliLuke/go-plane/internal/projectidentifiers"
	projectlabelreadsstore "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	projectmembershipreadsstore "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	projectstatereadsstore "github.com/CaliLuke/go-plane/internal/projectstatereads"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/ratelimit"
	useridentitystore "github.com/CaliLuke/go-plane/internal/useridentity"
	userpropertiesstore "github.com/CaliLuke/go-plane/internal/userproperties"
	workspacemetadatastore "github.com/CaliLuke/go-plane/internal/workspacemetadata"
	"github.com/CaliLuke/loom/clue/log"
)

func (assembly *systemAssembly) identityReads() error {
	var err error
	assembly.userIdentityRepository, err = useridentitystore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userIdentityService, err := systemapi.NewUserIdentity(assembly.userIdentityRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	userIdentityEndpoints := useridentityapi.NewEndpoints(userIdentityService)
	userIdentityEndpoints.Use(log.Endpoint)
	userPropertiesRepository, err := userpropertiesstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	userPropertiesService, err := systemapi.NewUserProperties(userPropertiesRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	userPropertiesEndpoints := userpropertiesapi.NewEndpoints(userPropertiesService)
	userPropertiesEndpoints.Use(log.Endpoint)
	workspaceMetadataRepository, err := workspacemetadatastore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	workspaceMetadataService, err := systemapi.NewWorkspaceMetadata(workspaceMetadataRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	workspaceMetadataEndpoints := workspacemetadataapi.NewEndpoints(workspaceMetadataService)
	workspaceMetadataEndpoints.Use(log.Endpoint)
	projectIdentifiersRepository, err := projectidentifiersstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	projectIdentifiersService, err := systemapi.NewProjectIdentifiers(projectIdentifiersRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	projectIdentifiersEndpoints := projectidentifiersapi.NewEndpoints(projectIdentifiersService)
	projectIdentifiersEndpoints.Use(log.Endpoint)
	assembly.routes.userIdentity = userIdentityEndpoints
	assembly.routes.userProperties = userPropertiesEndpoints
	assembly.routes.workspaceMetadata = workspaceMetadataEndpoints
	assembly.routes.projectIdentifiers = projectIdentifiersEndpoints
	return nil
}

func (assembly *systemAssembly) projectReads() error {
	var err error
	projectMembershipReadsRepository, err := projectmembershipreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	projectMembershipReadsService, err := systemapi.NewProjectMembershipReads(projectMembershipReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	projectMembershipReadsEndpoints := projectmembershipreadsapi.NewEndpoints(projectMembershipReadsService)
	projectMembershipReadsEndpoints.Use(log.Endpoint)
	projectStateWriteCache, err := projectstatereadsstore.NewStateWriteCache(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "project state write cache", projectStateWriteCache) })
	assembly.projectStateReadsRepository, err = projectstatereadsstore.NewWriteRepository(assembly.pool)
	if err != nil {
		return err
	}
	assembly.projectStateReadsRepository.SetCache(projectStateWriteCache)
	projectStateReadsService, err := systemapi.NewProjectStateReads(assembly.projectStateReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	projectStateReadsEndpoints := projectstatereadsapi.NewEndpoints(projectStateReadsService)
	projectStateReadsEndpoints.Use(log.Endpoint)
	assembly.projectLabelReadsRepository, err = projectlabelreadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	projectLabelReadsService, err := systemapi.NewProjectLabelReads(assembly.projectLabelReadsRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	projectLabelReadsEndpoints := projectlabelreadsapi.NewEndpoints(projectLabelReadsService)
	projectLabelReadsEndpoints.Use(log.Endpoint)
	assembly.routes.projectMembershipReads = projectMembershipReadsEndpoints
	assembly.routes.projectMembershipMetadata = projectMembershipReadsService.(systemapi.ProjectMembershipReadsMetadataAuthorizer)
	assembly.routes.projectStateReads = projectStateReadsEndpoints
	assembly.routes.projectStateMetadata = projectStateReadsService.(systemapi.ProjectStateReadsMetadataAuthorizer)
	assembly.routes.projectLabelReads = projectLabelReadsEndpoints
	return nil
}

func (assembly *systemAssembly) publicMetadata() error {
	var err error
	assembly.publicProjectMetadataReadsRepository, err = publicprojectmetadatareadsstore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsService, err := systemapi.NewPublicProjectMetadataReads(assembly.publicProjectMetadataReadsRepository)
	if err != nil {
		return err
	}
	publicProjectMetadataReadsEndpoints := publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsService)
	publicProjectMetadataReadsEndpoints.Use(log.Endpoint)
	assembly.publicProjectMetadataReadsCache, err = ratelimit.NewRedisStore(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) {
		closeResource(result, "public project metadata throttle cache", assembly.publicProjectMetadataReadsCache)
	})
	assembly.apiKeyThrottle, err = apikey.NewThrottle(assembly.publicProjectMetadataReadsCache, assembly.cfg.apiKeyRate)
	if err != nil {
		return fmt.Errorf("configure API-key throttle: %w", err)
	}
	assembly.apiKeyValidator, err = apikey.NewWithReaderWriter(assembly.activityReadsPool, assembly.pool)
	if err != nil {
		return err
	}
	assembly.routes.publicProjectMetadataReads = publicProjectMetadataReadsEndpoints
	return nil
}

func (assembly *systemAssembly) apiAuthentication() error {
	var err error
	apiV1StickiesService, err := systemapi.NewAPIV1Stickies(assembly.repository, assembly.apiKeyValidator, assembly.apiKeyThrottle, assembly.activityTaskPublisher)
	if err != nil {
		return err
	}
	apiV1StickiesEndpoints := apiv1stickiesapi.NewEndpoints(apiV1StickiesService)
	apiV1StickiesEndpoints.Use(log.Endpoint)
	assembly.activityReadsPrimaryAPIKeyValidator, err = apikey.NewWithReaderWriter(assembly.pool, assembly.pool)
	if err != nil {
		return err
	}
	assembly.milestoneRepository, err = milestonestore.NewRepository(assembly.pool)
	if err != nil {
		return err
	}
	milestoneService, err := systemapi.NewMilestones(assembly.milestoneRepository, assembly.activityReadsPrimaryAPIKeyValidator, assembly.apiKeyThrottle, os.Getenv("WEB_URL"))
	if err != nil {
		return err
	}
	milestoneEndpoints := milestoneapi.NewEndpoints(milestoneService)
	milestoneEndpoints.Use(log.Endpoint)
	assembly.routes.milestones = milestoneEndpoints
	assembly.routes.apiV1Stickies = apiV1StickiesEndpoints
	assembly.routes.apiV1StickiesProtocol = apiV1StickiesService.(systemapi.APIV1StickyProtocolAuthorizer)
	return nil
}

func (assembly *systemAssembly) mcpAuthentication() error {
	var err error
	assembly.mcpOptions = mcptransport.Options{Observer: mcptransport.LogObserver(assembly.logCtx)}
	if assembly.cfg.mcp.Enabled {
		memberships, membershipErr := mcpauth.NewWorkspaceReader(assembly.pool)
		if membershipErr != nil {
			return membershipErr
		}
		headerAuth, authErr := mcpauth.NewPersonalTokens(assembly.activityReadsPrimaryAPIKeyValidator, memberships, assembly.apiKeyThrottle)
		if authErr != nil {
			return authErr
		}
		assembly.mcpOptions.HeaderAuth = headerAuth.Authenticate
		if assembly.mcpOAuthProvider != nil {
			assembly.mcpLinks, err = mcpoauth.NewLinks(assembly.pool, assembly.cfg.mcpOAuth.Issuer)
			if err != nil {
				return err
			}
			oauthAuth, oauthErr := mcpauth.NewOAuth(assembly.mcpOAuthProvider, assembly.mcpLinks, memberships, assembly.apiKeyThrottle)
			if oauthErr != nil {
				return oauthErr
			}
			assembly.mcpOptions.OAuthAuth, assembly.mcpOptions.OAuthIssuer = oauthAuth.Authenticate, assembly.cfg.mcpOAuth.Issuer
		}
	}

	return nil
}

func (assembly *systemAssembly) assets() error {
	var err error
	var assetsLifecycleEndpoints *assetslifecycleapi.Endpoints
	var assetsLifecycleMetadata interface {
		systemapi.SessionResourceProtocolAuthorizer
		systemapi.AssetsAPIProtocolAuthorizer
	}
	assetDuplicateThrottle, err := apikey.NewThrottle(assembly.publicProjectMetadataReadsCache, "5/minute")
	if err != nil {
		return fmt.Errorf("configure duplicate asset throttle: %w", err)
	}
	assembly.assetRepository, err = assetsstore.NewPostgresRepository(assembly.pool)
	if err != nil {
		return err
	}
	storageConfig, err := assetsstore.StorageConfigFromEnv()
	if err != nil {
		return err
	}
	assembly.assetStorage, err = assetsstore.NewStorage(storageConfig)
	if err != nil {
		return err
	}
	assetCache, err := assetsstore.NewRedisCacheInvalidator(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "asset cache", assetCache) })
	assembly.assetAppBaseURL = os.Getenv("APP_BASE_URL")
	assetAppURL, assetAppURLErr := url.Parse(assembly.assetAppBaseURL)
	if assetAppURLErr != nil || assetAppURL.Scheme == "" || assetAppURL.Host == "" {
		assembly.assetAppBaseURL = os.Getenv("WEB_URL")
	}
	assetService, err := assetsstore.NewService(assembly.assetRepository, assembly.assetStorage, assembly.apiKeyValidator, assembly.apiKeyThrottle, assembly.cfg.fileSizeLimit, assetsstore.AssetEffects{
		Tasks:      assembly.activityTaskPublisher,
		Cache:      assetCache,
		AppBaseURL: assembly.assetAppBaseURL,
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
	assembly.routes.assetsLifecycle = assetsLifecycleEndpoints
	assembly.routes.assetsSessionMetadata = assetsLifecycleMetadata
	assembly.routes.assetsAPIMetadata = assetsLifecycleMetadata
	assembly.routes.assetsDuplicateThrottle = assetDuplicateThrottle
	return nil
}
