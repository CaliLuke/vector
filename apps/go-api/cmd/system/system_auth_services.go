package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	accountauthenticationapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	authredirectsapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	instancemanagementapi "github.com/CaliLuke/go-plane/gen/instance_management"
	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/authworkflow"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authentication/passwordstrength"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	instancemanagementstore "github.com/CaliLuke/go-plane/internal/instancemanagement"
	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/ratelimit"
	useraccountwritesstore "github.com/CaliLuke/go-plane/internal/useraccountwrites"
	mcpoauthlinksapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	oauthredirectsapi "github.com/CaliLuke/go-plane/oauthredirects/gen/oauth_redirects"
	useraccountwritesapi "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
	workspacelifecycleapi "github.com/CaliLuke/go-plane/workspacelifecycle/gen/workspace_lifecycle"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/redis/go-redis/v9"
)

func (assembly *systemAssembly) accountAuthentication() error {
	var err error
	assembly.instanceCipher, err = instancecipher.New(assembly.cfg.secretKey)
	if err != nil {
		return err
	}
	if assembly.mcpLinks != nil {
		linking, linkErr := mcpoauth.NewLinking(assembly.mcpLinks, assembly.mcpOAuthProvider, assembly.instanceCipher)
		if linkErr != nil {
			return linkErr
		}
		linkService, linkErr := systemapi.NewMCPOAuthLinks(linking, assembly.sessionValidator, assembly.cfg.mcpOAuth.Issuer, assembly.cfg.mcp.ResourceURL)
		if linkErr != nil {
			return linkErr
		}
		assembly.mcpLinkEndpoints = mcpoauthlinksapi.NewEndpoints(linkService)
		assembly.mcpLinkEndpoints.Use(log.Endpoint)
		stopCleanup := startMCPLinkCleanup(assembly.logCtx, assembly.mcpLinks)
		assembly.onClose(func(_ *error) { stopCleanup() })
	}
	assembly.useDatabaseConfiguration = os.Getenv("SKIP_ENV_VAR") == "1" || strings.EqualFold(os.Getenv("SKIP_ENV_VAR"), "true")
	assembly.accountRepository, err = accountauthentication.NewPoolRepository(assembly.pool, assembly.instanceCipher, assembly.useDatabaseConfiguration)
	if err != nil {
		return err
	}
	assembly.postAuthWorkflow, err = authworkflow.NewPostUserAuthWorkflow(assembly.invitationsRepository)
	if err != nil {
		return err
	}
	assembly.passwordHasher = password.Default()
	assembly.authRedirectService, err = authredirects.NewService(assembly.accountRepository, assembly.passwordHasher, passwordstrength.Scorer{}, assembly.sessionValidator, assembly.postAuthWorkflow)
	if err != nil {
		return err
	}
	redisOptions, err := redis.ParseURL(assembly.cfg.redisURL)
	if err != nil {
		return fmt.Errorf("configure authentication Redis client: %w", err)
	}
	authRedis := redis.NewClient(redisOptions)
	assembly.onClose(func(result *error) { closeResource(result, "authentication Redis client", authRedis) })
	magicCodeVerifier, err := authredirects.NewRedisMagicCodeVerifier(authRedis)
	if err != nil {
		return err
	}
	var authThrottleCache *ratelimit.RedisStore
	assembly.authThrottle, authThrottleCache, err = authredirects.NewAuthenticationThrottle(assembly.cfg.redisURL, "")
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "authentication throttle cache", authThrottleCache) })
	if setMagicCodeDependenciesErr := assembly.authRedirectService.SetMagicCodeDependencies(magicCodeVerifier, assembly.authThrottle); setMagicCodeDependenciesErr != nil {
		return setMagicCodeDependenciesErr
	}
	assembly.magicIssuer, err = newRedisMagicCodeIssuer(authRedis, func(ctx context.Context, email string) (bool, error) {
		_, userByEmailErr := assembly.accountRepository.UserByEmail(ctx, email)
		if errors.Is(userByEmailErr, accountauthentication.ErrUserNotFound) {
			return false, nil
		}
		return userByEmailErr == nil, userByEmailErr
	})
	if err != nil {
		return err
	}
	assembly.resetTokenIssuer, err = newPasswordResetTokenIssuer(assembly.cfg.secretKey)
	if err != nil {
		return err
	}
	if setPasswordErr := assembly.authRedirectService.SetPasswordResetTokenVerifier(assembly.resetTokenIssuer); setPasswordErr != nil {
		return setPasswordErr
	}
	assembly.routes.mcpLinks = assembly.mcpLinkEndpoints
	return nil
}

func (assembly *systemAssembly) accountEndpoints() error {
	var err error
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
	accountAuthenticationService, err := systemapi.NewAccountAuthentication(assembly.accountRepository, assembly.authThrottle, assembly.invitationsCache, assembly.activityTaskPublisher, assembly.magicIssuer, assembly.resetTokenIssuer, assembly.passwordHasher, assembly.sessionValidator, accountBaseURL, spaceBaseURL)
	if err != nil {
		return err
	}
	accountAuthenticationEndpoints := accountauthenticationapi.NewEndpoints(accountAuthenticationService)
	accountAuthenticationEndpoints.Use(log.Endpoint)
	authRedirectAdapter, err := systemapi.NewAuthRedirects(assembly.authRedirectService)
	if err != nil {
		return err
	}
	authRedirectEndpoints := authredirectsapi.NewEndpoints(authRedirectAdapter)
	authRedirectEndpoints.Use(log.Endpoint)
	oauthRepository, err := systemapi.NewOAuthRedirectPostgresRepository(assembly.pool, assembly.instanceCipher, assembly.useDatabaseConfiguration)
	if err != nil {
		return err
	}
	oauthRedirectService, err := systemapi.NewOAuthRedirects(oauthRepository, assembly.sessionValidator, assembly.postAuthWorkflow, &http.Client{Timeout: 15 * time.Second})
	if err != nil {
		return err
	}
	oauthRedirectEndpoints := oauthredirectsapi.NewEndpoints(oauthRedirectService)
	oauthRedirectEndpoints.Use(log.Endpoint)
	assembly.routes.accountAuthentication = accountAuthenticationEndpoints
	assembly.routes.accountAuthenticationMetadata = accountAuthenticationService.(systemapi.SessionProtocolAuthorizer)
	assembly.routes.authRedirects = authRedirectEndpoints
	assembly.routes.oauthRedirects = oauthRedirectEndpoints
	return nil
}

func (assembly *systemAssembly) accountManagement() error {
	var err error
	instanceRepository, err := instancemanagementstore.NewRepository(assembly.pool, assembly.instanceCipher)
	if err != nil {
		return err
	}
	workspaceLifecycleService, err := systemapi.NewWorkspaceLifecycle(instanceRepository, assembly.sessionValidator, assembly.activityTaskPublisher)
	if err != nil {
		return err
	}
	workspaceLifecycleEndpoints := workspacelifecycleapi.NewEndpoints(workspaceLifecycleService)
	workspaceLifecycleEndpoints.Use(log.Endpoint)
	instanceService, err := systemapi.NewInstanceManagement(instanceRepository, assembly.sessionValidator, assembly.instanceCipher, instancemail.Sender{}, assembly.invitationsCache, assembly.accountRepository, assembly.authRedirectService)
	if err != nil {
		return err
	}
	userAccountWritesRepository, err := useraccountwritesstore.NewRepository(assembly.pool, assembly.userIdentityRepository, assembly.sessionValidator)
	if err != nil {
		return err
	}
	userAccountWritesQuota, err := ratelimit.NewRedisStore(assembly.cfg.redisURL)
	if err != nil {
		return err
	}
	assembly.onClose(func(result *error) { closeResource(result, "user account request quota", userAccountWritesQuota) })
	userAccountWritesService, err := systemapi.NewUserAccountWrites(userAccountWritesRepository, assembly.sessionValidator, userAccountWritesQuota, assembly.assetAppBaseURL)
	if err != nil {
		return err
	}
	userAccountWritesEndpoints := useraccountwritesapi.NewEndpoints(userAccountWritesService)
	userAccountWritesEndpoints.Use(log.Endpoint)
	instanceEndpoints := instancemanagementapi.NewEndpoints(instanceService)
	instanceEndpoints.Use(log.Endpoint)
	assembly.routes.userAccountWrites = userAccountWritesEndpoints
	assembly.routes.userAccountWritesMetadata = userAccountWritesService.(sessionResourceProtocolAuthorizer)
	assembly.routes.instanceManagement = instanceEndpoints
	assembly.routes.instanceManagementMetadata = instanceService.(sessionResourceProtocolAuthorizer)
	assembly.routes.workspaceLifecycle = workspaceLifecycleEndpoints
	assembly.routes.workspaceLifecycleMetadata = workspaceLifecycleService.(systemapi.SessionResourceProtocolAuthorizer)
	return nil
}
