package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const cycleLifecycleUnauthenticatedDetail = "Authentication credentials were not provided."

type cycleLifecyclePrincipal struct {
	userID   uuid.UUID
	timezone string
	key      string
	apiKey   bool
}

type cycleLifecyclePrincipalKey struct{}

type cycleLifecycleService struct {
	sessions activityReadsSessions
	keys     activityReadsAPIKeys
	throttle activityReadsThrottle
	repo     cycleLifecycleRepository
}

var _ api.Service = (*cycleLifecycleService)(nil)

type cycleLifecycleRepository interface {
	Create(context.Context, string, string, string, store.CycleCreate, bool, map[string]any) (map[string]any, error)
	AuthorizeProject(context.Context, string, string, string) error
	AuthorizeAppProject(context.Context, string, string, string) error
	AuthorizeAppDelete(context.Context, string, string, string, string) error
	AuthorizeReadProject(context.Context, string, string, string) error
	DateCheck(context.Context, string, string, *string, *string, *string) (bool, error)
	SetArchived(context.Context, string, string, string, string, bool) (time.Time, error)
	AuthorizeDelete(context.Context, string, string, string, string, bool) error
	Delete(context.Context, string, string, string, string, bool) error
	Cycle(context.Context, string, string, string) (store.CycleLifecycleResource, error)
	Patch(context.Context, string, string, string, string, bool, map[string]any, map[string]any) (map[string]any, error)
	PutNoop(context.Context, string, string, string, string) (map[string]any, error)
}

func (service *cycleLifecycleService) APIKeyAuth(
	ctx context.Context,
	credential string,
	scheme *security.APIKeyScheme,
) (context.Context, error) {
	if scheme == nil {
		return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
	}
	switch scheme.Name {
	case "ApiKeyAuth":
		principal, err := service.keys.Validate(ctx, credential)
		if err != nil {
			if credential == "" {
				return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, cycleLifecycleError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, cycleLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
		return context.WithValue(ctx, cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
			userID: principal.UserID, timezone: principal.Timezone, key: credential, apiKey: true,
		}), nil
	case "PlaneSessionCookie", "SessionAuth", "DjangoSessionCookie":
		if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
			parsed, err := uuid.Parse(userID)
			if err != nil {
				return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
			}
			return context.WithValue(ctx, cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{userID: parsed, timezone: timezone}), nil
		}
		if credential == "" {
			credential = sessionCredentialFromContext(ctx)
		}
		principal, err := service.sessions.Validate(ctx, credential)
		if err != nil {
			return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
		}
		parsed, err := uuid.Parse(principal.UserID)
		if err != nil {
			return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
		}
		return context.WithValue(ctx, cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{
			userID: parsed, timezone: principal.UserTimezone,
		}), nil
	default:
		return ctx, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
	}
}

func sessionCredentialFromContext(ctx context.Context) string {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok {
		return ""
	}
	request := &http.Request{Header: metadata.Headers()}
	cookie, err := request.Cookie("session-id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func cycleLifecycleError(status int, body map[string]any) error {
	encoded, err := loom.JSONValueFrom(body)
	if err != nil {
		return &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	return &api.CycleLifecycleError{Kind: fmt.Sprintf("status_%d", status), Body: encoded}
}

func cycleLifecycleMissingLegacyObject() error {
	return cycleLifecycleError(404, map[string]any{"error": "The required object does not exist."})
}

func cycleLifecycleActor(ctx context.Context) (cycleLifecyclePrincipal, error) {
	principal, ok := ctx.Value(cycleLifecyclePrincipalKey{}).(cycleLifecyclePrincipal)
	if !ok {
		return cycleLifecyclePrincipal{}, cycleLifecycleError(401, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail})
	}
	return principal, nil
}

func (service *cycleLifecycleService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	if credential == "" {
		return http.StatusUnauthorized, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail}
	}
	principal, err := service.sessions.Validate(ctx, credential)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail}
	}
	if _, err := uuid.Parse(principal.UserID); err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail}
	}
	return 0, nil
}

func (service *cycleLifecycleService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	if status, body := service.AuthorizeSessionProtocol(ctx, credential); status != 0 {
		return status, body
	}
	principal, err := service.sessions.Validate(ctx, credential)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": cycleLifecycleUnauthenticatedDetail}
	}
	if resource.Method == http.MethodGet || resource.Method == http.MethodHead || resource.Method == http.MethodOptions {
		return 0, nil
	}
	if resource.Method == http.MethodPut {
		if err := service.repo.AuthorizeReadProject(ctx, principal.UserID, resource.Slug, resource.ProjectID); err != nil {
			return http.StatusNotFound, map[string]any{"error": "The requested resource does not exist."}
		}
		return 0, nil
	}
	var authorizationErr error
	if resource.Method == http.MethodDelete && resource.Pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/" {
		authorizationErr = service.repo.AuthorizeAppDelete(ctx, resource.Slug, resource.ProjectID, resource.ID, principal.UserID)
	} else {
		authorizationErr = service.repo.AuthorizeAppProject(ctx, principal.UserID, resource.Slug, resource.ProjectID)
	}
	if authorizationErr != nil {
		return http.StatusForbidden, map[string]any{"error": "You don't have the required permissions."}
	}
	return 0, nil
}

func (service *cycleLifecycleService) OwnsSessionProtocolMethod(pattern, method string) bool {
	switch {
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/":
		return method == http.MethodPost
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/date-check/":
		return method == http.MethodPost
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/":
		return method == http.MethodPost || method == http.MethodDelete
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/archived-cycles/":
		return method == http.MethodPost || method == http.MethodDelete
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/":
		return method == http.MethodPost || method == http.MethodDelete
	case pattern == "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/":
		return method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
	default:
		return false
	}
}

func (service *cycleLifecycleService) SessionProtocolResourceExists(ctx context.Context, resource SessionProtocolResource) (bool, error) {
	if resource.Pattern != "/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/" {
		return true, nil
	}
	userID, ok := ReadServiceSessionUserID(ctx)
	if !ok {
		return false, errors.New("cycle OPTIONS requires an authenticated session principal")
	}
	if err := service.repo.AuthorizeReadProject(ctx, userID, resource.Slug, resource.ProjectID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	_, err := service.repo.Cycle(ctx, resource.Slug, resource.ProjectID, resource.ID)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func NewCycleLifecycle(repository cycleLifecycleRepository, sessions activityReadsSessions, keys activityReadsAPIKeys, throttle activityReadsThrottle) (*cycleLifecycleService, error) {
	if repository == nil || sessions == nil || keys == nil || throttle == nil {
		return nil, errors.New("cycle lifecycle repository, session and API-key validators, and throttle are required")
	}
	return &cycleLifecycleService{repo: repository, sessions: sessions, keys: keys, throttle: throttle}, nil
}

func (service *cycleLifecycleService) create(ctx context.Context, slug, projectID string, body io.ReadCloser, apiV1 bool) (*map[string]any, cycleLifecyclePrincipal, error) {
	principal, err := cycleLifecycleActor(ctx)
	if err != nil {
		return nil, cycleLifecyclePrincipal{}, err
	}
	if authorizeProjectErr := service.authorizeProjectRepository(ctx, principal.userID.String(), slug, projectID, apiV1); authorizeProjectErr != nil {
		return nil, principal, cycleLifecycleRepoError(ctx, authorizeProjectErr, apiV1)
	}
	if apiV1 {
		rate, allowErr := service.throttle.Allow(ctx, principal.key)
		if allowErr != nil {
			return nil, principal, cycleLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return nil, principal, cycleLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
		}
		if validateUserTimezoneErr := djangosession.ValidateUserTimezone(principal.timezone); validateUserTimezoneErr != nil {
			return nil, principal, cycleLifecycleError(400, map[string]any{"error": "The required key does not exist."})
		}
	}
	metadata, hasMetadata := loomhttp.RequestMetadataFromContext(ctx)
	contentType := "application/json"
	if hasMetadata && metadata.Headers().Get("Content-Type") != "" {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, principal, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	if input.Kind != "" || input.Null {
		return nil, principal, cycleLifecycleError(400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(input.Kind) + "."}})
	}
	fields := input.Fields
	if (fields["start_date"] != nil) != (fields["end_date"] != nil) {
		return nil, principal, cycleLifecycleError(400, map[string]any{"error": "Both start date and end date are either required or are to be null"})
	}
	validated, validationErrors := validateCycleCreateInput(fields, input.Surrogates, apiV1)
	if len(validationErrors) > 0 {
		return nil, principal, cycleLifecycleError(400, validationErrors)
	}
	cycle, err := service.repo.Create(ctx, principal.userID.String(), slug, projectID, validated, apiV1, fields)
	if err != nil {
		return nil, principal, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	return &cycle, principal, nil
}

// cycleLifecycleLogError records only failure classification and PostgreSQL
// metadata. Request bodies, credentials, and error strings can contain user
// data and are deliberately excluded.
func cycleLifecycleLogError(ctx context.Context, operation, stage string, err error) {
	var httpError *djangoapi.HTTPError
	if errors.As(err, &httpError) && httpError.Status < 500 {
		return
	}
	fields := []log.Fielder{
		log.KV{K: "msg", V: "Cycle lifecycle operation failed"},
		log.KV{K: "operation", V: operation},
		log.KV{K: "stage", V: stage},
		log.KV{K: "error_type", V: fmt.Sprintf("%T", err)},
		log.KV{K: "error_kind", V: cycleLifecycleErrorKind(err)},
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		fields = append(fields,
			log.KV{K: "sqlstate", V: postgres.Code},
			log.KV{K: "table", V: postgres.TableName},
			log.KV{K: "column", V: postgres.ColumnName},
			log.KV{K: "constraint", V: postgres.ConstraintName},
		)
	}
	log.Print(ctx, fields...)
}

func cycleLifecycleErrorKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		return "postgres"
	}
	return "other"
}

func cycleLifecycleRepoError(ctx context.Context, err error, apiV1 bool) error {
	cycleLifecycleLogError(ctx, "request", "repository", err)
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return cycleLifecycleError(response.Status, response.Body)
	}
	if errors.Is(err, store.ErrForbidden) {
		if apiV1 {
			return cycleLifecycleError(403, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		return cycleLifecycleError(403, map[string]any{"error": "You don't have the required permissions."})
	}
	if errors.Is(err, store.ErrNotFound) {
		if !apiV1 {
			return cycleLifecycleError(404, map[string]any{"error": "Cycle not found"})
		}
		return cycleLifecycleError(404, map[string]any{"error": "The requested resource does not exist."})
	}
	return cycleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
}

func cycleLifecycleTypedResponse[T any](value map[string]any) (*T, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func validateCycleCreateInput(fields map[string]any, surrogates map[string][]string, apiV1 bool) (store.CycleCreate, map[string]any) {
	return store.ValidateCreateInput(fields, surrogates, apiV1)
}
func validateCyclePatchInput(fields map[string]any, surrogates map[string][]string, apiV1 bool) (map[string]any, map[string]any) {
	return store.ValidatePatchInput(fields, surrogates, apiV1)
}

func (service *cycleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDCycles(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesPayload, body io.ReadCloser) (*api.PlaneAppSerializersCycleCycleserializerResponse, error) {
	cycle, _, err := service.create(ctx, payload.Slug, payload.ProjectID, body, false)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, io.ErrUnexpectedEOF
	}
	return cycleLifecycleTypedResponse[api.PlaneAppSerializersCycleCycleserializerResponse](*cycle)
}

func (service *cycleLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDCycles(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesPayload, body io.ReadCloser) (*api.PlaneAPISerializersCycleCycleserializerResponse, error) {
	cycle, _, err := service.create(ctx, payload.Slug, payload.ProjectID, body, true)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, io.ErrUnexpectedEOF
	}
	return cycleLifecycleTypedResponse[api.PlaneAPISerializersCycleCycleserializerResponse](*cycle)
}

func (service *cycleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDCyclesDateCheck(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesDateCheckPayload, body io.ReadCloser) (*api.RestFrameworkSerializersCycledatecheckresponseResponse, error) {
	principal, err := cycleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeAppProjectErr := service.repo.AuthorizeAppProject(ctx, principal.userID.String(), payload.Slug, payload.ProjectID); authorizeAppProjectErr != nil {
		return nil, cycleLifecycleRepoError(ctx, authorizeAppProjectErr, false)
	}
	metadata, hasMetadata := loomhttp.RequestMetadataFromContext(ctx)
	contentType := "application/json"
	if hasMetadata && metadata.Headers().Get("Content-Type") != "" {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, false)
	}
	start, startOK := input.Fields["start_date"].(string)
	end, endOK := input.Fields["end_date"].(string)
	if !startOK || start == "" || !endOK || end == "" {
		message := "Start date and end date both are required"
		return &api.RestFrameworkSerializersCycledatecheckresponseResponse{Error: &message}, cycleLifecycleError(400, map[string]any{"error": message})
	}
	var cycleID *string
	if value, ok := input.Fields["cycle_id"].(string); ok && value != "" {
		cycleID = &value
	}
	clear, err := service.repo.DateCheck(ctx, payload.Slug, payload.ProjectID, &start, &end, cycleID)
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, false)
	}
	response := &api.RestFrameworkSerializersCycledatecheckresponseResponse{Status: clear}
	if !clear {
		message := "You have a cycle already on the given dates, if you want to create a draft cycle you can do that by removing dates"
		response.Error = &message
	}
	return response, nil
}

func (service *cycleLifecycleService) authorizeProject(ctx context.Context, slug, projectID string, apiV1 bool) (cycleLifecyclePrincipal, error) {
	principal, err := cycleLifecycleActor(ctx)
	if err != nil {
		return cycleLifecyclePrincipal{}, err
	}
	if err := service.authorizeProjectRepository(ctx, principal.userID.String(), slug, projectID, apiV1); err != nil {
		return principal, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	return principal, nil
}

func (service *cycleLifecycleService) authorizeProjectRepository(ctx context.Context, userID, slug, projectID string, apiV1 bool) error {
	if apiV1 {
		return service.repo.AuthorizeProject(ctx, userID, slug, projectID)
	}
	return service.repo.AuthorizeAppProject(ctx, userID, slug, projectID)
}

// allowAPIRequest applies the source API-key throttle after project permission
// checks, and records the rate result for the shared DRF response middleware.
func (service *cycleLifecycleService) allowAPIRequest(ctx context.Context, principal cycleLifecyclePrincipal) error {
	rate, err := service.throttle.Allow(ctx, principal.key)
	if err != nil {
		return cycleLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return cycleLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return cycleLifecycleError(400, map[string]any{"error": "The required key does not exist."})
	}
	return nil
}

func (service *cycleLifecycleService) archive(ctx context.Context, slug, projectID, cycleID string, apiV1, value bool) (time.Time, error) {
	principal, err := service.authorizeProject(ctx, slug, projectID, apiV1)
	if err != nil {
		return time.Time{}, err
	}
	if apiV1 {
		if allowAPIRequestErr := service.allowAPIRequest(ctx, principal); allowAPIRequestErr != nil {
			return time.Time{}, allowAPIRequestErr
		}
	}
	archivedAt, err := service.repo.SetArchived(ctx, slug, projectID, cycleID, principal.userID.String(), value)
	if err != nil {
		if !apiV1 && errors.Is(err, store.ErrNotFound) {
			return time.Time{}, cycleLifecycleMissingLegacyObject()
		}
		return time.Time{}, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	return archivedAt, nil
}

func (service *cycleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchivePayload) error {
	_, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.CycleID, false, false)
	return err
}

func (service *cycleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchivePayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	archivedAt, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.CycleID, false, true)
	if err != nil {
		return nil, err
	}
	return &api.RestFrameworkSerializersArchiveresponseResponse{ArchivedAt: pythoncompat.PythonDateTimeString(archivedAt)}, nil
}

func (service *cycleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCycles(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesPayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	principal, err := service.authorizeProject(ctx, payload.Slug, payload.ProjectID, false)
	if err != nil {
		return nil, err
	}
	_ = principal
	return nil, cycleLifecycleError(400, map[string]any{"error": "Cycle ID is required"})
}

func (service *cycleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCycles(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesPayload) error {
	_, err := service.authorizeProject(ctx, payload.Slug, payload.ProjectID, false)
	if err != nil {
		return err
	}
	return cycleLifecycleError(400, map[string]any{"error": "Cycle ID is required"})
}

func (service *cycleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPk(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPkPayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	archivedAt, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.Pk, false, true)
	if err != nil {
		return nil, err
	}
	return &api.RestFrameworkSerializersArchiveresponseResponse{ArchivedAt: pythoncompat.PythonDateTimeString(archivedAt)}, nil
}

func (service *cycleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedCyclesByPkPayload) error {
	_, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.Pk, false, false)
	return err
}

func (service *cycleLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesByCycleIDUnarchive(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedCyclesByCycleIDUnarchivePayload) error {
	_, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.CycleID, true, false)
	return err
}

func (service *cycleLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchive(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByCycleIDArchivePayload) error {
	_, err := service.archive(ctx, payload.Slug, payload.ProjectID, payload.CycleID, true, true)
	return err
}

func (service *cycleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDCyclesByPkPayload) error {
	return service.delete(ctx, payload.Slug, payload.ProjectID, payload.Pk, false)
}

func (service *cycleLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPkPayload) error {
	return service.delete(ctx, payload.Slug, payload.ProjectID, payload.Pk, true)
}

func (service *cycleLifecycleService) delete(ctx context.Context, slug, projectID, cycleID string, apiV1 bool) error {
	principal, err := cycleLifecycleActor(ctx)
	if err != nil {
		return err
	}
	if apiV1 {
		if err := service.repo.AuthorizeProject(ctx, principal.userID.String(), slug, projectID); err != nil {
			return cycleLifecycleRepoError(ctx, err, true)
		}
	} else if err := service.repo.AuthorizeAppDelete(ctx, slug, projectID, cycleID, principal.userID.String()); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return cycleLifecycleMissingLegacyObject()
		}
		return cycleLifecycleRepoError(ctx, err, false)
	}
	if apiV1 {
		if err := service.allowAPIRequest(ctx, principal); err != nil {
			return err
		}
	}
	if apiV1 {
		if err := service.repo.AuthorizeDelete(ctx, slug, projectID, cycleID, principal.userID.String(), true); err != nil {
			if apiV1 && errors.Is(err, store.ErrForbidden) {
				return cycleLifecycleError(403, map[string]any{"error": "Only admin or creator can delete the cycle"})
			}
			return cycleLifecycleRepoError(ctx, err, apiV1)
		}
	}
	if err := service.repo.Delete(ctx, slug, projectID, cycleID, principal.userID.String(), apiV1); err != nil {
		if !apiV1 && errors.Is(err, store.ErrNotFound) {
			return cycleLifecycleMissingLegacyObject()
		}
		return cycleLifecycleRepoError(ctx, err, apiV1)
	}
	return nil
}

func (service *cycleLifecycleService) patch(ctx context.Context, slug, projectID, cycleID string, body io.ReadCloser, apiV1 bool) (map[string]any, error) {
	principal, err := service.authorizeProject(ctx, slug, projectID, apiV1)
	if err != nil {
		return nil, err
	}
	if apiV1 {
		rate, allowErr := service.throttle.Allow(ctx, principal.key)
		if allowErr != nil {
			return nil, cycleLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return nil, cycleLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
		}
		if validateUserTimezoneErr := djangosession.ValidateUserTimezone(principal.timezone); validateUserTimezoneErr != nil {
			return nil, cycleLifecycleError(400, map[string]any{"error": "The required key does not exist."})
		}
	}
	cycle, err := service.repo.Cycle(ctx, slug, projectID, cycleID)
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	completed := cycle.EndDate != nil && cycle.EndDate.Before(time.Now())
	metadata, hasMetadata := loomhttp.RequestMetadataFromContext(ctx)
	contentType := "application/json"
	if hasMetadata && metadata.Headers().Get("Content-Type") != "" {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	if input.Kind != "" || input.Null {
		return nil, cycleLifecycleError(400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(input.Kind) + "."}})
	}
	validationInput := input.Fields
	validationSurrogates := input.Surrogates
	if completed {
		validationInput = map[string]any{}
		validationSurrogates = map[string][]string{}
		if value, present := input.Fields["sort_order"]; present {
			validationInput["sort_order"] = value
			validationSurrogates["sort_order"] = input.Surrogates["sort_order"]
		}
	}
	fields, validationErrors := validateCyclePatchInput(validationInput, validationSurrogates, apiV1)
	if len(validationErrors) > 0 {
		return nil, cycleLifecycleError(400, validationErrors)
	}
	activityRequest := maps.Clone(input.Fields)
	result, err := service.repo.Patch(ctx, slug, projectID, cycleID, principal.userID.String(), apiV1, fields, activityRequest)
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, apiV1)
	}
	return result, nil
}

func (service *cycleLifecycleService) PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugProjectsByProjectIDCyclesByPkPayload, body io.ReadCloser) (*api.PlaneAppSerializersCycleCycleserializerResponse, error) {
	cycle, err := service.patch(ctx, payload.Slug, payload.ProjectID, payload.Pk, body, false)
	if err != nil {
		return nil, err
	}
	return cycleLifecycleTypedResponse[api.PlaneAppSerializersCycleCycleserializerResponse](cycle)
}

func (service *cycleLifecycleService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDCyclesByPkPayload, body io.ReadCloser) (*api.PlaneAPISerializersCycleCycleserializerResponse, error) {
	cycle, err := service.patch(ctx, payload.Slug, payload.ProjectID, payload.Pk, body, true)
	if err != nil {
		return nil, err
	}
	return cycleLifecycleTypedResponse[api.PlaneAPISerializersCycleCycleserializerResponse](cycle)
}

func (service *cycleLifecycleService) PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByPk(ctx context.Context, payload *api.PutAPIWorkspacesBySlugProjectsByProjectIDCyclesByPkPayload, body io.ReadCloser) (*api.CycleAppReadonlyPutResponse, error) {
	principal, err := cycleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeReadProjectErr := service.repo.AuthorizeReadProject(ctx, principal.userID.String(), payload.Slug, payload.ProjectID); authorizeReadProjectErr != nil {
		if errors.Is(authorizeReadProjectErr, store.ErrNotFound) {
			return nil, cycleLifecycleError(404, map[string]any{"detail": "No Cycle matches the given query."})
		}
		return nil, cycleLifecycleRepoError(ctx, authorizeReadProjectErr, false)
	}
	if _, cycleErr := service.repo.Cycle(ctx, payload.Slug, payload.ProjectID, payload.Pk); cycleErr != nil {
		if errors.Is(cycleErr, store.ErrNotFound) {
			return nil, cycleLifecycleError(404, map[string]any{"detail": "No Cycle matches the given query."})
		}
		return nil, cycleLifecycleRepoError(ctx, cycleErr, false)
	}
	metadata, hasMetadata := loomhttp.RequestMetadataFromContext(ctx)
	contentType := "application/json"
	if hasMetadata && metadata.Headers().Get("Content-Type") != "" {
		contentType = metadata.Headers().Get("Content-Type")
	}
	if _, parseErr := djangoinput.Parse(body, contentType); parseErr != nil {
		return nil, cycleLifecycleRepoError(ctx, parseErr, false)
	}
	cycle, err := service.repo.PutNoop(ctx, payload.Slug, payload.ProjectID, payload.Pk, principal.userID.String())
	if err != nil {
		return nil, cycleLifecycleRepoError(ctx, err, false)
	}
	return cycleLifecycleTypedResponse[api.CycleAppReadonlyPutResponse](cycle)
}
