package systemapi

import (
	"context"
	"fmt"
	"io"

	api "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/loom/security"
)

type apiLabelRepository interface {
	AuthorizeAPIV1(context.Context, string, string, string, bool) error
	ReadAPIV1(context.Context, string, store.APIV1Request) (any, error)
}
type apiLabelReadsService struct {
	repository apiLabelRepository
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}
type apiLabelPrincipalKey struct{}

func NewAPILabelReads(repository apiLabelRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil {
		return nil, fmt.Errorf("API label repository, API-key validator and throttle are required")
	}
	return &apiLabelReadsService{repository: repository, keys: keys, throttle: throttle}, nil
}

func (s *apiLabelReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	return authenticateAPIRead(ctx, key, s.keys, apiLabelPrincipalKey{}, apiLabelError)
}
func (s *apiLabelReadsService) execute(ctx context.Context, operation string, payload any) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiLabelPrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiLabelError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	reader, err := (apiV1ReadPipeline{
		principal: principal.Principal, key: principal.Key, operation: operation, payload: payload, throttle: s.throttle,
		authorize: func(ctx context.Context, userID string, request activitystore.Request) error {
			return s.repository.AuthorizeAPIV1(ctx, userID, request.Params["slug"], request.Params["project_id"], false)
		},
		read: func(ctx context.Context, userID string, request activitystore.Request) (any, error) {
			return s.repository.ReadAPIV1(ctx, userID, store.APIV1Request{Operation: request.Operation, Params: request.Params, Present: request.Present})
		},
		newError: apiLabelError, mapError: mapAPILabelError,
	}).execute(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &api.APILabelReadsRawJSONResponseMetadata{ContentType: "application/json"}, reader, nil
}

func apiLabelError(status int, body map[string]any) error {
	result := &api.APILabelReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	return apicontract.ProjectError(body, result)
}
func mapAPILabelError(err error) error {
	return mapAPIReadError(err, store.ErrForbidden, store.ErrNotFound, apiLabelError)
}

func (s *apiLabelReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsPayload) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "list", payload)
}
func (s *apiLabelReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPkPayload) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "detail", payload)
}
