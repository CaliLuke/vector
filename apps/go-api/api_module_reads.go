package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	api "github.com/CaliLuke/go-plane/apimodulereads/gen/api_module_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
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
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, apiModuleError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, apiModuleError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiModuleError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, apiModulePrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
}

func (s *apiModuleReadsService) execute(ctx context.Context, operation string, payload any) (*api.APIModuleReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiModulePrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiModuleError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	raw := activityPayloadRequest(ctx, operation, payload)
	request := store.APIV1Request{Operation: operation, Params: raw.Params, Present: raw.Present}
	if err := s.repository.AuthorizeAPIV1(ctx, principal.UserID.String(), raw.Params["slug"], raw.Params["project_id"], false); err != nil {
		return nil, nil, mapAPIModuleError(err)
	}
	rate, err := s.throttle.Allow(ctx, principal.Key)
	if err != nil {
		return nil, nil, apiModuleError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return nil, nil, apiModuleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return nil, nil, apiModuleError(400, map[string]any{"error": "The required key does not exist."})
		}
		return nil, nil, mapAPIModuleError(err)
	}
	body, err := s.repository.ReadAPIV1(ctx, principal.UserID.String(), request)
	if err != nil {
		return nil, nil, mapAPIModuleError(err)
	}
	data, err := json.Marshal(activityTimestamps(body, principal.Timezone))
	if err != nil {
		return nil, nil, mapAPIModuleError(err)
	}
	return &api.APIModuleReadsRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}

func apiModuleError(status int, body map[string]any) error {
	result := &api.APIModuleReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, result)
	return result
}
func mapAPIModuleError(err error) error {
	var response *djangoapi.HTTPError
	switch {
	case errors.As(err, &response):
		return apiModuleError(response.Status, response.Body)
	case errors.Is(err, store.ErrForbidden):
		return apiModuleError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, store.ErrNotFound):
		return apiModuleError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return apiModuleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
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
