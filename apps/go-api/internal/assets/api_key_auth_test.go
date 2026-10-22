package assets

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type apiKeyValidatorFunc func(context.Context, string) (apikey.Principal, error)

func (validate apiKeyValidatorFunc) Validate(ctx context.Context, token string) (apikey.Principal, error) {
	return validate(ctx, token)
}

type apiKeyThrottleCache struct {
	values map[string][]byte
	gets   int
	sets   int
}

type genericPermissionRepository struct {
	assetRepository
	member bool
	calls  int
}

func (repository *genericPermissionRepository) WorkspaceMemberBySlug(context.Context, uuid.UUID, string) (bool, error) {
	repository.calls++
	return repository.member, nil
}

func newAPIKeyThrottleCache() *apiKeyThrottleCache {
	return &apiKeyThrottleCache{values: make(map[string][]byte)}
}

func (cache *apiKeyThrottleCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	cache.gets++
	value, ok := cache.values[key]
	return value, ok, nil
}

func (cache *apiKeyThrottleCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	cache.sets++
	cache.values[key] = append([]byte(nil), value...)
	return nil
}

func apiKeyAuthService(t *testing.T, validate apiKeyValidatorFunc, cache *apiKeyThrottleCache, rate string) *Service {
	t.Helper()
	throttle, err := apikey.NewThrottle(cache, rate)
	if err != nil {
		t.Fatalf("create API-key throttle: %v", err)
	}
	return &Service{apiKeys: validate, throttle: throttle}
}

func apiKeyAuthErrorStatus(err error) int {
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) {
		return 0
	}
	return httpErr.Status
}

func TestAPIKeyAuthMarksValidCredentialAndRateEffects(t *testing.T) {
	userID := uuid.MustParse("00112233-4455-6677-8899-aabbccddeeff")
	validatorCalls := 0
	cache := newAPIKeyThrottleCache()
	service := apiKeyAuthService(t, func(_ context.Context, token string) (apikey.Principal, error) {
		validatorCalls++
		if token != "valid-key" {
			t.Fatalf("validator received token %q, want valid-key", token)
		}
		return apikey.Principal{UserID: userID}, nil
	}, cache, "2/min")
	effects := &apikey.Effects{}
	ctx, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "valid-key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatalf("authenticate valid API key: %v", err)
	}
	if validatorCalls != 1 {
		t.Fatalf("validator calls = %d, want 1", validatorCalls)
	}
	if !effects.Authenticated {
		t.Fatal("valid API key did not mark request authenticated")
	}
	if effects.Rate == nil || !effects.Rate.Allowed {
		t.Fatalf("rate effect = %#v, want allowed rate result", effects.Rate)
	}
	if got, err := assetUserID(ctx); err != nil || got != userID {
		t.Fatalf("authenticated user = %s, %v; want %s", got, err, userID)
	}
}

func TestAPIKeyAuthInvalidOrMissingCredentialDoesNotThrottle(t *testing.T) {
	for _, test := range []struct {
		name             string
		key              string
		validationErr    error
		wantStatus       int
		wantValidatorHit int
	}{
		{name: "missing", key: "", wantStatus: http.StatusUnauthorized},
		{name: "invalid", key: "invalid-key", validationErr: apikey.ErrInvalid, wantStatus: http.StatusForbidden, wantValidatorHit: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			validatorCalls := 0
			cache := newAPIKeyThrottleCache()
			service := apiKeyAuthService(t, func(context.Context, string) (apikey.Principal, error) {
				validatorCalls++
				return apikey.Principal{}, test.validationErr
			}, cache, "1/min")
			effects := &apikey.Effects{}
			_, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), test.key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
			if got := apiKeyAuthErrorStatus(err); got != test.wantStatus {
				t.Fatalf("authentication status = %d, want %d (error %v)", got, test.wantStatus, err)
			}
			if validatorCalls != test.wantValidatorHit {
				t.Fatalf("validator calls = %d, want %d", validatorCalls, test.wantValidatorHit)
			}
			if cache.gets != 0 || cache.sets != 0 {
				t.Fatalf("invalid/missing key reached throttle cache: gets=%d sets=%d", cache.gets, cache.sets)
			}
			if effects.Authenticated || effects.Rate != nil {
				t.Fatalf("failed authentication changed response effects: %#v", effects)
			}
		})
	}
}

func TestAPIKeyAuthThrottleRejectionReports429AndDeniedRate(t *testing.T) {
	cache := newAPIKeyThrottleCache()
	service := apiKeyAuthService(t, func(context.Context, string) (apikey.Principal, error) {
		return apikey.Principal{UserID: uuid.MustParse("00112233-4455-6677-8899-aabbccddeeff")}, nil
	}, cache, "1/min")
	scheme := &security.APIKeyScheme{Name: "ApiKeyAuth"}
	if _, err := service.APIKeyAuth(context.Background(), "valid-key", scheme); err != nil {
		t.Fatalf("first request should pass the throttle: %v", err)
	}

	effects := &apikey.Effects{}
	_, err := service.APIKeyAuth(apikey.WithEffects(context.Background(), effects), "valid-key", scheme)
	if got := apiKeyAuthErrorStatus(err); got != http.StatusTooManyRequests {
		t.Fatalf("throttled status = %d, want %d (error %v)", got, http.StatusTooManyRequests, err)
	}
	if !effects.Authenticated {
		t.Fatal("valid but throttled API key did not retain authenticated effect")
	}
	if effects.Rate == nil || effects.Rate.Allowed {
		t.Fatalf("rate effect = %#v, want explicit denied rate result", effects.Rate)
	}
	if cache.gets != 2 || cache.sets != 1 {
		t.Fatalf("throttle cache calls = gets:%d sets:%d, want gets:2 sets:1", cache.gets, cache.sets)
	}
}

func TestGenericAssetWorkspacePermissionRunsBeforeAPIKeyThrottle(t *testing.T) {
	cache := newAPIKeyThrottleCache()
	repository := &genericPermissionRepository{member: false}
	service := apiKeyAuthService(t, func(context.Context, string) (apikey.Principal, error) {
		return apikey.Principal{UserID: uuid.MustParse("00112233-4455-6677-8899-aabbccddeeff")}, nil
	}, cache, "1/min")
	service.repository = repository
	resource := systemapi.SessionProtocolResource{
		Method: "POST", Pattern: "/api/v1/workspaces/{slug}/assets/", Slug: "private",
	}
	ctx := systemapi.WithProtocolResource(context.Background(), resource)
	effects := &apikey.Effects{}
	_, err := service.APIKeyAuth(apikey.WithEffects(ctx, effects), "valid-key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if got := apiKeyAuthErrorStatus(err); got != http.StatusForbidden {
		t.Fatalf("nonmember generic asset status = %d, want 403 (error %v)", got, err)
	}
	if repository.calls != 1 {
		t.Fatalf("workspace permission calls = %d, want 1", repository.calls)
	}
	if cache.gets != 0 || cache.sets != 0 || effects.Rate != nil {
		t.Fatalf("forbidden workspace request reached throttle: gets=%d sets=%d rate=%#v", cache.gets, cache.sets, effects.Rate)
	}
	if !effects.Authenticated {
		t.Fatal("valid API key did not retain authentication effect before permission denial")
	}
}
