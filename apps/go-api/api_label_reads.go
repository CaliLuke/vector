package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	api "github.com/CaliLuke/go-plane/apilabelreads/gen/api_label_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
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
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, apiLabelError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, apiLabelError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiLabelError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, apiLabelPrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
}

func (s *apiLabelReadsService) execute(ctx context.Context, operation string, payload any) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiLabelPrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiLabelError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	raw := activityPayloadRequest(ctx, operation, payload)
	request := store.APIV1Request{Operation: operation, Params: raw.Params, Present: raw.Present}
	if err := s.repository.AuthorizeAPIV1(ctx, principal.UserID.String(), raw.Params["slug"], raw.Params["project_id"], false); err != nil {
		return nil, nil, mapAPILabelError(err)
	}
	rate, err := s.throttle.Allow(ctx, principal.Key)
	if err != nil {
		return nil, nil, apiLabelError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return nil, nil, apiLabelError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return nil, nil, apiLabelError(400, map[string]any{"error": "The required key does not exist."})
		}
		return nil, nil, mapAPILabelError(err)
	}
	body, err := s.repository.ReadAPIV1(ctx, principal.UserID.String(), request)
	if err != nil {
		return nil, nil, mapAPILabelError(err)
	}
	data, err := json.Marshal(activityTimestamps(body, principal.Timezone))
	if err != nil {
		return nil, nil, mapAPILabelError(err)
	}
	return &api.APILabelReadsRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}

func apiLabelError(status int, body map[string]any) error {
	result := &api.APILabelReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, result)
	return result
}
func mapAPILabelError(err error) error {
	var response *djangoapi.HTTPError
	switch {
	case errors.As(err, &response):
		return apiLabelError(response.Status, response.Body)
	case errors.Is(err, store.ErrForbidden):
		return apiLabelError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, store.ErrNotFound):
		return apiLabelError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return apiLabelError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
}

func (s *apiLabelReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsPayload) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "list", payload)
}
func (s *apiLabelReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPkPayload) (*api.APILabelReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "detail", payload)
}
