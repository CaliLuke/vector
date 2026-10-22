package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	api "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type apiUserIdentityRepository interface {
	APIV1Identity(context.Context, uuid.UUID) (json.RawMessage, error)
}
type apiUserIdentityReadsService struct {
	repository apiUserIdentityRepository
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}
type apiUserIdentityPrincipalKey struct{}

func NewAPIUserIdentity(repository apiUserIdentityRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil {
		return nil, fmt.Errorf("API identity repository, API-key validator and throttle are required")
	}
	return &apiUserIdentityReadsService{repository: repository, keys: keys, throttle: throttle}, nil
}

func (s *apiUserIdentityReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, apiUserIdentityError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, apiUserIdentityError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiUserIdentityError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, apiUserIdentityPrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
}

func (s *apiUserIdentityReadsService) authorize(ctx context.Context) (apikey.Principal, error) {
	principal, ok := ctx.Value(apiUserIdentityPrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return apikey.Principal{}, apiUserIdentityError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}

	rate, err := s.throttle.Allow(ctx, principal.Key)
	if err != nil {
		return apikey.Principal{}, apiUserIdentityError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return apikey.Principal{}, apiUserIdentityError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return apikey.Principal{}, apiUserIdentityError(400, map[string]any{"error": "The required key does not exist."})
		}
		return apikey.Principal{}, mapAPIUserIdentityError(err)
	}
	return principal.Principal, nil
}

func (s *apiUserIdentityReadsService) GetAPIV1UsersMe(ctx context.Context, _ *api.GetAPIV1UsersMePayload) (*api.APIUserIdentityRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, err := s.authorize(ctx)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.repository.APIV1Identity(ctx, uuid.UUID(principal.UserID))
	if err != nil {
		return nil, nil, mapAPIUserIdentityError(err)
	}
	// Match DRF compact rendering before gzip decides whether the body reaches its size threshold.
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		return nil, nil, mapAPIUserIdentityError(err)
	}
	return &api.APIUserIdentityRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(compact.Bytes())), nil
}

// APIIdentityProtocolAuthorizer keeps GET-only rejection after source authentication,
// throttling and timezone, without introducing project permission checks.
type APIIdentityProtocolAuthorizer interface {
	AuthorizeAPIIdentityProtocol(context.Context, string) (int, map[string]any)
}

func (s *apiUserIdentityReadsService) AuthorizeAPIIdentityProtocol(ctx context.Context, key string) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, key, &security.APIKeyScheme{})
	if err == nil {
		_, err = s.authorize(ctx)
	}
	if err == nil {
		return 0, nil
	}
	var response *api.APIUserIdentityErrorBody
	if errors.As(err, &response) {
		var status int
		_, _ = fmt.Sscanf(response.Name, "status_%d", &status)
		raw, _ := json.Marshal(response)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		return status, body
	}
	return 500, map[string]any{"error": "Something went wrong please try again later"}
}

func apiUserIdentityError(status int, body map[string]any) error {
	result := &api.APIUserIdentityErrorBody{Name: fmt.Sprintf("status_%d", status)}
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, result)
	return result
}
func mapAPIUserIdentityError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return apiUserIdentityError(404, map[string]any{"error": "The requested resource does not exist."})
	}
	return apiUserIdentityError(500, map[string]any{"error": "Something went wrong please try again later"})
}
