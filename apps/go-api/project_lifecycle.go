package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	storeuuid "uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	store "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type projectLifecycleRepository interface {
	AuthorizeProjectWrite(context.Context, string, string, string, string, bool) error
	CheckProjectMutation(context.Context, string, string, string, bool) error
	CreateProject(context.Context, string, string, djangoinput.Input, bool) (json.RawMessage, error)
	UpdateProject(context.Context, string, string, string, djangoinput.Input, bool, bool) (json.RawMessage, error)
	DeleteProject(context.Context, string, string, string, bool) error
	SetProjectArchived(context.Context, string, string, string, bool) (time.Time, error)

	AuthorizeAPIProjectRead(context.Context, storeuuid.UUID, string, bool) error
	APIProjectList(context.Context, storeuuid.UUID, string, store.APIProjectListOptions) (json.RawMessage, error)
	APIProject(context.Context, storeuuid.UUID, string, storeuuid.UUID, store.APIProjectDetailOptions) (json.RawMessage, error)
	APIProjects(context.Context, storeuuid.UUID, string, store.APIProjectListOptions) (json.RawMessage, error)
	APIProjectSummary(context.Context, storeuuid.UUID, string, storeuuid.UUID, store.APIProjectSummaryOptions) (json.RawMessage, error)
}
type projectLifecycleService struct {
	repo     projectLifecycleRepository
	sessions activityReadsSessions
	keys     activityReadsAPIKeys
	throttle activityReadsThrottle
}
type projectLifecyclePrincipalKey struct{}

func NewProjectLifecycle(repo projectLifecycleRepository, sessions activityReadsSessions, keys activityReadsAPIKeys, throttle activityReadsThrottle) (*projectLifecycleService, error) {
	if repo == nil || sessions == nil || keys == nil || throttle == nil {
		return nil, errors.New("project lifecycle repository, session and API-key validators and throttle are required")
	}
	return &projectLifecycleService{repo: repo, sessions: sessions, keys: keys, throttle: throttle}, nil
}
func projectLifecycleError(status int, body map[string]any) error {
	value := map[string]loom.JSONValue{}
	for key, item := range body {
		encoded, err := loom.JSONValueFrom(item)
		if err != nil {
			panic(err)
		}
		value[key] = encoded
	}
	return &api.ProjectLifecycleErrorIdentity{Kind: fmt.Sprintf("status_%d", status), Body: value}
}
func projectLifecycleRepoError(ctx context.Context, err error) error {
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return projectLifecycleError(response.Status, response.Body)
	}
	if errors.Is(err, store.ErrForbidden) {
		return projectLifecycleError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	}
	if errors.Is(err, store.ErrNotFound) {
		return projectLifecycleError(404, map[string]any{"error": "The requested resource does not exist."})
	}
	moduleLifecycleLogError(ctx, err)
	return projectLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
}
func (s *projectLifecycleService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme != nil && scheme.Name == "ApiKeyAuth" {
		principal, err := s.keys.Validate(ctx, key)
		if err != nil {
			if key == "" {
				return ctx, projectLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, projectLifecycleError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, projectLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
		return context.WithValue(ctx, projectLifecyclePrincipalKey{}, moduleLifecyclePrincipal{userID: principal.UserID, timezone: principal.Timezone, key: key, apiKey: true}), nil
	}
	// Reuse the session authentication boundary, with this family's error type.
	helper := moduleLifecycleService{sessions: s.sessions}
	authenticated, err := helper.APIKeyAuth(ctx, key, scheme)
	if err != nil {
		return ctx, projectLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
	p, err := moduleLifecycleActor(authenticated)
	if err != nil {
		return ctx, projectLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
	return context.WithValue(authenticated, projectLifecyclePrincipalKey{}, p), nil
}
func (s *projectLifecycleService) read(ctx context.Context, operation string, payload any) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	p, ok := ctx.Value(projectLifecyclePrincipalKey{}).(moduleLifecyclePrincipal)
	if !ok {
		return nil, nil, projectLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
	request := activityPayloadRequest(ctx, operation, payload)
	slug := request.Params["slug"]
	if err := s.repo.AuthorizeAPIProjectRead(ctx, storeuuid.MustParse(p.userID.String()), slug, operation == "summary"); err != nil {
		return nil, nil, projectLifecycleRepoError(ctx, err)
	}
	rate, err := s.throttle.Allow(ctx, p.key)
	if err != nil {
		return nil, nil, projectLifecycleRepoError(ctx, err)
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return nil, nil, projectLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
		return nil, nil, projectLifecycleError(400, map[string]any{"error": "The required key does not exist."})
	}
	get := func(key string) *string {
		if !request.Present[key] {
			return nil
		}
		value := request.Params[key]
		return &value
	}
	fields := func(key string) []string {
		values := []string{}
		for _, v := range strings.Split(request.Params[key], ",") {
			if v != "" {
				values = append(values, v)
			}
		}
		return values
	}
	var body json.RawMessage
	switch operation {
	case "api-list":
		body, err = s.repo.APIProjectList(ctx, storeuuid.MustParse(p.userID.String()), slug, store.APIProjectListOptions{PerPage: get("per_page"), Cursor: get("cursor"), OrderBy: get("order_by"), Fields: request.Params["fields"], Expand: request.Params["expand"]})
	case "api-detail":
		id, parseErr := uuid.Parse(request.Params["pk"])
		if parseErr != nil {
			return nil, nil, projectLifecycleError(404, map[string]any{"error": "The requested resource does not exist."})
		}
		body, err = s.repo.APIProject(ctx, storeuuid.MustParse(p.userID.String()), slug, storeuuid.MustParse(id.String()), store.APIProjectDetailOptions{Fields: request.Params["fields"], Expand: request.Params["expand"]})
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, projectLifecycleError(404, map[string]any{"error": "The requested resource does not exist."})
		}

	case "lite":
		include := strings.ToLower(request.Params["include_archived"])
		body, err = s.repo.APIProjects(ctx, storeuuid.MustParse(p.userID.String()), slug, store.APIProjectListOptions{IncludeArchived: include == "true" || include == "1", PerPage: get("per_page"), Cursor: get("cursor"), OrderBy: get("order_by")})
	case "summary":
		id, parseErr := uuid.Parse(request.Params["project_id"])
		if parseErr != nil {
			return nil, nil, projectLifecycleRepoError(ctx, store.ErrNotFound)
		}
		body, err = s.repo.APIProjectSummary(ctx, storeuuid.MustParse(p.userID.String()), slug, storeuuid.MustParse(id.String()), store.APIProjectSummaryOptions{Fields: fields("fields")})
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, projectLifecycleError(404, map[string]any{"error": "Project not found"})
		}
	}
	if err != nil {
		return nil, nil, projectLifecycleRepoError(ctx, err)
	}
	return projectLifecycleRawResult(body, nil)
}
func (s *projectLifecycleService) GetAPIV1WorkspacesBySlugProjectsLite(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsLitePayload) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.read(ctx, "lite", p)
}

func (s *projectLifecycleService) GetAPIV1WorkspacesBySlugProjects(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsPayload) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.read(ctx, "api-list", p)
}

func (s *projectLifecycleService) GetAPIV1WorkspacesBySlugProjectsByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByPkPayload) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.read(ctx, "api-detail", p)
}

func (s *projectLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDSummary(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDSummaryPayload) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.read(ctx, "summary", p)
}

func (s *projectLifecycleService) prepareWrite(ctx context.Context, slug, id, operation string, app bool) (moduleLifecyclePrincipal, error) {
	p, ok := ctx.Value(projectLifecyclePrincipalKey{}).(moduleLifecyclePrincipal)
	if !ok {
		return p, projectLifecycleError(401, map[string]any{"detail": moduleLifecycleUnauthenticatedDetail})
	}
	if app {
		if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
			return p, projectLifecycleError(400, map[string]any{"error": "The required key does not exist."})
		}
	}
	if err := s.repo.AuthorizeProjectWrite(ctx, p.userID.String(), slug, id, operation, app); err != nil {
		return p, projectLifecycleWriteRepoError(ctx, err, app)
	}
	if !app {
		rate, err := s.throttle.Allow(ctx, p.key)
		if err != nil {
			return p, projectLifecycleRepoError(ctx, err)
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return p, projectLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
		}
		if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
			return p, projectLifecycleError(400, map[string]any{"error": "The required key does not exist."})
		}
	}
	if err := s.repo.CheckProjectMutation(ctx, slug, id, operation, app); err != nil {
		return p, projectLifecycleWriteRepoError(ctx, err, app)
	}
	return p, nil
}
func projectLifecycleWriteRepoError(ctx context.Context, err error, app bool) error {
	if app && errors.Is(err, store.ErrNotFound) {
		return projectLifecycleError(404, map[string]any{"error": "The required object does not exist."})
	}
	return projectLifecycleRepoError(ctx, err)
}
func (s *projectLifecycleService) parseWrite(ctx context.Context, body io.ReadCloser) (djangoinput.Input, error) {
	contentType := "application/json"
	if meta, ok := loomhttp.RequestMetadataFromContext(ctx); ok && meta.Headers().Get("Content-Type") != "" {
		contentType = meta.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return input, projectLifecycleRepoError(ctx, err)
	}
	if input.Kind != "" || input.Null {
		return input, projectLifecycleError(400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(input.Kind) + "."}})
	}
	return input, nil
}
func projectLifecycleRawResult(raw json.RawMessage, err error) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	if err != nil {
		return nil, nil, err
	}
	// PostgreSQL JSON whitespace must not change DRF's rendered length and
	// therefore Django's 200-byte gzip/Vary threshold.
	var rendered bytes.Buffer
	if err := json.Compact(&rendered, raw); err != nil {
		return nil, nil, err
	}
	raw = rendered.Bytes()
	return &api.ProjectLifecycleRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(raw)), nil
}
func (s *projectLifecycleService) create(ctx context.Context, slug string, body io.ReadCloser, app bool) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	p, err := s.prepareWrite(ctx, slug, "", "create", app)
	if err != nil {
		return nil, nil, err
	}
	input, err := s.parseWrite(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	raw, err := s.repo.CreateProject(ctx, p.userID.String(), slug, input, app)
	if err != nil {
		var response *djangoapi.HTTPError
		if !app && !errors.As(err, &response) {
			moduleLifecycleLogError(ctx, err)
			err = projectLifecycleError(500, map[string]any{"error": "An unexpected error occurred"})
		} else {
			err = projectLifecycleWriteRepoError(ctx, err, app)
		}
	}
	return projectLifecycleRawResult(raw, err)
}
func (s *projectLifecycleService) update(ctx context.Context, slug, id string, body io.ReadCloser, app, replace bool) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	operation := "patch"
	if replace {
		operation = "put"
	}
	p, err := s.prepareWrite(ctx, slug, id, operation, app)
	if err != nil {
		return nil, nil, err
	}
	input, err := s.parseWrite(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	raw, err := s.repo.UpdateProject(ctx, p.userID.String(), slug, id, input, app, replace)
	if err != nil {
		err = projectLifecycleWriteRepoError(ctx, err, app)
	}
	return projectLifecycleRawResult(raw, err)
}
func (s *projectLifecycleService) delete(ctx context.Context, slug, id string, app bool) error {
	p, err := s.prepareWrite(ctx, slug, id, "delete", app)
	if err != nil {
		return err
	}
	return projectLifecycleWriteRepoErrorOrNil(ctx, s.repo.DeleteProject(ctx, p.userID.String(), slug, id, app), app)
}
func projectLifecycleWriteRepoErrorOrNil(ctx context.Context, err error, app bool) error {
	if err == nil {
		return nil
	}
	return projectLifecycleWriteRepoError(ctx, err, app)
}
func (s *projectLifecycleService) archive(ctx context.Context, slug, id string, app, archive bool) (time.Time, error) {
	operation := "restore"
	if archive {
		operation = "archive"
	}
	p, err := s.prepareWrite(ctx, slug, id, operation, app)
	if err != nil {
		return time.Time{}, err
	}
	at, err := s.repo.SetProjectArchived(ctx, p.userID.String(), slug, id, archive)
	return at, projectLifecycleWriteRepoErrorOrNil(ctx, err, app)
}
func (s *projectLifecycleService) PostAPIV1WorkspacesBySlugProjects(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsPayload, b io.ReadCloser) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.create(ctx, p.Slug, b, false)
}
func (s *projectLifecycleService) PatchAPIV1WorkspacesBySlugProjectsByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByPkPayload, b io.ReadCloser) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.update(ctx, p.Slug, p.Pk, b, false, false)
}
func (s *projectLifecycleService) PostAPIWorkspacesBySlugProjects(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsPayload, b io.ReadCloser) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.create(ctx, p.Slug, b, true)
}
func (s *projectLifecycleService) PatchAPIWorkspacesBySlugProjectsByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByPkPayload, b io.ReadCloser) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.update(ctx, p.Slug, p.Pk, b, true, false)
}
func (s *projectLifecycleService) PutAPIWorkspacesBySlugProjectsByPk(ctx context.Context, p *api.PutAPIWorkspacesBySlugProjectsByPkPayload, b io.ReadCloser) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	return s.update(ctx, p.Slug, p.Pk, b, true, true)
}
func (s *projectLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByPkPayload) error {
	return s.delete(ctx, p.Slug, p.Pk, false)
}
func (s *projectLifecycleService) DeleteAPIWorkspacesBySlugProjectsByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByPkPayload) error {
	return s.delete(ctx, p.Slug, p.Pk, true)
}
func (s *projectLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDArchive(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDArchivePayload) error {
	_, err := s.archive(ctx, p.Slug, p.ProjectID, false, true)
	return err
}
func (s *projectLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchive(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivePayload) error {
	_, err := s.archive(ctx, p.Slug, p.ProjectID, false, false)
	return err
}
func (s *projectLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDArchive(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivePayload) error {
	_, err := s.archive(ctx, p.Slug, p.ProjectID, true, false)
	return err
}
func (s *projectLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDArchive(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDArchivePayload) (*api.ProjectLifecycleRawJSON, io.ReadCloser, error) {
	at, err := s.archive(ctx, p.Slug, p.ProjectID, true, true)
	if err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(map[string]any{"archived_at": pythoncompat.PythonDateTimeString(at)})
	return projectLifecycleRawResult(raw, err)
}

func (s *projectLifecycleService) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	helper := moduleLifecycleService{sessions: s.sessions}
	return helper.AuthorizeSessionProtocol(ctx, key)
}
func (s *projectLifecycleService) AuthorizeSessionResourceProtocol(ctx context.Context, key string, _ SessionProtocolResource) (int, map[string]any) {
	return s.AuthorizeSessionProtocol(ctx, key)
}
func (s *projectLifecycleService) AuthorizeProjectLifecycleAPIProtocol(ctx context.Context, key string, resource SessionProtocolResource) (int, map[string]any) {
	authorized, err := s.APIKeyAuth(ctx, key, &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err == nil {
		p := authorized.Value(projectLifecyclePrincipalKey{}).(moduleLifecyclePrincipal)
		summary := strings.HasSuffix(resource.Pattern, "/summary/")
		safe := resource.Method == "GET" || resource.Method == "HEAD" || resource.Method == "OPTIONS"
		if summary || safe {
			err = s.repo.AuthorizeAPIProjectRead(authorized, storeuuid.MustParse(p.userID.String()), resource.Slug, summary)
		} else {
			operation := "patch"
			if resource.Method == "POST" {
				operation = "create"
			}
			project := resource.ProjectID
			if project == "" {
				project = resource.ID
			}
			err = s.repo.AuthorizeProjectWrite(authorized, p.userID.String(), resource.Slug, project, operation, false)
		}
		if err != nil {
			err = projectLifecycleRepoError(ctx, err)
		} else {
			rate, limitErr := s.throttle.Allow(authorized, p.key)
			if effects := apikey.EffectsFromContext(ctx); effects != nil {
				effects.Rate = &rate
			}
			if limitErr != nil {
				err = projectLifecycleRepoError(ctx, limitErr)
			} else if !rate.Allowed {
				err = projectLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
			} else if zoneErr := djangosession.ValidateUserTimezone(p.timezone); zoneErr != nil {
				err = projectLifecycleError(400, map[string]any{"error": "The required key does not exist."})
			}
		}
	}
	if err == nil {
		return 0, nil
	}
	var response *api.ProjectLifecycleErrorIdentity
	if errors.As(err, &response) {
		status, _ := strconv.Atoi(strings.TrimPrefix(response.Kind, "status_"))
		raw, _ := json.Marshal(response.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		return status, body
	}
	return 500, map[string]any{"error": "Something went wrong please try again later"}
}

func (s *projectLifecycleService) SessionProtocolResourceExists(ctx context.Context, resource SessionProtocolResource) (bool, error) {
	if resource.ID == "" {
		return true, nil
	}
	err := s.repo.CheckProjectMutation(ctx, resource.Slug, resource.ID, "metadata", true)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}
