package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	api "github.com/CaliLuke/go-plane/apicyclereads/gen/api_cycle_reads"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
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
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, apiCycleError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, apiCycleError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiCycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, apiCyclePrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
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
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, result)
	return result
}
func mapAPICycleError(err error) error {
	var response *djangoapi.HTTPError
	switch {
	case errors.As(err, &response):
		return apiCycleError(response.Status, response.Body)
	case errors.Is(err, store.ErrForbidden):
		return apiCycleError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, store.ErrNotFound):
		return apiCycleError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return apiCycleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
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
