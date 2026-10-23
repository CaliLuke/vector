package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type apiV1ReadErrorFactory func(int, map[string]any) error
type apiV1ReadErrorMapper func(error) error

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

	if timezoneErr := djangosession.ValidateUserTimezone(pipeline.principal.Timezone); timezoneErr != nil {
		if errors.Is(timezoneErr, djangosession.ErrTimezoneNotFound) {
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
