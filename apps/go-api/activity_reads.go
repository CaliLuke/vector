package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	api "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	activitystore "github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type activityReadsRepository interface {
	Authorize(context.Context, activitystore.Principal, activitystore.Request) error
	Read(context.Context, activitystore.Principal, activitystore.Request) (activitystore.Result, error)
}

type activityReadsAPIKeys interface {
	Validate(context.Context, string) (apikey.Principal, error)
}
type activityReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type activityReadsThrottle interface {
	Allow(context.Context, string) (apikey.Rate, error)
}
type ActivityReadsProtocolAuthorizer interface {
	AuthorizeAPIActivityProtocol(context.Context, string, string, string, string) (int, map[string]any)
	AuthorizeUserActivityProtocol(context.Context, string) (int, map[string]any)
}
type activityReadsService struct {
	userRepository activityReadsRepository
	apiRepository  activityReadsRepository
	apiKeys        activityReadsAPIKeys
	sessions       activityReadsSessions
	throttle       activityReadsThrottle
}

type activityReadsPrincipalKey struct{}

func NewActivityReads(userRepository, apiRepository activityReadsRepository, apiKeys activityReadsAPIKeys, sessions activityReadsSessions, throttle activityReadsThrottle) (api.Service, error) {
	if userRepository == nil {
		return nil, fmt.Errorf("user activity reads repository is required")
	}
	if apiRepository == nil {
		return nil, fmt.Errorf("API activity reads repository is required")
	}
	if apiKeys == nil {
		return nil, fmt.Errorf("API-key validator is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	if throttle == nil {
		return nil, fmt.Errorf("API-key throttle is required")
	}
	return &activityReadsService{
		userRepository: userRepository,
		apiRepository:  apiRepository,
		apiKeys:        apiKeys,
		sessions:       sessions,
		throttle:       throttle,
	}, nil
}

func (service *activityReadsService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	switch scheme.Name {
	case "ApiKeyAuth":
		principal, err := service.apiKeys.Validate(ctx, key)
		if err != nil {
			if key == "" {
				return ctx, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, activityStatusError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, activityStatusError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
		return context.WithValue(ctx, activityReadsPrincipalKey{}, activitystore.Principal{
			UserID: principal.UserID, Authenticated: true, Timezone: principal.Timezone, APIKey: key,
		}), nil
	case "PlaneSessionCookie", "DjangoSessionCookie":
		if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
			parsed, err := uuid.Parse(userID)
			if err != nil {
				return ctx, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
			}
			return context.WithValue(ctx, activityReadsPrincipalKey{}, activitystore.Principal{UserID: parsed, Authenticated: true, Timezone: timezone}), nil
		}
		if key == "" {
			if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
				request := &http.Request{Header: metadata.Headers()}
				if cookie, cookieErr := request.Cookie("session-id"); cookieErr == nil {
					key = cookie.Value
				}
			}
		}
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		userID, err := uuid.Parse(principal.UserID)
		if err != nil {
			return ctx, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		return context.WithValue(ctx, activityReadsPrincipalKey{}, activitystore.Principal{UserID: userID, Authenticated: true, Timezone: principal.UserTimezone}), nil
	default:
		return ctx, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
}
func (service *activityReadsService) execute(ctx context.Context, operation string, payload any) (activitystore.Result, error) {
	principal, err := service.principal(ctx, operation, payload)
	if err != nil {
		return activitystore.Result{}, err
	}
	request := activityPayloadRequest(ctx, operation, payload)
	repository := service.apiRepository
	if operation == "get_api_users_me_activities" {
		repository = service.userRepository
	}
	if err := service.authorizeRequest(ctx, repository, principal, request); err != nil {
		return activitystore.Result{}, err
	}
	result, err := repository.Read(ctx, principal, request)
	if err != nil {
		return activitystore.Result{}, mapActivityReadError(err)
	}
	result.Body = activityTimestamps(result.Body, principal.Timezone)
	if operation != "get_api_users_me_activities" {
		result.Body = activityFields(result.Body, request.Params["fields"])
	}
	return result, nil
}

func (service *activityReadsService) authorizeRequest(ctx context.Context, repository activityReadsRepository, principal activitystore.Principal, request activitystore.Request) error {
	if err := repository.Authorize(ctx, principal, request); err != nil {
		return mapActivityReadError(err)
	}
	if principal.APIKey != "" {
		rate, err := service.throttle.Allow(ctx, principal.APIKey)
		if err != nil {
			return activityStatusError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return activityStatusError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
		}
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return activityStatusError(400, map[string]any{"error": "The required key does not exist."})
		}
		return activityStatusError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return nil
}

func activityProtocolRequiresProjectMemberRole(method string) bool {
	return method != "GET" && method != "HEAD" && method != "OPTIONS"
}

func (service *activityReadsService) AuthorizeAPIActivityProtocol(ctx context.Context, key, slug, projectID, method string) (int, map[string]any) {
	authorized, err := service.APIKeyAuth(ctx, key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err == nil {
		principal, _ := authorized.Value(activityReadsPrincipalKey{}).(activitystore.Principal)
		err = service.authorizeRequest(authorized, service.apiRepository, principal, activitystore.Request{
			Operation:                "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities",
			Params:                   map[string]string{"slug": slug, "project_id": projectID},
			RequireProjectMemberRole: activityProtocolRequiresProjectMemberRole(method),
		})
	}
	if err == nil {
		return 0, nil
	}
	return activityErrorStatusAndBody(err)
}

func (service *activityReadsService) AuthorizeUserActivityProtocol(ctx context.Context, session string) (int, map[string]any) {
	_, err := service.APIKeyAuth(ctx, session, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err == nil {
		return 0, nil
	}
	return activityErrorStatusAndBody(err)
}

func (service *activityReadsService) principal(ctx context.Context, operation string, _ any) (activitystore.Principal, error) {
	principal, _ := ctx.Value(activityReadsPrincipalKey{}).(activitystore.Principal)
	if principal.Authenticated || operation != "get_api_users_me_activities" {
		return principal, nil
	}
	if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return activitystore.Principal{}, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		return activitystore.Principal{UserID: parsed, Authenticated: true, Timezone: timezone}, nil
	}
	return activitystore.Principal{}, activityStatusError(401, map[string]any{"detail": "Authentication credentials were not provided."})
}
func mapActivityReadError(err error) error {
	var httpErr *activitystore.HTTPError
	switch {
	case errors.As(err, &httpErr):
		return activityStatusError(httpErr.Status, httpErr.Body)
	case errors.Is(err, activitystore.ErrForbidden):
		return activityStatusError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	case errors.Is(err, activitystore.ErrNotFound):
		return activityStatusError(404, map[string]any{"detail": "Not found."})
	default:
		return activityStatusError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
}

func activityFields(value any, raw string) any {
	if raw == "" {
		return value
	}
	fields := make(map[string]struct{})
	for _, field := range strings.Split(raw, ",") {
		if field != "" {
			fields[field] = struct{}{}
		}
	}
	if len(fields) == 0 {
		return value
	}
	filter := func(item map[string]any) map[string]any {
		projected := make(map[string]any, len(fields))
		for field := range fields {
			if fieldValue, ok := item[field]; ok {
				projected[field] = fieldValue
			}
		}
		return projected
	}
	projectItems := func(items []any) {
		for index, rawItem := range items {
			if object, ok := rawItem.(map[string]any); ok {
				items[index] = filter(object)
			}
		}
	}
	switch item := value.(type) {
	case []any:
		projectItems(item)
		return item
	case map[string]any:
		if results, paginated := item["results"].([]any); paginated {
			projectItems(results)
			item["results"] = results
			return item
		}
		return filter(item)
	default:
		return value
	}
}

func activityTimestamps(value any, timezone string) any { return djangoapi.Timestamps(value, timezone) }
func activityRawResult(body any, err error) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	if err != nil {
		return nil, nil, err
	}
	raw, marshalErr := json.Marshal(body)
	if marshalErr != nil {
		return nil, nil, fmt.Errorf("marshal activity result: %w", marshalErr)
	}
	return &api.ActivityRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(raw)), nil
}

func decodeActivityError(body map[string]any, name string) *api.ActivityErrorResponseBody {
	result := api.ActivityErrorResponseBody{Name: name}
	raw, _ := json.Marshal(body)
	_ = json.Unmarshal(raw, &result)
	return &result
}

func activityStatusError(status int, body map[string]any) error {
	if body == nil {
		body = map[string]any{"error": "Something went wrong please try again later"}
	}
	name := "status_500"
	switch status {
	case 400:
		name = "status_400"
	case 401:
		name = "status_401"
	case 403:
		name = "status_403"
	case 404:
		name = "status_404"
	case 429:
		name = "status_429"
	}
	return decodeActivityError(body, name)
}

func activityErrorStatusAndBody(err error) (int, map[string]any) {
	response, ok := err.(*api.ActivityErrorResponseBody)
	if !ok {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	status := 500
	switch response.Name {
	case "status_400":
		status = 400
	case "status_401":
		status = 401
	case "status_403":
		status = 403
	case "status_404":
		status = 404
	case "status_429":
		status = 429
	}
	raw, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	body := make(map[string]any)
	if unmarshalErr := json.Unmarshal(raw, &body); unmarshalErr != nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return status, body
}

func activityPayloadRequest(ctx context.Context, operation string, payload any) activitystore.Request {
	request := activitystore.Request{Operation: operation, Params: map[string]string{}, Present: map[string]bool{}}
	value := reflect.ValueOf(payload)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	typ := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := typ.Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || strings.HasSuffix(name, "Credential") {
			continue
		}
		item := value.Field(index)
		if item.Kind() == reflect.Pointer {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}
		name = snakeActivityParameter(name)
		request.Params[name] = fmt.Sprint(item.Interface())
		request.Present[name] = true
	}
	pathParameters := map[string]struct{}{"slug": {}, "project_id": {}, "issue_id": {}, "cycle_id": {}, "module_id": {}, "state_id": {}, "pk": {}}
	for name, values := range ReadServiceQueryValues(ctx) {
		if _, pathParameter := pathParameters[name]; pathParameter {
			continue
		}
		request.Present[name] = true
		if len(values) > 0 {
			request.Params[name] = values[len(values)-1]
		} else {
			request.Params[name] = ""
		}
	}
	return request
}

func snakeActivityParameter(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		upper := character >= 'A' && character <= 'Z'
		if upper && index > 0 {
			previousLowerOrDigit := (value[index-1] >= 'a' && value[index-1] <= 'z') || (value[index-1] >= '0' && value[index-1] <= '9')
			nextLower := index+1 < len(value) && value[index+1] >= 'a' && value[index+1] <= 'z'
			if previousLowerOrDigit || nextLower {
				result.WriteByte('_')
			}
		}
		if upper {
			character += 'a' - 'A'
		}
		result.WriteByte(character)
	}
	return result.String()
}

func (service *activityReadsService) GetAPIUsersMeActivities(ctx context.Context, payload *api.GetAPIUsersMeActivitiesPayload) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	result, err := service.execute(ctx, "get_api_users_me_activities", payload)
	return activityRawResult(result.Body, err)
}
func (service *activityReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesPayload) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	result, err := service.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities", payload)
	return activityRawResult(result.Body, err)
}
func (service *activityReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPkPayload) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	result, err := service.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities_by_pk", payload)
	return activityRawResult(result.Body, err)
}
func (service *activityReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivities(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesPayload) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	result, err := service.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities", payload)
	return activityRawResult(result.Body, err)
}
func (service *activityReadsService) GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesByPkPayload) (*api.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	result, err := service.execute(ctx, "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities_by_pk", payload)
	return activityRawResult(result.Body, err)
}
func (service *activityReadsService) GetAPIUsersMeActivitiesOptions(ctx context.Context) (*api.ActivityOptions, error) {
	principal, err := service.principal(ctx, "get_api_users_me_activities", nil)
	if err != nil {
		return nil, err
	}
	if err := service.userRepository.Authorize(ctx, principal, activitystore.Request{Operation: "get_api_users_me_activities"}); err != nil {
		return nil, mapActivityReadError(err)
	}
	return &api.ActivityOptions{
		Name:        "User Activity Endpoint",
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, nil
}

var _ ActivityReadsProtocolAuthorizer = (*activityReadsService)(nil)
