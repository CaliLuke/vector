package systemapi

import (
	"context"
	"fmt"
	"io"

	api "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/loom/security"
)

type apiCycleRepository interface {
	AuthorizeAPIV1(context.Context, string, string, string, bool) error
	ReadAPIV1(context.Context, string, store.APIV1Request) (any, error)
}
type apiCycleReadsService struct {
	repository apiCycleRepository
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}
type apiCyclePrincipalKey struct{}

func NewAPICycleReads(repository apiCycleRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil {
		return nil, fmt.Errorf("API cycle repository, API-key validator and throttle are required")
	}
	return &apiCycleReadsService{repository: repository, keys: keys, throttle: throttle}, nil
}

func (s *apiCycleReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	return authenticateAPIRead(ctx, key, s.keys, apiCyclePrincipalKey{}, apiCycleError)
}
func (s *apiCycleReadsService) execute(ctx context.Context, operation string, payload any) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiCyclePrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiCycleError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	reader, err := (apiV1ReadPipeline{
		principal: principal.Principal, key: principal.Key, operation: operation, payload: payload, throttle: s.throttle,
		authorize: func(ctx context.Context, userID string, request activitystore.Request) error {
			return s.repository.AuthorizeAPIV1(ctx, userID, request.Params["slug"], request.Params["project_id"], false)
		},
		read: func(ctx context.Context, userID string, request activitystore.Request) (any, error) {
			return s.repository.ReadAPIV1(ctx, userID, store.APIV1Request{Operation: request.Operation, Params: request.Params, Present: request.Present})
		},
		newError: apiCycleError, mapError: mapAPICycleError,
	}).execute(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &api.APICycleReadsRawJSONResponseMetadata{ContentType: "application/json"}, reader, nil
}

func apiCycleError(status int, body map[string]any) error {
	result := &api.APICycleReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	return apicontract.ProjectError(body, result)
}
func mapAPICycleError(err error) error {
	return mapAPIReadError(err, store.ErrForbidden, store.ErrNotFound, apiCycleError)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCycles(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesPayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "archived", payload)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLite(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesLitePayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "lite", payload)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDCycles(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesPayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "list", payload)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssues(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesPayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "issues", payload)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueID(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDCycleIssuesByIssueIDPayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "issue-detail", payload)
}

func (s *apiCycleReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPkPayload) (*api.APICycleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "detail", payload)
}
