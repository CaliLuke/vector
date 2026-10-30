package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/platform"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	api "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type sessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type keyValidator interface {
	Validate(context.Context, string) (apikey.Principal, error)
}
type configurationEffects interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}
type configurationThrottler interface {
	Allow(context.Context, string) (apikey.Rate, error)
}

type principal struct {
	userID, key, timezone string
	api                   bool
}
type principalKey struct{}

type projectConfigurationService struct {
	repository *store.Repository
	sessions   sessionValidator
	keys       keyValidator
	throttle   configurationThrottler
	effects    configurationEffects
}

func NewProjectConfiguration(repository *store.Repository, sessions sessionValidator, keys keyValidator, throttle configurationThrottler, effects configurationEffects) (api.Service, error) {
	if repository == nil || sessions == nil || keys == nil || throttle == nil || effects == nil {
		return nil, fmt.Errorf("project configuration dependencies are required")
	}
	return &projectConfigurationService{repository: repository, sessions: sessions, keys: keys, throttle: throttle, effects: effects}, nil
}

func (service *projectConfigurationService) APIKeyAuth(ctx context.Context, credential string, scheme *security.APIKeyScheme) (context.Context, error) {
	name := "ApiKeyAuth"
	if scheme != nil && scheme.Name != "" {
		name = scheme.Name
	}
	if name == "PlaneSessionCookie" {
		userID, timezone, inspected := readServiceSessionPrincipal(ctx)
		if !inspected {
			if credential == "" {
				credential = sessionCredentialFromContext(ctx)
			}
			if credential != "" {
				validated, err := service.sessions.Validate(ctx, credential)
				if err == nil {
					userID, timezone = validated.UserID, validated.UserTimezone
				}
			}
		}
		if userID == "" {
			return ctx, serviceError(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
		}
		return context.WithValue(ctx, principalKey{}, principal{userID: userID, timezone: timezone}), nil
	}
	if credential == "" {
		return ctx, serviceError(http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	validated, err := service.keys.Validate(ctx, credential)
	if err != nil {
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, serviceError(http.StatusForbidden, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, serviceError(http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, principalKey{}, principal{userID: validated.UserID.String(), key: credential, timezone: validated.Timezone, api: true}), nil
}

func (service *projectConfigurationService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := service.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	return protocolError(err)
}

func (service *projectConfigurationService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	if status, body := service.AuthorizeSessionProtocol(ctx, credential); status != 0 {
		return status, body
	}
	if resource.Slug == "" || resource.ProjectID == "" || resource.Method == http.MethodOptions || resource.Method == http.MethodHead {
		return 0, nil
	}
	identity := principalFromSessionContext(ctx, credential, service.sessions)
	if identity.userID == "" {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	if err := service.repository.Authorize(ctx, identity.userID, resource.Slug, resource.ProjectID, resource.Method != http.MethodGet); err != nil {
		return protocolError(service.mapError(err))
	}
	return 0, nil
}

func (service *projectConfigurationService) AuthorizeAPIV1LabelProtocol(ctx context.Context, key, slug, projectID string) (int, map[string]any) {
	authorized, err := service.APIKeyAuth(ctx, key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		return protocolError(err)
	}
	identity := principalFrom(authorized)
	if authorizeAPIV1Err := service.repository.AuthorizeAPIV1(authorized, identity.userID, slug, projectID); authorizeAPIV1Err != nil {
		return protocolError(service.mapError(authorizeAPIV1Err))
	}
	rate, err := service.throttle.Allow(authorized, key)
	if err != nil {
		return protocolError(serviceError(http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"}))
	}
	if effects := apikey.EffectsFromContext(authorized); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return http.StatusTooManyRequests, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait.Seconds()))}
	}
	return 0, nil
}

func protocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var response *api.ProjectConfigurationErrorResponse
	if errors.As(err, &response) {
		status, _ := strconv.Atoi(strings.TrimPrefix(response.Name, "status_"))
		var body map[string]any
		if json.Unmarshal(response.Body, &body) != nil {
			body = map[string]any{"detail": "Something went wrong please try again later"}
		}
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return status, body
	}
	return http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"}
}

func principalFromSessionContext(ctx context.Context, credential string, sessions sessionValidator) principal {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if inspected {
		return principal{userID: userID, timezone: timezone}
	}
	if credential != "" {
		if validated, err := sessions.Validate(ctx, credential); err == nil {
			return principal{userID: validated.UserID, timezone: validated.UserTimezone}
		}
	}
	return principal{}
}

func (service *projectConfigurationService) CreateEstimate(ctx context.Context, payload *api.CreateEstimatePayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.appPrincipal(ctx, payload.Slug, payload.ProjectID, false)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	contentType, _ := ctx.Value(loomhttp.ContentTypeKey).(string)
	if contentType == "" {
		contentType = "application/json"
	}
	result, err := service.repository.CreateEstimate(ctx, identity.userID, payload.Slug, payload.ProjectID, body, contentType)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}

func (service *projectConfigurationService) ListProjectEstimates(ctx context.Context, payload *api.ListProjectEstimatesPayload) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	identity, err := service.appPrincipal(ctx, payload.Slug, payload.ProjectID, true)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	var result []byte
	err = service.repository.DatabaseQueryRow(ctx, `SELECT CASE WHEN p.estimate_id IS NULL THEN '[]'::jsonb ELSE COALESCE((SELECT jsonb_agg(jsonb_build_object('id',ep.id,'created_at',ep.created_at,'updated_at',ep.updated_at,'created_by',ep.created_by_id,'updated_by',ep.updated_by_id,'deleted_at',ep.deleted_at,'estimate',ep.estimate_id,'workspace',ep.workspace_id,'project',ep.project_id,'key',ep.key,'value',ep.value,'description',ep.description) ORDER BY ep.value) FROM estimate_points ep WHERE ep.estimate_id=p.estimate_id AND ep.project_id=p.id AND ep.workspace_id=p.workspace_id AND ep.deleted_at IS NULL),'[]'::jsonb) END FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL`, payload.ProjectID, payload.Slug).Scan(&result)
	if err != nil {
		return nil, nil, serviceError(http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"})
	}
	location := time.UTC
	if identity.timezone != "" {
		if parsed, loadErr := time.LoadLocation(identity.timezone); loadErr == nil {
			location = parsed
		}
	}
	result, err = formatProjectEstimateDateTimes(result, location)
	if err != nil {
		return nil, nil, serviceError(http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"})
	}
	return jsonResponse(result)
}

func formatProjectEstimateDateTimes(raw []byte, location *time.Location) ([]byte, error) {
	var points []map[string]any
	if err := json.Unmarshal(raw, &points); err != nil {
		return nil, err
	}
	for _, point := range points {
		for _, key := range []string{"created_at", "updated_at", "deleted_at"} {
			text, ok := point[key].(string)
			if !ok {
				continue
			}
			if parsed, parseErr := time.Parse(time.RFC3339Nano, text); parseErr == nil {
				point[key] = pythoncompat.DRFDateTimeString(parsed, location)
			}
		}
	}
	return json.Marshal(points)
}

func (service *projectConfigurationService) PatchEstimate(ctx context.Context, payload *api.PatchEstimatePayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.appPrincipal(ctx, payload.Slug, payload.ProjectID, false)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	contentType := projectConfigurationContentType(ctx)
	result, err := service.repository.PatchEstimate(ctx, identity.userID, payload.Slug, payload.ProjectID, stringValue(payload.EstimateID), body, contentType)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}
func (service *projectConfigurationService) DeleteEstimate(ctx context.Context, payload *api.DeleteEstimatePayload) error {
	identity, err := service.appPrincipal(ctx, payload.Slug, payload.ProjectID, false)
	if err != nil {
		return service.mapError(err)
	}
	if err := service.repository.DeleteEstimate(ctx, identity.userID, payload.Slug, payload.ProjectID, stringValue(payload.EstimateID)); err != nil {
		return service.mapError(err)
	}
	return nil
}
func (service *projectConfigurationService) CreateEstimatePoint(ctx context.Context, payload *api.CreateEstimatePointPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.appPrincipal(ctx, payload.Slug, payload.ProjectID, true)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	result, err := service.repository.CreateEstimatePoint(ctx, identity.userID, payload.Slug, payload.ProjectID, stringValue(payload.EstimateID), body, projectConfigurationContentType(ctx))
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}
func (service *projectConfigurationService) PatchEstimatePoint(ctx context.Context, payload *api.PatchEstimatePointPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	return service.mutateEstimatePoint(ctx, payload.Slug, payload.ProjectID, body, func(actor string) (json.RawMessage, error) {
		return service.repository.PatchEstimatePoint(ctx, actor, payload.Slug, payload.ProjectID, stringValue(payload.EstimateID), stringValue(payload.EstimatePointID), body, projectConfigurationContentType(ctx))
	})
}
func (service *projectConfigurationService) DeleteEstimatePoint(ctx context.Context, payload *api.DeleteEstimatePointPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	return service.mutateEstimatePoint(ctx, payload.Slug, payload.ProjectID, body, func(actor string) (json.RawMessage, error) {
		return service.repository.DeleteEstimatePoint(ctx, actor, payload.Slug, payload.ProjectID, stringValue(payload.EstimateID), stringValue(payload.EstimatePointID), body, projectConfigurationContentType(ctx))
	})
}
func (service *projectConfigurationService) mutateEstimatePoint(ctx context.Context, slug, project string, body io.ReadCloser, operation func(string) (json.RawMessage, error)) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.appPrincipal(ctx, slug, project, true)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	result, err := operation(identity.userID)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}
func (service *projectConfigurationService) CreateAPILabel(ctx context.Context, payload *api.CreateAPILabelPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.apiLabelPrincipal(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	result, err := service.repository.WriteAPILabel(ctx, identity.userID, payload.Slug, payload.ProjectID, "", http.MethodPost, body, projectConfigurationContentType(ctx))
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}
func (service *projectConfigurationService) PatchAPILabel(ctx context.Context, payload *api.PatchAPILabelPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity, err := service.apiLabelPrincipal(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	result, err := service.repository.WriteAPILabel(ctx, identity.userID, payload.Slug, payload.ProjectID, stringValue(payload.Pk), http.MethodPatch, body, projectConfigurationContentType(ctx))
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}
func (service *projectConfigurationService) DeleteAPILabel(ctx context.Context, payload *api.DeleteAPILabelPayload) error {
	identity, err := service.apiLabelPrincipal(ctx, payload.Slug, payload.ProjectID)
	if err != nil {
		return service.mapError(err)
	}
	if _, err = service.repository.WriteAPILabel(ctx, identity.userID, payload.Slug, payload.ProjectID, stringValue(payload.Pk), http.MethodDelete, nil, ""); err != nil {
		return service.mapError(err)
	}
	if err = service.effects.SoftDelete(ctx, "db", "label", stringValue(payload.Pk)); err != nil {
		return service.mapError(err)
	}
	return nil
}
func (service *projectConfigurationService) BulkCreateLabels(ctx context.Context, payload *api.BulkCreateLabelsPayload, body io.ReadCloser) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	defer platform.CloseRequestBody(ctx, body)
	identity := principalFrom(ctx)
	if identity.userID == "" {
		return nil, nil, serviceError(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := validateConfigurationTimezone(identity.timezone); err != nil {
		return nil, nil, service.mapError(err)
	}
	if err := service.repository.AuthorizeBulkLabels(ctx, identity.userID, payload.Slug, payload.ProjectID); err != nil {
		return nil, nil, service.mapError(err)
	}
	result, err := service.repository.BulkCreateLabels(ctx, identity.userID, payload.Slug, payload.ProjectID, body, projectConfigurationContentType(ctx))
	if err != nil {
		return nil, nil, service.mapError(err)
	}
	return jsonResponse(result)
}

func principalFrom(ctx context.Context) principal {
	value, _ := ctx.Value(principalKey{}).(principal)
	return value
}
func jsonResponse(data []byte) (*api.ProjectConfigurationRawJSON, io.ReadCloser, error) {
	return &api.ProjectConfigurationRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}
func serviceError(status int, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		data = []byte(`{"detail":"Something went wrong please try again later"}`)
		status = 500
	}
	return &api.ProjectConfigurationErrorResponse{Name: "status_" + strconv.Itoa(status), Body: loom.JSONValue(data)}
}
func (service *projectConfigurationService) mapError(err error) error {
	var configurationErr *api.ProjectConfigurationErrorResponse
	if errors.As(err, &configurationErr) {
		return configurationErr
	}
	var apiErr *store.HTTPError
	if errors.As(err, &apiErr) {
		return serviceError(apiErr.Status, apiErr.Body)
	}
	var djangoErr *djangoapi.HTTPError
	if errors.As(err, &djangoErr) {
		return serviceError(djangoErr.Status, djangoErr.Body)
	}
	return serviceError(500, map[string]any{"detail": "Something went wrong please try again later"})
}

func (service *projectConfigurationService) appPrincipal(ctx context.Context, slug, project string, timezoneFirst bool) (principal, error) {
	identity := principalFrom(ctx)
	if identity.userID == "" {
		return principal{}, serviceError(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	checkTimezone := func() error { return validateConfigurationTimezone(identity.timezone) }
	if timezoneFirst {
		if err := checkTimezone(); err != nil {
			return principal{}, err
		}
	}
	if err := service.repository.Authorize(ctx, identity.userID, slug, project, true); err != nil {
		return principal{}, service.mapError(err)
	}
	if !timezoneFirst {
		if err := checkTimezone(); err != nil {
			return principal{}, err
		}
	}
	return identity, nil
}

func validateConfigurationTimezone(value string) error {
	err := djangosession.ValidateUserTimezone(value)
	if errors.Is(err, djangosession.ErrTimezoneNotFound) {
		return &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "The required key does not exist."}}
	}
	return err
}

func (service *projectConfigurationService) apiLabelPrincipal(ctx context.Context, slug, project string) (principal, error) {
	identity := principalFrom(ctx)
	if identity.userID == "" || !identity.api {
		return principal{}, serviceError(http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	if err := service.repository.AuthorizeAPIV1(ctx, identity.userID, slug, project); err != nil {
		return principal{}, err
	}
	rate, err := service.throttle.Allow(ctx, identity.key)
	if err != nil {
		return principal{}, err
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return principal{}, &djangoapi.HTTPError{Status: http.StatusTooManyRequests, Body: map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait.Seconds()))}}
	}
	if err := validateConfigurationTimezone(identity.timezone); err != nil {
		return principal{}, err
	}
	return identity, nil
}

func projectConfigurationContentType(ctx context.Context) string {
	contentType, _ := ctx.Value(loomhttp.ContentTypeKey).(string)
	if contentType == "" {
		return "application/json"
	}
	return contentType
}
