package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	api "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const moduleLifecycleUnauthenticatedDetail = "Authentication credentials were not provided."

type moduleLifecyclePrincipal struct {
	userID   uuid.UUID
	timezone string
	key      string
	apiKey   bool
}
type moduleLifecyclePrincipalKey struct{}

type moduleLifecycleRepository interface {
	AuthorizeEntityProject(context.Context, string, string, string) error
	AuthorizeAppProject(context.Context, string, string, string) error
	CheckModuleProject(context.Context, string, string, bool) error
	CheckModule(context.Context, string, string, string, bool, bool) error
	CreateModule(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error)
	CreateModuleV1(context.Context, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error)
	UpdateModule(context.Context, string, string, string, string, map[string]any, map[string][]string, bool, string) (json.RawMessage, error)
	UpdateModuleV1(context.Context, string, string, string, string, map[string]any, map[string][]string, string) (json.RawMessage, error)
	SetModuleArchived(context.Context, string, string, string, string, bool) (time.Time, error)
	DeleteModule(context.Context, string, string, string, string, bool, string) error
	CheckDeleteModule(context.Context, string, string, string, string, bool) error
	CreateLink(context.Context, string, string, string, string, map[string]any, map[string][]string) (json.RawMessage, error)
	CheckLink(context.Context, string, string, string, string, string) error
	UpdateLink(context.Context, string, string, string, string, string, map[string]any, map[string][]string, bool) (json.RawMessage, error)
	DeleteLink(context.Context, string, string, string, string, string) error
}
type moduleLifecycleService struct {
	repo     moduleLifecycleRepository
	sessions activityReadsSessions
	keys     activityReadsAPIKeys
	throttle activityReadsThrottle
}

var _ api.Service = (*moduleLifecycleService)(nil)

func NewModuleLifecycle(repo moduleLifecycleRepository, sessions activityReadsSessions, keys activityReadsAPIKeys, throttle activityReadsThrottle) (*moduleLifecycleService, error) {
	if repo == nil || sessions == nil || keys == nil || throttle == nil {
		return nil, errors.New("module lifecycle repository, session and API-key validators, and throttle are required")
	}
	return &moduleLifecycleService{repo: repo, sessions: sessions, keys: keys, throttle: throttle}, nil
}

func moduleLifecycleError(status int, body map[string]any) error {
	encoded, err := loom.JSONValueFrom(body)
	if err != nil {
		encoded, _ = loom.JSONValueFrom(map[string]any{"error": "Something went wrong please try again later"})
		status = 500
	}
	return &api.ModuleLifecycleError{Kind: fmt.Sprintf("status_%d", status), Body: encoded}
}
func moduleLifecycleRepoError(ctx context.Context, err error, apiV1 bool) error {
	var generated *api.ModuleLifecycleError
	if errors.As(err, &generated) {
		return err
	}
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return moduleLifecycleError(response.Status, response.Body)
	}
	if errors.Is(err, store.ErrForbidden) {
		if apiV1 {
			return moduleLifecycleError(403, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		return moduleLifecycleError(403, map[string]any{"error": "You don't have the required permissions."})
	}
	if errors.Is(err, store.ErrNotFound) {
		if apiV1 {
			return moduleLifecycleError(404, map[string]any{"error": "The requested resource does not exist."})
		}
		return moduleLifecycleError(404, map[string]any{"error": "The required object does not exist."})
	}
	var validation *store.ModuleValidationError
	if errors.As(err, &validation) {
		return moduleLifecycleError(400, validation.Body)
	}
	moduleLifecycleLogError(ctx, err)
	return moduleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
}

// moduleLifecycleLogError records only failure classification and PostgreSQL
// metadata. It omits request bodies, credentials, and error text.
func moduleLifecycleLogError(ctx context.Context, err error) {
	var response *djangoapi.HTTPError
	if errors.As(err, &response) && response.Status < 500 {
		return
	}
	kind := "other"
	switch {
	case errors.Is(err, context.Canceled):
		kind = "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		kind = "deadline_exceeded"
	}
	fields := []log.Fielder{
		log.KV{K: "msg", V: "Module lifecycle operation failed"},
		log.KV{K: "stage", V: "repository"},
		log.KV{K: "error_type", V: fmt.Sprintf("%T", err)},
		log.KV{K: "error_kind", V: kind},
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		fields[len(fields)-1] = log.KV{K: "error_kind", V: "postgres"}
		fields = append(fields,
			log.KV{K: "sqlstate", V: postgres.Code},
			log.KV{K: "table", V: postgres.TableName},
			log.KV{K: "column", V: postgres.ColumnName},
			log.KV{K: "constraint", V: postgres.ConstraintName},
		)
	}
	log.Print(ctx, fields...)
}
func moduleLifecycleActor(ctx context.Context) (moduleLifecyclePrincipal, error) {
	p, ok := ctx.Value(moduleLifecyclePrincipalKey{}).(moduleLifecyclePrincipal)
	if ok {
		return p, nil
	}
	if id, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		parsed, err := uuid.Parse(id)
		if err == nil {
			return moduleLifecyclePrincipal{userID: parsed, timezone: timezone}, nil
		}
	}
	return moduleLifecyclePrincipal{}, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
}
func (s *moduleLifecycleService) APIKeyAuth(ctx context.Context, credential string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme == nil {
		return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
	switch scheme.Name {
	case "ApiKeyAuth":
		p, err := s.keys.Validate(ctx, credential)
		if err != nil {
			if credential == "" {
				return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, moduleLifecycleError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, moduleLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
		return context.WithValue(ctx, moduleLifecyclePrincipalKey{}, moduleLifecyclePrincipal{userID: p.UserID, timezone: p.Timezone, key: credential, apiKey: true}), nil
	case "PlaneSessionCookie", "SessionAuth", "DjangoSessionCookie":
		if id, zone, inspected := readServiceSessionPrincipal(ctx); inspected {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
			}
			return context.WithValue(ctx, moduleLifecyclePrincipalKey{}, moduleLifecyclePrincipal{userID: parsed, timezone: zone}), nil
		}
		if credential == "" {
			if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
				request := &http.Request{Header: metadata.Headers()}
				if cookie, err := request.Cookie("session-id"); err == nil {
					credential = cookie.Value
				}
			}
		}
		p, err := s.sessions.Validate(ctx, credential)
		if err != nil {
			return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
		}
		id, err := uuid.Parse(p.UserID)
		if err != nil {
			return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
		}
		return context.WithValue(ctx, moduleLifecyclePrincipalKey{}, moduleLifecyclePrincipal{userID: id, timezone: p.UserTimezone}), nil
	default:
		return ctx, moduleLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
}

// AuthorizeSessionProtocol supplies the authenticated unsupported-method
// boundary for module lifecycle paths before generated route dispatch.
func (s *moduleLifecycleService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	if credential == "" {
		return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}
	}
	principal, err := s.sessions.Validate(ctx, credential)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}
	}
	if _, err := uuid.Parse(principal.UserID); err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}
	}
	return 0, nil
}

func (s *moduleLifecycleService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	if status, body := s.AuthorizeSessionProtocol(ctx, credential); status != 0 {
		return status, body
	}
	principal, err := s.sessions.Validate(ctx, credential)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}
	}
	if resource.Method == http.MethodOptions {
		return 0, nil
	}
	entityPermission := strings.Contains(resource.Pattern, "/module-links/") || strings.Contains(resource.Pattern, "/archive/") || strings.Contains(resource.Pattern, "/archived-modules")
	if !entityPermission {
		// Unsupported ordinary ModuleViewSet methods have no action decorator;
		// Django authenticates the request and returns 405 without resolving a
		// project or applying a write permission. Supported methods are checked
		// in their generated service methods before body parsing.
		return 0, nil
	}
	authorizeErr := s.repo.AuthorizeEntityProject(ctx, userID.String(), resource.Slug, resource.ProjectID)
	if authorizeErr != nil {
		if errors.Is(authorizeErr, store.ErrForbidden) {
			if entityPermission {
				return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
			}
			return http.StatusForbidden, map[string]any{"error": "You don't have the required permissions."}
		}
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
	}
	return 0, nil
}

// SessionProtocolResourceExists lets OPTIONS mirror DRF metadata responses for
// missing detail resources while leaving the status at 200.
func (s *moduleLifecycleService) SessionProtocolResourceExists(ctx context.Context, resource SessionProtocolResource) (bool, error) {
	switch resource.Pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":
		checker, ok := s.repo.(interface {
			ModuleExists(context.Context, string, string, string) (bool, error)
		})
		if !ok {
			return true, nil
		}
		return checker.ModuleExists(ctx, resource.Slug, resource.ProjectID, resource.ID)
	case "/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/":
		checker, ok := s.repo.(interface {
			LinkExists(context.Context, string, string, string, string) (bool, error)
		})
		if !ok {
			return true, nil
		}
		return checker.LinkExists(ctx, resource.Slug, resource.ProjectID, resource.PathParameters["module_id"], resource.ID)
	default:
		return true, nil
	}
}

// SessionProtocolOptionsMetadata selects the source metadata variant for
// module links. Guests can inspect the route but do not receive serializer
// action fields; app module metadata exposes its source fields to all roles.
func (s *moduleLifecycleService) SessionProtocolOptionsMetadata(ctx context.Context, credential string, resource SessionProtocolResource, metadata, guest json.RawMessage) (int, any, bool) {
	if !strings.Contains(resource.Pattern, "/module-links/") || len(guest) == 0 {
		return 0, nil, false
	}
	principal, err := moduleLifecycleActor(ctx)
	if err != nil {
		if credential == "" {
			return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}, true
		}
		validated, validateErr := s.sessions.Validate(ctx, credential)
		if validateErr != nil {
			return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}, true
		}
		userID, parseErr := uuid.Parse(validated.UserID)
		if parseErr != nil {
			return http.StatusUnauthorized, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail}, true
		}
		principal = moduleLifecyclePrincipal{userID: userID, timezone: validated.UserTimezone}
	}
	authorizeErr := s.repo.AuthorizeEntityProject(ctx, principal.userID.String(), resource.Slug, resource.ProjectID)
	if errors.Is(authorizeErr, store.ErrForbidden) {
		return 200, guest, true
	}
	if authorizeErr != nil {
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}, true
	}
	return 200, metadata, true
}

func (s *moduleLifecycleService) parse(ctx context.Context, body io.ReadCloser) (djangoinput.Input, error) {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	contentType := "application/json"
	if ok && metadata.Headers().Get("Content-Type") != "" {
		contentType = metadata.Headers().Get("Content-Type")
	}
	in, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return in, moduleLifecycleRepoError(ctx, err, false)
	}
	if in.Kind != "" || in.Null {
		return in, moduleLifecycleError(400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(in.Kind) + "."}})
	}
	return in, nil
}
func moduleLifecycleTyped[T any](raw json.RawMessage) (*T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, moduleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return &v, nil
}
func (s *moduleLifecycleService) authorize(ctx context.Context, p moduleLifecyclePrincipal, slug, project string, app bool) error {
	var err error
	if app {
		err = s.repo.AuthorizeAppProject(ctx, p.userID.String(), slug, project)
	} else {
		err = s.repo.AuthorizeEntityProject(ctx, p.userID.String(), slug, project)
	}
	if err != nil {
		return moduleLifecycleRepoError(ctx, err, !app)
	}
	return nil
}
func (s *moduleLifecycleService) moduleCreate(ctx context.Context, slug, project string, body io.ReadCloser, app bool) (json.RawMessage, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, app); err != nil {
		return nil, err
	}
	if !app {
		if moduleAPILimitErr := s.moduleAPILimit(ctx, p); moduleAPILimitErr != nil {
			return nil, moduleAPILimitErr
		}
	}
	if app {
		if checkModuleProjectErr := s.repo.CheckModuleProject(ctx, slug, project, false); checkModuleProjectErr != nil {
			return nil, moduleLifecycleRepoError(ctx, checkModuleProjectErr, false)
		}
	}
	in, err := s.parse(ctx, body)
	if err != nil {
		return nil, err
	}
	raw, err := s.repo.CreateModule(ctx, p.userID.String(), slug, project, in.Fields, in.Surrogates, moduleLifecycleOrigin(ctx))
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, !app)
	}
	return raw, nil
}

func (s *moduleLifecycleService) moduleCreateV1(ctx context.Context, slug, project string, body io.ReadCloser) (json.RawMessage, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, false); err != nil {
		return nil, err
	}
	if err = s.moduleAPILimit(ctx, p); err != nil {
		return nil, err
	}
	if err = s.repo.CheckModuleProject(ctx, slug, project, true); err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, true)
	}
	in, err := s.parse(ctx, body)
	if err != nil {
		return nil, err
	}
	raw, err := s.repo.CreateModuleV1(ctx, p.userID.String(), slug, project, in.Fields, in.Surrogates, moduleLifecycleOrigin(ctx))
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, true)
	}
	return raw, nil
}

func (s *moduleLifecycleService) moduleUpdateV1(ctx context.Context, slug, project, id string, body io.ReadCloser) (json.RawMessage, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, false); err != nil {
		return nil, err
	}
	if err = s.moduleAPILimit(ctx, p); err != nil {
		return nil, err
	}
	if err = s.repo.CheckModule(ctx, slug, project, id, true, false); err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, true)
	}
	in, err := s.parse(ctx, body)
	if err != nil {
		return nil, err
	}
	raw, err := s.repo.UpdateModuleV1(ctx, p.userID.String(), slug, project, id, in.Fields, in.Surrogates, moduleLifecycleOrigin(ctx))
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, true)
	}
	return raw, nil
}

func (s *moduleLifecycleService) moduleAPILimit(ctx context.Context, p moduleLifecyclePrincipal) error {
	rate, err := s.throttle.Allow(ctx, p.key)
	if err != nil {
		return moduleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return moduleLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
		return moduleLifecycleError(400, map[string]any{"error": "The required key does not exist."})
	}
	return nil
}

func moduleLifecycleOrigin(context.Context) string {
	// Django's base_host(request, is_app=True) is configured from APP_BASE_URL
	// or WEB_URL; it does not trust the request Origin header. The repository
	// supplies that configured value for durable activity payloads.
	return ""
}
func (s *moduleLifecycleService) moduleUpdate(ctx context.Context, slug, project, id string, body io.ReadCloser, replace bool) (json.RawMessage, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, true); err != nil {
		return nil, err
	}
	if checkModuleErr := s.repo.CheckModule(ctx, slug, project, id, false, replace); checkModuleErr != nil {
		return nil, moduleLifecycleRepoError(ctx, checkModuleErr, false)
	}
	in, err := s.parse(ctx, body)
	if err != nil {
		return nil, err
	}
	raw, err := s.repo.UpdateModule(ctx, p.userID.String(), slug, project, id, in.Fields, in.Surrogates, replace, moduleLifecycleOrigin(ctx))
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, false)
	}
	return raw, nil
}
func (s *moduleLifecycleService) archive(ctx context.Context, slug, project, id string, archive bool, app bool) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, false); err != nil {
		return nil, err
	}
	if !app {
		if err = s.moduleAPILimit(ctx, p); err != nil {
			return nil, err
		}
	}
	if id == "" {
		return nil, moduleLifecycleError(400, map[string]any{"error": "Module ID is required"})
	}
	t, err := s.repo.SetModuleArchived(ctx, p.userID.String(), slug, project, id, archive)
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, !app)
	}
	if !archive {
		return &api.RestFrameworkSerializersArchiveresponseResponse{ArchivedAt: api.PlanePythonDateTimeString("")}, nil
	}
	return &api.RestFrameworkSerializersArchiveresponseResponse{ArchivedAt: api.PlanePythonDateTimeString(pythoncompat.PythonDateTimeString(t))}, nil
}
func (s *moduleLifecycleService) deleteModule(ctx context.Context, slug, project, id string, apiV1 bool) error {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return err
	}
	if apiV1 {
		if err = s.authorize(ctx, p, slug, project, false); err != nil {
			return err
		}
		if err = s.moduleAPILimit(ctx, p); err != nil {
			return err
		}
	}
	if err = s.repo.CheckDeleteModule(ctx, p.userID.String(), slug, project, id, apiV1); err != nil {
		return moduleLifecycleRepoError(ctx, err, apiV1)
	}
	if err = s.repo.DeleteModule(ctx, p.userID.String(), slug, project, id, apiV1, moduleLifecycleOrigin(ctx)); err != nil {
		return moduleLifecycleRepoError(ctx, err, apiV1)
	}
	return nil
}
func (s *moduleLifecycleService) link(ctx context.Context, slug, project, module string, body io.ReadCloser, method string, linkID string) (json.RawMessage, error) {
	p, err := moduleLifecycleActor(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, p, slug, project, false); err != nil {
		return nil, err
	}
	if method != "POST" {
		if err = s.repo.CheckLink(ctx, p.userID.String(), slug, project, module, linkID); err != nil {
			return nil, moduleLifecycleRepoError(ctx, err, false)
		}
	}
	in, err := s.parse(ctx, body)
	if err != nil {
		return nil, err
	}
	if method == "POST" {
		raw, createLinkErr := s.repo.CreateLink(ctx, p.userID.String(), slug, project, module, in.Fields, in.Surrogates)
		if createLinkErr != nil {
			return nil, moduleLifecycleRepoError(ctx, createLinkErr, false)
		}
		return raw, nil
	}
	raw, err := s.repo.UpdateLink(ctx, p.userID.String(), slug, project, module, linkID, in.Fields, in.Surrogates, method == "PUT")
	if err != nil {
		return nil, moduleLifecycleRepoError(ctx, err, false)
	}
	return raw, nil
}

func (s *moduleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDModules(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDModulesPayload, b io.ReadCloser) (*api.AppModuleComputedResponse, error) {
	r, e := s.moduleCreate(ctx, p.Slug, p.ProjectID, b, true)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.AppModuleComputedResponse](r)
}
func (s *moduleLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDModules(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesPayload, b io.ReadCloser) (*api.PlaneAPISerializersModuleModuleserializerResponse, error) {
	r, e := s.moduleCreateV1(ctx, p.Slug, p.ProjectID, b)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.PlaneAPISerializersModuleModuleserializerResponse](r)
}
func (s *moduleLifecycleService) PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByPkPayload, b io.ReadCloser) (*api.AppModuleComputedResponse, error) {
	r, e := s.moduleUpdate(ctx, p.Slug, p.ProjectID, p.Pk, b, false)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.AppModuleComputedResponse](r)
}
func (s *moduleLifecycleService) PutAPIWorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, p *api.PutAPIWorkspacesBySlugProjectsByProjectIDModulesByPkPayload, b io.ReadCloser) (*api.PlaneAppSerializersModuleModulewriteserializerResponse, error) {
	r, e := s.moduleUpdate(ctx, p.Slug, p.ProjectID, p.Pk, b, true)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.PlaneAppSerializersModuleModulewriteserializerResponse](r)
}
func (s *moduleLifecycleService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkPayload, b io.ReadCloser) (*api.ModuleUpdateSerializerResponse, error) {
	r, e := s.moduleUpdateV1(ctx, p.Slug, p.ProjectID, p.Pk, b)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.ModuleUpdateSerializerResponse](r)
}

func (s *moduleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinks(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksPayload, b io.ReadCloser) (*api.AppModuleLinkCreateResponse, error) {
	r, e := s.link(ctx, p.Slug, p.ProjectID, p.ModuleID, b, "POST", "")
	if e != nil {
		return nil, e
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(r, &body); err != nil {
		return nil, moduleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	body["location"] = append(json.RawMessage(nil), body["url"]...)
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, moduleLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return moduleLifecycleTyped[api.AppModuleLinkCreateResponse](encoded)
}
func (s *moduleLifecycleService) PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPkPayload, b io.ReadCloser) (*api.PlaneAppSerializersModuleModulelinkserializerResponse, error) {
	r, e := s.link(ctx, p.Slug, p.ProjectID, p.ModuleID, b, "PATCH", p.Pk)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.PlaneAppSerializersModuleModulelinkserializerResponse](r)
}
func (s *moduleLifecycleService) PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk(ctx context.Context, p *api.PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPkPayload, b io.ReadCloser) (*api.PlaneAppSerializersModuleModulelinkserializerResponse, error) {
	r, e := s.link(ctx, p.Slug, p.ProjectID, p.ModuleID, b, "PUT", p.Pk)
	if e != nil {
		return nil, e
	}
	return moduleLifecycleTyped[api.PlaneAppSerializersModuleModulelinkserializerResponse](r)
}
func (s *moduleLifecycleService) deleteLink(ctx context.Context, slug, project, module, id string) error {
	p, e := moduleLifecycleActor(ctx)
	if e != nil {
		return e
	}
	if e = s.authorize(ctx, p, slug, project, false); e != nil {
		return e
	}
	if e = s.repo.DeleteLink(ctx, p.userID.String(), slug, project, module, id); e != nil {
		if errors.Is(e, store.ErrNotFound) {
			return moduleLifecycleError(404, map[string]any{"detail": "No ModuleLink matches the given query."})
		}
		return moduleLifecycleRepoError(ctx, e, false)
	}
	return nil
}
func (s *moduleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPkPayload) error {
	return s.deleteLink(ctx, p.Slug, p.ProjectID, p.ModuleID, p.Pk)
}

func (s *moduleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByPkPayload) error {
	return s.deleteModule(ctx, p.Slug, p.ProjectID, p.Pk, false)
}
func (s *moduleLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkPayload) error {
	return s.deleteModule(ctx, p.Slug, p.ProjectID, p.Pk, true)
}
func (s *moduleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchive(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchivePayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	return s.archive(ctx, p.Slug, p.ProjectID, p.ModuleID, true, true)
}
func (s *moduleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchive(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchivePayload) error {
	_, e := s.archive(ctx, p.Slug, p.ProjectID, p.ModuleID, false, true)
	return e
}
func (s *moduleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPk(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPkPayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	return s.archive(ctx, p.Slug, p.ProjectID, p.Pk, true, true)
}
func (s *moduleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPkPayload) error {
	_, e := s.archive(ctx, p.Slug, p.ProjectID, p.Pk, false, true)
	return e
}
func (s *moduleLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkArchive(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkArchivePayload) error {
	_, e := s.archive(ctx, p.Slug, p.ProjectID, p.Pk, true, false)
	return e
}
func (s *moduleLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesByPkUnarchive(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesByPkUnarchivePayload) error {
	_, e := s.archive(ctx, p.Slug, p.ProjectID, p.Pk, false, false)
	return e
}
func (s *moduleLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModules(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesPayload) (*api.RestFrameworkSerializersArchiveresponseResponse, error) {
	return s.archive(ctx, p.Slug, p.ProjectID, "", true, true)
}
func (s *moduleLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModules(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesPayload) error {
	_, e := s.archive(ctx, p.Slug, p.ProjectID, "", false, true)
	return e
}
