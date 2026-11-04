package systemapi

import (
	"context"
	"fmt"
	"io"

	api "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/loom/security"
)

type apiModuleRepository interface {
	AuthorizeAPIV1(context.Context, string, string, string, bool) error
	ReadAPIV1(context.Context, string, store.APIV1Request) (any, error)
}
type apiModuleReadsService struct {
	repository apiModuleRepository
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}
type apiModulePrincipalKey struct{}

func NewAPIModuleReads(repository apiModuleRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil {
		return nil, fmt.Errorf("API module repository, API-key validator and throttle are required")
	}
	return &apiModuleReadsService{repository: repository, keys: keys, throttle: throttle}, nil
}

func (s *apiModuleReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	return authenticateAPIRead(ctx, key, s.keys, apiModulePrincipalKey{}, apiModuleError)
}
func (s *apiModuleReadsService) execute(ctx context.Context, operation string, payload any) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiModulePrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiModuleError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	reader, err := (apiV1ReadPipeline{
		principal: principal.Principal, key: principal.Key, operation: operation, payload: payload, throttle: s.throttle,
		authorize: func(ctx context.Context, userID string, request activitystore.Request) error {
			return s.repository.AuthorizeAPIV1(ctx, userID, request.Params["slug"], request.Params["project_id"], false)
		},
		read: func(ctx context.Context, userID string, request activitystore.Request) (any, error) {
			return s.repository.ReadAPIV1(ctx, userID, store.APIV1Request{Operation: request.Operation, Params: request.Params, Present: request.Present})
		},
		newError: apiModuleError, mapError: mapAPIModuleError,
	}).execute(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &api.APIModuleReadsRawJSONResponseMetadata{ContentType: "application/json"}, reader, nil
}

func apiModuleError(status int, body map[string]any) error {
	result := &api.APIModuleReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	return apicontract.ProjectError(body, result)
}
func mapAPIModuleError(err error) error {
	return mapAPIReadError(err, store.ErrForbidden, store.ErrNotFound, apiModuleError)
}

func (s *apiModuleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModules(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesPayload) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "archived", payload)
}

func (s *apiModuleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLite(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLitePayload) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "lite", payload)
}

func (s *apiModuleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDModules(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesPayload) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "list", payload)
}

func (s *apiModuleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssuesPayload) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "issues", payload)
}

func (s *apiModuleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkPayload) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "detail", payload)
}
