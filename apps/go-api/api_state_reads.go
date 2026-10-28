package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/apistatereads/gen/api_state_reads"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type apiStateRepository interface {
	AuthorizeAPIV1(context.Context, string, string, string, bool) error
	ReadAPIV1(context.Context, string, store.APIV1Request) (any, error)
}

type apiStateWriteRepository interface {
	ResolveForUpdate(context.Context, store.StateWriteInput) error
	Create(context.Context, store.StateWriteInput) (json.RawMessage, error)
	Update(context.Context, store.StateWriteInput) (json.RawMessage, error)
	Delete(context.Context, store.StateWriteInput) error
	SerializeAPIV1(context.Context, json.RawMessage, string, string) (map[string]any, error)
}

type apiStateReadsService struct {
	repository apiStateRepository
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}

type apiStatePrincipalKey struct{}

func NewAPIStateReads(repository apiStateRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil {
		return nil, fmt.Errorf("API state repository, API-key validator and throttle are required")
	}
	return &apiStateReadsService{repository: repository, keys: keys, throttle: throttle}, nil
}

func (s *apiStateReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if apiStateWriteMethod(ctx) {
			if key == "" {
				return ctx, apiStateWriteError(401, map[string]any{"detail": "Authentication credentials were not provided."})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, apiStateWriteError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, apiStateWriteError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if key == "" {
			return ctx, apiStateError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, apiStateError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiStateError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, apiStatePrincipalKey{}, struct {
		apikey.Principal
		Key string
	}{principal, key}), nil
}

func (s *apiStateReadsService) execute(ctx context.Context, operation string, payload any) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiStatePrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiStateError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	reader, err := (apiV1ReadPipeline{
		principal: principal.Principal, key: principal.Key, operation: operation, payload: payload, throttle: s.throttle,
		authorize: func(ctx context.Context, userID string, request activitystore.Request) error {
			return s.repository.AuthorizeAPIV1(ctx, userID, request.Params["slug"], request.Params["project_id"], false)
		},
		read: func(ctx context.Context, userID string, request activitystore.Request) (any, error) {
			return s.repository.ReadAPIV1(ctx, userID, store.APIV1Request{Operation: request.Operation, Params: request.Params, Present: request.Present})
		},
		newError: apiStateError, mapError: mapAPIStateError,
	}).execute(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &api.APIStateReadsRawJSONResponseMetadata{ContentType: "application/json"}, reader, nil
}

func apiStateError(status int, body map[string]any) error {
	result := &api.APIStateReadsErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, result)
	return result
}

func mapAPIStateError(err error) error {
	var response *djangoapi.HTTPError
	switch {
	case errors.As(err, &response):
		return apiStateError(response.Status, response.Body)
	case errors.Is(err, store.ErrForbidden):
		return apiStateError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, store.ErrNotFound):
		return apiStateError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return apiStateError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
}

func (s *apiStateReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDStates(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesPayload) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "list", payload)
}

func (s *apiStateReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDPayload) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.execute(ctx, "detail", payload)
}

func (s *apiStateReadsService) executeWrite(ctx context.Context, operation, stateID string, payload any, body io.ReadCloser) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	principal, ok := ctx.Value(apiStatePrincipalKey{}).(struct {
		apikey.Principal
		Key string
	})
	if !ok {
		return nil, nil, apiStateWriteError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	writes, ok := s.repository.(apiStateWriteRepository)
	if !ok {
		return nil, nil, apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	raw := activityPayloadRequest(ctx, operation, payload)
	if err := s.repository.AuthorizeAPIV1(ctx, principal.UserID.String(), raw.Params["slug"], raw.Params["project_id"], true); err != nil {
		return nil, nil, mapAPIStateWriteError(err)
	}
	rate, err := s.throttle.Allow(ctx, principal.Key)
	if err != nil {
		return nil, nil, apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return nil, nil, apiStateWriteError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if validateUserTimezoneErr := djangosession.ValidateUserTimezone(principal.Timezone); validateUserTimezoneErr != nil {
		if errors.Is(validateUserTimezoneErr, djangosession.ErrTimezoneNotFound) {
			return nil, nil, apiStateWriteError(400, map[string]any{"error": "The required key does not exist."})
		}
		return nil, nil, mapAPIStateWriteError(validateUserTimezoneErr)
	}
	input := store.StateWriteInput{UserID: principal.UserID.String(), Slug: raw.Params["slug"], ProjectID: raw.Params["project_id"], StateID: stateID, Method: map[string]string{"create": "POST", "update": "PATCH", "delete": "DELETE"}[operation]}
	if operation == "update" {
		if resolveForUpdateErr := writes.ResolveForUpdate(ctx, input); resolveForUpdateErr != nil {
			return nil, nil, apiStateErrorFromWrite(resolveForUpdateErr)
		}
	}
	parsed := djangoinput.Input{Fields: map[string]any{}}
	if operation != "delete" {
		parsed, err = store.ParseStateWriteBody(body, apiStateWriteContentType(payload))
		if err != nil {
			return nil, nil, apiStateErrorFromWrite(err)
		}
	}
	input.Fields, input.Parsed = parsed.Fields, &parsed
	var data json.RawMessage
	switch operation {
	case "create":
		data, err = writes.Create(ctx, input)
	case "update":
		data, err = writes.Update(ctx, input)
	case "delete":
		err = writes.Delete(ctx, input)
	default:
		err = fmt.Errorf("unknown API state write operation %q", operation)
	}
	if err != nil {
		return nil, nil, apiStateErrorFromWrite(err)
	}
	if operation == "delete" {
		return nil, nil, nil
	}
	// StateSerializer is instantiated without fields/expand kwargs on writes;
	// the query parameters are accepted but do not project mutation responses.
	projected, err := writes.SerializeAPIV1(ctx, data, "", "")
	if err != nil {
		return nil, nil, apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	encoded, err := json.Marshal(activityTimestamps(projected, principal.Timezone))
	if err != nil {
		return nil, nil, apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return &api.APIStateReadsRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(encoded)), nil
}

func apiStateWriteContentType(payload any) string {
	switch value := payload.(type) {
	case *api.CreateAPIV1WorkspacesBySlugProjectsByProjectIDStatesPayload:
		if value.ContentType != nil {
			return *value.ContentType
		}
	case *api.UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDPayload:
		if value.ContentType != nil {
			return *value.ContentType
		}
	}
	return ""
}

func apiStateErrorFromWrite(err error) error {
	var failure *store.WriteFailure
	if errors.As(err, &failure) {
		return apiStateWriteError(failure.Status, asAPIErrorBody(failure.Body))
	}
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return apiStateWriteError(response.Status, asAPIErrorBody(response.Body))
	}
	return apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
}

func apiStateWriteMethod(ctx context.Context) bool {
	method, _ := ctx.Value(loom.MethodKey).(string)
	return strings.HasPrefix(method, "CreateAPIV1") || strings.HasPrefix(method, "UpdateAPIV1") || strings.HasPrefix(method, "DeleteAPIV1")
}

func apiStateWriteError(status int, body map[string]any) error {
	result := &api.APIStateWritesErrorResponseBody{Kind: fmt.Sprintf("status_%d", status)}
	raw, _ := json.Marshal(body)
	result.Body = json.RawMessage(raw)
	return result
}

func mapAPIStateWriteError(err error) error {
	var response *djangoapi.HTTPError
	switch {
	case errors.As(err, &response):
		return apiStateWriteError(response.Status, asAPIErrorBody(response.Body))
	case errors.Is(err, store.ErrForbidden):
		return apiStateWriteError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, store.ErrNotFound):
		return apiStateWriteError(404, map[string]any{"error": "The requested resource does not exist."})
	default:
		return apiStateWriteError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
}

func asAPIErrorBody(body any) map[string]any {
	if value, ok := body.(map[string]string); ok {
		result := map[string]any{}
		for key, item := range value {
			result[key] = item
		}
		return result
	}
	if value, ok := body.(map[string][]string); ok {
		result := map[string]any{}
		for key, item := range value {
			result[key] = item
		}
		return result
	}
	if value, ok := body.(map[string]any); ok {
		return value
	}
	return map[string]any{"error": body}
}

func (s *apiStateReadsService) CreateAPIV1WorkspacesBySlugProjectsByProjectIDStates(ctx context.Context, payload *api.CreateAPIV1WorkspacesBySlugProjectsByProjectIDStatesPayload, body io.ReadCloser) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.executeWrite(ctx, "create", "", payload, body)
}
func (s *apiStateReadsService) UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID(ctx context.Context, payload *api.UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDPayload, body io.ReadCloser) (*api.APIStateReadsRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.executeWrite(ctx, "update", payload.StateID, payload, body)
}
func (s *apiStateReadsService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDPayload) error {
	_, _, err := s.executeWrite(ctx, "delete", payload.StateID, payload, nil)
	return err
}
