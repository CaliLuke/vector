package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/CaliLuke/go-plane/internal/apicontract"

	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
)

type apiV1ReadErrorFactory func(int, map[string]any) error
type apiV1ReadErrorMapper func(error) error

// Keep the service-specific context key and error factory at the generated
// adapter boundary; token validation and authentication effects are shared.
func authenticateAPIRead(ctx context.Context, key string, keys activityReadsAPIKeys, principalKey any, newError apiV1ReadErrorFactory) (context.Context, error) {
	principal, err := keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, newError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, newError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, newError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, principalKey, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
}

type apiV1ReadPipeline struct {
	principal apikey.Principal
	key       string
	operation string
	payload   any
	throttle  activityReadsThrottle
	authorize func(context.Context, string, activitystore.Request) error
	read      func(context.Context, string, activitystore.Request) (any, error)
	newError  apiV1ReadErrorFactory
	mapError  apiV1ReadErrorMapper
}

func (pipeline apiV1ReadPipeline) execute(ctx context.Context) (io.ReadCloser, error) {
	request := activityPayloadRequest(ctx, pipeline.operation, pipeline.payload)
	userID := pipeline.principal.UserID.String()
	if err := pipeline.authorize(ctx, userID, request); err != nil {
		return nil, pipeline.mapError(err)
	}

	rate, err := pipeline.throttle.Allow(ctx, pipeline.key)
	if err != nil {
		return nil, pipeline.newError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return nil, pipeline.newError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}

	if timezoneErr := browsersession.ValidateUserTimezone(pipeline.principal.Timezone); timezoneErr != nil {
		if errors.Is(timezoneErr, browsersession.ErrTimezoneNotFound) {
			return nil, pipeline.newError(400, map[string]any{"error": "The required key does not exist."})
		}
		return nil, pipeline.mapError(timezoneErr)
	}

	body, err := pipeline.read(ctx, userID, request)
	if err != nil {
		return nil, pipeline.mapError(err)
	}
	data, err := json.Marshal(activityTimestamps(body, pipeline.principal.Timezone))
	if err != nil {
		return nil, pipeline.mapError(err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func mapAPIReadError(err, forbidden, notFound error, newError apiV1ReadErrorFactory) error {
	var response *apicontract.HTTPError
	switch {
	case errors.As(err, &response):
		return newError(response.Status, response.Body)
	case errors.Is(err, forbidden):
		return newError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, notFound):
		return newError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return newError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
}
